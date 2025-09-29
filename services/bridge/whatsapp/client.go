package whatsapp

import (
	"context"
	"fmt"
	"os"

	"github.com/mdp/qrterminal/v3"
	"github.com/tennex/bridge/db"
	backendGRPC "github.com/tennex/bridge/internal/grpc"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waCompanionReg"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"

	_ "github.com/lib/pq" // PostgreSQL driver
)

type WhatsAppConnector struct {
	storage           *db.Storage // Still needed for whatsmeow store
	backendClient     *backendGRPC.BackendClient
	integrationClient *backendGRPC.RecordingIntegrationClient
	eventsProcessor   *EventsProcessor
}

func NewWhatsAppConnector(storage *db.Storage, backendClient *backendGRPC.BackendClient, integrationClient *backendGRPC.RecordingIntegrationClient) *WhatsAppConnector {
	return &WhatsAppConnector{
		storage:           storage,
		backendClient:     backendClient,
		integrationClient: integrationClient,
	}
}

type QRCodeData string

func (c *WhatsAppConnector) RunWhatsAppConnectionFlow(ctx context.Context, accountID string, callbackChan chan<- QRCodeData) error {
	fmt.Println("Starting WhatsApp connection flow...")

	// Create events processor for this connection
	c.eventsProcessor = NewEventsProcessor(c.integrationClient, c.backendClient, accountID)

	dsn := db.GetConnectionString()
	dbLogger := waLog.Stdout("whatsapp", "DEBUG", true)

	container, err := sqlstore.New(ctx, "postgres", dsn, dbLogger)
	if err != nil {
		return fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	store.DeviceProps.Os = proto.String("Tennex")
	store.DeviceProps.PlatformType = waCompanionReg.DeviceProps_DESKTOP.Enum()
	store.DeviceProps.RequireFullSync = proto.Bool(false)

	device := container.NewDevice()
	client := whatsmeow.NewClient(device, dbLogger)

	// Use the events processor instead of the generic event handler
	client.AddEventHandler(func(evt interface{}) {
		c.eventsProcessor.ProcessEvent(ctx, evt)
	})

	qrChan, err := client.GetQRChannel(ctx)
	if err != nil {
		return fmt.Errorf("failed to get QR channel: %w", err)
	}

	if err := client.Connect(); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	go func() {
		fmt.Printf("🔄 [WA CLIENT DEBUG] Starting QR handler goroutine\n")
		defer fmt.Printf("🔄 [WA CLIENT DEBUG] QR handler goroutine exiting\n")

		qrHandled := false

		// Handle QR events
		for evt := range qrChan {
			fmt.Printf("🔄 [WA CLIENT DEBUG] Received QR event: %s\n", evt.Event)

			switch evt.Event {
			case "code":
				fmt.Println("\nScan this QR with WhatsApp:")
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
				fmt.Println()
				fmt.Println("(If it expires, just run again.)")

				callbackChan <- QRCodeData(evt.Code)

			case "success":
				jid := ""
				displayName := ""
				avatarURL := ""

				if client.Store != nil && client.Store.ID != nil {
					jid = client.Store.ID.String()
				}

				fmt.Printf("\n🎉 QR scan successful! Session established.\n")
				fmt.Printf("👤 User ID: %s\n", accountID)
				fmt.Printf("📱 WhatsApp JID: %s\n", jid)

				// Start recording session if recording mode is enabled
				if err := c.integrationClient.StartRecordingSession(accountID, "whatsapp"); err != nil {
					fmt.Printf("⚠️  Failed to start recording session: %v\n", err)
				}

				// Create user integration in backend
				userIntegrationID, err := c.integrationClient.CreateUserIntegration(
					ctx,
					accountID,
					jid,
					displayName,
					avatarURL,
					map[string]string{
						"device_id":     device.ID.String(),
						"platform_type": "desktop",
					},
				)
				if err != nil {
					fmt.Printf("❌ Failed to create user integration: %v\n", err)
					// Continue anyway - don't fail the entire flow for this
				} else {
					fmt.Printf("✅ User integration created: ID=%d\n", userIntegrationID)

					// Set integration context in events processor
					c.eventsProcessor.SetIntegrationContext(userIntegrationID, jid)
				}

				// Also notify backend about connection via old bridge service (for compatibility)
				if err := c.backendClient.UpdateAccountStatus(ctx, accountID, jid, displayName, avatarURL); err != nil {
					fmt.Printf("❌ Failed to notify backend of WhatsApp connection: %v\n", err)
					// Continue anyway - don't fail the entire flow for this
				} else {
					fmt.Printf("✅ Backend notified of WhatsApp connection!\n")
				}
				qrHandled = true
			}
		}

		fmt.Printf("🔄 [WA CLIENT DEBUG] QR channel closed, qrHandled=%v\n", qrHandled)

		// Keep connection alive if QR was successfully handled
		if qrHandled {
			fmt.Printf("🔄 [WA CLIENT DEBUG] Keeping WhatsApp connection alive...\n")
			// Keep the client connected and handle events
			<-ctx.Done()
			fmt.Printf("🔄 [WA CLIENT DEBUG] Context cancelled, disconnecting WhatsApp client\n")

			// End recording session before disconnecting
			if err := c.integrationClient.EndRecordingSession(); err != nil {
				fmt.Printf("⚠️  Failed to end recording session: %v\n", err)
			}

			client.Disconnect()
		}
	}()

	return nil
}
