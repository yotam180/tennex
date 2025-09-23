package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/tennex/bridge/db"
	"github.com/tennex/bridge/whatsapp"
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

	fmt.Println("🚀 Starting Tennex WhatsApp Bridge HTTP Server...")

	storage, err := db.NewStorage()
	if err != nil {
		fmt.Printf("❌ Failed to create storage: %v\n", err)
		os.Exit(1)
	}

	whatsappConnector := whatsapp.NewWhatsAppConnector(storage)

	gin.SetMode(gin.DebugMode)
	router := gin.Default()
	router.POST("/connect", func(c *gin.Context) {
		createWhatsappConnection(c, whatsappConnector, ctx)
	})

	if err := router.Run(":8081"); err != nil {
		fmt.Printf("❌ Failed to start HTTP server: %v\n", err)
		os.Exit(1)
	}
}

func createWhatsappConnection(c *gin.Context, whatsappConnector *whatsapp.WhatsAppConnector, ctx context.Context) {
	accountID := c.Query("account_id")
	if accountID == "" {
		accountID = "sample-account-id" // TODO: Remove this logic
	}

	qrChan := make(chan whatsapp.QRCodeData, 1)

	if err := whatsappConnector.RunWhatsAppConnectionFlow(ctx, accountID, qrChan); err != nil {
		fmt.Printf("❌ WhatsApp connection failed: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("WhatsApp connection failed: %v", err),
		})
		return
	}

	qrCode := <-qrChan

	c.JSON(http.StatusOK, gin.H{
		"qr_code": qrCode,
	})
}
