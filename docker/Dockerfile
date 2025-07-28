# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies and set working directory
RUN apk add --no-cache git && \
    mkdir -p /app && \
    cd /app

WORKDIR /app

# Copy go module files first for better caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy only the source code needed for building
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY pkg/ ./pkg/
COPY plugin.go ./

# Build all binaries in a single layer
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o tailscale-coredns ./cmd/tailscale-coredns && \
    CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o splitdns ./cmd/splitdns

# Build custom CoreDNS with the tailscale plugin in a single layer
RUN git clone https://github.com/coredns/coredns.git && \
    cd coredns && \
    git checkout v1.11.3 && \
    sed -i '/^forward:/i tailscale:tailscale-coredns' plugin.cfg && \
    go mod edit -require=tailscale-coredns@v0.0.0 && \
    go mod edit -replace=tailscale-coredns=/app && \
    go generate && \
    go mod tidy && \
    go build -o coredns .

# Runtime stage
FROM alpine:3.22

# Install all packages and create directories in a single layer
RUN apk add --no-cache \
    ca-certificates \
    iptables \
    ip6tables \
    curl \
    jq && \
    mkdir -p \
    /run/tailscale \
    /state \
    /etc/ts-dns/hosts \
    /etc/ts-dns/additional \
    /etc/ts-dns/rewrite

# Install Tailscale in a single layer
ENV TAILSCALE_VERSION=1.84.0
RUN ARCH=$(uname -m) && \
    case $ARCH in \
        x86_64) ARCH_NAME=amd64 ;; \
        aarch64) ARCH_NAME=arm64 ;; \
        *) echo "Unsupported architecture: $ARCH" && exit 1 ;; \
    esac && \
    curl -fsSL https://pkgs.tailscale.com/stable/tailscale_${TAILSCALE_VERSION}_${ARCH_NAME}.tgz | \
    tar -xzf - --strip-components=1 -C /usr/local/bin/ tailscale_${TAILSCALE_VERSION}_${ARCH_NAME}/tailscale tailscale_${TAILSCALE_VERSION}_${ARCH_NAME}/tailscaled

# Copy all binaries in a single layer
COPY --from=builder /app/tailscale-coredns /usr/local/bin/
COPY --from=builder /app/splitdns /usr/local/bin/
COPY --from=builder /app/coredns/coredns /usr/local/bin/

# Copy configuration files last (changes here won't invalidate previous layers)
COPY docker/ts-dns/ /etc/ts-dns/

# Set working directory
WORKDIR /

# Expose DNS port
EXPOSE 53/udp 53/tcp

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
    CMD tailscale --socket=/run/tailscale/tailscaled.sock status || exit 1

# Set entrypoint
ENTRYPOINT ["/usr/local/bin/tailscale-coredns"]
