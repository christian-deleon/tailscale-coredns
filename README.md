# Tailscale CoreDNS Plugin

This project provides a CoreDNS plugin that resolves DNS names to Tailscale IPs, using tags for nested subdomains.

## Authentication

The plugin requires a Tailscale OAuth key. You can get one from the Tailscale admin console.

The OAuth client requires the following permissions with the tag `tag:tailscale-coredns` (or any other tag you want to use):

- 'auth_keys' (Both read and write)
- 'dns' (Both read and write)
- 'devices:core' (Both read and write)
