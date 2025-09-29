package grpc

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"

	proto "github.com/tennex/shared/proto/gen/proto"
)

// BackendClient wraps the gRPC client for communicating with the backend
type BackendClient struct {
	client proto.BridgeServiceClient
	conn   *grpc.ClientConn
}

// NewBackendClient creates a new backend gRPC client
func NewBackendClient(backendAddr string) (*BackendClient, error) {
	conn, err := grpc.Dial(backendAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to backend at %s: %w", backendAddr, err)
	}

	client := proto.NewBridgeServiceClient(conn)

	return &BackendClient{
		client: client,
		conn:   conn,
	}, nil
}

// Close closes the gRPC connection
func (c *BackendClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// UpdateAccountStatus updates the account status in the backend
func (c *BackendClient) UpdateAccountStatus(ctx context.Context, accountID, waJid, displayName, avatarUrl string) error {
	req := &proto.UpdateAccountStatusRequest{
		AccountId: accountID,
		Status:    proto.AccountStatus_ACCOUNT_STATUS_CONNECTED,
		LastSeen:  timestamppb.New(time.Now()),
		Info: &proto.AccountInfo{
			WaJid:       waJid,
			DisplayName: displayName,
			AvatarUrl:   avatarUrl,
		},
	}

	resp, err := c.client.UpdateAccountStatus(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to update account status: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("backend reported failure updating account status")
	}

	return nil
}

// UpdateAccountDisconnected updates the account status to disconnected
func (c *BackendClient) UpdateAccountDisconnected(ctx context.Context, accountID string) error {
	req := &proto.UpdateAccountStatusRequest{
		AccountId: accountID,
		Status:    proto.AccountStatus_ACCOUNT_STATUS_DISCONNECTED,
		LastSeen:  timestamppb.New(time.Now()),
	}

	resp, err := c.client.UpdateAccountStatus(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to update account disconnection status: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("backend reported failure updating account disconnection")
	}

	return nil
}
