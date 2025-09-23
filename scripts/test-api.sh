#!/bin/bash
# Test script for Tennex Bridge API

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

# Configuration
BRIDGE_URL=${BRIDGE_URL:-"http://localhost:8080"}
CLIENT_ID=${CLIENT_ID:-"test-client-$(date +%s)"}

log_info "Testing Tennex Bridge API"
log_info "Bridge URL: $BRIDGE_URL"
log_info "Client ID: $CLIENT_ID"
echo

# Test 1: Health Check
log_info "Test 1: Health Check"
if curl -s -f "$BRIDGE_URL/health" >/dev/null; then
    log_success "Health check passed"
    curl -s "$BRIDGE_URL/health" | jq .
else
    log_error "Health check failed"
    exit 1
fi
echo

# Test 2: Service Stats
log_info "Test 2: Service Stats"
if curl -s -f "$BRIDGE_URL/stats" >/dev/null; then
    log_success "Stats endpoint accessible"
    curl -s "$BRIDGE_URL/stats" | jq .
else
    log_error "Stats endpoint failed"
fi
echo

# Test 3: Debug Config
log_info "Test 3: Debug Configuration"
if curl -s -f "$BRIDGE_URL/debug/config" >/dev/null; then
    log_success "Debug config endpoint accessible"
    curl -s "$BRIDGE_URL/debug/config" | jq .
else
    log_warning "Debug config endpoint not accessible"
fi
echo

# Test 4: Connect Client (Updated to use connect-minimal)
log_info "Test 4: Connect Client"
RESPONSE=$(curl -s -w "\nHTTP_STATUS:%{http_code}" \
    -X POST "$BRIDGE_URL/connect-minimal" \
    -H "Content-Type: application/json" \
    -d "{\"client_id\":\"$CLIENT_ID\"}")

HTTP_STATUS=$(echo "$RESPONSE" | tail -n1 | cut -d: -f2)
BODY=$(echo "$RESPONSE" | sed '$d')

if [ "$HTTP_STATUS" -eq 200 ]; then
    log_success "Client connection initiated successfully"
    echo "$BODY" | jq .
    
    # Extract session info
    SESSION_ID=$(echo "$BODY" | jq -r '.session_id')
    QR_CODE=$(echo "$BODY" | jq -r '.qr_code')
    
    log_info "Session ID: $SESSION_ID"
    log_info "QR Code: ${QR_CODE:0:50}..."
    
else
    log_error "Client connection failed (HTTP $HTTP_STATUS)"
    echo "$BODY"
fi
echo

# Test 5: Service Readiness
log_info "Test 5: Service Readiness"
if curl -s -f "$BRIDGE_URL/ready" >/dev/null; then
    log_success "Service is ready"
    curl -s "$BRIDGE_URL/ready" | jq .
else
    log_warning "Service readiness check failed"
    curl -s "$BRIDGE_URL/ready" | jq . || true
fi
echo

log_success "API tests completed!"

# Show additional information
echo
log_info "Additional Information:"
log_info "  To connect a WhatsApp account, scan the QR code above with your phone"
log_info "  Monitor logs with: docker compose logs -f bridge"
log_info "  Check PostgreSQL data: docker compose exec postgres psql -U tennex -d tennex"
log_info "  View service metrics: $BRIDGE_URL/metrics"
