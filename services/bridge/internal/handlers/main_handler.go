package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	api "github.com/tennex/bridge/api/gen"
	"github.com/tennex/bridge/db"
	"github.com/tennex/shared/auth"
)

type MainHandler struct {
	storage         *db.Storage
	whatsappHandler *WhatsAppHandler
	jwtConfig       *auth.JWTConfig
}

func NewMainHandler(storage *db.Storage, whatsappHandler *WhatsAppHandler, jwtConfig *auth.JWTConfig) *MainHandler {
	return &MainHandler{
		storage:         storage,
		whatsappHandler: whatsappHandler,
		jwtConfig:       jwtConfig,
	}
}

// Routes sets up all bridge service routes
func (h *MainHandler) Routes() chi.Router {
	r := chi.NewRouter()

	// Public routes (no auth required)
	r.Get("/health", h.GetHealth)

	// Protected routes (JWT required)
	r.Route("/", func(r chi.Router) {
		// Apply JWT authentication middleware
		r.Use(h.jwtConfig.ChiMiddleware())

		// Mount WhatsApp routes
		r.Mount("/whatsapp", h.whatsappHandler.Routes())

		// General connection management
		r.Get("/connections", h.ListConnections)
	})

	return r
}

// GetHealth implements GET /health
func (h *MainHandler) GetHealth(w http.ResponseWriter, r *http.Request) {
	response := api.HealthResponse{
		Status:    "ok",
		Timestamp: time.Now(),
		Version:   stringPtr("1.0.0"),
	}

	h.writeJSON(w, http.StatusOK, response)
}

// ListConnections implements GET /connections
func (h *MainHandler) ListConnections(w http.ResponseWriter, r *http.Request) {
	// Get authenticated user ID
	userID, err := auth.GetUserIDFromContext(r.Context())
	if err != nil {
		h.writeError(w, http.StatusUnauthorized, "authentication_required", "User must be authenticated", nil)
		return
	}

	// For now, just return WhatsApp connection info
	// In the future, this could call the backend to get full connection details
	var apiConnections []api.Connection

	whatsappConn := api.Connection{
		Platform:  "whatsapp",
		Connected: false, // Default to false, would need backend call to get actual status
		UserId:    userID,
	}

	apiConnections = append(apiConnections, whatsappConn)

	response := api.ConnectionsResponse{
		Connections: apiConnections,
		UserId:      userID,
		Total:       intPtr(len(apiConnections)),
	}

	h.writeJSON(w, http.StatusOK, response)
}

// Helper functions
func (h *MainHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *MainHandler) writeError(w http.ResponseWriter, status int, code, message string, details map[string]interface{}) {
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

func getPlatformFromIntegrationID(integrationID string) api.ConnectionPlatform {
	switch integrationID {
	case "whatsapp":
		return api.Whatsapp
	case "telegram":
		return api.Telegram
	case "discord":
		return api.Discord
	default:
		return api.Whatsapp // Default fallback
	}
}

func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}
