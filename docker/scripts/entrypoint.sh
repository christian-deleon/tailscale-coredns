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

echo "Starting ts-dns container..."

# Set default values
TS_HOSTS_FILE=${TS_HOSTS_FILE:-""}
TS_FORWARD_TO=${TS_FORWARD_TO:-"/etc/resolv.conf"}
TS_EPHEMERAL=${TS_EPHEMERAL:-"true"}

# Read additional configuration file if mounted
if [ -f "/etc/ts-dns/additional/additional.conf" ]; then
    echo "Found additional configuration file"
    TS_ADDITIONAL_CONFIG=$(cat /etc/ts-dns/additional/additional.conf)
    export TS_ADDITIONAL_CONFIG
fi

# Generate Corefile from environment variables using Jinja2
echo "Generating Corefile with domain: $TS_DOMAIN"
echo "Hosts file: $TS_HOSTS_FILE"
echo "Forward to: $TS_FORWARD_TO"
if [ -n "$TS_ADDITIONAL_CONFIG" ]; then
    echo "Additional configuration will be appended"
fi

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
  --authkey="${TS_AUTHKEY}?ephemeral=${TS_EPHEMERAL}" \
  --advertise-tags=tag:ts-dns \
  --hostname="$TS_HOSTNAME"

# Wait for connection to be established
echo "Waiting for connection to be established..."
while ! tailscale --socket=/run/tailscale/tailscaled.sock status >/dev/null 2>&1; do
  sleep 0.1
done
echo "Connection established"

echo "Starting CoreDNS..."
exec /coredns -conf /Corefile
