# Stage 1: Build custom CoreDNS with tailscale plugin
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git

WORKDIR /build

# Copy the plugin source code
COPY go.mod go.sum setup.go serve.go tailscale.go /plugin/

# Clone CoreDNS repository
RUN git clone https://github.com/coredns/coredns.git

WORKDIR /build/coredns

# Checkout the specific version matching the dependency
RUN git checkout v1.11.3

# Add the tailscale plugin to go.mod with replace to local path
RUN go mod edit -require=tailscale-coredns@v0.0.0
RUN go mod edit -replace=tailscale-coredns=/plugin

# Insert the plugin into plugin.cfg before the forward plugin
RUN sed -i '/^forward:/i tailscale:tailscale-coredns' plugin.cfg

# Generate and build CoreDNS
RUN go generate
RUN go mod tidy
RUN go build -o /coredns .

# Stage 2: Runtime image
FROM alpine:latest

# Install necessary packages
RUN apk add --no-cache ca-certificates iptables ip6tables

# Copy the built CoreDNS binary
COPY --from=builder /coredns /coredns

# Download and install Tailscale binaries
ARG TAILSCALE_VERSION=1.84.0
ARG TARGETARCH
RUN wget https://pkgs.tailscale.com/stable/tailscale_${TAILSCALE_VERSION}_${TARGETARCH}.tgz && \
    tar xzf tailscale_${TAILSCALE_VERSION}_${TARGETARCH}.tgz --strip-components=1 \
    tailscale_${TAILSCALE_VERSION}_${TARGETARCH}/tailscale \
    tailscale_${TAILSCALE_VERSION}_${TARGETARCH}/tailscaled && \
    mv tailscale /usr/bin/tailscale && \
    mv tailscaled /usr/bin/tailscaled && \
    rm tailscale_${TAILSCALE_VERSION}_${TARGETARCH}.tgz

# Copy the entry script
COPY docker/entry.sh /entry.sh
RUN chmod +x /entry.sh

# Expose DNS port
EXPOSE 53/udp

# Set entrypoint
ENTRYPOINT ["/entry.sh"]
