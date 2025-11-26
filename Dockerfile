# Multi-stage Dockerfile for HTTP Proxy Server

# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build applications
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o proxy cmd/proxy/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o backend cmd/backend/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o traffic-gen cmd/traffic-gen/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o config-gen cmd/config-gen/main.go

# Generate configuration files
RUN ./config-gen

# Production stage
FROM alpine:3.18

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata curl

# Create non-root user
RUN addgroup -g 1001 proxy && \
    adduser -u 1001 -G proxy -s /bin/sh -D proxy

# Create necessary directories
RUN mkdir -p /app/logs /app/config /app/examples && \
    chown -R proxy:proxy /app

# Set working directory
WORKDIR /app

# Copy binaries from builder stage
COPY --from=builder --chown=proxy:proxy /app/proxy /app/backend /app/traffic-gen /app/config-gen ./
COPY --from=builder --chown=proxy:proxy /app/examples/ ./examples/

# Switch to non-root user
USER proxy

# Expose ports
EXPOSE 8080 8090

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
    CMD curl -f http://localhost:8080/proxy/health || exit 1

# Default command
CMD ["./proxy", "-config", "examples/proxy.yaml"]

# Labels
LABEL maintainer="HTTP Proxy Team"
LABEL version="1.0"
LABEL description="High-performance HTTP proxy server with rule-based filtering"
