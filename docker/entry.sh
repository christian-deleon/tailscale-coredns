#!/bin/sh

set -e

# Generate Corefile from environment variables
cat <<EOF > /Corefile
. {
    tailscale $TS_DOMAIN
    forward . /etc/resolv.conf
    log
    errors
}
EOF

# Run tailscaled in the background
tailscaled --tun=userspace-networking --state=/state/tailscaled.state --socket=/run/tailscale/tailscaled.sock &

# Wait for tailscaled to be ready
until [ -S /run/tailscale/tailscaled.sock ]; do
  sleep 0.1
done

# Authenticate with Tailscale
tailscale --socket=/run/tailscale/tailscaled.sock up --authkey="$TS_AUTHKEY" --hostname="$TS_HOSTNAME"

# Wait for connection to be established
until tailscale --socket=/run/tailscale/tailscaled.sock status; do
  sleep 0.1
done

# Run CoreDNS
exec /coredns -conf /Corefile
