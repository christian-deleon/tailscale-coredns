# Tailscale CoreDNS High-Availability Service

This project provides a high-availability Go service that integrates a CoreDNS plugin for resolving DNS names to Tailscale IPs, using tags for nested subdomains. The service includes automatic process management, graceful shutdown, and split DNS functionality for HA deployments.

## Features

- **High Availability**: Full Go service with automatic process management and graceful shutdown
- **Tailscale Integration**: Automatically resolves Tailscale hostnames to their IP addresses
- **Subdomain Tags**: Support for custom subdomains using Tailscale tags (`tag:subdomain-*`)
- **Hosts File Support**: Works with CoreDNS's built-in `hosts` plugin for custom DNS entries
- **Forward Server**: Works with CoreDNS's built-in `forward` plugin for unresolved queries
- **IPv4/IPv6 Support**: Full support for both IPv4 and IPv6 addresses
- **Periodic Refresh**: Configurable refresh interval to keep DNS records up-to-date
- **Process Management**: Monitors and manages CoreDNS and Tailscale processes
- **Graceful Shutdown**: Proper cleanup and signal handling for container orchestration
- **Split DNS Support**: Optional split DNS management for high availability deployments
- **OAuth Authentication**: Secure authentication using Tailscale OAuth credentials
- **Go Native**: Pure Go implementation with no Python or shell script dependencies
- **Configuration Templating**: Built-in Corefile generation with Go templates

## Authentication

The plugin uses Tailscale OAuth for authentication. You can create OAuth credentials from the [Tailscale admin console](https://login.tailscale.com/admin/settings/oauth).

The OAuth client requires the following permissions:

- `dns:read` - Read DNS configuration
- `dns:write` - Write DNS configuration (for split DNS functionality)

## Split DNS Functionality

The plugin supports optional split DNS management for high availability deployments. When enabled, each instance will:

- Add its own Tailscale IP to the split DNS configuration on startup
- Remove its IP from the split DNS configuration on shutdown
- Only manage its own IP, preserving other domains in the split DNS configuration

This ensures that multiple instances can run simultaneously without conflicts.

## Installation

### Docker Deployment (Recommended)

1. **Clone the repository**:
   ```bash
   git clone https://github.com/christian-deleon/tailscale-coredns.git
   cd tailscale-coredns
   ```

2. **Create environment file**:
   ```bash
   cp docker/example.env docker/.env
   ```

3. **Configure custom files** (optional):
   ```bash
   # Custom hosts file
   cp docker/ts-dns/hosts/custom_hosts docker/ts-dns/hosts/custom_hosts.example
   # Edit the file with your custom DNS entries

   # Additional configuration
   cp docker/additional.conf.example docker/ts-dns/additional/additional.conf
   # Edit the file with your additional CoreDNS configuration
   ```

   **Hosts File Format**: The hosts file uses standard hosts file format:
   ```
   # Custom DNS entries
   192.168.1.100    serviceA.mydomain.com
   192.168.1.101    serviceB.mydomain.com
   192.168.1.102    serviceC.mydomain.com
   ```

4. **Edit the environment file** with your Tailscale credentials:
   ```bash
   # Required: Tailscale OAuth credentials
   # Get these from https://login.tailscale.com/admin/settings/oauth
   TS_CLIENT_ID=tskey-client-abc123def456ghi
   TS_CLIENT_SECRET=tskey-client-abc123def456ghi-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx

   # Required: Your domain
   TS_DOMAIN=mydomain.com

   # Required: Hostname for this CoreDNS instance
   TS_HOSTNAME=coredns

   # Optional: Enable split DNS functionality (default: false)
   # When enabled, this instance will add its IP to the split DNS configuration
   TS_ENABLE_SPLIT_DNS=false

   # Optional: Path to hosts file (default: /etc/ts-dns/hosts/custom_hosts)
   TS_HOSTS_FILE=/etc/ts-dns/hosts/custom_hosts

   # Optional: Forward server for unresolved queries (default: 8.8.8.8)
   TS_FORWARD_TO=8.8.8.8

   # Optional: Enable ephemeral mode for Tailscale (default: true)
   TS_EPHEMERAL=true
   ```

4. **Start the service**:
   ```bash
   cd docker
   docker compose --env-file .env up --build
   ```

### Manual Build

1. **Prerequisites**:
   - Go 1.22 or later
   - Git

2. **Build the service**:
   ```bash
   # Clone the repository
   git clone https://github.com/christian-deleon/tailscale-coredns.git
   cd tailscale-coredns

   # Build the main service
   go build -o tailscale-coredns ./cmd/tailscale-coredns

   # Build the split DNS tool
   go build -o splitdns ./cmd/splitdns

   # Build custom CoreDNS with plugin (optional, for standalone use)
   git clone https://github.com/coredns/coredns.git
   cd coredns
   git checkout v1.11.3
   sed -i '/^forward:/i tailscale:tailscale-coredns' plugin.cfg
   go mod edit -require=tailscale-coredns@v0.0.0
   go mod edit -replace=tailscale-coredns=/path/to/tailscale-coredns
   go generate
   go mod tidy
   go build -o coredns .
   ```

3. **Run the service**:
   ```bash
   # Set required environment variables
   export TS_CLIENT_ID="your-client-id"
   export TS_CLIENT_SECRET="your-client-secret"
   export TS_DOMAIN="your-domain.com"
   export TS_HOSTNAME="coredns-server"

   # Run the service
   ./tailscale-coredns
   ```

## Configuration

### Environment Variables

- `TS_CLIENT_ID` (required): Tailscale OAuth client ID
- `TS_CLIENT_SECRET` (required): Tailscale OAuth client secret
- `TS_DOMAIN` (required): Your domain name for DNS resolution
- `TS_HOSTNAME` (required): Hostname for this CoreDNS instance
- `TS_ENABLE_SPLIT_DNS` (optional): Enable split DNS functionality (default: false)
- `TS_TAILNET` (optional): Explicit tailnet name (auto-detected if not set). Accepts multiple formats: `tail326daa`, `tail326daa.ts.net`, or `hostname.tail326daa.ts.net`
- `TS_HOSTS_FILE` (optional): Path to hosts file for custom DNS entries (default: /etc/ts-dns/hosts/custom_hosts)
- `TS_FORWARD_TO` (optional): Forward server for unresolved queries (default: 8.8.8.8)
- `TS_EPHEMERAL` (optional): Enable ephemeral mode for Tailscale (default: true). When set to true, the node will be automatically removed when it goes offline
- `TSC_REFRESH_INTERVAL` (optional): Refresh interval in seconds (default: 30)

### Split DNS Configuration

When `TS_ENABLE_SPLIT_DNS` is set to `true`, the plugin will:

1. **On Startup**: Add the current instance's Tailscale IP to the split DNS configuration
2. **On Shutdown**: Remove the current instance's IP from the split DNS configuration
3. **High Availability**: Multiple instances can run simultaneously, each managing only its own IP

**Requirements for Split DNS**:
- The OAuth client must have `dns:read` and `dns:write` permissions
- The domain name must be in the format `tailnet.com` (e.g., `mydomain.com`)

**Finding Your Tailnet Name**:
If split DNS operations fail with 404 errors, you may need to explicitly set your tailnet name:
1. Run `tailscale status` on any device in your tailnet
2. Look for the DNS name (e.g., `hostname.tail326daa.ts.net`)
3. You can set `TS_TAILNET` to any of these formats (all will be normalized automatically):
   - `TS_TAILNET=tail326daa` (just the tailnet name)
   - `TS_TAILNET=tail326daa.ts.net` (full format) 
   - `TS_TAILNET=hostname.tail326daa.ts.net` (hostname format)

### Directory Structure

The plugin uses `/etc/ts-dns/` as the base directory for configuration files:

```
/etc/ts-dns/
├── hosts/
│   └── custom_hosts          # Custom DNS entries (hosts file format)
└── additional/
    └── additional.conf       # Additional CoreDNS configuration for plugins
```

### Volume Mounts

- `/etc/ts-dns/hosts/custom_hosts` (optional): Custom hosts file for DNS entries
- `/etc/ts-dns/additional/additional.conf` (optional): Additional CoreDNS configuration for built-in plugins like route53, etcd, kubernetes

### Corefile Configuration

The plugin supports a simple Corefile configuration:

```
. {
    tailscale mydomain.com
    hosts /etc/ts-dns/hosts/custom_hosts {
        fallthrough
    }
    forward . 8.8.8.8
    log
    errors
}
```

### Additional Plugins

You can extend the CoreDNS configuration with additional built-in plugins by mounting an `additional.conf` file. This allows you to use plugins like `route53`, `etcd`, `kubernetes`, `cache`, and `prometheus`.

#### Setup

1. **Create an additional configuration file**:
   ```bash
   cp docker/additional.conf.example docker/ts-dns/additional/additional.conf
   ```

2. **Edit the configuration** with your specific plugin settings:
   ```bash
   # Example: Route53 plugin
   example.private. {
       route53 example.private.:Z0123456789ABCDEF
       fallthrough
       log
       errors
   }
   ```

3. **Mount the file** in your docker-compose.yml (already configured):
   ```yaml
   volumes:
     - ./ts-dns/additional/additional.conf:/etc/ts-dns/additional/additional.conf:ro
   ```

#### Examples

**Route53 Plugin**:
```
example.private. {
    route53 example.private.:Z0123456789ABCDEF
    fallthrough
    log
    errors
}
```

## Usage

### Basic DNS Resolution

Once running, the plugin will automatically resolve:

- **Tailscale hostnames**: `hostname.mydomain.com` → Tailscale IP
- **IPv4 and IPv6**: Both A and AAAA records are supported
- **Custom hosts**: Entries from the hosts file
- **Forwarded queries**: Unresolved queries are forwarded to the configured server

### Subdomain Tags

The plugin supports custom subdomains using Tailscale tags:

1. **Add a tag to your Tailscale device**: `tag:subdomain-web-server`
2. **DNS resolution**: `hostname.web.server.mydomain.com` → Tailscale IP

The plugin converts hyphens in tag names to dots in the subdomain.

### Examples

```bash
# Basic hostname resolution
dig hostname.mydomain.com @localhost

# Subdomain tag resolution (device tagged with tag:subdomain-web-server)
dig hostname.web.server.mydomain.com @localhost

# IPv6 resolution
dig AAAA hostname.mydomain.com @localhost

# Custom hosts file entry
dig serviceA.mydomain.com @localhost
```

### Graceful Shutdown

The Go service automatically handles graceful shutdown when receiving SIGTERM or SIGINT signals:

- **Automatic Logout**: When the service stops, it automatically logs out from Tailscale, removing the device from the network
- **Process Management**: Monitors both CoreDNS and tailscaled processes, triggering cleanup if either exits unexpectedly
- **Split DNS Cleanup**: Automatically removes the instance from split DNS configuration during shutdown
- **Timeout Handling**: Implements graceful shutdown with timeouts, falling back to force-kill if processes don't respond
- **Signal Handling**: Proper signal handling for container orchestration (Docker, Kubernetes)
- **Ephemeral Mode**: Works especially well with `TS_EPHEMERAL=true` (default) to ensure devices are automatically removed when offline

This ensures that your Tailscale network stays clean and devices are properly removed when containers are stopped or restarted.

## Docker Management

The project includes a `justfile` for common operations:

```bash
# Build the Docker image
just build

# Start the service
just start

# Stop the service
just stop

# Clean up (remove containers, volumes, and images)
just clean
```

## Development

### Project Structure

```
tailscale-coredns/
├── cmd/                      # Command-line applications
│   ├── tailscale-coredns/    # Main HA service application
│   │   └── main.go
│   └── splitdns/             # Split DNS management tool
│       └── main.go
├── internal/                 # Internal packages
│   ├── config/               # Configuration management
│   │   └── config.go
│   ├── plugin/               # CoreDNS plugin implementation
│   │   ├── plugin.go         # Main plugin logic
│   │   ├── serve.go          # DNS request handler
│   │   ├── setup.go          # Plugin initialization
│   │   └── splitdns.go       # Split DNS management
│   ├── process/              # Process management
│   │   └── manager.go
│   └── template/             # Configuration templating
│       └── corefile.go
├── pkg/                      # Public packages
│   └── api/                  # Tailscale API client
│       └── client.go
├── docker/                   # Docker deployment files
│   ├── Dockerfile            # Go-based container
│   ├── docker-compose.yml
│   ├── example.env
│   ├── justfile              # Common commands
│   └── ts-dns/               # Default configuration files
│       ├── hosts/
│       │   └── custom_hosts
│       └── additional/
│           └── additional.conf
├── go.mod                    # Go module dependencies
├── go.sum                    # Go module checksums
└── README.md
```

### Local Development

1. **Run tests**:
   ```bash
   go test ./...
   ```

2. **Build the applications**:
   ```bash
   # Build the main service
   go build ./cmd/tailscale-coredns

   # Build the split DNS tool
   go build ./cmd/splitdns
   ```

3. **Test locally**:
   ```bash
   # Set environment variables
   export TS_CLIENT_ID="your-client-id"
   export TS_CLIENT_SECRET="your-client-secret"
   export TS_DOMAIN="your-domain.com"
   export TS_HOSTNAME="test-server"

   # Run the service
   ./tailscale-coredns
   ```

4. **Development tools**:
   ```bash
   # Check split DNS status
   ./splitdns -action=status -domain=your-domain.com

   # Initialize split DNS manually
   ./splitdns -action=init -domain=your-domain.com

   # Cleanup split DNS manually
   ./splitdns -action=cleanup -domain=your-domain.com
   ```

## Troubleshooting

### Common Issues

1. **Plugin not resolving Tailscale hostnames**:
   - Verify Tailscale is running and authenticated
   - Check the Tailscale socket path: `/run/tailscale/tailscaled.sock`
   - Ensure the OAuth key has the correct permissions

2. **DNS queries timing out**:
   - Check the forward server configuration
   - Verify network connectivity
   - Review CoreDNS logs for errors

3. **Subdomain tags not working**:
   - Ensure tags are properly formatted: `tag:subdomain-web-server`
   - Check that devices have the correct tags applied
   - Verify the tag conversion (hyphens become dots)

### Debug Commands

```bash
# Check Tailscale status
tailscale status

# Test DNS resolution
dig hostname.mydomain.com @localhost

# View CoreDNS logs
docker logs tailscale-coredns

# Check Tailscale connectivity
tailscale ping hostname
```

## Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature-name`
3. Make your changes
4. Submit a pull request

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

## Author

Created by [Christian De Leon](https://github.com/christian-deleon)

## Support

For issues and questions:
- [GitHub Issues](https://github.com/christian-deleon/tailscale-coredns/issues)
- [Tailscale Documentation](https://tailscale.com/docs/)
- [CoreDNS Documentation](https://coredns.io/manual/toc/)
