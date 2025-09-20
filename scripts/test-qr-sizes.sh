#!/bin/bash
# Test different QR code sizes to find the optimal WhatsApp-scannable size

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_header() {
    echo -e "${PURPLE}==== $1 ====${NC}"
}

# Test QR data (realistic WhatsApp format)
TEST_QR="1@s.whatsapp.net,s4l6A8B9C2d3E4f5G6h7I8j9K0l1M2n3O4p5Q6r7S8t9U0v1W2x3Y4z5A6B7C8D9E0F1G2H3I4J5K6L7M8N9O0P1Q2R3S4T5U6V7W8X9Y0Z1a2b3c4d5e6f7g8h9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z7A8B9C0D1E2F3G4H5I6J7K8L9M0N1O2P3Q4R5S6T7U8V9W0X1Y2Z3a4b5c6d7e8f9g0h1i2j3k4l5m6n7o8p9q0r1s2t3u4v5w6x7y8z9A0B1C2D3E4F5G6H7I8J9K0L1M2N3O4P5Q6R7S8T9U0V1W2X3Y4Z5a6b7c8d9e0f1g2h3i4j5k6l7m8n9o0p1q2r3s4t5u6v7w8x9y0z1A2B3C4D3E4F5G6H7I8J9K0L1M2,ARjnnJIjW2Q+6SV/ZtjV+uCKt4o="

log_header "QR CODE SIZE TESTING FOR WHATSAPP"

if ! command -v qrencode >/dev/null 2>&1; then
    log_info "Installing qrencode..."
    if command -v brew >/dev/null 2>&1; then
        brew install qrencode
    else
        echo "Please install qrencode manually"
        exit 1
    fi
fi

echo
log_info "${CYAN}Testing different QR code sizes for optimal WhatsApp scanning...${NC}"
echo

# Test 1: Default size (usually too big)
log_header "1. DEFAULT SIZE (Usually too big for WhatsApp)"
echo
qrencode -t ansiutf8 "$TEST_QR"
echo
read -p "Press Enter to see smaller versions..."

clear

# Test 2: Small module size
log_header "2. SMALL MODULES (Recommended for WhatsApp)"
echo
log_info "${GREEN}Parameters: -s 3 -m 1 -l L${NC}"
log_info "  -s 3: Small module size (3x3 pixels per module)"
log_info "  -m 1: Minimal border (1 module width)"
log_info "  -l L: Low error correction (smaller QR code)"
echo
qrencode -t ansiutf8 -s 3 -m 1 -l L "$TEST_QR"
echo
read -p "Press Enter to see very small version..."

clear

# Test 3: Very small
log_header "3. VERY SMALL (Minimal size)"
echo
log_info "${GREEN}Parameters: -s 2 -m 1 -l L${NC}"
echo
qrencode -t ansiutf8 -s 2 -m 1 -l L "$TEST_QR"
echo
read -p "Press Enter to see medium version..."

clear

# Test 4: Medium with higher error correction
log_header "4. MEDIUM WITH ERROR CORRECTION (Balance)"
echo
log_info "${GREEN}Parameters: -s 4 -m 2 -l M${NC}"
log_info "  -l M: Medium error correction (better reliability)"
echo
qrencode -t ansiutf8 -s 4 -m 2 -l M "$TEST_QR"
echo
read -p "Press Enter for recommendations..."

clear

log_header "RECOMMENDATIONS FOR WHATSAPP QR CODES"
echo
log_info "${GREEN}✅ BEST FOR WHATSAPP:${NC} qrencode -t ansiutf8 -s 3 -m 1 -l L"
log_info "   • Small enough for phone screens"
log_info "   • Fast to scan"
log_info "   • Reliable for WhatsApp data format"
echo
log_info "${YELLOW}⚠️  FALLBACK:${NC} qrencode -t ansiutf8 -s 2 -m 1 -l L"
log_info "   • Use if primary is still too big"
log_info "   • May be harder to scan in poor lighting"
echo
log_info "${RED}❌ AVOID:${NC} Default qrencode settings"
log_info "   • Usually too large for mobile screens"
log_info "   • WhatsApp may have trouble focusing"
echo

log_header "IMPLEMENTATION"
echo
log_info "Update your scripts with:"
echo
cat << 'EOF'
# WhatsApp-optimized QR generation
qrencode -t ansiutf8 -s 3 -m 1 -l L "$qr_code_data"
EOF

echo
log_info "${CYAN}💡 Pro tips:${NC}"
log_info "• Test with your specific phone and lighting conditions"
log_info "• Adjust terminal font size if needed"
log_info "• Some terminals display QR codes better than others"
log_info "• Keep phone 6-12 inches from screen for optimal scanning"
echo

log_header "TESTING WITH REAL WHATSAPP"
echo
log_info "To test with actual WhatsApp QR scanning:"
log_info "1. Start your bridge service"
log_info "2. Call POST /connect-client"
log_info "3. Use the optimized QR display"
log_info "4. Test scanning with WhatsApp camera"
echo
log_info "${GREEN}Happy QR scanning! 📱${NC}"
