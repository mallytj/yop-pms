# =============================================================================
# Production-Ready Multi-Stage Dockerfile for Yop PMS
# =============================================================================

# -----------------------------------------------------------------------------
# Stage 1: Build Environment
# -----------------------------------------------------------------------------
FROM golang:1.25-alpine AS builder

# Install build dependencies and security certificates
RUN apk --no-cache add \
    ca-certificates \
    tzdata \
    git

# Create non-root user for the runtime stage
RUN adduser -D -g '' -u 10001 appuser

# Set working directory
WORKDIR /app

# Copy dependency files first (Docker layer caching optimization)
COPY go.mod go.sum ./

# Download dependencies with mount cache for faster rebuilds
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copy source code
COPY . .

# Build the server binary with optimizations
# For local Mac (arm64): use GOARCH=arm64
# For production (amd64): use GOARCH=amd64
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build \
    -ldflags="-w -s -X main.version=${VERSION:-dev}" \
    -a \
    -installsuffix cgo \
    -o server ./cmd/server

# Verify the binary is static
RUN file server | grep -q "statically linked" && echo "Binary is static" || echo "Warning: Binary may not be fully static"

# -----------------------------------------------------------------------------
# Stage 2: Production Runtime
# -----------------------------------------------------------------------------
FROM scratch

# Add metadata labels following OCI conventions
LABEL org.opencontainers.image.title="Yop PMS Backend API" \
      org.opencontainers.image.description="Production Yop PMS backend server" \
      org.opencontainers.image.vendor="Yop" \
      org.opencontainers.image.source="https://github.com/lexxcode1/yop-pms"

# Copy CA certificates for HTTPS support
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy timezone data for time operations
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy user/group files for non-root execution
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group

# Copy the compiled binary
COPY --from=builder /app/server /server

# Copy migrations
COPY migrations/ /migrations/

# Set default timezone
ENV TZ=UTC

# Switch to non-root user
USER appuser:appuser

# Document the exposed port
EXPOSE 8080

# Run the application
ENTRYPOINT ["/server"]