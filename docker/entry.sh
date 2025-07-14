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
echo "Waiting for connection to be established..."
while ! tailscale --socket=/run/tailscale/tailscaled.sock status >/dev/null 2>&1; do
  sleep 0.1
done
echo "Connection established"

# Run CoreDNS
exec /coredns -conf /Corefile
