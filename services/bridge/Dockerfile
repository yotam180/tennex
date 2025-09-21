# Tennex Bridge Service Dockerfile
# Multi-stage build for optimal image size and security

# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies including C compiler for CGO
RUN apk add --no-cache git ca-certificates tzdata gcc musl-dev

# Set working directory
WORKDIR /build

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
ARG VERSION=latest
ARG BUILD_TIME
ARG GIT_COMMIT
RUN CGO_ENABLED=1 GOOS=linux go build \
    -a -installsuffix cgo \
    -ldflags "-X main.Version=dev -X main.BuildTime=docker-build -X main.GitCommit=dev -s -w" \
    -o dist/bridge ./cmd/bridge

# Runtime stage
FROM alpine:3.19

# Install runtime dependencies
RUN apk --no-cache add ca-certificates curl

# Create non-root user
RUN addgroup -g 1001 -S tennex && \
    adduser -u 1001 -S tennex -G tennex

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /build/dist/bridge .

# Copy configuration files (optional - using env vars primarily)
COPY config/ ./config/

# Create necessary directories
RUN mkdir -p /app/sessions /app/logs && \
    chown -R tennex:tennex /app

# Switch to non-root user
USER tennex

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=40s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Default command
CMD ["./bridge"]
