# Tailscale CoreDNS Plugin

This project provides a CoreDNS plugin that resolves DNS names to Tailscale IPs, using tags for nested subdomains. The plugin works alongside CoreDNS's built-in `hosts` and `forward` plugins. The primary use case is to configure Tailscale's Split DNS settings to point to this CoreDNS instance, allowing your Tailscale network to use custom DNS resolution for specific domains.

## Features

- **Tailscale Integration**: Automatically resolves Tailscale hostnames to their IP addresses
- **Split DNS Support**: Configure Tailscale's DNS settings to use this CoreDNS instance for specific domains
- **Subdomain Tags**: Support for custom subdomains using Tailscale tags (`tag:subdomain-*`)
- **Hosts File Support**: Works with CoreDNS's built-in `hosts` plugin for custom DNS entries
- **Forward Server**: Works with CoreDNS's built-in `forward` plugin for unresolved queries
- **IPv4/IPv6 Support**: Full support for both IPv4 and IPv6 addresses
- **Periodic Refresh**: Configurable refresh interval to keep DNS records up-to-date
- **Docker Ready**: Complete Docker setup with Tailscale client included
- **Kubernetes Ready**: Helm chart for easy deployment in Kubernetes clusters

## Authentication

The plugin requires a Tailscale OAuth key. You can get one from the [Tailscale admin console](https://login.tailscale.com/admin/settings/oauth).

The OAuth client requires the following permissions with the tag `tag:tailscale-coredns` (or any other tag you want to use):

- `auth_keys` (Both read and write)
- `devices:core` (Both read and write)

## Split DNS Configuration

This CoreDNS instance is designed to work with Tailscale's Split DNS feature. This allows you to:

1. **Restrict DNS traffic** to specific domains
2. **Use custom DNS resolution** for your internal services
3. **Maintain security** by only routing specific domains through this DNS server

### Setting up Split DNS

1. **Deploy the CoreDNS instance** (see installation instructions below)

2. **Get the Tailscale IP address** (Tailnet IP):
   ```bash
   # From any Tailscale device
   tailscale ip ts-dns

   # Or get it from the Tailscale admin console
   # Go to https://login.tailscale.com/admin/machines
   ```

3. **Configure Tailscale DNS settings**:
   - Go to your [Tailscale admin console](https://login.tailscale.com/admin/dns)
   - Navigate to **DNS** settings
   - Add the **Tailscale IP** (not the server IP) as a **DNS server**
   - Configure **Split DNS** for your domain(s)

### Split DNS Example

If your domain is `mydomain.com`, you can configure Tailscale to:
- Use this CoreDNS instance for `*.mydomain.com`
- Use your regular DNS servers for all other domains

This ensures that:
- `serviceA.mydomain.com` → Resolved by this CoreDNS instance (via Tailscale IP)
- `google.com` → Resolved by your regular DNS servers
- `github.com` → Resolved by your regular DNS servers

**Important**: Use the Tailscale IP (Tailnet IP) of the CoreDNS device, not the server's internal IP address.

### Benefits of Split DNS

- **Security**: Only specific domains are routed through your custom DNS
- **Performance**: Other domains use fast, reliable DNS servers
- **Flexibility**: Mix custom and standard DNS resolution
- **Control**: Fine-grained control over DNS resolution

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
   # Required: Tailscale OAuth key
   TS_AUTHKEY=tskey-auth-xxxxx-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx

   # Required: Your domain
   TS_DOMAIN=mydomain.com

   # Required: Hostname for this CoreDNS instance
   TS_HOSTNAME=coredns

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

### Kubernetes Deployment (Helm Chart)

1. **Clone the repository**:
   ```bash
   git clone https://github.com/christian-deleon/tailscale-coredns.git
   cd tailscale-coredns
   ```

2. **Install using Helm**:
   ```bash
   # Basic installation (Split DNS focused)
   helm install tailscale-coredns ./chart \
     --set tailscale.authKey="your-oauth-key" \
     --set tailscale.domain="your-domain.com"

   # Or use the interactive installer
   cd chart
   ./install.sh
   ```

3. **Get the Tailscale IP for Split DNS**:
   ```bash
   # From any Tailscale device
   tailscale ip ts-dns

   # Or get it from the Tailscale admin console
   # Go to https://login.tailscale.com/admin/machines
   ```

4. **Configure Tailscale DNS settings** (see Split DNS Configuration above)

For detailed Kubernetes deployment options, see the [chart documentation](./chart/README.md).

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
- `TS_HOSTS_FILE` (optional): Path to hosts file for custom DNS entries (default: /etc/ts-dns/hosts/custom_hosts)
- `TS_FORWARD_TO` (optional): Forward server for unresolved queries (default: 8.8.8.8)
- `TS_EPHEMERAL` (optional): Enable ephemeral mode for Tailscale (default: true). When set to true, the node will be automatically removed when it goes offline
- `TSC_REFRESH_INTERVAL` (optional): Refresh interval in seconds (default: 30)

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

### Split DNS with Tailscale

The primary use case is to configure Tailscale's Split DNS settings to use this CoreDNS instance for specific domains. This provides:

- **Domain-specific DNS resolution**: Only certain domains use this DNS server
- **Security**: Other domains continue using your regular DNS servers
- **Flexibility**: Mix custom and standard DNS resolution

### Basic DNS Resolution

Once running and configured with Split DNS, the plugin will automatically resolve:

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
# Basic hostname resolution (when using Split DNS)
dig hostname.mydomain.com

# Subdomain tag resolution (device tagged with tag:subdomain-web-server)
dig hostname.web.server.mydomain.com

# IPv6 resolution
dig AAAA hostname.mydomain.com

# Custom hosts file entry
dig serviceA.mydomain.com

# Test from within the CoreDNS container
docker exec -it ts-dns-ts-dns-1 nslookup hostname.mydomain.com
```

### Graceful Shutdown

The container automatically handles graceful shutdown when receiving SIGTERM or SIGINT signals:

- **Automatic Logout**: When the container stops, it automatically logs out from Tailscale, removing the device from the network
- **Process Monitoring**: Monitors both CoreDNS and tailscaled processes, triggering cleanup if either exits unexpectedly
- **Timeout Handling**: Implements graceful shutdown with timeouts, falling back to force-kill if processes don't respond
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
├── serve.go          # Main DNS request handler
├── setup.go          # Plugin initialization and configuration
├── tailscale.go      # Tailscale integration and record management
├── go.mod           # Go module dependencies
├── docker/          # Docker deployment files
│   ├── Dockerfile
│   ├── docker-compose.yml
│   ├── example.env
│   ├── hosts        # Example hosts file
│   ├── additional.conf.example  # Example additional plugins config
│   ├── justfile     # Common commands
│   ├── scripts/
│   │   ├── entrypoint.sh
│   │   └── generate-corefile.py
│   └── templates/
│       └── Corefile.j2
├── chart/           # Kubernetes Helm chart
│   ├── Chart.yaml
│   ├── values.yaml
│   ├── templates/   # Kubernetes manifests
│   ├── install.sh   # Interactive installer
│   └── README.md    # Chart documentation
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

4. **Split DNS not working**:
   - Verify the **Tailscale IP** (not server IP) is correctly configured in Tailscale DNS settings
   - Check that Split DNS is enabled for your domain
   - Ensure the domain matches your `TS_DOMAIN` configuration
   - Test connectivity from Tailscale devices to the CoreDNS instance

### Debug Commands

```bash
# Check Tailscale status
tailscale status

# Test DNS resolution (when using Split DNS)
dig hostname.mydomain.com

# View CoreDNS logs
docker logs tailscale-coredns

# Check Tailscale connectivity
tailscale ping hostname

# Test from within the CoreDNS container
docker exec -it ts-dns-ts-dns-1 nslookup hostname.mydomain.com

# Check Split DNS configuration
docker exec -it ts-dns-ts-dns-1 tailscale ip -4
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
