# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.work* ./
COPY services/backend/go.mod services/backend/go.su[m] ./services/backend/
COPY services/eventstream/go.mod services/eventstream/go.su[m] ./services/eventstream/

# Copy pkg directory for shared dependencies
COPY pkg/ ./pkg/

# Download dependencies
RUN cd services/eventstream && go mod download

# Copy source code
COPY services/eventstream/ ./services/eventstream/

# Build binary
RUN cd services/eventstream && CGO_ENABLED=0 GOOS=linux go build -o /app/eventstream ./cmd/eventstream

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

COPY --from=builder /app/eventstream .

# Create non-root user
RUN addgroup -g 1001 -S tennex && \
    adduser -u 1001 -S tennex -G tennex

USER tennex

EXPOSE 6002

CMD ["./eventstream"]
