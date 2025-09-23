package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
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

	// Setup Gin router
	gin.SetMode(gin.ReleaseMode) // Reduce logging noise
	router := gin.Default()

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "tennex-whatsapp-bridge",
		})
	})

	// WhatsApp connection endpoint
	router.POST("/connect", func(c *gin.Context) {
		fmt.Println("📱 WhatsApp connection request received...")

		// Call our whatsapp connection function
		if err := whatsapp.ConnectToWhatsApp(ctx); err != nil {
			fmt.Printf("❌ WhatsApp connection failed: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("WhatsApp connection failed: %v", err),
			})
			return
		}

		fmt.Println("🎉 WhatsApp connection completed successfully!")
		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"message": "WhatsApp connection completed successfully",
		})
	})

	// Start HTTP server
	port := ":8081"
	fmt.Printf("🌐 HTTP Server starting on port %s\n", port)
	fmt.Printf("📋 Endpoints:\n")
	fmt.Printf("   GET  /health  - Health check\n")
	fmt.Printf("   POST /connect - Start WhatsApp connection\n")
	fmt.Printf("\n💡 To connect WhatsApp: curl -X POST http://localhost:8080/connect\n")

	if err := router.Run(port); err != nil {
		fmt.Printf("❌ Failed to start HTTP server: %v\n", err)
		os.Exit(1)
	}
}
