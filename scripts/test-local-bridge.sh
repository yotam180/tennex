#!/bin/bash
# Test script for running bridge locally without Docker

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
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

log_header() {
    echo -e "${PURPLE}==== $1 ====${NC}"
}

# Change to project root
cd "$(dirname "$0")/.."

log_header "LOCAL BRIDGE SERVICE TEST"

# Check if MongoDB is available
if ! nc -z localhost 27017 2>/dev/null; then
    log_warning "MongoDB not available. Starting with Docker Compose..."
    log_info "Starting only MongoDB and NATS..."
    docker-compose up -d mongodb nats
    sleep 10
fi

# Build the bridge service
log_info "Building bridge service..."
cd services/bridge
if ! go build -o ../../bridge ./cmd/bridge; then
    log_error "Failed to build bridge service"
    exit 1
fi
cd ../..

log_success "Bridge service built successfully"

# Set environment variables for local run
export TENNEX_BRIDGE_HTTP_PORT=8080
export TENNEX_BRIDGE_LOG_LEVEL=info
export TENNEX_BRIDGE_MONGODB_URI="mongodb://admin:password123@localhost:27017/tennex?authSource=admin"
export TENNEX_BRIDGE_MONGODB_DATABASE="tennex"
export TENNEX_BRIDGE_WHATSAPP_SESSION_PATH="./sessions"
export TENNEX_BRIDGE_DEV_ENABLE_PPROF=true
export TENNEX_BRIDGE_DEV_ENABLE_METRICS=true
export TENNEX_BRIDGE_DEV_QR_IN_TERMINAL=false
export TENNEX_ENV=development

# Create session directory
mkdir -p sessions

log_info "Starting bridge service locally..."
log_info "Environment configured:"
log_info "  Port: $TENNEX_BRIDGE_HTTP_PORT"
log_info "  MongoDB: $TENNEX_BRIDGE_MONGODB_URI"
log_info "  Sessions: $TENNEX_BRIDGE_WHATSAPP_SESSION_PATH"

# Start bridge service in background
./bridge &
BRIDGE_PID=$!

# Function to cleanup on exit
cleanup() {
    log_info "Cleaning up..."
    if [ ! -z "$BRIDGE_PID" ]; then
        kill $BRIDGE_PID 2>/dev/null || true
    fi
    log_info "Bridge service stopped"
}
trap cleanup EXIT

# Wait for service to start
log_info "Waiting for bridge service to start..."
for i in {1..30}; do
    if curl -s http://localhost:8080/health >/dev/null 2>&1; then
        break
    fi
    sleep 1
    if [ $i -eq 30 ]; then
        log_error "Bridge service failed to start within 30 seconds"
        exit 1
    fi
done

log_success "Bridge service started successfully!"

# Test health endpoint
log_info "Testing health endpoint..."
HEALTH_RESPONSE=$(curl -s http://localhost:8080/health)
echo "$HEALTH_RESPONSE" | jq . || echo "$HEALTH_RESPONSE"

# Test stats endpoint
log_info "Testing stats endpoint..."
STATS_RESPONSE=$(curl -s http://localhost:8080/stats)
echo "$STATS_RESPONSE" | jq . || echo "$STATS_RESPONSE"

# Test connect-client endpoint
log_header "TESTING CONNECT-CLIENT ENDPOINT"
CLIENT_ID="test-local-$(date +%s)"

log_info "Connecting client: $CLIENT_ID"

CONNECT_RESPONSE=$(curl -s -w "\nHTTP_STATUS:%{http_code}" \
    -X POST http://localhost:8080/connect-client \
    -H "Content-Type: application/json" \
    -d "{\"client_id\":\"$CLIENT_ID\"}")

HTTP_STATUS=$(echo "$CONNECT_RESPONSE" | tail -n1 | cut -d: -f2)
BODY=$(echo "$CONNECT_RESPONSE" | sed '$d')

if [ "$HTTP_STATUS" -eq 200 ]; then
    log_success "✅ Client connection successful!"
    
    echo "$BODY" | jq . 2>/dev/null || echo "$BODY"
    
    # Extract QR code
    QR_CODE=$(echo "$BODY" | jq -r '.qr_code' 2>/dev/null || echo "")
    SESSION_ID=$(echo "$BODY" | jq -r '.session_id' 2>/dev/null || echo "")
    
    if [ ! -z "$QR_CODE" ] && [ "$QR_CODE" != "null" ]; then
        log_header "WHATSAPP QR CODE"
        
        if command -v qrencode >/dev/null 2>&1; then
            log_info "Scan this QR code with WhatsApp:"
            echo
            qrencode -t ansiutf8 "$QR_CODE"
            echo
            log_info "Instructions:"
            log_info "1. Open WhatsApp on your phone"
            log_info "2. Go to Settings > Linked Devices"
            log_info "3. Tap 'Link a Device'"
            log_info "4. Scan the QR code above"
        else
            log_warning "qrencode not found. QR code data:"
            echo "$QR_CODE"
        fi
        
        echo
        log_info "Session ID: $SESSION_ID"
        log_info "Monitoring for connection... (Press Ctrl+C to stop)"
        
        # Monitor for connection
        for i in {1..60}; do
            STATS_RESPONSE=$(curl -s http://localhost:8080/stats 2>/dev/null || echo "{}")
            ACTIVE_CLIENTS=$(echo "$STATS_RESPONSE" | jq -r '.active_clients // 0' 2>/dev/null || echo "0")
            
            if [ "$ACTIVE_CLIENTS" -gt 0 ]; then
                echo
                log_success "🎉 WhatsApp client connected!"
                log_success "Active clients: $ACTIVE_CLIENTS"
                break
            fi
            
            printf "\r${BLUE}[INFO]${NC} Waiting for WhatsApp scan... (${i}/60)"
            sleep 1
        done
        echo
        
    else
        log_warning "No QR code in response"
    fi
    
else
    log_error "❌ Client connection failed (HTTP $HTTP_STATUS)"
    echo "$BODY"
fi

echo
log_header "TEST COMPLETED"
log_info "Bridge service will stop when you press Ctrl+C"

# Keep running until interrupted
wait $BRIDGE_PID
