package whatsapp

import (
	"context"
	"fmt"
	"os"
	"reflect"

	"github.com/mdp/qrterminal/v3"
	"github.com/tennex/bridge/db"
	backendGRPC "github.com/tennex/bridge/internal/grpc"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waCompanionReg"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"

	_ "github.com/lib/pq" // PostgreSQL driver
)

type WhatsAppConnector struct {
	storage       *db.Storage // Still needed for whatsmeow store
	backendClient *backendGRPC.BackendClient
}

func NewWhatsAppConnector(storage *db.Storage, backendClient *backendGRPC.BackendClient) *WhatsAppConnector {
	return &WhatsAppConnector{
		storage:       storage,
		backendClient: backendClient,
	}
}

type QRCodeData string

func (c *WhatsAppConnector) RunWhatsAppConnectionFlow(ctx context.Context, accountID string, callbackChan chan<- QRCodeData) error {
	fmt.Println("Starting WhatsApp connection flow...")

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

	client.AddEventHandler(eventHandler)

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
				if client.Store != nil && client.Store.ID != nil {
					jid = client.Store.ID.String()
				}
				fmt.Printf("\n🎉 QR scan successful! Session established.\n")
				fmt.Printf("👤 User ID: %s\n", accountID)
				fmt.Printf("📱 WhatsApp JID: %s\n", jid)

				// Notify backend about the WhatsApp connection via gRPC
				if err := c.backendClient.UpdateAccountStatus(ctx, accountID, jid, "", ""); err != nil {
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
			client.Disconnect()
		}
	}()

	return nil
}

func eventHandler(evt interface{}) {
	// Get event type name
	eventType := reflect.TypeOf(evt).String()

	// Print basic event info
	fmt.Printf("\n🔔 Event received: %s\n", eventType)

	// Handle specific event types
	switch v := evt.(type) {
	case *events.Message:
		msgID := "unknown"
		if v.Info.ID != "" {
			msgID = v.Info.ID
		}
		conversation := v.Message.GetConversation()
		sender := v.Info.Sender.String()

		fmt.Printf("📨 Message ID: %s\n", msgID)
		fmt.Printf("👤 From: %s\n", sender)
		fmt.Printf("💬 Content: %s\n", conversation)

	case *events.Receipt:
		fmt.Printf("✅ Receipt: %s for message %s\n", v.Type, v.MessageIDs)

	case *events.Presence:
		fmt.Printf("👁️  Presence: %s is %s\n", v.From.String(), v.LastSeen.String())

	case *events.ChatPresence:
		fmt.Printf("👥 Chat presence: %s in %s\n", v.State, v.Chat.String())

	case *events.HistorySync:
		fmt.Printf("🔄 History sync: %s (%d conversations)\n", v.Data.SyncType, len(v.Data.Conversations))

	case *events.AppStateSyncComplete:
		fmt.Printf("🔄 App state sync complete: %s\n", v.Name)

	case *events.Connected:
		fmt.Printf("🔗 Connected to WhatsApp!\n")

	case *events.Disconnected:
		fmt.Printf("❌ Disconnected from WhatsApp\n")

	case *events.LoggedOut:
		fmt.Printf("👋 Logged out from WhatsApp\n")

	default:
		fmt.Printf("❓ Unknown event type: %s\n", eventType)
	}

	fmt.Printf("───────────────────────────────────────\n")
}
