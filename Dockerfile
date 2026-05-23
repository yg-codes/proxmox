# Multi-stage build for pve CLI

# Build stage
FROM golang:1.21-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata make

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN make build

# Runtime stage
FROM alpine:3.18

RUN apk add --no-cache ca-certificates tzdata

RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

WORKDIR /app

COPY --from=builder /app/build/pve /usr/local/bin/pve

USER appuser

HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD pve --help > /dev/null || exit 1

ENTRYPOINT ["pve"]
CMD ["--help"]

LABEL maintainer="YG Codes <support@yg.codes>" \
      org.opencontainers.image.title="pve" \
      org.opencontainers.image.description="Proxmox VE administration CLI" \
      org.opencontainers.image.version="1.2.0" \
      org.opencontainers.image.source="https://github.com/yg-codes/proxmox" \
      org.opencontainers.image.licenses="MIT"
