# Tennex Bridge Service

WhatsApp bridge service using whatsmeow for QR-based authentication and session management.

## Features

- **WhatsApp Integration**: QR-based authentication via whatsmeow
- **HTTP API**: Health checks and QR generation endpoints  
- **Session Management**: Persistent session storage with SQLite
- **Live Reload**: Development with Air hot reload
- **Structured Logging**: Comprehensive logging with different levels

## Quick Start

Use the shell shortcuts from the project root:

```bash
# Source shortcuts
. ../../scripts/shell_shortcuts.sh

# Start development environment
txdev

# Test QR generation  
txconnect my-client

# View logs
txl
```

## Configuration

Configure via environment variables:

- `TENNEX_BRIDGE_LOG_LEVEL` - Log level (debug, info, warn, error, fatal)
- `TENNEX_BRIDGE_HTTP_PORT` - HTTP server port (default: 8080)
- `TENNEX_BRIDGE_WHATSAPP_SESSION_PATH` - Session storage path
- `TENNEX_BRIDGE_DEV_QR_IN_TERMINAL` - Show QR in terminal (true/false)

## API Endpoints

### Core Endpoints
- `GET /health` - Basic health check
- `GET /ready` - Readiness check
- `GET /stats` - Runtime statistics
- `POST /connect-minimal` - Generate QR for WhatsApp authentication

### Debug Endpoints  
- `GET /debug/config` - Configuration information
- `GET /debug/whatsapp` - WhatsApp client status
- `GET /debug/pprof/*` - Go profiling endpoints (if enabled)

## Development

The service uses Air for live reload during development:

- Code changes automatically trigger rebuilds
- Session data persists across restarts
- Logs stream in real-time

## Session Management

WhatsApp session data is stored in SQLite databases under `/app/sessions/`:
- Each client gets its own session database
- Sessions persist across container restarts
- Re-authentication required if session data is lost

## Project Structure

```
bridge/
├── cmd/bridge/         # Main application entry point
├── internal/
│   ├── config/         # Configuration management  
│   ├── logging/        # Structured logging utilities
│   └── server/         # HTTP server and endpoints
├── config/             # Configuration files
└── Dockerfile.dev      # Development container with Air
```