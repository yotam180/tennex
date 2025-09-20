#!/bin/bash
# Tennex Development Environment Shutdown Script

set -e

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

# Change to project root
cd "$(dirname "$0")/.."

log_info "Stopping Tennex Development Environment..."

# Stop and remove containers
docker-compose down --remove-orphans

# Optionally remove volumes (uncomment to clear all data)
# docker-compose down --volumes

log_success "Tennex Development Environment stopped!"
log_info "To remove all data as well, run: docker-compose down --volumes"
