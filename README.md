# Tennex - WhatsApp Bridge System

A minimal WhatsApp bridge service for QR-based authentication and session management.

## Current Status

This is a simplified, working WhatsApp bridge that:
- Generates QR codes for WhatsApp authentication
- Provides HTTP API for basic operations
- Uses Docker for development with live reload

## Project Structure

```
tennex/
├── services/bridge/        # WhatsApp bridge service (Go)
├── deployments/local/      # Local development Docker setup
├── scripts/               # Development shell shortcuts
└── test/                  # Minimal whatsmeow PoC
```

## Quick Start

1. **Start development environment:**
   ```bash
   # Source shell shortcuts
   . scripts/shell_shortcuts.sh
   
   # Start with live reload
   txdev
   ```

2. **Test QR generation:**
   ```bash
   # Generate QR and get session
   txconnect my-client
   ```

3. **Check health:**
   ```bash
   txhealth
   ```

## Development

- `txdev` - Start development environment with live reload
- `txup` - Start services
- `txdown` - Stop services  
- `txl` - View logs
- `txconnect <client>` - Test QR generation
- `txhelp` - Show all shortcuts

## API Endpoints

- `GET /health` - Health check
- `GET /ready` - Readiness check  
- `GET /stats` - Runtime stats
- `POST /connect-minimal` - Generate QR for client
- `GET /debug/config` - Configuration info

## Architecture

The bridge service is intentionally minimal:
- Single HTTP server with essential endpoints
- QR generation via whatsmeow library
- Session management with SQLite
- Live reload development with Air
- Docker-based deployment