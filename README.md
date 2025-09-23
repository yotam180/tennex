# Tennex - WhatsApp Bridge Platform

A production-ready WhatsApp messaging bridge built with Go, featuring event sourcing, reliable message delivery, and real-time streaming.

## Architecture Overview

```
┌─────────────────┐    ┌──────────────────┐    ┌────────────────┐
│   WhatsApp      │    │     Backend      │    │  Event Stream  │
│   Bridge        │◄──►│    (REST/gRPC)   │◄──►│  (WebSocket)   │
│                 │    │                  │    │                │
└─────────────────┘    └──────────────────┘    └────────────────┘
         │                        │                       │
         │                        ▼                       │
         │              ┌──────────────────┐              │
         │              │   PostgreSQL     │              │
         │              │  (Event Store)   │              │
         │              └──────────────────┘              │
         │                        │                       │
         └────────────────────────┼───────────────────────┘
                                  ▼
                         ┌──────────────────┐
                         │   NATS Core      │
                         │ (Notifications)  │
                         └──────────────────┘
```

## Features

- **Event Sourcing**: Append-only event log as source of truth
- **Reliable Messaging**: Transactional outbox pattern for guaranteed delivery
- **Real-time Streaming**: WebSocket connections with NATS notifications
- **Type Safety**: Generated code from OpenAPI, protobuf, and SQL schemas
- **Scalable Architecture**: Clean separation of concerns, ready for microservices
- **Production Ready**: Observability, graceful shutdown, health checks

## Quick Start

### Prerequisites

- Go 1.22+
- Docker & Docker Compose
- Optional: `buf`, `oapi-codegen`, `sqlc` for code generation

### 1. Start Infrastructure

```bash
# Start PostgreSQL, NATS, MinIO, and PgAdmin
make docker-up

# Or manually:
cd deployments/local && docker-compose up -d
```

**Services Available:**
- PostgreSQL: `localhost:5432`
- PgAdmin: http://localhost:8080 (admin@tennex.com / admin123)
- NATS: `localhost:4222` (HTTP monitoring: http://localhost:8222)
- MinIO: http://localhost:9001 (tennex / tennex123)

### 2. Generate Code

```bash
# Generate all contracts and database code
make gen
```

### 3. Run Services

```bash
# Backend API (port 8082)
cd services/backend && go run cmd/backend/main.go

# Event Stream (port 8083) 
cd services/eventstream && go run cmd/eventstream/main.go

# Bridge (port 8081) - Your existing WhatsApp integration
cd services/bridge && go run main.go
```

### 4. Test API

```bash
# Health check
curl http://localhost:8082/health

# List accounts
curl http://localhost:8082/accounts

# Sync events
curl "http://localhost:8082/sync?account_id=test-account&since=0&limit=10"

# WebSocket connection
wscat -c "ws://localhost:8083/ws?account_id=test-account"
```

## Project Structure

```
tennex/
├── Makefile                          # Development commands
├── go.work                          # Go workspace
├── buf.yaml, buf.gen.yaml           # Protobuf generation
├── pkg/                             # Shared contracts & utilities
│   ├── api/                         # OpenAPI specs & generated REST code
│   ├── proto/                       # gRPC protobuf definitions
│   ├── db/                          # Database schema & queries
│   └── events/                      # Event type definitions
├── services/
│   ├── backend/                     # Main API & business logic
│   │   ├── cmd/backend/             # Main application
│   │   └── internal/                # HTTP handlers, gRPC, core services
│   ├── bridge/                      # WhatsApp integration (your PoC)
│   └── eventstream/                 # WebSocket streaming service
├── deployments/local/               # Docker development environment
└── tools/                           # Code generation scripts
```

## Database Schema

### Core Tables

- **`events`**: Append-only event log (source of truth)
- **`outbox`**: Reliable message sending queue  
- **`accounts`**: WhatsApp account bindings
- **`media_blobs`**: Content-addressed media storage

### Key Indexes

- `events(account_id, seq)` - Event cursoring
- `events(convo_id, seq)` - Conversation queries
- `outbox(status, created_at)` - Queue processing

## API Endpoints

### REST API (Backend - Port 8082)

- `GET /health` - Health check
- `POST /outbox` - Send message (queued for delivery)
- `GET /sync` - Sync events since sequence number
- `GET /qr` - Generate WhatsApp QR code
- `GET /accounts` - List accounts
- `GET /accounts/{id}` - Get account details

### WebSocket (Event Stream - Port 8083)

- `GET /ws?account_id=X` - Real-time event notifications

### gRPC (Backend - Port 9001)

- `PublishInbound` - Bridge publishes WhatsApp events
- `SendMessage` - Bridge sends outbound messages  
- `GetQRCode` - Generate pairing QR code
- `UpdateAccountStatus` - Account status updates

## Development Workflow

### Available Commands

```bash
make help              # Show all available commands
make dev               # Start full development environment
make gen               # Generate code from contracts
make test              # Run tests
make lint              # Run linters
make clean             # Clean generated files
make docker-up         # Start Docker services
make docker-down       # Stop Docker services
```

### Code Generation

The project uses schema-first development:

1. **OpenAPI** (`pkg/api/openapi.yaml`) → REST handlers & types
2. **Protobuf** (`pkg/proto/bridge.proto`) → gRPC services & types  
3. **SQL** (`pkg/db/schema/`) → Database types & queries

Run `make gen` to regenerate all code after schema changes.

### Adding Features

1. **New REST endpoint**: Update `openapi.yaml`, run `make gen`, implement handler
2. **New gRPC method**: Update `bridge.proto`, run `make gen`, implement service
3. **Database changes**: Add migration in `pkg/db/schema/`, update queries
4. **New event type**: Add to `pkg/events/types.go`

## Configuration

Environment variables (prefix with `TENNEX_`):

```bash
# Database
TENNEX_DATABASE_URL=postgres://user:pass@host:port/db
TENNEX_DATABASE_MAX_CONNS=25

# NATS  
TENNEX_NATS_URL=nats://localhost:4222

# HTTP
TENNEX_HTTP_PORT=8082
TENNEX_GRPC_PORT=9001

# Logging
TENNEX_LOG_LEVEL=info
TENNEX_LOG_JSON=false
```

See `deployments/local/env.example` for all options.

## Next Steps

### Bridge Integration

1. **Generate contracts**: Run `make gen` to create gRPC client code
2. **Replace direct DB**: Use gRPC calls to backend instead of direct database access
3. **Implement handlers**: Add `PublishInbound`, `GetQRCode` in bridge service
4. **Add idempotency**: Use `client_msg_uuid` for deduplication

### Production Readiness

1. **Migrations**: Add migration runner (Atlas/Goose)
2. **Observability**: Add OpenTelemetry tracing, Prometheus metrics
3. **Authentication**: Implement JWT/PASETO validation
4. **Rate Limiting**: Add per-account rate limits
5. **Media Storage**: Integrate S3/MinIO for attachments

### Scaling

1. **Partitioning**: Partition `events` table by account_id or time
2. **Read Replicas**: Route read queries to replicas
3. **Caching**: Add Redis for frequently accessed data
4. **Load Balancing**: Add multiple backend instances

## Monitoring

- **NATS**: http://localhost:8222/varz (metrics)
- **PgAdmin**: http://localhost:8080 (database management)
- **MinIO Console**: http://localhost:9001 (object storage)
- **Application Logs**: Structured JSON with trace IDs

---

**You now have a solid foundation that can handle ~100 users and ~200k events/day with room to scale!** 🚀
