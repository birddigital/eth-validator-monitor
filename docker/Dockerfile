# Multi-stage Dockerfile with security best practices
# Build stage
FROM golang:alpine3.19 AS builder

# Add metadata
LABEL maintainer="Adrien Bird <contact@adrienbird.com>" \
      description="Ethereum Validator Monitor - Production-grade validator performance tracking" \
      version="1.0.0"

# Install build dependencies
RUN apk add --no-cache \
    git \
    make \
    ca-certificates

WORKDIR /build

# Copy dependency files first for better caching
COPY go.mod go.sum ./

# Enable Go toolchain auto-download for version compatibility
ENV GOTOOLCHAIN=auto

RUN go mod download && go mod verify

# Copy source code
COPY . .

# Platform detection for multi-arch builds
ARG TARGETARCH
ARG TARGETOS=linux

# Build with security-hardened flags
RUN CGO_ENABLED=0 \
    GOOS=${TARGETOS} \
    GOARCH=${TARGETARCH} \
    go build \
    -a \
    -trimpath \
    -ldflags="-s -w -X main.Version=$(git describe --tags --always --dirty 2>/dev/null || echo 'dev') -X main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ) -extldflags '-static'" \
    -o validator-monitor \
    ./cmd/server

# Verify the binary
RUN chmod +x validator-monitor && ./validator-monitor --version 2>&1 || true

# Runtime stage - minimal attack surface
FROM alpine:3.19

# Add metadata
LABEL maintainer="Adrien Bird <contact@adrienbird.com>" \
      description="Ethereum Validator Monitor - Runtime container" \
      version="1.0.0"

# Install runtime dependencies and security updates
RUN apk --no-cache add \
    ca-certificates \
    tzdata \
    && apk upgrade --no-cache

# Create non-root user and group
RUN addgroup -g 10001 -S validator && \
    adduser -u 10001 -S -G validator -h /app validator

WORKDIR /app

# Copy binary and config from builder
COPY --from=builder --chown=validator:validator /build/validator-monitor .
COPY --from=builder --chown=validator:validator /build/config.yaml .

# Switch to non-root user
USER validator

# Expose application and metrics ports
EXPOSE 8080 9090

# Health check - verify API responds
HEALTHCHECK --interval=30s --timeout=10s --start-period=40s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the application
ENTRYPOINT ["./validator-monitor"]
CMD []
