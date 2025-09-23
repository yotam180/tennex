#!/bin/bash
# Code generation script for Tennex
# Generates Go code from OpenAPI, protobuf, and SQL schemas

set -euo pipefail

echo "🔄 Starting code generation..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if required tools are installed
check_tool() {
    if ! command -v "$1" &> /dev/null; then
        echo -e "${RED}❌ $1 is not installed. Please install it first.${NC}"
        exit 1
    fi
}

echo "🔍 Checking required tools..."
check_tool "buf"
check_tool "oapi-codegen" 
check_tool "sqlc"

# Get project root
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

echo "📁 Working in: $PROJECT_ROOT"

# Generate protobuf code
echo -e "${YELLOW}🔧 Generating protobuf code...${NC}"
if [ -f "buf.yaml" ] && [ -f "pkg/proto/bridge.proto" ]; then
    buf generate
    echo -e "${GREEN}✅ Protobuf generation complete${NC}"
else
    echo -e "${YELLOW}⚠️  Skipping protobuf generation (buf.yaml or proto files not found)${NC}"
fi

# Generate OpenAPI code
echo -e "${YELLOW}🔧 Generating OpenAPI code...${NC}"
if [ -f "pkg/api/openapi.yaml" ]; then
    oapi-codegen -package api -generate types,chi-server,spec -o pkg/api/gen/api.go pkg/api/openapi.yaml
    echo -e "${GREEN}✅ OpenAPI generation complete${NC}"
else
    echo -e "${YELLOW}⚠️  Skipping OpenAPI generation (openapi.yaml not found)${NC}"
fi

# Generate sqlc code
echo -e "${YELLOW}🔧 Generating sqlc code...${NC}"
if [ -f "pkg/db/sqlc.yaml" ] && [ -d "pkg/db/queries" ]; then
    sqlc generate -f pkg/db/sqlc.yaml
    echo -e "${GREEN}✅ sqlc generation complete${NC}"
else
    echo -e "${YELLOW}⚠️  Skipping sqlc generation (sqlc.yaml or queries not found)${NC}"
fi

echo -e "${GREEN}🎉 Code generation complete!${NC}"
