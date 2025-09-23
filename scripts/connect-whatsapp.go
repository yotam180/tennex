package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/mdp/qrterminal/v3"
)

// ConnectRequest represents the request to the bridge service
type ConnectRequest struct {
	ClientID string `json:"client_id"`
}

// ConnectResponse represents the response from the bridge service
type ConnectResponse struct {
	SessionID string `json:"session_id"`
	QRCode    string `json:"qr_code"`
	Status    string `json:"status"`
	ExpiresAt string `json:"expires_at"`
}

// StatsResponse represents the stats endpoint response
type StatsResponse struct {
	ActiveClients int `json:"active_clients,omitempty"`
}

// Config holds the application configuration
type Config struct {
	BridgeURL string
	ClientID  string
	Monitor   bool
}

func main() {
	// Parse command line flags
	config := parseFlags()

	fmt.Printf("🚀 Tennex WhatsApp Connection Script\n")
	fmt.Printf("📱 Client ID: %s\n", config.ClientID)
	fmt.Printf("🌐 Bridge URL: %s\n", config.BridgeURL)
	fmt.Println()

	// Test bridge connection
	if err := testBridgeConnection(config.BridgeURL); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Bridge connection failed: %v\n", err)
		fmt.Fprintf(os.Stderr, "💡 Make sure the bridge service is running:\n")
		fmt.Fprintf(os.Stderr, "   - Docker: ./scripts/dev-start.sh\n")
		fmt.Fprintf(os.Stderr, "   - Local: cd services/bridge && go run ./cmd/bridge\n")
		os.Exit(1)
	}

	fmt.Println("✅ Bridge service is accessible")

	// Connect client and get QR code
	response, err := connectClient(config.BridgeURL, config.ClientID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to connect client: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Client connection request successful\n")
	fmt.Printf("📋 Session ID: %s\n", response.SessionID)
	fmt.Printf("📊 Status: %s\n", response.Status)
	fmt.Printf("⏰ Expires: %s\n", response.ExpiresAt)
	fmt.Println()

	// Display QR code using the same qrterminal package as test/main.go
	displayQRCode(response.QRCode)

	// Monitor connection if requested
	if config.Monitor {
		fmt.Println()
		monitorConnection(config.BridgeURL, response.SessionID)
	} else {
		fmt.Println()
		fmt.Printf("📊 To monitor connection status, run with -monitor flag\n")
		fmt.Printf("📈 Check service stats: curl %s/debug/clients | jq\n", config.BridgeURL)
	}

	fmt.Println("🎉 Script completed successfully!")
}

// parseFlags parses command line arguments
func parseFlags() Config {
	var config Config

	flag.StringVar(&config.BridgeURL, "url", getEnvOrDefault("BRIDGE_URL", "http://localhost:8080"), "Bridge service URL")
	flag.StringVar(&config.ClientID, "client", getEnvOrDefault("CLIENT_ID", fmt.Sprintf("whatsapp-%d", time.Now().Unix())), "Client ID")
	flag.BoolVar(&config.Monitor, "monitor", false, "Monitor connection status after QR display")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s                                    # Connect with default settings\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -client my-phone -monitor         # Connect with custom ID and monitor\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -url http://remote:8080           # Connect to remote bridge\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
		fmt.Fprintf(os.Stderr, "  CLIENT_ID             Custom client identifier\n")
		fmt.Fprintf(os.Stderr, "  BRIDGE_URL            Bridge service URL\n")
	}

	flag.Parse()
	return config
}

// getEnvOrDefault returns environment variable value or default
func getEnvOrDefault(envVar, defaultValue string) string {
	if value := os.Getenv(envVar); value != "" {
		return value
	}
	return defaultValue
}

// testBridgeConnection tests if the bridge service is accessible
func testBridgeConnection(bridgeURL string) error {
	resp, err := http.Get(bridgeURL + "/health")
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed with status: %d", resp.StatusCode)
	}

	return nil
}

// connectClient sends a connection request to the bridge service
func connectClient(bridgeURL, clientID string) (*ConnectResponse, error) {
	reqBody := ConnectRequest{
		ClientID: clientID,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := http.Post(bridgeURL+"/connect-minimal", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var response ConnectResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response, nil
}

// displayQRCode displays the QR code using qrterminal exactly like test/main.go
func displayQRCode(qrData string) {
	fmt.Println("📱 Scan this QR code with WhatsApp:")
	fmt.Println()

	// Use exactly the same qrterminal settings as test/main.go
	qrterminal.GenerateHalfBlock(qrData, qrterminal.L, os.Stdout)

	fmt.Println()
	fmt.Println("📲 How to scan:")
	fmt.Println("   1. Open WhatsApp on your phone")
	fmt.Println("   2. Settings → Linked Devices")
	fmt.Println("   3. Link a Device → Scan QR")
	fmt.Println("   4. Point camera at the QR code above")
	fmt.Println()
	fmt.Println("💡 If QR expires, just run this script again")
}

// monitorConnection monitors the connection status
func monitorConnection(bridgeURL, sessionID string) {
	fmt.Printf("👀 Monitoring connection status for session: %s\n", sessionID)
	fmt.Println("⏹️  Press Ctrl+C to stop monitoring")
	fmt.Println()

	checkCount := 0
	for {
		checkCount++

		// Check debug/clients endpoint to see active clients
		resp, err := http.Get(bridgeURL + "/debug/clients")
		if err == nil {
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				body, err := io.ReadAll(resp.Body)
				if err == nil {
					var debugResp map[string]interface{}
					if json.Unmarshal(body, &debugResp) == nil {
						if totalCount, ok := debugResp["total_count"].(float64); ok && totalCount > 0 {
							fmt.Printf("\n🎉 WhatsApp client connected! Active clients: %.0f\n", totalCount)
							fmt.Println("✅ You can now use the WhatsApp bridge API")

							// Show client details if available
							if clients, ok := debugResp["active_clients"].([]interface{}); ok && len(clients) > 0 {
								for _, client := range clients {
									if clientMap, ok := client.(map[string]interface{}); ok {
										if jid, exists := clientMap["jid"].(string); exists && jid != "" {
											fmt.Printf("📞 JID: %s\n", jid)
										}
										if uptime, exists := clientMap["uptime"].(string); exists {
											fmt.Printf("⏱️  Uptime: %s\n", uptime)
										}
									}
								}
							}

							return
						}
					}
				}
			}
		}

		fmt.Printf("\r⏳ Waiting for WhatsApp scan... (check %d)", checkCount)

		time.Sleep(2 * time.Second)

		// Timeout after 5 minutes
		if checkCount > 150 {
			fmt.Println()
			fmt.Println("⏰ Timeout waiting for WhatsApp connection")
			fmt.Println("💡 QR code may have expired. Try running the script again.")
			return
		}
	}
}
