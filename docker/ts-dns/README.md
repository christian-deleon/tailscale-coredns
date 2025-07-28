# TS-DNS Configuration Directory

This directory contains the configuration files for the Tailscale CoreDNS plugin.

## Directory Structure

```text
/etc/ts-dns/
├── hosts/
│   └── custom_hosts          # Custom DNS entries (hosts file format)
├── rewrite/
│   └── rewrite.conf          # Rewrite rules for CoreDNS rewrite plugin
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

### `/etc/ts-dns/rewrite/rewrite.conf`

Rewrite rules for CoreDNS rewrite plugin. This file allows you to define DNS rewrite rules that will be applied before other plugins.

Example:

```text
# Rewrite rules for CoreDNS
# Each line should contain a rewrite rule in the format expected by CoreDNS rewrite plugin
# Lines starting with # are comments and will be ignored

# Example: Rewrite www.example.com to example.com
name www.example.com example.com

# Example: Rewrite with regex pattern
name regex (.*)\.example\.com {1}.example.com

# Example: Rewrite with response rewrite
answer name example.com www.example.com

# Example: Rewrite with CNAME
name example.com cname.example.com
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
2. **Rewrite Rules**: Configure DNS rewrite rules in `rewrite/rewrite.conf`
3. **Additional Plugins**: Configure additional CoreDNS plugins in `additional/additional.conf`
4. **Docker Compose**: The compose.yml file automatically mounts these files to the correct locations

## File Permissions

All files should be readable by the CoreDNS process. The Docker container mounts these files as read-only (`:ro`) for security.

## Deployment Considerations

The included `compose.yml` file is configured with 3 replicas for demonstration and testing purposes. However, this is **NOT** true high availability since all replicas run on the same host. For production deployments, comment out the `deploy` section to run a single instance, or refer to the [High Availability Deployment section](../../README.md#high-availability-deployment) in the main README for best practices.
