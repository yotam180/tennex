package server

import (
	"go.uber.org/zap"

	"github.com/tennex/backend/internal/core"
)

// BridgeServer implements the bridge gRPC service
type BridgeServer struct {
	eventService   *core.EventService
	outboxService  *core.OutboxService
	accountService *core.AccountService
	logger         *zap.Logger
	// TODO: Add generated UnimplementedBridgeServiceServer when protobuf is generated
}

// NewBridgeServer creates a new bridge gRPC server
func NewBridgeServer(eventService *core.EventService, outboxService *core.OutboxService, accountService *core.AccountService, logger *zap.Logger) *BridgeServer {
	return &BridgeServer{
		eventService:   eventService,
		outboxService:  outboxService,
		accountService: accountService,
		logger:         logger.Named("bridge_server"),
	}
}

// TODO: Implement gRPC methods once protobuf is generated
//
// PublishInbound publishes an inbound event from the bridge
// func (s *BridgeServer) PublishInbound(ctx context.Context, req *bridgev1.PublishInboundRequest) (*bridgev1.PublishInboundResponse, error) {
//     event := convertProtoToEvent(req.Event)
//     seq, created, err := s.eventService.PublishInbound(ctx, event)
//     if err != nil {
//         return nil, err
//     }
//
//     return &bridgev1.PublishInboundResponse{
//         Seq:     seq,
//         Created: created,
//     }, nil
// }
//
// SendMessage sends a message through WhatsApp
// func (s *BridgeServer) SendMessage(ctx context.Context, req *bridgev1.SendMessageRequest) (*bridgev1.SendMessageResponse, error) {
//     // TODO: Implement message sending logic
//     return &bridgev1.SendMessageResponse{
//         Success:     true,
//         WaMessageId: "mock-wa-message-id",
//     }, nil
// }
//
// GetQRCode generates a QR code for WhatsApp pairing
// func (s *BridgeServer) GetQRCode(ctx context.Context, req *bridgev1.GetQRCodeRequest) (*bridgev1.GetQRCodeResponse, error) {
//     // TODO: Implement QR code generation
//     return &bridgev1.GetQRCodeResponse{
//         QrCodePng:        []byte("mock-qr-code"),
//         PairingSessionId: "mock-session-id",
//         ExpiresAt:        timestamppb.New(time.Now().Add(5 * time.Minute)),
//     }, nil
// }
//
// UpdateAccountStatus updates account status
// func (s *BridgeServer) UpdateAccountStatus(ctx context.Context, req *bridgev1.UpdateAccountStatusRequest) (*bridgev1.UpdateAccountStatusResponse, error) {
//     var lastSeen *time.Time
//     if req.LastSeen != nil {
//         t := req.LastSeen.AsTime()
//         lastSeen = &t
//     }
//
//     status := convertProtoStatusToString(req.Status)
//     err := s.accountService.UpdateAccountStatus(ctx, req.AccountId, status, lastSeen)
//     if err != nil {
//         return nil, err
//     }
//
//     return &bridgev1.UpdateAccountStatusResponse{
//         Success: true,
//     }, nil
// }

// Helper methods (will be implemented once protobuf is generated)

// func convertProtoToEvent(protoEvent *bridgev1.Event) *repo.Event {
//     return &repo.Event{
//         ID:            uuid.MustParse(protoEvent.Id),
//         Type:          protoEvent.Type,
//         AccountID:     protoEvent.AccountId,
//         DeviceID:      sql.NullString{String: protoEvent.DeviceId, Valid: protoEvent.DeviceId != ""},
//         ConvoID:       protoEvent.ConvoId,
//         WaMessageID:   sql.NullString{String: protoEvent.WaMessageId, Valid: protoEvent.WaMessageId != ""},
//         SenderJid:     sql.NullString{String: protoEvent.SenderJid, Valid: protoEvent.SenderJid != ""},
//         Payload:       json.RawMessage(protoEvent.Payload),
//         AttachmentRef: json.RawMessage(protoEvent.AttachmentRef),
//     }
// }
//
// func convertProtoStatusToString(status bridgev1.AccountStatus) string {
//     switch status {
//     case bridgev1.AccountStatus_ACCOUNT_STATUS_CONNECTED:
//         return events.AccountStatusConnected
//     case bridgev1.AccountStatus_ACCOUNT_STATUS_CONNECTING:
//         return events.AccountStatusConnecting
//     case bridgev1.AccountStatus_ACCOUNT_STATUS_DISCONNECTED:
//         return events.AccountStatusDisconnected
//     case bridgev1.AccountStatus_ACCOUNT_STATUS_ERROR:
//         return events.AccountStatusError
//     default:
//         return events.AccountStatusDisconnected
//     }
// }
