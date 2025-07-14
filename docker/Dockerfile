# Stage 1: Build custom CoreDNS with tailscale plugin
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git

WORKDIR /build

# Copy the plugin source code
COPY go.mod go.sum setup.go serve.go tailscale.go /plugin/

# Clone CoreDNS repository and build
RUN git clone https://github.com/coredns/coredns.git && \
    cd coredns && \
    git checkout v1.11.3 && \
    go mod edit -require=tailscale-coredns@v0.0.0 && \
    go mod edit -replace=tailscale-coredns=/plugin && \
    sed -i '/^forward:/i tailscale:tailscale-coredns' plugin.cfg && \
    go generate && \
    go mod tidy && \
    go build -o /coredns .

# Stage 2: Runtime image
FROM alpine:latest

# Install necessary packages
RUN apk add --no-cache ca-certificates iptables ip6tables

# Install necessary packages and download Tailscale
ARG TAILSCALE_VERSION=1.84.0
ARG TARGETARCH
RUN apk add --no-cache ca-certificates iptables ip6tables && \
    wget https://pkgs.tailscale.com/stable/tailscale_${TAILSCALE_VERSION}_${TARGETARCH}.tgz && \
    tar xzf tailscale_${TAILSCALE_VERSION}_${TARGETARCH}.tgz --strip-components=1 tailscale_${TAILSCALE_VERSION}_${TARGETARCH}/tailscale tailscale_${TAILSCALE_VERSION}_${TARGETARCH}/tailscaled && \
    mv tailscale /usr/bin/tailscale && \
    mv tailscaled /usr/bin/tailscaled && \
    rm tailscale_${TAILSCALE_VERSION}_${TARGETARCH}.tgz

# Copy the built CoreDNS binary
COPY --from=builder /coredns /coredns

# Copy the entry script with executable permissions
COPY --chmod=755 docker/entry.sh /entry.sh

# Expose DNS port
EXPOSE 53/udp

# Set entrypoint
ENTRYPOINT ["/entry.sh"]
