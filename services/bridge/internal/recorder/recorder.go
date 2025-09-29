package recorder

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"google.golang.org/protobuf/proto"
)

type RecordingMode int

const (
	ModeOff RecordingMode = iota
	ModeRecord
	ModeReplay
)

// Recording represents a single gRPC request recording
type Recording struct {
	ID          int                    `json:"id"`
	Timestamp   time.Time              `json:"timestamp"`
	RequestType string                 `json:"request_type"`
	Metadata    map[string]interface{} `json:"metadata"`
	PayloadFile string                 `json:"payload_file"`
}

// Session represents a recording session
type Session struct {
	SessionID       string      `json:"session_id"`
	UserID          string      `json:"user_id"`
	IntegrationType string      `json:"integration_type"`
	StartedAt       time.Time   `json:"started_at"`
	CompletedAt     *time.Time  `json:"completed_at,omitempty"`
	Recordings      []Recording `json:"recordings"`
	TotalRecordings int         `json:"total_recordings"`
}

// Recorder handles recording and replaying gRPC requests
type Recorder struct {
	mode       RecordingMode
	sessionDir string
	session    *Session
	nextID     int
}

// NewRecorder creates a new recorder instance
func NewRecorder(mode RecordingMode, sessionDir string) *Recorder {
	return &Recorder{
		mode:       mode,
		sessionDir: sessionDir,
		nextID:     1,
	}
}

// StartSession initializes a new recording session
func (r *Recorder) StartSession(userID, integrationType string) error {
	if r.mode != ModeRecord {
		return nil
	}

	sessionID := fmt.Sprintf("%s-%s-%d", integrationType, userID[:8], time.Now().Unix())
	sessionPath := filepath.Join(r.sessionDir, sessionID)

	if err := os.MkdirAll(sessionPath, 0755); err != nil {
		return fmt.Errorf("failed to create session directory: %w", err)
	}

	r.session = &Session{
		SessionID:       sessionID,
		UserID:          userID,
		IntegrationType: integrationType,
		StartedAt:       time.Now(),
		Recordings:      []Recording{},
		TotalRecordings: 0,
	}

	r.sessionDir = sessionPath
	fmt.Printf("📼 Recording session started: %s\n", sessionPath)
	return nil
}

// Record saves a gRPC request to disk
func (r *Recorder) Record(ctx context.Context, requestType string, request proto.Message, metadata map[string]interface{}) error {
	if r.mode != ModeRecord || r.session == nil {
		return nil
	}

	// Serialize protobuf to bytes
	bytes, err := proto.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal protobuf: %w", err)
	}

	// Create recording
	recording := Recording{
		ID:          r.nextID,
		Timestamp:   time.Now(),
		RequestType: requestType,
		Metadata:    metadata,
		PayloadFile: fmt.Sprintf("%03d-%s.pb", r.nextID, requestType),
	}

	// Save protobuf payload
	payloadPath := filepath.Join(r.sessionDir, recording.PayloadFile)
	if err := os.WriteFile(payloadPath, bytes, 0644); err != nil {
		return fmt.Errorf("failed to write payload file: %w", err)
	}

	// Add to session
	r.session.Recordings = append(r.session.Recordings, recording)
	r.session.TotalRecordings++
	r.nextID++

	// Save session metadata after each recording
	if err := r.saveSessionMetadata(); err != nil {
		return fmt.Errorf("failed to save session metadata: %w", err)
	}

	fmt.Printf("📼 Recorded: #%03d %s (metadata: %v)\n", recording.ID, requestType, metadata)
	return nil
}

// EndSession completes the recording session
func (r *Recorder) EndSession() error {
	if r.mode != ModeRecord || r.session == nil {
		return nil
	}

	now := time.Now()
	r.session.CompletedAt = &now

	if err := r.saveSessionMetadata(); err != nil {
		return fmt.Errorf("failed to save final session metadata: %w", err)
	}

	fmt.Printf("📼 Recording session completed: %d recordings saved to %s\n", r.session.TotalRecordings, r.sessionDir)
	return nil
}

// saveSessionMetadata writes the session metadata to manifest.json
func (r *Recorder) saveSessionMetadata() error {
	manifestPath := filepath.Join(r.sessionDir, "manifest.json")
	data, err := json.MarshalIndent(r.session, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(manifestPath, data, 0644)
}

// LoadSession loads a recording session from disk
func LoadSession(sessionDir string) (*Session, error) {
	manifestPath := filepath.Join(sessionDir, "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	return &session, nil
}

// LoadRecording loads a specific recording from disk
func LoadRecording(sessionDir string, recordingID int) (*Recording, []byte, error) {
	session, err := LoadSession(sessionDir)
	if err != nil {
		return nil, nil, err
	}

	// Find recording by ID
	var recording *Recording
	for i := range session.Recordings {
		if session.Recordings[i].ID == recordingID {
			recording = &session.Recordings[i]
			break
		}
	}

	if recording == nil {
		return nil, nil, fmt.Errorf("recording #%d not found", recordingID)
	}

	// Load payload
	payloadPath := filepath.Join(sessionDir, recording.PayloadFile)
	payload, err := os.ReadFile(payloadPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read payload: %w", err)
	}

	return recording, payload, nil
}

// ListSessions lists all available recording sessions
func ListSessions(recordingsDir string) ([]string, error) {
	entries, err := os.ReadDir(recordingsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	sessions := []string{}
	for _, entry := range entries {
		if entry.IsDir() {
			sessions = append(sessions, entry.Name())
		}
	}
	return sessions, nil
}

// PrintSession displays session information
func PrintSession(w io.Writer, session *Session) {
	fmt.Fprintf(w, "📼 Session: %s\n", session.SessionID)
	fmt.Fprintf(w, "   User ID: %s\n", session.UserID)
	fmt.Fprintf(w, "   Integration: %s\n", session.IntegrationType)
	fmt.Fprintf(w, "   Started: %s\n", session.StartedAt.Format(time.RFC3339))
	if session.CompletedAt != nil {
		fmt.Fprintf(w, "   Completed: %s\n", session.CompletedAt.Format(time.RFC3339))
		duration := session.CompletedAt.Sub(session.StartedAt)
		fmt.Fprintf(w, "   Duration: %s\n", duration.Round(time.Second))
	}
	fmt.Fprintf(w, "   Total Recordings: %d\n\n", session.TotalRecordings)

	fmt.Fprintf(w, "Recordings:\n")
	for _, rec := range session.Recordings {
		fmt.Fprintf(w, "  #%03d [%s] %s\n", rec.ID, rec.Timestamp.Format("15:04:05"), rec.RequestType)
		if len(rec.Metadata) > 0 {
			fmt.Fprintf(w, "       Metadata: %v\n", rec.Metadata)
		}
	}
}
