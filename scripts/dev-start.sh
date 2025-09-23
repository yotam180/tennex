#!/bin/bash
# Tennex Development Environment Startup Script

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Change to deployments/local directory for docker-compose
cd "$(dirname "$0")/../deployments/local"

log_info "Starting Tennex Development Environment..."

# Check if Docker is running
if ! docker info >/dev/null 2>&1; then
    log_error "Docker is not running. Please start Docker and try again."
    exit 1
fi

# Check if docker compose is available
if ! docker compose version >/dev/null 2>&1; then
    log_error "docker compose is not available. Please install Docker Compose v2."
    exit 1
fi

# Set build arguments
export BUILD_TIME=$(date -u '+%Y-%m-%d %H:%M:%S UTC')
export GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
export VERSION="dev-$(date +%Y%m%d-%H%M%S)"

log_info "Build information:"
log_info "  Version: $VERSION"
log_info "  Build Time: $BUILD_TIME"
log_info "  Git Commit: $GIT_COMMIT"

# Stop any existing containers
log_info "Stopping existing containers..."
docker compose down --remove-orphans

# Build and start services
log_info "Building and starting services..."
docker compose up --build -d

# Wait for services to be healthy
log_info "Waiting for services to start..."

# Wait for PostgreSQL
log_info "Waiting for PostgreSQL..."
timeout=60
counter=0
while ! docker compose exec -T postgres pg_isready -U tennex -d tennex >/dev/null 2>&1; do
    if [ $counter -ge $timeout ]; then
        log_error "PostgreSQL did not start within ${timeout} seconds"
        docker compose logs postgres
        exit 1
    fi
    sleep 1
    counter=$((counter + 1))
done
log_success "PostgreSQL is ready!"

# Wait for Bridge Service
log_info "Waiting for Bridge Service..."
timeout=120
counter=0
while ! curl -s http://localhost:8080/health >/dev/null; do
    if [ $counter -ge $timeout ]; then
        log_error "Bridge Service did not start within ${timeout} seconds"
        docker compose logs bridge
        exit 1
    fi
    sleep 2
    counter=$((counter + 2))
done
log_success "Bridge Service is ready!"

# Show service status
log_info "Service Status:"
docker compose ps

log_success "Tennex Development Environment is ready!"
echo
log_info "Available services:"
log_info "  Bridge Service:    http://localhost:8080"
log_info "  - Health Check:    http://localhost:8080/health"
log_info "  - API Stats:       http://localhost:8080/stats"
log_info "  - Connect Client:  POST http://localhost:8080/connect-minimal"
log_info "  - Debug Info:      http://localhost:8080/debug/config"
log_info ""
log_info "  PostgreSQL:        postgres://tennex:tennex123@localhost:5432/tennex"
echo
log_info "Useful commands:"
log_info "  View logs:         docker compose logs -f [service]"
log_info "  Stop services:     docker compose down"
log_info "  Restart service:   docker compose restart [service]"
log_info "  Shell access:      docker compose exec [service] sh"
echo
log_info "To test the API, try:"
log_info "  curl -X POST http://localhost:8080/connect-minimal -H 'Content-Type: application/json' -d '{\"client_id\":\"test123\"}'"
