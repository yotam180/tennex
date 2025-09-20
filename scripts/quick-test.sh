#!/bin/bash
# Quick test script for Tennex Bridge without full Docker setup

set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m' # No Color

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

# Change to project root
cd "$(dirname "$0")/.."

log_info "Quick Tennex Bridge Test"

# Test 1: Build the bridge service
log_info "Test 1: Building bridge service..."
if cd services/bridge && go build ./cmd/bridge; then
    log_success "Bridge service builds successfully"
    cd ../..
else
    log_error "Bridge service failed to build"
    exit 1
fi

# Test 2: Check if MongoDB is running (for docker-compose test)
log_info "Test 2: Checking for running services..."
if curl -s http://localhost:8080/health >/dev/null 2>&1; then
    log_success "Bridge service is already running at :8080"
    BRIDGE_RUNNING=true
else
    log_warning "Bridge service is not running"
    BRIDGE_RUNNING=false
fi

if nc -z localhost 27017 2>/dev/null; then
    log_success "MongoDB is accessible on :27017"
    MONGODB_RUNNING=true
else
    log_warning "MongoDB is not running on :27017"
    MONGODB_RUNNING=false
fi

# Test 3: Test API if service is running
if [ "$BRIDGE_RUNNING" = true ]; then
    log_info "Test 3: Testing API endpoints..."
    
    # Health check
    if curl -s http://localhost:8080/health | jq . >/dev/null 2>&1; then
        log_success "Health endpoint works"
    else
        log_warning "Health endpoint returned non-JSON"
    fi
    
    # Stats endpoint
    if curl -s http://localhost:8080/stats | jq . >/dev/null 2>&1; then
        log_success "Stats endpoint works"
        curl -s http://localhost:8080/stats | jq -r '. | "Active clients: \(.active_clients // 0)"'
    else
        log_warning "Stats endpoint issues"
    fi
    
    # Test connect-client endpoint (only if MongoDB is available)
    if [ "$MONGODB_RUNNING" = true ]; then
        log_info "Testing /connect-client endpoint..."
        RESPONSE=$(curl -s -w "\nHTTP_STATUS:%{http_code}" \
            -X POST http://localhost:8080/connect-client \
            -H "Content-Type: application/json" \
            -d '{"client_id":"test-quick"}')
        
        HTTP_STATUS=$(echo "$RESPONSE" | tail -n1 | cut -d: -f2)
        BODY=$(echo "$RESPONSE" | sed '$d')
        
        if [ "$HTTP_STATUS" -eq 200 ]; then
            log_success "Connect-client endpoint works!"
            echo "$BODY" | jq .
        else
            log_warning "Connect-client endpoint returned HTTP $HTTP_STATUS"
            echo "$BODY"
        fi
    else
        log_warning "Skipping /connect-client test (MongoDB not available)"
    fi
    
else
    log_info "Test 3: Starting temporary bridge for testing..."
    log_warning "Full API test requires MongoDB. Consider running: ./scripts/dev-start.sh"
fi

# Test 4: Check dependencies for QR script
log_info "Test 4: Checking QR script dependencies..."

if command -v curl >/dev/null 2>&1; then
    log_success "curl is available"
else
    log_error "curl is required but not installed"
fi

if command -v jq >/dev/null 2>&1; then
    log_success "jq is available"
else
    log_warning "jq is not available (install with: brew install jq)"
fi

if command -v qrencode >/dev/null 2>&1; then
    log_success "qrencode is available for QR display"
else
    log_warning "qrencode not found (install with: brew install qrencode)"
fi

# Test 5: Quick QR generation test
if command -v qrencode >/dev/null 2>&1; then
    log_info "Test 5: QR code generation test..."
    echo
    log_info "Sample QR code (test data):"
    qrencode -t ansiutf8 "TEST:This is a test QR code for Tennex"
    echo
    log_success "QR code generation works!"
else
    log_info "Test 5: Skipped (qrencode not available)"
fi

echo
log_info "Summary:"
echo "  Bridge Build: ✅"
echo "  Bridge Running: $([ "$BRIDGE_RUNNING" = true ] && echo "✅" || echo "❌")"
echo "  MongoDB Running: $([ "$MONGODB_RUNNING" = true ] && echo "✅" || echo "❌")"
echo "  Dependencies: $(command -v curl >/dev/null && command -v jq >/dev/null && echo "✅" || echo "⚠️")"
echo "  QR Support: $(command -v qrencode >/dev/null && echo "✅" || echo "⚠️")"

echo
if [ "$BRIDGE_RUNNING" = false ]; then
    log_info "To start the full environment:"
    log_info "  ./scripts/dev-start.sh"
    echo
fi

log_info "To connect WhatsApp (after services are running):"
log_info "  ./scripts/connect-whatsapp.sh"
echo

log_success "Quick test completed!"
