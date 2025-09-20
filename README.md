# Tennex - WhatsApp Bridge & Management System

A comprehensive system for WhatsApp automation and management with event-sourced architecture.

## Architecture Overview

- **Bridge Service**: WhatsApp connectivity via whatsmeow (Go)
- **Event Store API**: Durable event storage with cursor-based sync (Go) 
- **Management API**: OAuth, access control, scheduling (Go)
- **Client App**: Local-first UI with offline support (TypeScript/Electron)

## Project Structure

```
tennex/
├── services/           # Backend services
│   ├── bridge/         # WhatsApp bridge (whatsmeow)
│   ├── event-store/    # Event storage & sync API
│   └── management/     # Management & auth API
├── client/             # Client application
├── shared/             # Shared schemas & protocols
├── deployments/        # Docker & deployment configs
├── scripts/            # Development scripts
├── docs/              # Documentation
└── tools/             # Development tools
```

## Getting Started

1. Start with the bridge service: `cd services/bridge`
2. Run with Docker Compose: `docker-compose up -d`

## Development

See individual service README files for specific development instructions.
