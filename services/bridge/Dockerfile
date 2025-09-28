# Development Dockerfile with air hot reloading
FROM golang:1.25-alpine

# Install air for hot reloading
RUN go install github.com/air-verse/air@latest

WORKDIR /app

# Copy all workspace files for go.work
COPY go.work* ./
COPY services/backend/go.mod services/backend/go.sum ./services/backend/
COPY services/eventstream/go.mod services/eventstream/go.sum ./services/eventstream/
COPY services/bridge/go.mod services/bridge/go.sum ./services/bridge/
COPY shared/go.mod shared/go.sum ./shared/
COPY pkg/ ./pkg/

# Copy shared dependencies
COPY shared/ ./shared/

# Download dependencies
RUN cd services/bridge && go mod download

# Set working directory to bridge service
WORKDIR /app/services/bridge

# Expose port
EXPOSE 6003

# Run air for hot reloading
CMD ["air", "-c", ".air.toml"]
