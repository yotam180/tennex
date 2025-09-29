package grpc

import (
	"fmt"
	"os"
	"path/filepath"
)

// IntegrationClientInterface defines the interface for integration clients
// This allows us to use either IntegrationClient or RecordingIntegrationClient interchangeably
type IntegrationClientInterface interface {
	Close() error
	// Add other methods as needed - for now, we'll use the concrete types
}

// NewIntegrationClientWithRecording creates an integration client with optional recording
func NewIntegrationClientWithRecording(backendAddr string) (*RecordingIntegrationClient, error) {
	// Determine recordings directory
	recordingsDir := os.Getenv("RECORDINGS_DIR")
	if recordingsDir == "" {
		// Default to ./recordings or /app/recordings in Docker
		if _, err := os.Stat("/app"); err == nil {
			recordingsDir = "/app/recordings"
		} else {
			cwd, _ := os.Getwd()
			recordingsDir = filepath.Join(cwd, "recordings")
		}
	}

	// Create recordings directory if it doesn't exist
	if err := os.MkdirAll(recordingsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create recordings directory: %w", err)
	}

	return NewRecordingIntegrationClient(backendAddr, recordingsDir)
}
