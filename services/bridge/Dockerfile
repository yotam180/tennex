# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy workspace and module files
COPY go.work* ./
COPY services/backend/go.mod services/backend/go.sum ./services/backend/
COPY services/eventstream/go.mod services/eventstream/go.sum ./services/eventstream/
COPY services/bridge/go.mod services/bridge/go.sum ./services/bridge/
COPY shared/go.mod shared/go.sum ./shared/

# Copy pkg directory and shared dependencies
COPY pkg/ ./pkg/
COPY shared/ ./shared/

# Copy bridge source code
COPY services/bridge/ ./services/bridge/

# Download dependencies
RUN cd services/bridge && go mod download

# Build binary
RUN cd services/bridge && CGO_ENABLED=0 GOOS=linux go build -o /app/bridge .

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

COPY --from=builder /app/bridge .

# Create non-root user
RUN addgroup -g 1001 -S tennex && \
    adduser -u 1001 -S tennex -G tennex

USER tennex

EXPOSE 6003

CMD ["./bridge"]
