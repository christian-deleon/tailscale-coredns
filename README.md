# Tailscale CoreDNS Plugin

This project provides a CoreDNS plugin that resolves DNS names to Tailscale IPs, using tags for nested subdomains. The plugin works alongside CoreDNS's built-in `hosts` and `forward` plugins.

## Features

- **Tailscale Integration**: Automatically resolves Tailscale hostnames to their IP addresses
- **Subdomain Tags**: Support for custom subdomains using Tailscale tags (`tag:subdomain-*`)
- **Hosts File Support**: Works with CoreDNS's built-in `hosts` plugin for custom DNS entries
- **Forward Server**: Works with CoreDNS's built-in `forward` plugin for unresolved queries
- **IPv4/IPv6 Support**: Full support for both IPv4 and IPv6 addresses
- **Periodic Refresh**: Configurable refresh interval to keep DNS records up-to-date
- **Docker Ready**: Complete Docker setup with Tailscale client included

## Authentication

The plugin requires a Tailscale OAuth key. You can get one from the [Tailscale admin console](https://login.tailscale.com/admin/settings/oauth).

The OAuth client requires the following permissions with the tag `tag:tailscale-coredns` (or any other tag you want to use):

- `auth_keys` (Both read and write)
- `dns` (Both read and write)
- `devices:core` (Both read and write)

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

3. **Edit the environment file** with your Tailscale credentials:
   ```bash
   # Required: Tailscale OAuth key
   TS_AUTHKEY=tskey-auth-xxxxx-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx

   # Required: Your domain
   TS_DOMAIN=mydomain.com

   # Required: Hostname for this CoreDNS instance
   TS_HOSTNAME=coredns

   # Optional: Path to hosts file (default: /etc/coredns/custom_hosts)
   TS_HOSTS_FILE=/etc/coredns/custom_hosts

   # Optional: Forward server for unresolved queries (default: 8.8.8.8)
   TS_FORWARD_TO=8.8.8.8

   # Optional: Enable ephemeral mode for Tailscale (default: false)
   TS_EPHEMERAL=false
   ```

4. **Start the service**:
   ```bash
   cd docker
   docker compose --env-file .env up --build
   ```

### Manual Build

1. **Prerequisites**:
   - Go 1.22 or later
   - CoreDNS v1.11.3

2. **Build custom CoreDNS with plugin**:
   ```bash
   git clone https://github.com/coredns/coredns.git
   cd coredns
   git checkout v1.11.3

   # Add plugin to CoreDNS
   go mod edit -require=tailscale-coredns@v0.0.0
   go mod edit -replace=tailscale-coredns=/path/to/this/plugin

   # Add plugin to plugin.cfg
   sed -i '/^forward:/i tailscale:tailscale-coredns' plugin.cfg

   go generate
   go mod tidy
   go build -o coredns .
   ```

## Configuration

### Environment Variables

- `TS_AUTHKEY` (required): Tailscale OAuth authentication key
- `TS_DOMAIN` (required): Your domain name for DNS resolution
- `TS_HOSTNAME` (required): Hostname for this CoreDNS instance
- `TS_HOSTS_FILE` (optional): Path to hosts file for custom DNS entries (default: /etc/coredns/custom_hosts)
- `TS_FORWARD_TO` (optional): Forward server for unresolved queries (default: 8.8.8.8)
- `TS_EPHEMERAL` (optional): Enable ephemeral mode for Tailscale (default: false). When set to true, the node will be automatically removed when it goes offline
- `TSC_REFRESH_INTERVAL` (optional): Refresh interval in seconds (default: 30)

### Corefile Configuration

The plugin supports a simple Corefile configuration:

```
. {
    tailscale mydomain.com
    hosts /etc/coredns/custom_hosts {
        fallthrough
    }
    forward . 8.8.8.8
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
├── serve.go          # Main DNS request handler
├── setup.go          # Plugin initialization and configuration
├── tailscale.go      # Tailscale integration and record management
├── go.mod           # Go module dependencies
├── docker/          # Docker deployment files
│   ├── Dockerfile
│   ├── docker-compose.yml
│   ├── example.env
│   ├── hosts        # Example hosts file
│   ├── justfile     # Common commands
│   ├── scripts/
│   │   ├── entrypoint.sh
│   │   └── generate-corefile.py
│   └── templates/
│       └── Corefile.j2
└── README.md
```

### Local Development

1. **Run tests**:
   ```bash
   go test ./...
   ```

2. **Build the plugin**:
   ```bash
   go build .
   ```

3. **Test with local CoreDNS**:
   ```bash
   # Build custom CoreDNS with plugin
   # Use the manual build instructions above
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
