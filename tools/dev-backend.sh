#!/bin/bash
# Development helper script for running backend with infrastructure

set -euo pipefail

# Get project root directory
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${GREEN}🚀 Tennex Backend Development Helper${NC}"

# Check if Docker is running
if ! docker info >/dev/null 2>&1; then
    echo -e "${RED}❌ Docker is not running. Please start Docker Desktop first.${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Docker is running${NC}"

# Check if infrastructure containers are running
if ! docker ps --format "table {{.Names}}" | grep -q "tennex-postgres"; then
    echo -e "${YELLOW}📊 Starting infrastructure containers...${NC}"
    make docker-up
    echo -e "${YELLOW}⏳ Waiting for services to be ready...${NC}"
    sleep 8
else
    echo -e "${GREEN}✅ Infrastructure containers are running${NC}"
fi

# Check if database has our tables
echo -e "${YELLOW}🗄️  Checking database schema...${NC}"
if ! docker exec tennex-postgres psql -U tennex -d tennex -c "\dt" 2>/dev/null | grep -q "users"; then
    echo -e "${YELLOW}📊 Running database migrations...${NC}"
    make migrate-all
else
    echo -e "${GREEN}✅ Database schema is up to date${NC}"
fi

echo -e "${GREEN}🎯 Infrastructure Status:${NC}"
echo "  📊 Postgres: http://localhost:8080 (admin@tennex.com / admin123)"
echo "  🔄 NATS Monitor: http://localhost:8222"  
echo "  📦 MinIO Console: http://localhost:9001 (tennex / tennex123)"
echo ""

echo -e "${GREEN}🚀 Starting Backend Service...${NC}"
echo -e "${YELLOW}Press Ctrl+C to stop${NC}"
echo ""

cd services/backend
exec go run cmd/backend/main.go
