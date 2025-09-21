#!/bin/sh
# Tennex shell shortcuts - source this file to get handy functions
# Usage: . scripts/shell_shortcuts.sh

TENNX_ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
export TENNX_ROOT_DIR

# Defaults
export BRIDGE_URL="${BRIDGE_URL:-http://localhost:8080}"
export DEFAULT_SERVICE="bridge"

# [tx] - tennex - navigate to the Tennex home directory
function tx() {
    cd $TENNX_ROOT_DIR
}

# [txrss] - tennex refresh shell shortcuts - refresh the shell shortcuts
function txrss() {
    source $TENNX_ROOT_DIR/scripts/shell_shortcuts.sh
}

# [txb] - tennex build - build a development container using docker-compose
function txb() {
    (
        cd $TENNX_ROOT_DIR/deployments/local
        
        SERVICE_NAME=$1
        if [ -z "$SERVICE_NAME" ]; then
            docker compose build
        else
            docker compose build $SERVICE_NAME
        fi
    )
}

# [txup] - tennex up - start the development environment locally using docker-compose
function txup() {
    (
        cd $TENNX_ROOT_DIR/deployments/local
        
        SERVICE_NAME=$1
        if [ -z "$SERVICE_NAME" ]; then
            docker compose up -d
        else
            docker compose up -d $SERVICE_NAME
        fi
    )
}

# [txrestart] - tennex restart - restart the development environment using docker-compose
function txrestart() {
    (
        cd $TENNX_ROOT_DIR/deployments/local
        SERVICE_NAME=$1
        if [ -z "$SERVICE_NAME" ]; then
            txdown
            txup
        else
            txdown $SERVICE_NAME
            txup $SERVICE_NAME
        fi
    )
}

# [txdown] - tennex down - stop the development environment using docker-compose
function txdown() {
    (
        cd $TENNX_ROOT_DIR/deployments/local
        SERVICE_NAME=$1
        if [ -z "$SERVICE_NAME" ]; then
            docker compose down --remove-orphans
        else
            docker compose down $SERVICE_NAME
        fi
    )
}

# [txps] - tennex ps - show the status of the development environment using docker-compose
function txps() {
    (
        cd $TENNX_ROOT_DIR/deployments/local
        docker compose ps
    )
}

# [txl] - tennex logs - show the logs of the development environment using docker-compose
function txl() {
    (
        cd $TENNX_ROOT_DIR/deployments/local
        
        if [ -z "$1" ]; then
            SERVICE_NAME=$DEFAULT_SERVICE
        else
            if [[ ! "$1" =~ ^- ]]; then
                SERVICE_NAME=$1
                shift
            else
                SERVICE_NAME=$DEFAULT_SERVICE
            fi
        fi

        DOCKER_ID=$(docker compose ps $SERVICE_NAME -q)
        if [ -z "$DOCKER_ID" ]; then
            # For dockers that failed and are not currently running
            docker compose logs $SERVICE_NAME -f --tail 200 2>&1
        else
            docker logs $DOCKER_ID -f --tail 200 2>&1
        fi
    )
}

# [txbrl] - tennex build, run, logs - build the service, run it, and show the logs
function txbrl() {
    SERVICE_NAME=${1:-$DEFAULT_SERVICE}
    txb $SERVICE_NAME || return $?
    txup $SERVICE_NAME || return $?
    txl $SERVICE_NAME || return $?
}

# [txrl] - tennex restart, logs - refresh a service and show the logs
function txrl() {
    SERVICE_NAME=${1:-$DEFAULT_SERVICE}
    txrestart $SERVICE_NAME || return $?
    txl $SERVICE_NAME || return $?
}

# [txx] - tennex execute - execute a command inside a container
function txx() {
    (
        cd $TENNX_ROOT_DIR/deployments/local
        
        SERVICE_NAME=${1:-$DEFAULT_SERVICE}
        shift
        docker exec -it $(docker compose ps -q $SERVICE_NAME) "$@"
    )
}

# [txsh] - tennex shell - open a shell inside the bridge container
function txsh() {
    txx bridge sh
}

# [txdev] - tennex development - start development environment with live reload
function txdev() {
    (
        cd $TENNX_ROOT_DIR/deployments/local
        echo "Starting development environment with live reload..."
        docker compose up -d
    )
}

# --- API helpers ---
txhealth() { curl -s "$BRIDGE_URL/health" | jq .; }
txready() { curl -s "$BRIDGE_URL/ready" | jq .; }
txstats() { curl -s "$BRIDGE_URL/stats" | jq .; }
txdebug_config() { curl -s "$BRIDGE_URL/debug/config" | jq .; }
txdebug_whatsapp() { curl -s "$BRIDGE_URL/debug/whatsapp" | jq .; }

# --- Minimal connect ---
txconnect() {
    local client_id="${1:-whatsapp-test}"
    curl -sS -X POST "$BRIDGE_URL/connect-minimal" \
        -H 'Content-Type: application/json' \
        -d "{\"client_id\":\"$client_id\"}" | jq .
}

# --- Script wrappers ---
txconnect_whatsapp() { (cd "$TENNX_ROOT_DIR" && ./scripts/connect-whatsapp.sh "$@"); }
txdemo_qr() { (cd "$TENNX_ROOT_DIR" && ./scripts/demo-qr-only.sh "$@"); }
txdev_start() { (cd "$TENNX_ROOT_DIR" && ./scripts/dev-start.sh "$@"); }
txdev_stop() { (cd "$TENNX_ROOT_DIR" && ./scripts/dev-stop.sh "$@"); }
txquick_test() { (cd "$TENNX_ROOT_DIR" && ./scripts/quick-test.sh "$@"); }
txtest_api() { (cd "$TENNX_ROOT_DIR" && ./scripts/test-api.sh "$@"); }
txtest_local_bridge() { (cd "$TENNX_ROOT_DIR" && ./scripts/test-local-bridge.sh "$@"); }
txtest_qr_integration() { (cd "$TENNX_ROOT_DIR" && ./scripts/test-qr-integration.sh "$@"); }
txtest_qr_sizes() { (cd "$TENNX_ROOT_DIR" && ./scripts/test-qr-sizes.sh "$@"); }

# --- PoC ---
txqr_poc() { (cd "$TENNX_ROOT_DIR/test" && go run .); }

# --- Help ---
txhelp() {
  cat <<EOF
Tennex shortcuts loaded. Examples:
  tx                    # navigate to project root
  txup [svc]            # docker compose up -d
  txdev                 # start development environment with live reload
  txdown                # docker compose down
  txb [svc]             # docker compose build
  txbrl [svc]           # build, run, logs
  txrl [svc]            # restart, logs
  txl [svc]             # show logs
  txps                  # docker compose ps
  txx [svc] <cmd>       # execute command in container
  txsh                  # shell into bridge container
  txconnect [client]    # call POST /connect-minimal
  txhealth|txready|txstats  # quick API checks
  txqr_poc              # run minimal whatsmeow PoC (prints QR)
  txrss                 # refresh shortcuts
EOF
}

echo "[tennex] Shortcuts loaded. Run 'txhelp' for commands. BRIDGE_URL=$BRIDGE_URL"