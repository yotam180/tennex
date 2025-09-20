package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	qrterminal "github.com/mdp/qrterminal/v3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"

	_ "github.com/mattn/go-sqlite3"
)

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

	device, err := container.GetFirstDevice(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get device store: %v\n", err)
		os.Exit(1)
	}

	client := whatsmeow.NewClient(device, nil)

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

	// Timeout guard for QR flow
	timeoutCtx, cancelTimeout := context.WithTimeout(ctx, 2*time.Minute)
	defer cancelTimeout()

	for {
		select {
		case evt := <-qrChan:
			switch evt.Event {
			case "code":
				fmt.Println("\nScan this QR with WhatsApp:")
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
				fmt.Println()
				fmt.Println("(If it expires, just run again.)")
			case "timeout":
				fmt.Fprintln(os.Stderr, "QR timed out. Re-run to retry.")
				client.Disconnect()
				os.Exit(2)
			case "success":
				jid := ""
				if client.Store != nil && client.Store.ID != nil {
					jid = client.Store.ID.String()
				}
				fmt.Printf("\nQR scan successful. Session established. JID: %s\n", jid)
				client.Disconnect()
				os.Exit(0)
			default:
				// ignore other events
			}
		case <-timeoutCtx.Done():
			fmt.Fprintln(os.Stderr, "Timed out waiting for QR scan. Exiting.")
			client.Disconnect()
			os.Exit(3)
		}
	}
}
