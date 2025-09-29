package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/tennex/backend/internal/core"
	"github.com/tennex/backend/internal/repo"
	dbgen "github.com/tennex/pkg/db/gen"
	"github.com/tennex/pkg/events"
	"github.com/tennex/shared/auth"
)

// APIHandler handles HTTP API requests
type APIHandler struct {
	eventService       *core.EventService
	outboxService      *core.OutboxService
	accountService     *core.AccountService
	integrationService *core.IntegrationService
	queries            *dbgen.Queries
	authHandler        *AuthHandler
	jwtConfig          *auth.JWTConfig
	logger             *zap.Logger
}

// NewAPIHandler creates a new API handler
func NewAPIHandler(eventService *core.EventService, outboxService *core.OutboxService, accountService *core.AccountService, integrationService *core.IntegrationService, queries *dbgen.Queries, jwtSecret string, logger *zap.Logger) *APIHandler {
	authHandler := NewAuthHandler(queries, jwtSecret, logger)
	jwtConfig := auth.DefaultJWTConfig(jwtSecret)

	return &APIHandler{
		eventService:       eventService,
		outboxService:      outboxService,
		accountService:     accountService,
		integrationService: integrationService,
		queries:            queries,
		authHandler:        authHandler,
		jwtConfig:          jwtConfig,
		logger:             logger.Named("api_handler"),
	}
}

// Routes returns the HTTP routes
func (h *APIHandler) Routes() chi.Router {
	r := chi.NewRouter()

	// Public routes
	r.Get("/health", h.GetHealth)

	// Authentication routes
	r.Mount("/auth", h.authHandler.Routes())

	// Protected routes (in a real app, you'd add JWT middleware here)
	r.Post("/outbox", h.CreateOutboxMessage)
	r.Get("/sync", h.SyncEvents)
	r.Get("/qr", h.GetQRCode)
	r.Get("/accounts", h.ListAccounts)
	r.Get("/accounts/{account_id}", h.GetAccount)
	r.Get("/settings", h.GetSettings)

	// Data sync endpoints
	r.Get("/sync/conversations/{integration_id}", h.SyncConversations)
	r.Get("/sync/messages/{integration_id}", h.SyncMessages)
	r.Get("/sync/contacts/{integration_id}", h.SyncContacts)
	r.Get("/sync/status/{integration_id}", h.GetSyncStatus)

	return r
}

// GetHealth handles health check requests
func (h *APIHandler) GetHealth(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now().UTC(),
		"version":   "1.0.0",
	}

	h.writeJSON(w, http.StatusOK, response)
}

// CreateOutboxMessage handles message sending requests
func (h *APIHandler) CreateOutboxMessage(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ClientMsgUUID string                 `json:"client_msg_uuid"`
		AccountID     string                 `json:"account_id"`
		ConvoID       string                 `json:"convo_id"`
		MessageType   string                 `json:"message_type"`
		Content       map[string]interface{} `json:"content"`
		ReplyTo       string                 `json:"reply_to,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid JSON", err)
		return
	}

	// Validate required fields
	if req.ClientMsgUUID == "" || req.AccountID == "" || req.ConvoID == "" || req.MessageType == "" {
		h.writeError(w, http.StatusBadRequest, "Missing required fields", nil)
		return
	}

	clientUUID, err := uuid.Parse(req.ClientMsgUUID)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid client_msg_uuid", err)
		return
	}

	// Create message payload
	payload := events.MessageOutPayload{
		ContentType:      req.MessageType,
		Content:          req.Content,
		ClientMsgUUID:    req.ClientMsgUUID,
		ReplyToMessageID: req.ReplyTo,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "Failed to marshal payload", err)
		return
	}

	// Create event and outbox entry
	serverMsgID, err := h.eventService.CreateMessageOutEvent(r.Context(), req.AccountID, req.ConvoID, req.ClientMsgUUID, payloadBytes)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "Failed to create event", err)
		return
	}

	if err := h.outboxService.CreateOutboxEntry(r.Context(), clientUUID, req.AccountID, req.ConvoID, serverMsgID); err != nil {
		h.writeError(w, http.StatusInternalServerError, "Failed to create outbox entry", err)
		return
	}

	response := map[string]interface{}{
		"server_msg_id":   serverMsgID,
		"status":          events.OutboxStatusQueued,
		"client_msg_uuid": req.ClientMsgUUID,
	}

	h.logger.Info("Message queued for sending",
		zap.String("client_msg_uuid", req.ClientMsgUUID),
		zap.Int64("server_msg_id", serverMsgID),
		zap.String("account_id", req.AccountID))

	h.writeJSON(w, http.StatusCreated, response)
}

// SyncEvents handles event synchronization requests
func (h *APIHandler) SyncEvents(w http.ResponseWriter, r *http.Request) {
	accountID := r.URL.Query().Get("account_id")
	if accountID == "" {
		h.writeError(w, http.StatusBadRequest, "Missing account_id parameter", nil)
		return
	}

	sinceStr := r.URL.Query().Get("since")
	since := int64(0)
	if sinceStr != "" {
		var err error
		since, err = strconv.ParseInt(sinceStr, 10, 64)
		if err != nil {
			h.writeError(w, http.StatusBadRequest, "Invalid since parameter", err)
			return
		}
	}

	limitStr := r.URL.Query().Get("limit")
	limit := int32(100)
	if limitStr != "" {
		limitInt, err := strconv.Atoi(limitStr)
		if err != nil || limitInt < 1 || limitInt > 1000 {
			h.writeError(w, http.StatusBadRequest, "Invalid limit parameter (must be 1-1000)", nil)
			return
		}
		limit = int32(limitInt)
	}

	events, err := h.eventService.GetEventsSince(r.Context(), accountID, since, limit)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "Failed to get events", err)
		return
	}

	// Get next sequence number
	nextSeq := since
	if len(events) > 0 {
		nextSeq = events[len(events)-1].Seq
	}

	// Check if there are more events
	hasMore := len(events) == int(limit)

	response := map[string]interface{}{
		"events":   h.convertEventsToAPI(events),
		"next_seq": nextSeq,
		"has_more": hasMore,
	}

	h.logger.Debug("Sync events response",
		zap.String("account_id", accountID),
		zap.Int64("since", since),
		zap.Int("count", len(events)),
		zap.Bool("has_more", hasMore))

	h.writeJSON(w, http.StatusOK, response)
}

// GetQRCode handles QR code generation requests
func (h *APIHandler) GetQRCode(w http.ResponseWriter, r *http.Request) {
	accountID := r.URL.Query().Get("account_id")
	if accountID == "" {
		h.writeError(w, http.StatusBadRequest, "Missing account_id parameter", nil)
		return
	}

	// TODO: Call bridge service to generate QR code
	// For now, return a placeholder response
	response := map[string]interface{}{
		"qr_code_png":        "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg==", // 1x1 transparent PNG
		"pairing_session_id": uuid.New().String(),
		"expires_at":         time.Now().Add(5 * time.Minute).UTC(),
	}

	h.logger.Info("QR code requested",
		zap.String("account_id", accountID))

	h.writeJSON(w, http.StatusOK, response)
}

// ListAccounts handles account listing requests
func (h *APIHandler) ListAccounts(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := int32(20)
	if limitStr != "" {
		limitInt, err := strconv.Atoi(limitStr)
		if err != nil || limitInt < 1 || limitInt > 100 {
			h.writeError(w, http.StatusBadRequest, "Invalid limit parameter (must be 1-100)", nil)
			return
		}
		limit = int32(limitInt)
	}

	offsetStr := r.URL.Query().Get("offset")
	offset := int32(0)
	if offsetStr != "" {
		offsetInt, err := strconv.Atoi(offsetStr)
		if err != nil || offsetInt < 0 {
			h.writeError(w, http.StatusBadRequest, "Invalid offset parameter", nil)
			return
		}
		offset = int32(offsetInt)
	}

	accounts, err := h.accountService.ListAccounts(r.Context(), limit, offset)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "Failed to list accounts", err)
		return
	}

	response := map[string]interface{}{
		"accounts": h.convertAccountsToAPI(accounts),
		"total":    len(accounts), // TODO: Get actual total count
	}

	h.writeJSON(w, http.StatusOK, response)
}

// GetAccount handles individual account requests
func (h *APIHandler) GetAccount(w http.ResponseWriter, r *http.Request) {
	accountID := chi.URLParam(r, "account_id")
	if accountID == "" {
		h.writeError(w, http.StatusBadRequest, "Missing account_id", nil)
		return
	}

	account, err := h.accountService.GetAccount(r.Context(), accountID)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "Account not found", err)
		return
	}

	h.writeJSON(w, http.StatusOK, h.convertAccountToAPI(*account))
}

// Helper methods

func (h *APIHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode JSON response", zap.Error(err))
	}
}

func (h *APIHandler) writeError(w http.ResponseWriter, status int, message string, err error) {
	h.logger.Error("API error",
		zap.String("message", message),
		zap.Error(err),
		zap.Int("status", status))

	response := map[string]interface{}{
		"error":     message,
		"timestamp": time.Now().UTC(),
	}

	if err != nil {
		response["details"] = err.Error()
	}

	h.writeJSON(w, status, response)
}

func (h *APIHandler) convertEventsToAPI(events []repo.Event) []map[string]interface{} {
	result := make([]map[string]interface{}, len(events))
	for i, event := range events {
		result[i] = map[string]interface{}{
			"seq":            event.Seq,
			"id":             event.ID,
			"timestamp":      event.Ts,
			"type":           event.Type,
			"account_id":     event.AccountID,
			"device_id":      event.DeviceID,
			"convo_id":       event.ConvoID,
			"wa_message_id":  event.WaMessageID,
			"sender_jid":     event.SenderJid,
			"payload":        json.RawMessage(event.Payload),
			"attachment_ref": json.RawMessage(event.AttachmentRef),
		}
	}
	return result
}

func (h *APIHandler) convertAccountsToAPI(accounts []repo.Account) []map[string]interface{} {
	result := make([]map[string]interface{}, len(accounts))
	for i, account := range accounts {
		result[i] = h.convertAccountToAPI(account)
	}
	return result
}

// GetSettings handles user settings requests
func (h *APIHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from JWT token (same pattern as auth handler)
	userID, err := h.extractUserFromToken(r)
	if err != nil {
		h.writeError(w, http.StatusUnauthorized, "Invalid or missing token", err)
		return
	}

	// Get user's WhatsApp integration information
	whatsappIntegration, err := h.integrationService.GetWhatsAppIntegration(r.Context(), userID)
	if err != nil {
		// If integration doesn't exist, return default settings (no WhatsApp connected)
		h.logger.Debug("WhatsApp integration not found, returning default settings",
			zap.String("user_id", userID.String()),
			zap.Error(err))

		response := map[string]interface{}{
			"user_id": userID,
			"whatsapp": map[string]interface{}{
				"connected": false,
				"status":    "disconnected",
			},
		}
		h.writeJSON(w, http.StatusOK, response)
		return
	}

	// Build WhatsApp connection info
	whatsappInfo := map[string]interface{}{
		"connected": whatsappIntegration.Status == "connected",
		"status":    whatsappIntegration.Status,
		"wa_jid":    whatsappIntegration.ExternalID,
	}

	if whatsappIntegration.DisplayName.Valid {
		whatsappInfo["display_name"] = whatsappIntegration.DisplayName.String
	}
	if whatsappIntegration.AvatarUrl.Valid {
		whatsappInfo["avatar_url"] = whatsappIntegration.AvatarUrl.String
	}
	if whatsappIntegration.LastSeen.Valid {
		whatsappInfo["last_seen"] = whatsappIntegration.LastSeen.Time
	}

	response := map[string]interface{}{
		"user_id":  userID,
		"whatsapp": whatsappInfo,
	}

	h.logger.Debug("Settings retrieved", zap.String("user_id", userID.String()))
	h.writeJSON(w, http.StatusOK, response)
}

// Helper methods

func (h *APIHandler) extractUserFromToken(r *http.Request) (uuid.UUID, error) {
	// Extract token from Authorization header
	authHeader := r.Header.Get("Authorization")
	tokenString, err := auth.ExtractTokenFromHeader(authHeader)
	if err != nil {
		return uuid.Nil, errors.New("missing or malformed token")
	}

	// Validate token using JWT config
	claims, err := h.jwtConfig.ValidateToken(tokenString)
	if err != nil {
		return uuid.Nil, errors.New("invalid token")
	}

	return claims.UserID, nil
}

func (h *APIHandler) convertAccountToAPI(account repo.Account) map[string]interface{} {
	result := map[string]interface{}{
		"id":         account.ID,
		"status":     account.Status,
		"created_at": account.CreatedAt,
		"updated_at": account.UpdatedAt,
	}

	if account.WaJid.Valid {
		result["wa_jid"] = account.WaJid.String
	}
	if account.DisplayName.Valid {
		result["display_name"] = account.DisplayName.String
	}
	if account.AvatarUrl.Valid {
		result["avatar_url"] = account.AvatarUrl.String
	}
	if account.LastSeen.Valid {
		result["last_seen"] = account.LastSeen.Time
	}

	return result
}

// SyncConversations handles conversation sync requests
func (h *APIHandler) SyncConversations(w http.ResponseWriter, r *http.Request) {
	integrationIDStr := chi.URLParam(r, "integration_id")
	integrationID, err := strconv.Atoi(integrationIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid integration_id", err)
		return
	}

	sinceSeqStr := r.URL.Query().Get("since_seq")
	sinceSeq := int64(0)
	if sinceSeqStr != "" {
		sinceSeq, err = strconv.ParseInt(sinceSeqStr, 10, 64)
		if err != nil {
			h.writeError(w, http.StatusBadRequest, "Invalid since_seq parameter", err)
			return
		}
	}

	limitStr := r.URL.Query().Get("limit")
	limit := int32(100)
	if limitStr != "" {
		limitInt, err := strconv.Atoi(limitStr)
		if err != nil || limitInt < 1 || limitInt > 1000 {
			h.writeError(w, http.StatusBadRequest, "Invalid limit parameter (must be 1-1000)", nil)
			return
		}
		limit = int32(limitInt)
	}

	conversations, err := h.queries.ListUserIntegrationConversationsSinceSeq(r.Context(), dbgen.ListUserIntegrationConversationsSinceSeqParams{
		UserIntegrationID: int32(integrationID),
		SinceSeq:          sinceSeq,
		LimitCount:        limit,
	})
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "Failed to fetch conversations", err)
		return
	}

	// Get latest seq
	latestSeq := sinceSeq
	if len(conversations) > 0 {
		latestSeq = conversations[len(conversations)-1].Seq.Int64
	}

	hasMore := len(conversations) == int(limit)

	response := map[string]interface{}{
		"conversations": conversations,
		"latest_seq":    latestSeq,
		"has_more":      hasMore,
		"total_count":   len(conversations),
	}

	h.logger.Debug("Sync conversations response",
		zap.Int("integration_id", integrationID),
		zap.Int64("since_seq", sinceSeq),
		zap.Int("count", len(conversations)),
		zap.Bool("has_more", hasMore))

	h.writeJSON(w, http.StatusOK, response)
}

// SyncMessages handles message sync requests
func (h *APIHandler) SyncMessages(w http.ResponseWriter, r *http.Request) {
	integrationIDStr := chi.URLParam(r, "integration_id")
	integrationID, err := strconv.Atoi(integrationIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid integration_id", err)
		return
	}

	sinceSeqStr := r.URL.Query().Get("since_seq")
	sinceSeq := int64(0)
	if sinceSeqStr != "" {
		sinceSeq, err = strconv.ParseInt(sinceSeqStr, 10, 64)
		if err != nil {
			h.writeError(w, http.StatusBadRequest, "Invalid since_seq parameter", err)
			return
		}
	}

	limitStr := r.URL.Query().Get("limit")
	limit := int32(1500)
	if limitStr != "" {
		limitInt, err := strconv.Atoi(limitStr)
		if err != nil || limitInt < 1 || limitInt > 2000 {
			h.writeError(w, http.StatusBadRequest, "Invalid limit parameter (must be 1-2000)", nil)
			return
		}
		limit = int32(limitInt)
	}

	messages, err := h.queries.ListUserIntegrationMessagesSinceSeq(r.Context(), dbgen.ListUserIntegrationMessagesSinceSeqParams{
		UserIntegrationID: int32(integrationID),
		SinceSeq:          sinceSeq,
		LimitCount:        limit,
	})
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "Failed to fetch messages", err)
		return
	}

	// Get latest seq
	latestSeq := sinceSeq
	if len(messages) > 0 {
		latestSeq = messages[len(messages)-1].Seq.Int64
	}

	hasMore := len(messages) == int(limit)

	response := map[string]interface{}{
		"messages":    messages,
		"latest_seq":  latestSeq,
		"has_more":    hasMore,
		"total_count": len(messages),
	}

	h.logger.Debug("Sync messages response",
		zap.Int("integration_id", integrationID),
		zap.Int64("since_seq", sinceSeq),
		zap.Int("count", len(messages)),
		zap.Bool("has_more", hasMore))

	h.writeJSON(w, http.StatusOK, response)
}

// SyncContacts handles contact sync requests
func (h *APIHandler) SyncContacts(w http.ResponseWriter, r *http.Request) {
	integrationIDStr := chi.URLParam(r, "integration_id")
	integrationID, err := strconv.Atoi(integrationIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid integration_id", err)
		return
	}

	sinceSeqStr := r.URL.Query().Get("since_seq")
	sinceSeq := int64(0)
	if sinceSeqStr != "" {
		sinceSeq, err = strconv.ParseInt(sinceSeqStr, 10, 64)
		if err != nil {
			h.writeError(w, http.StatusBadRequest, "Invalid since_seq parameter", err)
			return
		}
	}

	limitStr := r.URL.Query().Get("limit")
	limit := int32(500)
	if limitStr != "" {
		limitInt, err := strconv.Atoi(limitStr)
		if err != nil || limitInt < 1 || limitInt > 1000 {
			h.writeError(w, http.StatusBadRequest, "Invalid limit parameter (must be 1-1000)", nil)
			return
		}
		limit = int32(limitInt)
	}

	contacts, err := h.queries.ListUserIntegrationContactsSinceSeq(r.Context(), dbgen.ListUserIntegrationContactsSinceSeqParams{
		UserIntegrationID: int32(integrationID),
		SinceSeq:          sinceSeq,
		LimitCount:        limit,
	})
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "Failed to fetch contacts", err)
		return
	}

	// Get latest seq
	latestSeq := sinceSeq
	if len(contacts) > 0 {
		latestSeq = contacts[len(contacts)-1].Seq.Int64
	}

	hasMore := len(contacts) == int(limit)

	response := map[string]interface{}{
		"contacts":    contacts,
		"latest_seq":  latestSeq,
		"has_more":    hasMore,
		"total_count": len(contacts),
	}

	h.logger.Debug("Sync contacts response",
		zap.Int("integration_id", integrationID),
		zap.Int64("since_seq", sinceSeq),
		zap.Int("count", len(contacts)),
		zap.Bool("has_more", hasMore))

	h.writeJSON(w, http.StatusOK, response)
}

// GetSyncStatus handles sync status requests
func (h *APIHandler) GetSyncStatus(w http.ResponseWriter, r *http.Request) {
	integrationIDStr := chi.URLParam(r, "integration_id")
	integrationID, err := strconv.Atoi(integrationIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid integration_id", err)
		return
	}

	// Get latest seq numbers for each entity type
	latestConvSeq, err := h.queries.GetUserIntegrationLatestConversationSeq(r.Context(), int32(integrationID))
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "Failed to get latest conversation seq", err)
		return
	}

	latestMsgSeq, err := h.queries.GetUserIntegrationLatestMessageSeq(r.Context(), int32(integrationID))
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "Failed to get latest message seq", err)
		return
	}

	latestContactSeq, err := h.queries.GetUserIntegrationLatestContactSeq(r.Context(), int32(integrationID))
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "Failed to get latest contact seq", err)
		return
	}

	response := map[string]interface{}{
		"latest_conversation_seq": latestConvSeq,
		"latest_message_seq":      latestMsgSeq,
		"latest_contact_seq":      latestContactSeq,
	}

	h.logger.Debug("Sync status response",
		zap.Int("integration_id", integrationID),
		zap.Any("latest_conv_seq", latestConvSeq),
		zap.Any("latest_msg_seq", latestMsgSeq),
		zap.Any("latest_contact_seq", latestContactSeq))

	h.writeJSON(w, http.StatusOK, response)
}
