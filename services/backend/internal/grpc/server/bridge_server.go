package server

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/tennex/backend/internal/core"
	bridgev1 "github.com/tennex/pkg/proto/gen/pkg/proto"
)

// BridgeServer implements the bridge gRPC service
type BridgeServer struct {
	bridgev1.UnimplementedBridgeServiceServer
	eventService       *core.EventService
	outboxService      *core.OutboxService
	accountService     *core.AccountService
	integrationService *core.IntegrationService
	logger             *zap.Logger
}

// NewBridgeServer creates a new bridge gRPC server
func NewBridgeServer(eventService *core.EventService, outboxService *core.OutboxService, accountService *core.AccountService, integrationService *core.IntegrationService, logger *zap.Logger) *BridgeServer {
	return &BridgeServer{
		eventService:       eventService,
		outboxService:      outboxService,
		accountService:     accountService,
		integrationService: integrationService,
		logger:             logger.Named("bridge_server"),
	}
}

// UpdateAccountStatus updates account status and info via gRPC
func (s *BridgeServer) UpdateAccountStatus(ctx context.Context, req *bridgev1.UpdateAccountStatusRequest) (*bridgev1.UpdateAccountStatusResponse, error) {
	s.logger.Debug("UpdateAccountStatus gRPC call received",
		zap.String("account_id", req.AccountId),
		zap.String("status", req.Status.String()))

	var lastSeen *time.Time
	if req.LastSeen != nil {
		t := req.LastSeen.AsTime()
		lastSeen = &t
	}

	// Extract account info from request
	var waJid, displayName, avatarUrl string
	if req.Info != nil {
		waJid = req.Info.WaJid
		displayName = req.Info.DisplayName
		avatarUrl = req.Info.AvatarUrl
	}

	// Convert proto status to string
	status := convertProtoStatusToString(req.Status)

	// Parse account ID as UUID
	userID, err := uuid.Parse(req.AccountId)
	if err != nil {
		s.logger.Error("Failed to parse account_id as UUID",
			zap.String("account_id", req.AccountId),
			zap.Error(err))
		return nil, err
	}

	// Use UpsertUserIntegration to create or update the WhatsApp integration
	_, err = s.integrationService.UpsertUserIntegration(ctx, userID, core.IntegrationTypeWhatsApp, waJid, displayName, avatarUrl, status, nil, lastSeen)
	if err != nil {
		s.logger.Error("Failed to upsert WhatsApp integration",
			zap.String("account_id", req.AccountId),
			zap.Error(err))
		return nil, err
	}

	s.logger.Info("WhatsApp integration updated successfully",
		zap.String("account_id", req.AccountId),
		zap.String("wa_jid", waJid),
		zap.String("status", status))

	return &bridgev1.UpdateAccountStatusResponse{
		Success: true,
	}, nil
}

// Helper functions for protobuf conversion

func convertProtoStatusToString(status bridgev1.AccountStatus) string {
	switch status {
	case bridgev1.AccountStatus_ACCOUNT_STATUS_CONNECTED:
		return "connected"
	case bridgev1.AccountStatus_ACCOUNT_STATUS_CONNECTING:
		return "connecting"
	case bridgev1.AccountStatus_ACCOUNT_STATUS_DISCONNECTED:
		return "disconnected"
	case bridgev1.AccountStatus_ACCOUNT_STATUS_ERROR:
		return "error"
	default:
		return "disconnected"
	}
}
