package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"google.golang.org/protobuf/proto"

	backendGRPC "github.com/tennex/bridge/internal/grpc"
	"github.com/tennex/bridge/internal/recorder"
	pb "github.com/tennex/shared/proto/gen/proto"
)

func main() {
	recordingsDir := flag.String("dir", "./recordings", "Recordings directory")
	sessionID := flag.String("session", "", "Session ID to replay")
	recordingRange := flag.String("range", "", "Recording range (e.g. '1', '1-5', '10-20')")
	backendAddr := flag.String("backend", "localhost:6001", "Backend gRPC address")
	flag.Parse()

	if *sessionID == "" {
		fmt.Println("❌ Error: --session is required")
		fmt.Println("\nUsage:")
		fmt.Println("  replay --session <session-id> [--range <start>-<end>] [--backend <addr>]")
		fmt.Println("\nExamples:")
		fmt.Println("  replay --session whatsapp-abc123-1234567890")
		fmt.Println("  replay --session whatsapp-abc123-1234567890 --range 1-10")
		fmt.Println("  replay --session whatsapp-abc123-1234567890 --range 5")
		os.Exit(1)
	}

	sessionDir := filepath.Join(*recordingsDir, *sessionID)

	// Load session
	session, err := recorder.LoadSession(sessionDir)
	if err != nil {
		fmt.Printf("❌ Failed to load session: %v\n", err)
		os.Exit(1)
	}

	// Print session info
	fmt.Println("📼 Loaded recording session:")
	recorder.PrintSession(os.Stdout, session)

	// Parse range
	startID, endID, err := parseRange(*recordingRange, session.TotalRecordings)
	if err != nil {
		fmt.Printf("❌ Invalid range: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("🎬 Replaying recordings #%d to #%d...\n\n", startID, endID)

	// Connect to backend
	fmt.Printf("🔌 Connecting to backend at %s...\n", *backendAddr)
	client, err := backendGRPC.NewIntegrationClient(*backendAddr)
	if err != nil {
		fmt.Printf("❌ Failed to connect to backend: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()
	fmt.Println("✅ Connected to backend\n")

	// Replay recordings
	ctx := context.Background()
	successCount := 0
	failureCount := 0

	for id := startID; id <= endID; id++ {
		fmt.Printf("📼 Replaying recording #%03d...\n", id)

		rec, payload, err := recorder.LoadRecording(sessionDir, id)
		if err != nil {
			fmt.Printf("❌ Failed to load recording: %v\n", err)
			failureCount++
			break
		}

		// Replay the recording
		if err := replayRecording(ctx, client, rec, payload); err != nil {
			fmt.Printf("❌ Failed to replay: %v\n", err)
			failureCount++
			break
		}

		fmt.Printf("✅ Successfully replayed #%03d\n\n", id)
		successCount++

		// Small delay between recordings
		time.Sleep(100 * time.Millisecond)
	}

	// Summary
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("📊 Replay Summary:\n")
	fmt.Printf("   ✅ Successful: %d\n", successCount)
	fmt.Printf("   ❌ Failed: %d\n", failureCount)
	fmt.Printf("   📼 Total: %d\n", successCount+failureCount)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	if failureCount > 0 {
		os.Exit(1)
	}
}

func parseRange(rangeStr string, maxID int) (start, end int, err error) {
	if rangeStr == "" {
		return 1, maxID, nil
	}

	// Check if it's a range (e.g., "1-5") or single ID (e.g., "3")
	if strings.Contains(rangeStr, "-") {
		parts := strings.Split(rangeStr, "-")
		if len(parts) != 2 {
			return 0, 0, fmt.Errorf("invalid range format, expected 'start-end'")
		}

		start, err = strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			return 0, 0, fmt.Errorf("invalid start ID: %v", err)
		}

		end, err = strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil {
			return 0, 0, fmt.Errorf("invalid end ID: %v", err)
		}

		if start < 1 || end > maxID || start > end {
			return 0, 0, fmt.Errorf("range must be between 1 and %d, and start <= end", maxID)
		}

		return start, end, nil
	}

	// Single ID
	id, err := strconv.Atoi(strings.TrimSpace(rangeStr))
	if err != nil {
		return 0, 0, fmt.Errorf("invalid ID: %v", err)
	}

	if id < 1 || id > maxID {
		return 0, 0, fmt.Errorf("ID must be between 1 and %d", maxID)
	}

	return id, id, nil
}

func replayRecording(ctx context.Context, client *backendGRPC.IntegrationClient, rec *recorder.Recording, payload []byte) error {
	fmt.Printf("   Type: %s\n", rec.RequestType)
	fmt.Printf("   Timestamp: %s\n", rec.Timestamp.Format(time.RFC3339))
	if len(rec.Metadata) > 0 {
		fmt.Printf("   Metadata: %v\n", rec.Metadata)
	}

	switch rec.RequestType {
	case "SyncConversations":
		return replaySyncConversations(ctx, client, payload)
	case "SyncMessages":
		return replaySyncMessages(ctx, client, payload)
	case "SyncContacts":
		return replaySyncContacts(ctx, client, payload)
	case "ProcessMessage":
		return replayProcessMessage(ctx, client, payload)
	case "UpdateConnectionStatus":
		return replayUpdateConnectionStatus(ctx, client, payload)
	case "CreateUserIntegration":
		return replayCreateUserIntegration(ctx, client, payload)
	default:
		return fmt.Errorf("unknown request type: %s", rec.RequestType)
	}
}

func replaySyncConversations(ctx context.Context, client *backendGRPC.IntegrationClient, payload []byte) error {
	var req pb.SyncConversationsRequest
	if err := proto.Unmarshal(payload, &req); err != nil {
		return fmt.Errorf("failed to unmarshal: %w", err)
	}

	// Extract integration context and conversations
	integrationCtx := req.Context
	conversations := req.Conversations
	syncType := req.SyncType

	return client.SyncConversations(ctx, integrationCtx, conversations, syncType)
}

func replaySyncMessages(ctx context.Context, client *backendGRPC.IntegrationClient, payload []byte) error {
	var req pb.SyncMessagesRequest
	if err := proto.Unmarshal(payload, &req); err != nil {
		return fmt.Errorf("failed to unmarshal: %w", err)
	}

	return client.SyncMessages(ctx, req.Context, req.ConversationExternalId, req.Messages)
}

func replaySyncContacts(ctx context.Context, client *backendGRPC.IntegrationClient, payload []byte) error {
	var req pb.SyncContactsRequest
	if err := proto.Unmarshal(payload, &req); err != nil {
		return fmt.Errorf("failed to unmarshal: %w", err)
	}

	return client.SyncContacts(ctx, req.Context, req.Contacts)
}

func replayProcessMessage(ctx context.Context, client *backendGRPC.IntegrationClient, payload []byte) error {
	var req pb.ProcessMessageRequest
	if err := proto.Unmarshal(payload, &req); err != nil {
		return fmt.Errorf("failed to unmarshal: %w", err)
	}

	return client.ProcessMessage(ctx, req.Context, req.Message)
}

func replayUpdateConnectionStatus(ctx context.Context, client *backendGRPC.IntegrationClient, payload []byte) error {
	var req pb.UpdateConnectionStatusRequest
	if err := proto.Unmarshal(payload, &req); err != nil {
		return fmt.Errorf("failed to unmarshal: %w", err)
	}

	return client.UpdateConnectionStatus(ctx, req.Context, req.Status, req.QrCode, req.Metadata)
}

func replayCreateUserIntegration(ctx context.Context, client *backendGRPC.IntegrationClient, payload []byte) error {
	var req pb.CreateUserIntegrationRequest
	if err := proto.Unmarshal(payload, &req); err != nil {
		return fmt.Errorf("failed to unmarshal: %w", err)
	}

	_, err := client.CreateUserIntegration(ctx, req.UserId, req.PlatformUserId, req.DisplayName, req.AvatarUrl, req.Metadata)
	return err
}
