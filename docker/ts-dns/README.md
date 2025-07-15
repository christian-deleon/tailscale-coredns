# TS-DNS Configuration Directory

This directory contains the configuration files for the Tailscale CoreDNS plugin.

## Directory Structure

```text
/etc/ts-dns/
├── hosts/
│   └── custom_hosts          # Custom DNS entries (hosts file format)
└── additional/
    └── additional.conf       # Additional CoreDNS configuration for plugins
```

## Files

### `/etc/ts-dns/hosts/custom_hosts`

Custom DNS entries in hosts file format. This file allows you to add static DNS entries that will be resolved alongside Tailscale hostnames.

Example:

```text
# Custom DNS entries
192.168.1.100    serviceA.mydomain.com
192.168.1.101    serviceB.mydomain.com
192.168.1.102    serviceC.mydomain.com
```

### `/etc/ts-dns/additional/additional.conf`

Additional CoreDNS configuration for built-in plugins like route53, etcd, kubernetes, cache, and prometheus.

Example:

```text
# Route53 plugin configuration
example.private. {
    route53 example.private.:Z0123456789ABCDEF
    fallthrough
    log
    errors
}
```

## Usage

1. **Custom Hosts**: Place your custom DNS entries in `hosts/custom_hosts`
2. **Additional Plugins**: Configure additional CoreDNS plugins in `additional/additional.conf`
3. **Docker Compose**: The docker-compose.yml file automatically mounts these files to the correct locations

## File Permissions

All files should be readable by the CoreDNS process. The Docker container mounts these files as read-only (`:ro`) for security.
