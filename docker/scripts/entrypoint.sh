#!/bin/sh -e

cat <<EOF
████████╗ █████╗ ██╗██╗     ███████╗ ██████╗ █████╗ ██╗     ███████╗     ██████╗ ██████╗ ██████╗ ███████╗██████╗ ███╗   ██╗███████╗
╚══██╔══╝██╔══██╗██║██║     ██╔════╝██╔════╝██╔══██╗██║     ██╔════╝    ██╔════╝██╔═══██╗██╔══██╗██╔════╝██╔══██╗████╗  ██║██╔════╝
   ██║   ███████║██║██║     ███████╗██║     ███████║██║     █████╗      ██║     ██║   ██║██████╔╝█████╗  ██║  ██║██╔██╗ ██║███████╗
   ██║   ██╔══██║██║██║     ╚════██║██║     ██╔══██║██║     ██╔══╝      ██║     ██║   ██║██╔══██╗██╔══╝  ██║  ██║██║╚██╗██║╚════██║
   ██║   ██║  ██║██║███████╗███████║╚██████╗██║  ██║███████╗███████╗    ╚██████╗╚██████╔╝██║  ██║███████╗██████╔╝██║ ╚████║███████║
   ╚═╝   ╚═╝  ╚═╝╚═╝╚══════╝╚══════╝ ╚═════╝╚═╝  ╚═╝╚══════╝╚══════╝     ╚═════╝ ╚═════╝ ╚═╝  ╚═╝╚══════╝╚═════╝ ╚═╝  ╚═══╝╚══════╝

A CoreDNS plugin that allows you to resolve DNS names to Tailscale IPs.

By Christian De Leon (https://github.com/christian-deleon/tailscale-coredns)

EOF

echo "Starting tailscale-coredns container..."

# Set default values
TS_HOSTS_FILE=${TS_HOSTS_FILE:-""}
TS_FORWARD_TO=${TS_FORWARD_TO:-"/etc/resolv.conf"}

# Generate Corefile from environment variables using Jinja2
echo "Generating Corefile with domain: $TS_DOMAIN"
echo "Hosts file: $TS_HOSTS_FILE"
echo "Forward to: $TS_FORWARD_TO"

# Generate Corefile using Python script with Jinja2
python3 /generate-corefile.py

# Run tailscaled in the background
echo "Starting tailscaled..."
tailscaled --tun=userspace-networking --state=/state/tailscaled.state --socket=/run/tailscale/tailscaled.sock &

# Wait for tailscaled to be ready
echo "Waiting for tailscaled socket..."
until [ -S /run/tailscale/tailscaled.sock ]; do
  sleep 0.1
done
echo "tailscaled socket ready"

# Authenticate with Tailscale
echo "Authenticating with Tailscale..."
tailscale --socket=/run/tailscale/tailscaled.sock up \
  --authkey="${TS_AUTHKEY}?ephemeral=false" \
  --advertise-tags=tag:tailscale-coredns \
  --hostname="$TS_HOSTNAME"

# Wait for connection to be established
echo "Waiting for connection to be established..."
while ! tailscale --socket=/run/tailscale/tailscaled.sock status >/dev/null 2>&1; do
  sleep 0.1
done
echo "Connection established"

echo "Starting CoreDNS..."
exec /coredns -conf /Corefile
