#!/bin/bash
# Development helper script for running bridge service

set -euo pipefail

# Get project root directory
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${GREEN}🚀 Starting Tennex Bridge Development Environment...${NC}"

# Ensure Docker is running
if ! docker info &> /dev/null; then
    echo -e "${RED}❌ Docker is not running. Please start Docker Desktop and try again.${NC}"
    exit 1
fi

# Check if infrastructure is running
if ! docker ps | grep -q tennex-postgres; then
    echo -e "${YELLOW}🐳 Infrastructure not running, starting it...${NC}"
    make docker-up
    sleep 5
fi

# Build bridge service
echo -e "${YELLOW}🔨 Building bridge service...${NC}"
make build-bridge

echo -e "${GREEN}✅ Bridge service ready. Starting...${NC}"
echo -e "${YELLOW}Press Ctrl+C to stop${NC}"
echo ""

# Run bridge service
exec ./bin/bridge
