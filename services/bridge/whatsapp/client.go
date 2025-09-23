package whatsapp

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"reflect"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/mdp/qrterminal/v3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waCompanionReg"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"

	_ "github.com/lib/pq" // PostgreSQL driver
)

var (
	sessionStartTime string
	eventCounter     int64
)

func init() {
	// Initialize session timestamp for this run
	sessionStartTime = time.Now().Format("02_01_06_15_04_05") // DD_MM_YY_HH_MM_SS
}

func saveEventToBinary(evt interface{}) {
	// Create event_history directory if it doesn't exist
	eventDir := filepath.Join("event_history", sessionStartTime)
	if err := os.MkdirAll(eventDir, 0755); err != nil {
		fmt.Printf("⚠️  Failed to create event directory: %v\n", err)
		return
	}

	// Increment counter and get file number
	fileNum := atomic.AddInt64(&eventCounter, 1)
	filename := fmt.Sprintf("%d.bin", fileNum)
	filepath := filepath.Join(eventDir, filename)

	// Try to serialize the event to binary
	var data []byte
	var err error

	// Check if it's a protobuf message
	if protoMsg, ok := evt.(proto.Message); ok {
		data, err = proto.Marshal(protoMsg)
	} else {
		// For non-protobuf events, we'll skip binary serialization
		fmt.Printf("🔍 Event #%d: Non-protobuf event, skipping binary save\n", fileNum)
		return
	}

	if err != nil {
		fmt.Printf("⚠️  Failed to marshal event #%d: %v\n", fileNum, err)
		return
	}

	// Save to file
	if err := os.WriteFile(filepath, data, 0644); err != nil {
		fmt.Printf("⚠️  Failed to save event #%d to file: %v\n", fileNum, err)
		return
	}

	fmt.Printf("💾 Saved event #%d to %s (%d bytes)\n", fileNum, filepath, len(data))
}

func eventHandler(evt interface{}) {
	// Get event type name
	eventType := reflect.TypeOf(evt).String()

	// Print basic event info
	fmt.Printf("\n🔔 Event received: %s\n", eventType)

	// Save event to binary file
	saveEventToBinary(evt)

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

// ConnectToWhatsApp creates a WhatsApp client and handles QR code scanning
// This is copied directly from the working test/main.go
func ConnectToWhatsApp(ctx context.Context) error {
	fmt.Println("Starting whatsmeow QR PoC (prints QR to terminal, exits on success)...")

	// Use PostgreSQL database from docker-compose
	dsn := "postgres://tennex:tennex123@localhost:5432/tennex?sslmode=disable"
	dbLogger := waLog.Noop
	container, err := sqlstore.New(ctx, "postgres", dsn, dbLogger)
	if err != nil {
		return fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	// Set device properties exactly like test/main.go
	store.DeviceProps.Os = proto.String("Temple OS")
	store.DeviceProps.PlatformType = waCompanionReg.DeviceProps_DESKTOP.Enum()
	store.DeviceProps.RequireFullSync = proto.Bool(true)

	device, err := container.GetFirstDevice(ctx)
	if err != nil {
		return fmt.Errorf("failed to get device store: %w", err)
	}

	// Patch for debug
	if device.ID != nil {
		fmt.Println("Deleting device")
		device.Delete(ctx)

		fmt.Println("Getting new device")
		device, err = container.GetFirstDevice(ctx)
		if err != nil {
			return fmt.Errorf("failed to get device store: %w", err)
		}
	}

	client := whatsmeow.NewClient(device, nil)
	client.AddEventHandler(eventHandler)

	// Get QR channel BEFORE connect, as per whatsmeow pattern
	qrChan, err := client.GetQRChannel(ctx)
	if err != nil {
		return fmt.Errorf("failed to get QR channel: %w", err)
	}

	// Connect to start QR generation
	if err := client.Connect(); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	for evt := range qrChan {
		switch evt.Event {
		case "code":
			fmt.Println("\nScan this QR with WhatsApp:")
			qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
			fmt.Println()
			fmt.Println("(If it expires, just run again.)")
		case "timeout":
			return fmt.Errorf("QR timed out. Re-run to retry")
		case "success":
			jid := ""
			if client.Store != nil && client.Store.ID != nil {
				jid = client.Store.ID.String()
			}
			fmt.Printf("\nQR scan successful. Session established. JID: %s\n", jid)

			// In the test, it would block here waiting for signals
			// For now, we'll just return success
			fmt.Println("✅ WhatsApp connection successful!")
		default:
			// ignore other events
		}
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	return nil
}
