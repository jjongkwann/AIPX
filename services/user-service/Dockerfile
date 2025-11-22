# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git ca-certificates

# Copy shared module first (if exists)
COPY shared/go ./shared/go

# Copy service go mod files
COPY services/user-service/go.mod services/user-service/go.sum ./services/user-service/

# Download dependencies
WORKDIR /app/services/user-service
RUN go mod download

# Copy source
WORKDIR /app
COPY services/user-service ./services/user-service

# Build binary
WORKDIR /app/services/user-service
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /app/bin/server \
    ./cmd/server

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/bin/server .

# Copy config and migrations
COPY --from=builder /app/services/user-service/config ./config
COPY --from=builder /app/services/user-service/migrations ./migrations

# Non-root user
RUN addgroup -g 1000 app && \
    adduser -D -u 1000 -G app app && \
    chown -R app:app /app

USER app

EXPOSE 8082
EXPOSE 9091

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD ["/app/server", "healthcheck"] || exit 1

CMD ["./server"]
