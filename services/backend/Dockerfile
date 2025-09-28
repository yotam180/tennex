# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.work* ./
COPY services/backend/go.mod services/backend/go.su[m] ./services/backend/
COPY services/eventstream/go.mod services/eventstream/go.su[m] ./services/eventstream/
COPY services/bridge/go.mod services/bridge/go.su[m] ./services/bridge/
COPY shared/go.mod shared/go.su[m] ./shared/

# Copy pkg directory and shared dependencies
COPY pkg/ ./pkg/
COPY shared/ ./shared/

# Download dependencies
RUN cd services/backend && go mod download

# Copy source code
COPY services/backend/ ./services/backend/

# Build binary
RUN cd services/backend && CGO_ENABLED=0 GOOS=linux go build -o /app/backend ./cmd/backend

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

COPY --from=builder /app/backend .

# Create non-root user
RUN addgroup -g 1001 -S tennex && \
    adduser -u 1001 -S tennex -G tennex

USER tennex

EXPOSE 8000 6001

CMD ["./backend"]
