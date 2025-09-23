# Tennex - WhatsApp Bridge Platform
# Development tooling and workflow automation

.PHONY: help dev gen migrate migrate-all db-reset test lint clean docker-up docker-down

# Default target
help: ## Show this help message
	@echo "Tennex Development Commands:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

# Development workflow
dev-infra: docker-up ## Start infrastructure only (for local development)
	@echo "🚀 Starting infrastructure services..."
	@echo ""
	@echo "📊 Infrastructure:"
	@echo "  Postgres: http://localhost:8080 (admin@tennex.com / admin123)"
	@echo "  NATS Monitor: http://localhost:8222"
	@echo "  MinIO Console: http://localhost:9001 (tennex / tennex123)"
	@echo ""
	@echo "🔧 Next steps for local development:"
	@echo "  make gen                    # Generate contracts"
	@echo "  make migrate                # Apply database migrations"
	@echo "  cd services/backend && go run cmd/backend/main.go     # Port 6000"
	@echo "  cd services/eventstream && go run cmd/eventstream/main.go # Port 6002"
	@echo "  cd services/bridge && go run main.go                  # Port 6003"
	@echo ""
	@echo "🐳 Or run everything in Docker:"
	@echo "  make dev               # Run all services in containers"

dev: ## Start full environment in Docker (including Go services)
	@echo "🚀 Starting full Docker environment..."
	@cd deployments/local && docker-compose --profile full up --build -d
	@echo ""
	@echo "📊 All services running:"
	@echo "  Backend API: http://localhost:6000"
	@echo "  Event Stream: http://localhost:6002"
	@echo "  Bridge API: http://localhost:6003"
	@echo "  Postgres: http://localhost:8080 (admin@tennex.com / admin123)"
	@echo "  NATS Monitor: http://localhost:8222"
	@echo "  MinIO Console: http://localhost:9001 (tennex / tennex123)"

gen: ## Generate code from contracts (OpenAPI, protobuf, sqlc)
	@echo "🔄 Generating code from contracts..."
	@./tools/codegen.sh

migrate: ## Run database migrations
	@echo "📊 Running database migrations..."
	@echo "Applying schema files to local database..."
	@docker exec -i tennex-postgres psql -U tennex -d tennex < pkg/db/schema/001_initial_schema.sql
	@echo "✅ Migrations completed successfully"

migrate-all: ## Run all database migrations (useful for new schema files)
	@echo "📊 Running all database migrations..."
	@for file in pkg/db/schema/*.sql; do \
		echo "Applying $$file..."; \
		docker exec -i tennex-postgres psql -U tennex -d tennex < "$$file"; \
	done
	@echo "✅ All migrations completed successfully"

db-reset: ## Reset database (drop and recreate all tables)
	@echo "🔄 Resetting database..."
	@echo "⚠️  This will destroy all data!"
	@docker exec -i tennex-postgres psql -U tennex -d tennex -c "DROP SCHEMA IF EXISTS public CASCADE; CREATE SCHEMA public;"
	@$(MAKE) migrate-all
	@echo "✅ Database reset completed"

test: ## Run tests with dockertest
	@echo "🧪 Running tests..."
	go test -race ./...

lint: ## Run linters and static analysis
	@echo "🔍 Running linters..."
	@./tools/lint.sh

clean: ## Clean generated files and build artifacts
	@echo "🧹 Cleaning up..."
	@rm -rf pkg/api/gen/
	@rm -rf pkg/proto/gen/
	@rm -rf pkg/db/gen/
	@find . -name "*.log" -delete
	@docker system prune -f

# Docker operations
docker-up: ## Start docker services (Postgres, NATS, MinIO)
	@echo "🐳 Starting Docker services..."
	@cd deployments/local && docker-compose up -d
	@echo "Waiting for services to be ready..."
	@sleep 5

docker-down: ## Stop docker services
	@echo "🛑 Stopping Docker services..."
	@cd deployments/local && docker-compose down

# Build operations
build-backend: ## Build backend service
	@echo "🔨 Building backend..."
	@cd services/backend && go build -o ../../bin/backend ./cmd/backend

build-bridge: ## Build bridge service  
	@echo "🔨 Building bridge..."
	@cd services/bridge && go build -o ../../bin/bridge .

build-eventstream: ## Build event stream service
	@echo "🔨 Building event stream..."
	@cd services/eventstream && go build -o ../../bin/eventstream ./cmd/eventstream

build-all: build-backend build-bridge build-eventstream ## Build all services
