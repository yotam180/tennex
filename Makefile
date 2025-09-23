# Tennex - WhatsApp Bridge Platform
# Development tooling and workflow automation

.PHONY: help dev gen migrate test lint clean docker-up docker-down

# Default target
help: ## Show this help message
	@echo "Tennex Development Commands:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

# Development workflow
dev: docker-up ## Start full development environment
	@echo "🚀 Starting development environment..."
	@echo "Postgres: http://localhost:8080 (admin@tennex.com / admin123)"
	@echo "Backend will be available at: http://localhost:8082"
	@echo "Bridge will be available at: http://localhost:8081"

gen: ## Generate code from contracts (OpenAPI, protobuf, sqlc)
	@echo "🔄 Generating code from contracts..."
	@./tools/codegen.sh

migrate: ## Run database migrations
	@echo "📊 Running database migrations..."
	@echo "TODO: Implement migration runner"

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
