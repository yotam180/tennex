package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	qrterminal "github.com/mdp/qrterminal/v3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waCompanionReg"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"

	_ "github.com/mattn/go-sqlite3"
)

func eventHandler(evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		fmt.Println("Received a message!", v.Message.GetConversation())
	}
}

func main() {
	// Context with cancel for the whole run
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle OS signals for graceful exit
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nInterrupted. Exiting.")
		cancel()
		os.Exit(1)
	}()

	fmt.Println("Starting whatsmeow QR PoC (prints QR to terminal, exits on success)...")

	// Prepare a local SQLite-backed store under ./test
	dsn := "file:session.db?_foreign_keys=on"
	dbLogger := waLog.Noop
	container, err := sqlstore.New(ctx, "sqlite3", dsn, dbLogger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create session container: %v\n", err)
		os.Exit(1)
	}

	store.DeviceProps.Os = proto.String("Temple OS")
	store.DeviceProps.PlatformType = waCompanionReg.DeviceProps_CHROME.Enum()
	store.DeviceProps.RequireFullSync = proto.Bool(true)

	device, err := container.GetFirstDevice(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get device store: %v\n", err)
		os.Exit(1)
	}

	client := whatsmeow.NewClient(device, nil)
	client.AddEventHandler(eventHandler)

	// Get QR channel BEFORE connect, as per whatsmeow pattern
	qrChan, err := client.GetQRChannel(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get QR channel: %v\n", err)
		os.Exit(1)
	}

	// Connect to start QR generation
	if err := client.Connect(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect: %v\n", err)
		os.Exit(1)
	}

	for evt := range qrChan {
		switch evt.Event {
		case "code":
			fmt.Println("\nScan this QR with WhatsApp:")
			qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
			fmt.Println()
			fmt.Println("(If it expires, just run again.)")
		case "timeout":
			fmt.Fprintln(os.Stderr, "QR timed out. Re-run to retry.")
		case "success":
			jid := ""
			if client.Store != nil && client.Store.ID != nil {
				jid = client.Store.ID.String()
			}
			fmt.Printf("\nQR scan successful. Session established. JID: %s\n", jid)
		default:
			// ignore other events
		}
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
}
