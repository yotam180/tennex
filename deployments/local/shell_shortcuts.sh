#!/bin/zsh

## Tennex Development Environment Setup

# Set TENNEX_HOME if not already set
if [ -z "$TENNEX_HOME" ]; then
    # Get the directory containing this script, then go up two levels (from deployments/local to project root)
    SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]:-${(%):-%N}}" )" && pwd )"
    export TENNEX_HOME="$(cd "$SCRIPT_DIR/../.." && pwd)"
fi

## Settings
export DEFAULT_SERVICE="backend" # Default service for operations
export DEFAULT_ENV="dev"

# [tx] - tennex - navigate to the Tennex home directory
function tx() {
    cd ${TENNEX_HOME}
}

# [txrss] - tennex refresh shell shortcuts - refresh the shell shortcuts (mainly for development purposes)
function txrss() {
    source ${TENNEX_HOME}/local/shell_shortcuts.sh
}

# [txgen] - tennex generate - generate code from contracts (OpenAPI, protobuf, sqlc)
function txgen() {
    (
        cd ${TENNEX_HOME}
        echo "🔄 Generating code from contracts..."
        
        echo "🔧 Generating OpenAPI code for backend..."
        if [ -f "pkg/api/openapi.yaml" ]; then
            oapi-codegen -package api -generate types,chi-server,spec -o pkg/api/gen/api.go pkg/api/openapi.yaml
            echo "✅ Backend OpenAPI generation complete"
        else
            echo "⚠️  Skipping Backend OpenAPI generation (pkg/api/openapi.yaml not found)"
        fi
        
        echo "🔧 Generating OpenAPI code for bridge..."
        if [ -f "services/bridge/api/openapi.yaml" ]; then
            oapi-codegen -package api -generate types,chi-server,spec -o services/bridge/api/gen/api.go services/bridge/api/openapi.yaml
            echo "✅ Bridge OpenAPI generation complete"
        else
            echo "⚠️  Skipping Bridge OpenAPI generation (services/bridge/api/openapi.yaml not found)"
        fi
        
        echo "🔧 Generating sqlc code..."
        if [ -f "pkg/db/sqlc.yaml" ] && [ -d "pkg/db/queries" ]; then
            sqlc generate -f pkg/db/sqlc.yaml
            echo "✅ sqlc generation complete"
        else
            echo "⚠️  Skipping sqlc generation (sqlc.yaml or queries not found)"
        fi
        
        echo "🎉 Code generation complete!"
    )
}

# [txmigrate] - tennex migrate - run database migrations
function txmigrate() {
    (
        cd ${TENNEX_HOME}
        echo "📊 Running database migrations..."
        echo "Applying schema files to local database..."
        docker exec -i tennex-postgres psql -U tennex -d tennex < pkg/db/schema/001_initial_schema.sql
        echo "✅ Migrations completed successfully"
    )
}

# [txmigrateall] - tennex migrate all - run all database migrations
function txmigrateall() {
    (
        cd ${TENNEX_HOME}
        echo "📊 Running all database migrations..."
        for file in pkg/db/schema/*.sql; do
            echo "Applying $file..."
            docker exec -i tennex-postgres psql -U tennex -d tennex < "$file"
        done
        echo "✅ All migrations completed successfully"
    )
}

# [txdbreset] - tennex database reset - reset database (drop and recreate all tables)
function txdbreset() {
    (
        cd ${TENNEX_HOME}
        echo "🔄 Resetting database..."
        echo "⚠️  This will destroy all data!"
        docker exec -i tennex-postgres psql -U tennex -d tennex -c "DROP SCHEMA IF EXISTS public CASCADE; CREATE SCHEMA public;"
        txmigrateall
        echo "✅ Database reset completed"
    )
}

# [txb] - tennex build - build a development container using docker-compose
function txb() {
    (
        cd ${TENNEX_HOME}/deployments/local
        
        SERVICE_NAME=$1
        if [ -z "$SERVICE_NAME" ]; then
            docker-compose build
        else
            docker-compose build $SERVICE_NAME
        fi
    )
}

# [txinfra] - tennex infrastructure up - start infrastructure services only (postgres, nats, minio, pgadmin)
function txinfra() {
    (
        cd ${TENNEX_HOME}/deployments/local
        echo "🚀 Starting infrastructure services..."
        docker-compose up -d postgres nats minio pgadmin
        echo "Waiting for services to be ready..."
        sleep 5
        echo ""
        echo "📊 Infrastructure services running:"
        echo "  Postgres: localhost:5432"
        echo "  PgAdmin: http://localhost:8080 (admin@tennex.com / admin123)"
        echo "  NATS: localhost:4222"
        echo "  NATS Monitor: http://localhost:8222"
        echo "  MinIO API: localhost:9000"
        echo "  MinIO Console: http://localhost:9001 (tennex / tennex123)"
        echo ""
        echo "🔧 Next steps for local Go development:"
        echo "  txgen                    # Generate contracts"
        echo "  txmigrate               # Apply database migrations"
        echo "  cd services/backend && go run cmd/backend/main.go     # Port 8000"
        echo "  cd services/eventstream && go run cmd/eventstream/main.go # Port 6002"
        echo "  cd services/bridge && go run main.go                  # Port 6003"
        echo ""
        echo "🐳 Or run all services in Docker:"
        echo "  txup                    # Run all services in containers"
    )
}

# [txup] - tennex up - start the full environment using docker-compose (including Go services)
function txup() {
    (
        cd ${TENNEX_HOME}/deployments/local
        echo "🚀 Starting full Docker environment..."
        
        SERVICE_NAME=$1
        if [ -z "$SERVICE_NAME" ]; then
            docker-compose --profile full up -d
        else
            docker-compose up -d $SERVICE_NAME
        fi
        
        if [ -z "$SERVICE_NAME" ]; then
            echo ""
            echo "📊 All services running:"
            echo "  Backend API: http://localhost:8000"
            echo "  Event Stream: http://localhost:6002"
            echo "  Bridge API: http://localhost:6003"
            echo "  Postgres: localhost:5432"
            echo "  PgAdmin: http://localhost:8080 (admin@tennex.com / admin123)"
            echo "  NATS: localhost:4222"
            echo "  NATS Monitor: http://localhost:8222"
            echo "  MinIO API: localhost:9000"
            echo "  MinIO Console: http://localhost:9001 (tennex / tennex123)"
        fi
    )
}

# [txrestart] - tennex restart - restart services using docker-compose
function txrestart() {
    (
        cd ${TENNEX_HOME}/deployments/local
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

# [txdown] - tennex down - stop services using docker-compose
function txdown() {
    (
        cd ${TENNEX_HOME}/deployments/local
        SERVICE_NAME=$1
        if [ -z "$SERVICE_NAME" ]; then
            docker-compose down --remove-orphans
        else
            docker-compose down $SERVICE_NAME
        fi
    )
}

# [txps] - tennex ps - show the status of services using docker-compose
function txps() {
    (
        cd ${TENNEX_HOME}/deployments/local
        docker-compose ps
    )
}

# [txl] - tennex logs - show the logs of services using docker-compose
function txl() {
    (
        cd ${TENNEX_HOME}/deployments/local
        
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

# [txlr] - tennex logs raw - show the logs without filtering and formatting
function txlr() {
    (
        cd ${TENNEX_HOME}/deployments/local
        
        if [[ ! "$1" =~ ^- ]]; then
            SERVICE_NAME=$1
            shift
        else
            SERVICE_NAME=$DEFAULT_SERVICE
        fi

        docker logs $(docker compose ps $SERVICE_NAME -q) -f "$@" 2>&1
    )
}

# [txbrl] - tennex build, run, logs - build the service, run it, and show the logs
function txbrl() {
    SERVICE_NAME=${1:-$DEFAULT_SERVICE}
    txb $SERVICE_NAME || return $?
    txup $SERVICE_NAME || return $?
    txl $SERVICE_NAME || return $?
}

# [txrl] - tennex restart, logs - restart a service and show the logs
function txrl() {
    SERVICE_NAME=${1:-$DEFAULT_SERVICE}
    txrestart $SERVICE_NAME || return $?
    txl $SERVICE_NAME || return $?
}

# [txx] - tennex execute - execute a command inside a container
function txx() {
    (
        cd ${TENNEX_HOME}/deployments/local
        
        SERVICE_NAME=${1:-$DEFAULT_SERVICE}
        shift
        docker exec -it $(docker-compose ps -q $SERVICE_NAME) "$@"
    )
}

# [txsh] - tennex shell - open a shell inside a container
function txsh() {
    SERVICE_NAME=${1:-$DEFAULT_SERVICE}
    txx $SERVICE_NAME sh
}

# [txshb] - tennex shell backend - open a shell inside the backend container
function txshb() {
    txx backend sh
}

# [txtest] - tennex test - run tests
function txtest() {
    (
        cd ${TENNEX_HOME}
        echo "🧪 Running tests..."
        go test -race ./...
    )
}

# [txlint] - tennex lint - run linters and static analysis
function txlint() {
    (
        cd ${TENNEX_HOME}
        echo "🔍 Running linters..."
        if [ -f "./tools/lint.sh" ]; then
            ./tools/lint.sh
        else
            echo "Running go vet..."
            go vet ./...
            echo "Running go fmt check..."
            if [ -n "$(gofmt -l .)" ]; then
                echo "Code is not formatted. Run 'go fmt ./...' to fix."
                return 1
            fi
            echo "✅ Linting complete"
        fi
    )
}

# [txfmt] - tennex format - format Go code
function txfmt() {
    (
        cd ${TENNEX_HOME}
        echo "🎨 Formatting Go code..."
        go fmt ./...
        echo "✅ Code formatting complete"
    )
}

# [txclean] - tennex clean - clean generated files and build artifacts
function txclean() {
    (
        cd ${TENNEX_HOME}
        echo "🧹 Cleaning up..."
        rm -rf pkg/api/gen/
        rm -rf pkg/proto/gen/
        rm -rf pkg/db/gen/
        rm -rf services/bridge/api/gen/
        rm -rf bin/
        find . -name "*.log" -delete
        find . -name "bridge" -type f -delete 2>/dev/null || true
        find . -name "backend" -type f -delete 2>/dev/null || true
        find . -name "eventstream" -type f -delete 2>/dev/null || true
        echo "✅ Cleanup complete"
    )
}

# Build operations
# [txbb] - tennex build backend - build backend service
function txbb() {
    (
        cd ${TENNEX_HOME}
        txgen
        echo "🔨 Building backend..."
        mkdir -p bin
        cd services/backend && go build -o ../../bin/backend ./cmd/backend
        echo "✅ Backend built: bin/backend"
    )
}

# [txbridge] - tennex build bridge - build bridge service  
function txbridge() {
    (
        cd ${TENNEX_HOME}
        txgen
        echo "🔨 Building bridge..."
        mkdir -p bin
        cd services/bridge && go build -o ../../bin/bridge .
        echo "✅ Bridge built: bin/bridge"
    )
}

# [txbes] - tennex build eventstream - build event stream service
function txbes() {
    (
        cd ${TENNEX_HOME}
        txgen
        echo "🔨 Building event stream..."
        mkdir -p bin
        cd services/eventstream && go build -o ../../bin/eventstream ./cmd/eventstream
        echo "✅ Event stream built: bin/eventstream"
    )
}

# [txbuildall] - tennex build all - build all services
function txbuildall() {
    txbb && txbridge && txbes
}

# Run operations  
# [txrb] - tennex run backend - build and run backend service
function txrb() {
    txbb || return $?
    echo "🚀 Starting backend service..."
    ./bin/backend
}

# [txrbridge] - tennex run bridge - build and run bridge service
function txrbridge() {
    txbridge || return $?
    echo "🚀 Starting bridge service..."
    ./bin/bridge
}

# [txres] - tennex run eventstream - build and run eventstream service
function txres() {
    txbes || return $?
    echo "🚀 Starting eventstream service..."
    ./bin/eventstream
}

# [txdev] - tennex development mode - run backend locally with infrastructure in Docker
function txdev() {
    (
        cd ${TENNEX_HOME}
        echo "🚀 Starting development mode..."
        echo "Starting infrastructure services..."
        txinfra
        echo ""
        echo "Generating code and running migrations..."
        txgen
        txmigrate
        echo ""
        echo "✅ Development environment ready!"
        echo ""
        echo "🔧 To run services locally:"
        echo "  txrb                    # Run backend (port 8000)"
        echo "  txres                   # Run eventstream (port 6002)" 
        echo "  txrbridge              # Run bridge (port 6003)"
        echo ""
        echo "💡 Or run individual services in separate terminals:"
        echo "  cd services/backend && go run cmd/backend/main.go"
        echo "  cd services/eventstream && go run cmd/eventstream/main.go"
        echo "  cd services/bridge && go run main.go"
    )
}

# [txcode] - tennex code - open Cursor in the Tennex directory
function txcode() {
    (
        cd ${TENNEX_HOME}
        cursor .
    )
}

# [ting] - play sound (useful for notifying that a long running command ended)
function ting() {
    afplay /System/Library/Sounds/Funk.aiff
}

# Git utilities
# [gchash] - git commit hash - copy to clipboard the hash of the last commit
function gchash() {
    git rev-parse --short HEAD | tr -d '\n' | pbcopy
}

function gitcommit() {
    git rev-parse --short HEAD
}

# Help function
# [txhelp] - tennex help - show available commands
function txhelp() {
    echo "Tennex Development Shell Shortcuts:"
    echo ""
    echo "🏠 Navigation:"
    echo "  tx                      # Navigate to Tennex home"
    echo "  txcode                  # Open Cursor editor"
    echo ""
    echo "🔄 Code Generation:"
    echo "  txgen                   # Generate code from contracts"
    echo "  txrss                   # Refresh shell shortcuts"
    echo ""
    echo "📊 Database:"
    echo "  txmigrate              # Run database migrations"
    echo "  txmigrateall           # Run all database migrations"
    echo "  txdbreset              # Reset database (destructive!)"
    echo ""
    echo "🐳 Docker Operations:"
    echo "  txinfra                # Start infrastructure only"
    echo "  txup [service]         # Start all services (or specific service)"
    echo "  txdown [service]       # Stop all services (or specific service)"
    echo "  txrestart [service]    # Restart services"
    echo "  txps                   # Show service status"
    echo "  txb [service]          # Build containers"
    echo ""
    echo "📝 Logs:"
    echo "  txl [service]          # Show logs (default: backend)"
    echo "  txlr [service]         # Show raw logs"
    echo ""
    echo "🔨 Local Build & Run:"
    echo "  txbb                   # Build backend"
    echo "  txbridge               # Build bridge"
    echo "  txbes                  # Build eventstream" 
    echo "  txbuildall             # Build all services"
    echo "  txrb                   # Run backend locally"
    echo "  txrbridge              # Run bridge locally"
    echo "  txres                  # Run eventstream locally"
    echo ""
    echo "🚀 Development Workflows:"
    echo "  txdev                  # Start dev mode (infra + local services)"
    echo "  txbrl [service]        # Build, run, and show logs"
    echo "  txrl [service]         # Restart and show logs"
    echo ""
    echo "🔧 Container Operations:"
    echo "  txx [service] [cmd]    # Execute command in container"
    echo "  txsh [service]         # Open shell in container"
    echo "  txshb                  # Open shell in backend container"
    echo ""
    echo "🧪 Quality:"
    echo "  txtest                 # Run tests"
    echo "  txlint                 # Run linters"
    echo "  txfmt                  # Format code"
    echo "  txclean               # Clean generated files"
    echo ""
    echo "🎵 Utilities:"
    echo "  ting                   # Play notification sound"
    echo "  gchash                 # Copy git commit hash to clipboard"
    echo ""
}

echo "🎯 Tennex shell shortcuts loaded! Type 'txhelp' to see available commands."
echo "✅ Tennex development environment loaded!"
echo "🏠 TENNEX_HOME: $TENNEX_HOME"
echo "💡 Type 'txhelp' to see available commands"
