#!/bin/bash
# Script to connect a WhatsApp client and display QR code in terminal

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
WHITE='\033[1;37m'
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

log_header() {
    echo -e "${PURPLE}==== $1 ====${NC}"
}

# Configuration
BRIDGE_URL=${BRIDGE_URL:-"http://localhost:8080"}
CLIENT_ID=${CLIENT_ID:-"whatsapp-$(date +%s)"}

# Check if required tools are available
check_dependencies() {
    local missing_deps=()
    
    if ! command -v curl >/dev/null 2>&1; then
        missing_deps+=("curl")
    fi
    
    if ! command -v jq >/dev/null 2>&1; then
        missing_deps+=("jq")
    fi
    
    if [ ${#missing_deps[@]} -gt 0 ]; then
        log_error "Missing required dependencies: ${missing_deps[*]}"
        log_info "Please install missing dependencies:"
        for dep in "${missing_deps[@]}"; do
            case $dep in
                "curl")
                    log_info "  - macOS: brew install curl"
                    log_info "  - Ubuntu/Debian: apt install curl"
                    ;;
                "jq")
                    log_info "  - macOS: brew install jq"
                    log_info "  - Ubuntu/Debian: apt install jq"
                    ;;
            esac
        done
        exit 1
    fi
}

# Check if QR code tools are available
check_qr_tools() {
    if command -v qrencode >/dev/null 2>&1; then
        QR_TOOL="qrencode"
        log_info "Using qrencode for QR display"
    else
        QR_TOOL="none"
        log_warning "qrencode not found. QR code will be shown as text only"
        log_info "To display QR codes visually, install qrencode:"
        log_info "  - macOS: brew install qrencode"
        log_info "  - Ubuntu/Debian: apt install qrencode"
    fi
}

# Display QR code in terminal
display_qr() {
    local qr_data="$1"
    
    if [ "$QR_TOOL" = "qrencode" ]; then
        log_header "SCAN THIS QR CODE WITH WHATSAPP"
        echo
        log_info "${CYAN}Optimized QR code size for WhatsApp scanning:${NC}"
        echo
        # Generate smaller, WhatsApp-friendly QR code
        # -s 3: module size 3 (smaller than default)
        # -m 1: margin of 1 module (smaller border)
        # -l L: Low error correction (smaller code)
        qrencode -t ansiutf8 -s 3 -m 1 -l L "$qr_data"
        echo
        log_info "${CYAN}📱 Scan with WhatsApp mobile app:${NC}"
        log_info "   1. Open WhatsApp"
        log_info "   2. Settings → Linked Devices"  
        log_info "   3. Link a Device → Scan QR"
        log_info "   4. Point camera at the QR code above"
        echo
        log_info "${YELLOW}💡 Tip: If QR is still too big, adjust your terminal font size${NC}"
    else
        log_header "QR CODE DATA (Install qrencode for visual display)"
        echo
        echo -e "${WHITE}$qr_data${NC}"
        echo
        log_info "You can generate a QR code at: https://qr.io/"
        log_info "Or install qrencode: brew install qrencode (macOS) / apt install qrencode (Ubuntu)"
    fi
}

# Test bridge service connectivity
test_bridge_connection() {
    log_info "Testing bridge service connectivity..."
    
    if ! curl -s -f "$BRIDGE_URL/health" >/dev/null; then
        log_error "Bridge service is not accessible at $BRIDGE_URL"
        log_info "Make sure the service is running:"
        log_info "  - Docker: ./scripts/dev-start.sh"
        log_info "  - Local: cd services/bridge && make run"
        exit 1
    fi
    
    log_success "Bridge service is accessible"
}

# Connect WhatsApp client
connect_client() {
    log_info "Connecting WhatsApp client with ID: $CLIENT_ID"
    
    local response
    response=$(curl -s -w "\nHTTP_STATUS:%{http_code}" \
        -X POST "$BRIDGE_URL/connect-minimal" \
        -H "Content-Type: application/json" \
        -d "{\"client_id\":\"$CLIENT_ID\"}")
    
    local http_status
    http_status=$(echo "$response" | tail -n1 | cut -d: -f2)
    local body
    body=$(echo "$response" | sed '$d')
    
    if [ "$http_status" -ne 200 ]; then
        log_error "Failed to connect client (HTTP $http_status)"
        echo "$body" | jq . 2>/dev/null || echo "$body"
        exit 1
    fi
    
    log_success "Client connection request successful"
    
    # Parse response
    local session_id qr_code status expires_at
    session_id=$(echo "$body" | jq -r '.session_id')
    qr_code=$(echo "$body" | jq -r '.qr_code')
    status=$(echo "$body" | jq -r '.status')
    expires_at=$(echo "$body" | jq -r '.expires_at')
    
    log_info "Session ID: $session_id"
    log_info "Status: $status"
    log_info "Expires: $expires_at"
    
    # Display QR code
    display_qr "$qr_code"
    
    # Store session info for potential later use
    echo "$session_id" > /tmp/tennex_last_session_id
    
    return 0
}

# Monitor connection status
monitor_connection() {
    local session_id="$1"
    
    if [ -z "$session_id" ] && [ -f /tmp/tennex_last_session_id ]; then
        session_id=$(cat /tmp/tennex_last_session_id)
    fi
    
    if [ -z "$session_id" ]; then
        log_error "No session ID provided or found"
        return 1
    fi
    
    log_info "Monitoring connection status for session: $session_id"
    log_info "Press Ctrl+C to stop monitoring"
    
    local check_count=0
    while true; do
        check_count=$((check_count + 1))
        
        # Check service stats to see if client is connected
        local stats_response
        if stats_response=$(curl -s "$BRIDGE_URL/stats"); then
            local active_clients
            active_clients=$(echo "$stats_response" | jq -r '.active_clients // 0')
            
            if [ "$active_clients" -gt 0 ]; then
                log_success "🎉 WhatsApp client connected! Active clients: $active_clients"
                log_info "You can now use the WhatsApp bridge API"
                break
            fi
        fi
        
        printf "\r${BLUE}[INFO]${NC} Waiting for WhatsApp scan... (check $check_count)"
        sleep 2
        
        # Timeout after 5 minutes
        if [ $check_count -gt 150 ]; then
            echo
            log_warning "Timeout waiting for WhatsApp connection"
            log_info "QR code may have expired. Try running the script again."
            break
        fi
    done
    
    echo
}

# Show usage
show_usage() {
    echo "Usage: $0 [OPTIONS]"
    echo
    echo "Options:"
    echo "  -c, --client-id ID    Set custom client ID (default: whatsapp-timestamp)"
    echo "  -u, --url URL         Set bridge service URL (default: http://localhost:8080)"
    echo "  -m, --monitor         Monitor connection status after QR display"
    echo "  -h, --help            Show this help message"
    echo
    echo "Examples:"
    echo "  $0                                    # Connect with default settings"
    echo "  $0 -c my-phone -m                    # Connect with custom ID and monitor"
    echo "  $0 -u http://remote:8080             # Connect to remote bridge"
    echo
    echo "Environment Variables:"
    echo "  CLIENT_ID             Custom client identifier"
    echo "  BRIDGE_URL            Bridge service URL"
}

# Parse command line arguments
MONITOR=false
while [[ $# -gt 0 ]]; do
    case $1 in
        -c|--client-id)
            CLIENT_ID="$2"
            shift 2
            ;;
        -u|--url)
            BRIDGE_URL="$2"
            shift 2
            ;;
        -m|--monitor)
            MONITOR=true
            shift
            ;;
        -h|--help)
            show_usage
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"
            show_usage
            exit 1
            ;;
    esac
done

# Main execution
main() {
    log_header "TENNEX WHATSAPP CONNECTION SCRIPT"
    
    echo -e "${CYAN}This script will help you connect a WhatsApp account to the Tennex bridge${NC}"
    echo -e "${CYAN}Client ID: $CLIENT_ID${NC}"
    echo -e "${CYAN}Bridge URL: $BRIDGE_URL${NC}"
    echo
    
    check_dependencies
    check_qr_tools
    test_bridge_connection
    connect_client
    
    if [ "$MONITOR" = true ]; then
        echo
        monitor_connection
    else
        echo
        log_info "Connection initiated! Use -m flag to monitor connection status"
        log_info "Check service stats: curl $BRIDGE_URL/stats | jq"
    fi
    
    log_success "Script completed successfully!"
}

# Run main function
main "$@"
