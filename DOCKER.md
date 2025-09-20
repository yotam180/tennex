# Tennex Docker Setup

This document describes how to run Tennex using Docker and Docker Compose for development and production environments.

## Quick Start (Development)

1. **Start the development environment:**
   ```bash
   ./scripts/dev-start.sh
   ```

2. **Test the API:**
   ```bash
   ./scripts/test-api.sh
   ```

3. **Stop the environment:**
   ```bash
   ./scripts/dev-stop.sh
   ```

## Architecture Overview

The Docker setup includes the following services:

- **Bridge Service** (Port 8080): Multi-tenant WhatsApp bridge
- **MongoDB** (Port 27017): Database for client sessions and events
- **Mongo Express** (Port 8081): MongoDB admin interface
- **NATS** (Port 4222, 8222): Message queue for future event distribution

## Development Environment

### Configuration

The development environment uses:
- `docker-compose.yml` - Base configuration
- `docker-compose.override.yml` - Development-specific overrides

### Starting Services

**Option 1: Use the helper script (recommended)**
```bash
./scripts/dev-start.sh
```

**Option 2: Manual startup**
```bash
# Set build variables
export BUILD_TIME=$(date -u '+%Y-%m-%d %H:%M:%S UTC')
export GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Start services
docker-compose up --build -d
```

### Accessing Services

| Service | URL | Credentials |
|---------|-----|-------------|
| Bridge API | http://localhost:8080 | None |
| Health Check | http://localhost:8080/health | None |
| Stats | http://localhost:8080/stats | None |
| Debug Info | http://localhost:8080/debug/config | None |
| MongoDB | mongodb://localhost:27017 | admin:password123 |
| Mongo Express | http://localhost:8081 | admin:admin123 |
| NATS | nats://localhost:4222 | None |
| NATS Monitor | http://localhost:8222 | None |

### Testing the API

**Connect a WhatsApp client:**
```bash
curl -X POST http://localhost:8080/connect-client \
  -H "Content-Type: application/json" \
  -d '{"client_id":"test123"}'
```

**Check service stats:**
```bash
curl http://localhost:8080/stats | jq
```

**Run comprehensive tests:**
```bash
./scripts/test-api.sh
```

### Managing Services

```bash
# View logs
docker-compose logs -f bridge
docker-compose logs -f mongodb

# Restart a service
docker-compose restart bridge

# Get shell access
docker-compose exec bridge sh
docker-compose exec mongodb mongosh

# Stop services
docker-compose down

# Stop and remove data
docker-compose down --volumes
```

## Production Environment

### Configuration

Production environment uses:
- `docker-compose.prod.yml` - Production configuration
- Docker secrets for sensitive data
- Optimized resource limits and security settings

### Setup

1. **Create secrets directory:**
   ```bash
   mkdir -p secrets
   ```

2. **Create secret files:**
   ```bash
   echo "admin" > secrets/mongo_root_username.txt
   echo "your-secure-password" > secrets/mongo_root_password.txt
   echo "mongodb://admin:your-secure-password@mongodb:27017/tennex?authSource=admin" > secrets/mongodb_uri.txt
   ```

3. **Set production variables:**
   ```bash
   export VERSION=v1.0.0
   export BUILD_TIME=$(date -u '+%Y-%m-%d %H:%M:%S UTC')
   export GIT_COMMIT=$(git rev-parse --short HEAD)
   ```

4. **Start production environment:**
   ```bash
   docker-compose -f docker-compose.prod.yml up -d
   ```

### Security Considerations

- Services bind to localhost only (except bridge on 8080)
- Non-root user in containers
- Docker secrets for sensitive data
- Resource limits and health checks
- Structured logging with rotation

## Environment Variables

### Bridge Service

| Variable | Default | Description |
|----------|---------|-------------|
| `TENNEX_BRIDGE_HTTP_PORT` | 8080 | HTTP server port |
| `TENNEX_BRIDGE_LOG_LEVEL` | info | Log level (debug, info, warn, error) |
| `TENNEX_BRIDGE_MONGODB_URI` | - | MongoDB connection string |
| `TENNEX_BRIDGE_MONGODB_DATABASE` | tennex | Database name |
| `TENNEX_BRIDGE_NATS_URLS` | nats://nats:4222 | NATS server URLs |
| `TENNEX_BRIDGE_WHATSAPP_SESSION_PATH` | /app/sessions | Session storage path |
| `TENNEX_BRIDGE_DEV_ENABLE_PPROF` | false | Enable pprof endpoints |
| `TENNEX_BRIDGE_DEV_ENABLE_METRICS` | true | Enable metrics |
| `TENNEX_ENV` | development | Environment (development, production) |

## Data Persistence

### Development Volumes

- `mongodb_data`: MongoDB data files
- `bridge_sessions`: WhatsApp session data
- `bridge_logs`: Application logs

### Backup and Restore

**Backup MongoDB:**
```bash
docker-compose exec mongodb mongodump --out /backup
docker cp $(docker-compose ps -q mongodb):/backup ./backup
```

**Restore MongoDB:**
```bash
docker cp ./backup $(docker-compose ps -q mongodb):/backup
docker-compose exec mongodb mongorestore /backup
```

**Backup Sessions:**
```bash
docker cp $(docker-compose ps -q bridge):/app/sessions ./sessions-backup
```

## Troubleshooting

### Common Issues

1. **Port conflicts:**
   ```bash
   # Check what's using the ports
   lsof -i :8080
   lsof -i :27017
   
   # Change ports in docker-compose.override.yml if needed
   ```

2. **MongoDB connection issues:**
   ```bash
   # Check MongoDB logs
   docker-compose logs mongodb
   
   # Test connection
   docker-compose exec mongodb mongosh --eval "db.runCommand('ping')"
   ```

3. **Bridge service not starting:**
   ```bash
   # Check logs
   docker-compose logs bridge
   
   # Rebuild with no cache
   docker-compose build --no-cache bridge
   ```

4. **Out of disk space:**
   ```bash
   # Clean up Docker
   docker system prune -a
   
   # Remove unused volumes
   docker volume prune
   ```

### Debug Mode

Enable debug logging:
```bash
# Edit docker-compose.override.yml
TENNEX_BRIDGE_LOG_LEVEL: debug

# Restart service
docker-compose restart bridge
```

### Health Checks

All services include health checks:
```bash
# Check health status
docker-compose ps

# Manual health check
curl http://localhost:8080/health
```

## Development Workflow

1. **Make code changes** in `services/bridge/`
2. **Rebuild and restart:**
   ```bash
   docker-compose build bridge
   docker-compose restart bridge
   ```
3. **Test changes:**
   ```bash
   ./scripts/test-api.sh
   ```
4. **View logs:**
   ```bash
   docker-compose logs -f bridge
   ```

## Performance Tuning

### Resource Limits

Edit docker-compose files to adjust:
- Memory limits
- CPU limits  
- Database connections
- Log retention

### Monitoring

- Use `/metrics` endpoint for Prometheus
- Use `/stats` endpoint for runtime statistics
- Monitor container resources with `docker stats`
- Check health endpoints regularly

## Next Steps

After getting the Docker environment running:

1. **Test WhatsApp connectivity** by scanning QR codes
2. **Implement event distribution** via NATS
3. **Add client application** for message management
4. **Set up monitoring** with Prometheus/Grafana
5. **Configure backup strategy** for production data
