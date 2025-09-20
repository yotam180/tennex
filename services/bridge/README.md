# Tennex Bridge Service

The WhatsApp Bridge Service is the core component that handles WhatsApp connectivity using the `whatsmeow` library. It maintains persistent WhatsApp sessions, processes incoming messages, and converts them into internal events.

## Features

- **WhatsApp Integration**: Full WhatsApp connectivity via whatsmeow
- **Event Processing**: Converts WhatsApp events to internal event format
- **Session Management**: Persistent session storage with SQLite
- **HTTP API**: Health checks, metrics, and debugging endpoints
- **Structured Logging**: Comprehensive logging with different levels
- **Graceful Shutdown**: Clean shutdown handling
- **Development Tools**: Live reload, profiling, and debugging support

## Getting Started

### Prerequisites

- Go 1.21+
- WhatsApp account for authentication

### Quick Start

1. **Build and run the service:**
   ```bash
   make run
   ```

2. **Scan QR code:** When prompted, scan the QR code with your WhatsApp mobile app

3. **Check status:** Visit `http://localhost:8080/health` to verify the service is running

### Development Setup

1. **Install development tools:**
   ```bash
   make dev-tools
   ```

2. **Run with live reload:**
   ```bash
   make dev
   ```

## Configuration

The service can be configured via environment variables or YAML config file:

### Environment Variables

- `TENNEX_BRIDGE_LOG_LEVEL` - Log level (debug, info, warn, error, fatal)
- `TENNEX_BRIDGE_HTTP_PORT` - HTTP server port (default: 8080)
- `TENNEX_BRIDGE_WHATSAPP_SESSION_PATH` - Session storage path
- `TENNEX_BRIDGE_DEV_QR_IN_TERMINAL` - Show QR code in terminal (true/false)

### Configuration File

See `config/bridge.yaml` for a complete configuration example.

## API Endpoints

### Health & Status
- `GET /health` - Basic health check
- `GET /ready` - Readiness check (includes WhatsApp connectivity)
- `GET /stats` - Runtime statistics
- `GET /metrics` - Prometheus metrics (if enabled)

### Debug Endpoints
- `GET /debug/config` - Configuration information
- `GET /debug/whatsapp` - WhatsApp client status
- `GET /debug/pprof/*` - Go profiling endpoints (if enabled)

## Event Types

The bridge service processes the following WhatsApp events:

- **msg_in** - Incoming messages
- **msg_out** - Outgoing messages
- **delivery** - Delivery and read receipts
- **contact** - Contact information updates
- **presence** - User presence (online/offline/typing)
- **thread_meta** - Group/chat metadata changes
- **connection** - Connection status changes

## Project Structure

```
bridge/
├── cmd/bridge/         # Main application entry point
├── internal/
│   ├── config/         # Configuration management
│   ├── events/         # Event processing and handling
│   ├── logging/        # Structured logging utilities
│   ├── publisher/      # Event publishing (console, NATS, etc.)
│   ├── server/         # HTTP server and endpoints
│   └── whatsapp/       # WhatsApp client wrapper
├── config/             # Configuration files
├── build/              # Build artifacts
└── session/            # WhatsApp session data (created at runtime)
```

## Building

### Local Development
```bash
# Build only
make build

# Build and run
make run

# Development with live reload
make dev

# Run tests
make test

# Format and lint
make check
```

### Production Builds
```bash
# Build for current platform
make build

# Build for multiple platforms
make build-all

# Build Docker image
make docker-build
```

## Session Management

The service stores WhatsApp session data in the `session/` directory:
- `session.db` - SQLite database with session information
- Encryption keys and device registration data

**Important**: Keep the session directory secure and backed up. Losing it requires re-authentication.

## Logging

The service uses structured logging with configurable levels:

- **debug** - Verbose debugging information
- **info** - General information (default)
- **warn** - Warning messages
- **error** - Error messages
- **fatal** - Fatal errors (causes shutdown)

Example log entry:
```json
{
  "level": "info",
  "ts": "2024-01-15T10:30:45.123Z",
  "caller": "events/handler.go:45",
  "msg": "Publishing event",
  "service": "tennex-bridge",
  "component": "events",
  "event_id": "550e8400-e29b-41d4-a716-446655440000",
  "event_type": "msg_in",
  "convo_id": "1234567890@s.whatsapp.net"
}
```

## Debugging

### Debug Mode
Run with debug logging:
```bash
TENNEX_BRIDGE_LOG_LEVEL=debug make run
```

### Profiling
Enable pprof endpoints:
```bash
TENNEX_BRIDGE_DEV_ENABLE_PPROF=true make run
# Visit http://localhost:8080/debug/pprof/
```

### Common Issues

1. **QR Code Authentication**
   - Ensure `qr_in_terminal: true` in config
   - Check terminal supports UTF-8
   - Scan with WhatsApp mobile app

2. **Connection Issues**
   - Check internet connectivity
   - Verify WhatsApp account is not banned
   - Check logs for specific error messages

3. **Session Problems**
   - Delete `session/` directory to re-authenticate
   - Check file permissions on session directory

## Environment Variables

For a complete list of configuration options, see the configuration section above or run:
```bash
make help
```

## Contributing

1. Follow Go code formatting: `make fmt`
2. Run tests: `make test`
3. Run linting: `make lint`
4. Check all: `make check`

## License

This project is part of the Tennex system. See the main project LICENSE file for details.
