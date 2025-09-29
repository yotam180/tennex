package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	api "github.com/tennex/bridge/api/gen"
	"github.com/tennex/bridge/db"
	backendGRPC "github.com/tennex/bridge/internal/grpc"
	"github.com/tennex/bridge/whatsapp"
	"github.com/tennex/shared/auth"
)

type WhatsAppHandler struct {
	storage           *db.Storage
	whatsappConnector *whatsapp.WhatsAppConnector
	backendClient     *backendGRPC.BackendClient
	integrationClient *backendGRPC.IntegrationClient
}

func NewWhatsAppHandler(storage *db.Storage, whatsappConnector *whatsapp.WhatsAppConnector, backendClient *backendGRPC.BackendClient, integrationClient *backendGRPC.IntegrationClient) *WhatsAppHandler {
	return &WhatsAppHandler{
		storage:           storage,
		whatsappConnector: whatsappConnector,
		backendClient:     backendClient,
		integrationClient: integrationClient,
	}
}

// Routes sets up WhatsApp-specific routes
func (h *WhatsAppHandler) Routes() chi.Router {
	r := chi.NewRouter()

	// All WhatsApp routes require authentication
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Printf("🔵 [WA HANDLER DEBUG] Processing WhatsApp route: %s %s\n", r.Method, r.URL.Path)

			// Ensure user is authenticated (JWT middleware should have run first)
			userID, err := auth.GetUserIDFromContext(r.Context())
			if err != nil {
				log.Printf("🔵 [WA HANDLER DEBUG] ❌ Failed to get user ID from context: %v\n", err)
				log.Printf("🔵 [WA HANDLER DEBUG] ❌ Context keys available: %+v\n", r.Context())
				h.writeError(w, http.StatusUnauthorized, "authentication_required", "User must be authenticated", nil)
				return
			}

			log.Printf("🔵 [WA HANDLER DEBUG] ✅ User authenticated successfully: %s\n", userID.String())

			// Add user ID to context for convenience
			ctx := context.WithValue(r.Context(), "user_id", userID.String())
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})

	r.Post("/connect", h.ConnectWhatsApp)
	r.Get("/status", h.GetWhatsAppStatus)
	r.Post("/disconnect", h.DisconnectWhatsApp)

	return r
}

// ConnectWhatsApp implements POST /whatsapp/connect
func (h *WhatsAppHandler) ConnectWhatsApp(w http.ResponseWriter, r *http.Request) {
	// Get authenticated user ID
	userIDStr, ok := r.Context().Value("user_id").(string)
	if !ok {
		h.writeError(w, http.StatusInternalServerError, "context_error", "Failed to get user ID from context", nil)
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_user_id", "Invalid user ID format", nil)
		return
	}

	fmt.Printf("🔐 User %s requesting WhatsApp connection\n", userID)

	// Create QR channel for this connection attempt
	qrChan := make(chan whatsapp.QRCodeData, 1)
	sessionID := uuid.New()

	fmt.Printf("📱 Starting WhatsApp connection flow for user %s (session: %s)\n", userID, sessionID)

	// Start WhatsApp connection flow in background
	// Use background context so connection survives HTTP request completion
	connCtx := context.Background()
	go func() {
		fmt.Printf("🚀 [WA DEBUG] Starting WhatsApp connection with background context\n")
		if err := h.whatsappConnector.RunWhatsAppConnectionFlow(connCtx, userIDStr, qrChan); err != nil {
			fmt.Printf("❌ WhatsApp connection failed for user %s: %v\n", userID, err)
		}
		fmt.Printf("🔚 [WA DEBUG] WhatsApp connection flow completed for user %s\n", userID)
	}()

	// Wait for QR code (with timeout)
	select {
	case qrCode := <-qrChan:
		fmt.Printf("📲 QR code generated for user %s\n", userID)

		response := api.WhatsAppConnectResponse{
			QrCode:       string(qrCode),
			SessionId:    sessionID,
			ExpiresAt:    timePtr(time.Now().Add(2 * time.Minute)), // QR codes typically expire quickly
			Instructions: stringPtr("Open WhatsApp on your phone, tap Menu > Linked Devices > Link a Device, and scan this QR code"),
		}

		h.writeJSON(w, http.StatusOK, response)

	case <-time.After(30 * time.Second):
		fmt.Printf("⏰ QR code generation timeout for user %s\n", userID)
		h.writeError(w, http.StatusRequestTimeout, "qr_timeout", "QR code generation timed out", nil)

	case <-r.Context().Done():
		fmt.Printf("🚫 Request cancelled for user %s\n", userID)
		h.writeError(w, http.StatusRequestTimeout, "request_cancelled", "Request was cancelled", nil)
	}
}

// GetWhatsAppStatus implements GET /whatsapp/status
func (h *WhatsAppHandler) GetWhatsAppStatus(w http.ResponseWriter, r *http.Request) {
	// Get authenticated user ID
	userIDStr, ok := r.Context().Value("user_id").(string)
	if !ok {
		h.writeError(w, http.StatusInternalServerError, "context_error", "Failed to get user ID from context", nil)
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_user_id", "Invalid user ID format", nil)
		return
	}

	// TODO: Check WhatsApp connection status from backend
	// For now, returning disconnected status - should call backend to get actual status
	response := api.WhatsAppStatusResponse{
		Connected: false,
		UserId:    userID,
	}

	fmt.Printf("📊 WhatsApp status for user %s: connected=%v\n", userID, response.Connected)
	h.writeJSON(w, http.StatusOK, response)
}

// DisconnectWhatsApp implements POST /whatsapp/disconnect
func (h *WhatsAppHandler) DisconnectWhatsApp(w http.ResponseWriter, r *http.Request) {
	// Get authenticated user ID
	userIDStr, ok := r.Context().Value("user_id").(string)
	if !ok {
		h.writeError(w, http.StatusInternalServerError, "context_error", "Failed to get user ID from context", nil)
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_user_id", "Invalid user ID format", nil)
		return
	}

	// Notify backend about WhatsApp disconnection
	if err := h.backendClient.UpdateAccountDisconnected(r.Context(), userIDStr); err != nil {
		fmt.Printf("⚠️  Failed to notify backend of WhatsApp disconnection: %v\n", err)
		// Continue anyway - don't fail the API call for this
	}

	fmt.Printf("🔌 WhatsApp disconnected for user %s\n", userID)

	response := api.SuccessResponse{
		Success:   true,
		Message:   "WhatsApp disconnected successfully",
		Timestamp: timePtr(time.Now()),
	}

	h.writeJSON(w, http.StatusOK, response)
}

// Helper functions
func (h *WhatsAppHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *WhatsAppHandler) writeError(w http.ResponseWriter, status int, code, message string, details map[string]interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	response := api.ErrorResponse{
		Error:     message,
		Code:      &code,
		Details:   &details,
		Timestamp: time.Now(),
	}

	json.NewEncoder(w).Encode(response)
}

func timePtr(t time.Time) *time.Time {
	return &t
}
