# Multi-stage build for Proxmox Snapshot Manager Go

# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata make

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN make build

# Runtime stage
FROM alpine:3.18

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/build/proxmox-snapshot-manager /usr/local/bin/proxmox-snapshot-manager

# Copy sample configuration
COPY --from=builder /app/README.md /app/

# Create config directory
RUN mkdir -p /app/config && \
    chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD proxmox-snapshot-manager --help > /dev/null || exit 1

# Default command
ENTRYPOINT ["proxmox-snapshot-manager"]
CMD ["--help"]

# Labels
LABEL maintainer="YG Codes <support@yg.codes>" \
      org.opencontainers.image.title="Proxmox Snapshot Manager" \
      org.opencontainers.image.description="Fast and efficient Proxmox VM snapshot management tool" \
      org.opencontainers.image.version="1.0.0" \
      org.opencontainers.image.source="https://github.com/yg-codes/proxmox-admin-cli" \
      org.opencontainers.image.licenses="MIT"