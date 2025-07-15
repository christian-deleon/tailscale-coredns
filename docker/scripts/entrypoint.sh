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

# Function to cleanup Tailscale on exit
cleanup() {
    echo "Received shutdown signal, cleaning up..."

    # Kill CoreDNS gracefully first
    if [ -n "$COREDNS_PID" ] && kill -0 $COREDNS_PID 2>/dev/null; then
        echo "Stopping CoreDNS gracefully..."
        kill -TERM $COREDNS_PID
        # Wait up to 10 seconds for graceful shutdown
        for i in $(seq 1 10); do
            if ! kill -0 $COREDNS_PID 2>/dev/null; then
                break
            fi
            sleep 1
        done
        # Force kill if still running
        if kill -0 $COREDNS_PID 2>/dev/null; then
            echo "Force killing CoreDNS..."
            kill -KILL $COREDNS_PID
        fi
    fi

    # Logout from Tailscale
    if [ -S /run/tailscale/tailscaled.sock ]; then
        echo "Logging out from Tailscale..."
        tailscale --socket=/run/tailscale/tailscaled.sock logout
        echo "Tailscale logout completed"
    fi

        # Stop tailscaled gracefully
    if [ -n "$TAILSCALED_PID" ] && kill -0 $TAILSCALED_PID 2>/dev/null; then
        echo "Stopping tailscaled..."
        kill -TERM $TAILSCALED_PID
        # Wait up to 5 seconds for graceful shutdown
        for i in $(seq 1 5); do
            if ! kill -0 $TAILSCALED_PID 2>/dev/null; then
                break
            fi
            sleep 1
        done
        # Force kill if still running
        if kill -0 $TAILSCALED_PID 2>/dev/null; then
            echo "Force killing tailscaled..."
            kill -KILL $TAILSCALED_PID
        fi
    fi

    # Stop the monitor process
    if [ -n "$MONITOR_PID" ] && kill -0 $MONITOR_PID 2>/dev/null; then
        echo "Stopping process monitor..."
        kill -TERM $MONITOR_PID
    fi

    echo "Cleanup completed"
    exit 0
}

# Set up signal handlers
trap cleanup SIGTERM SIGINT

# Run tailscaled in the background
echo "Starting tailscaled..."
tailscaled --tun=userspace-networking --state=/state/tailscaled.state --socket=/run/tailscale/tailscaled.sock &
TAILSCALED_PID=$!

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
# Start CoreDNS in the background and wait for it
/coredns -conf /Corefile &
COREDNS_PID=$!

# Function to monitor processes and cleanup if one exits
monitor_processes() {
    while true; do
        # Check if CoreDNS is still running
        if [ -n "$COREDNS_PID" ] && ! kill -0 $COREDNS_PID 2>/dev/null; then
            echo "CoreDNS process exited unexpectedly"
            cleanup
        fi

        # Check if tailscaled is still running
        if [ -n "$TAILSCALED_PID" ] && ! kill -0 $TAILSCALED_PID 2>/dev/null; then
            echo "tailscaled process exited unexpectedly"
            cleanup
        fi

        sleep 1
    done
}

# Start process monitoring in background
monitor_processes &
MONITOR_PID=$!

# Wait for either process to exit
wait $COREDNS_PID $TAILSCALED_PID

# If we reach here, one of the processes has exited
echo "One of the main processes has exited, triggering cleanup..."
cleanup
