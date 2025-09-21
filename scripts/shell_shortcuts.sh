#!/bin/sh
# Tennex shell shortcuts - source this file to get handy functions
# Usage: . scripts/shell_shortcuts.sh

TENNX_ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
export TENNX_ROOT_DIR

# Defaults
export BRIDGE_URL="${BRIDGE_URL:-http://localhost:8080}"

# --- Docker Compose helpers ---
up() { (cd "$TENNX_ROOT_DIR" && docker compose up -d "$@"); }
down() { (cd "$TENNX_ROOT_DIR" && docker compose down --remove-orphans); }
rebuild() { (cd "$TENNX_ROOT_DIR" && docker compose build "$@" && docker compose up -d "$@"); }
logs() { (cd "$TENNX_ROOT_DIR" && docker compose logs -f "$@"); }
ps() { (cd "$TENNX_ROOT_DIR" && docker compose ps); }

# --- Service specific ---
bridge_up() { up bridge; }
bridge_rebuild() { rebuild bridge; }
bridge_logs() { logs bridge; }
bridge_shell() { (cd "$TENNX_ROOT_DIR" && docker compose exec bridge sh); }

mongo_up() { up mongodb; }
mongo_logs() { logs mongodb; }

nats_up() { up nats; }
nats_logs() { logs nats; }

# --- Scripts wrappers ---
connect_whatsapp() { (cd "$TENNX_ROOT_DIR" && ./scripts/connect-whatsapp.sh "$@"); }
demo_qr() { (cd "$TENNX_ROOT_DIR" && ./scripts/demo-qr-only.sh "$@"); }
dev_start() { (cd "$TENNX_ROOT_DIR" && ./scripts/dev-start.sh "$@"); }
dev_stop() { (cd "$TENNX_ROOT_DIR" && ./scripts/dev-stop.sh "$@"); }
quick_test() { (cd "$TENNX_ROOT_DIR" && ./scripts/quick-test.sh "$@"); }
test_api() { (cd "$TENNX_ROOT_DIR" && ./scripts/test-api.sh "$@"); }
test_local_bridge() { (cd "$TENNX_ROOT_DIR" && ./scripts/test-local-bridge.sh "$@"); }
test_qr_integration() { (cd "$TENNX_ROOT_DIR" && ./scripts/test-qr-integration.sh "$@"); }
test_qr_sizes() { (cd "$TENNX_ROOT_DIR" && ./scripts/test-qr-sizes.sh "$@"); }

# --- Curl helpers ---
health() { curl -s "$BRIDGE_URL/health" | jq .; }
ready() { curl -s "$BRIDGE_URL/ready" | jq .; }
stats() { curl -s "$BRIDGE_URL/stats" | jq .; }
debug_config() { curl -s "$BRIDGE_URL/debug/config" | jq .; }
debug_whatsapp() { curl -s "$BRIDGE_URL/debug/whatsapp" | jq .; }

connect_min() {
  cid="${1:-whatsapp-$(date +%s)}"
  curl -s -X POST "$BRIDGE_URL/connect-minimal" \
    -H 'Content-Type: application/json' \
    -d "{\"client_id\":\"$cid\"}" | jq .
}

# --- Go PoC ---
qr_poc() { (cd "$TENNX_ROOT_DIR/test" && go run .); }

# --- Help ---
shortcuts_help() {
  cat <<EOF
Tennex shortcuts loaded. Examples:
  up [svc]              # docker compose up -d
  down                  # docker compose down
  bridge_rebuild        # rebuild and restart bridge
  bridge_logs           # tail bridge logs
  connect_whatsapp      # run script to connect client and show QR
  connect_min [client]  # call POST /connect-minimal
  health|ready|stats    # quick API checks
  qr_poc                # run minimal whatsmeow PoC (prints QR)
EOF
}

echo "[tennex] Shortcuts loaded. Run 'shortcuts_help' for commands. BRIDGE_URL=$BRIDGE_URL"
