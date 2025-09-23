#!/bin/bash
# Linting and static analysis for Tennex

set -euo pipefail

echo "🔍 Running linters and static analysis..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Get project root
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

echo "📁 Working in: $PROJECT_ROOT"

# Run go vet
echo -e "${YELLOW}🔧 Running go vet...${NC}"
if go vet ./...; then
    echo -e "${GREEN}✅ go vet passed${NC}"
else
    echo -e "${RED}❌ go vet failed${NC}"
    exit 1
fi

# Run go fmt check
echo -e "${YELLOW}🔧 Checking go fmt...${NC}"
if [ -z "$(gofmt -l .)" ]; then
    echo -e "${GREEN}✅ go fmt check passed${NC}"
else
    echo -e "${RED}❌ Files need formatting:${NC}"
    gofmt -l .
    echo -e "${YELLOW}Run: go fmt ./...${NC}"
    exit 1
fi

# Run staticcheck if available
if command -v staticcheck &> /dev/null; then
    echo -e "${YELLOW}🔧 Running staticcheck...${NC}"
    if staticcheck ./...; then
        echo -e "${GREEN}✅ staticcheck passed${NC}"
    else
        echo -e "${RED}❌ staticcheck failed${NC}"
        exit 1
    fi
else
    echo -e "${YELLOW}⚠️  staticcheck not installed, skipping${NC}"
fi

# Run golangci-lint if available
if command -v golangci-lint &> /dev/null; then
    echo -e "${YELLOW}🔧 Running golangci-lint...${NC}"
    if golangci-lint run; then
        echo -e "${GREEN}✅ golangci-lint passed${NC}"
    else
        echo -e "${RED}❌ golangci-lint failed${NC}"
        exit 1
    fi
else
    echo -e "${YELLOW}⚠️  golangci-lint not installed, skipping${NC}"
fi

echo -e "${GREEN}🎉 All linters passed!${NC}"
