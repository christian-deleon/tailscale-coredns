# Tailscale CoreDNS Helm Chart Summary

## Overview

This Helm chart deploys the Tailscale CoreDNS plugin in Kubernetes, providing DNS resolution for Tailscale hostnames and custom domains. The primary use case is to configure Tailscale's Split DNS settings to point to this CoreDNS instance, allowing your Tailscale network to use custom DNS resolution without exposing services externally.

## What's Included

### Core Components

1. **Deployment**: Main application deployment with proper security context
2. **Service**: Kubernetes service (disabled by default for Split DNS)
3. **Secret**: Secure storage for Tailscale OAuth key
4. **ConfigMaps**: For custom hosts and additional CoreDNS configuration
5. **ServiceAccount**: Dedicated service account for the deployment
6. **PVC**: Persistent volume for Tailscale state storage

### Optional Components

1. **Horizontal Pod Autoscaler**: Automatic scaling based on CPU/memory usage
2. **Pod Disruption Budget**: Ensures availability during cluster maintenance
3. **Network Policy**: Network security controls
4. **Service Monitor**: Prometheus monitoring integration
5. **Ingress**: External access configuration
6. **Health Checks**: Liveness, readiness, and startup probes

## Key Features

### Split DNS Focus
- Service disabled by default (no external exposure needed)
- Tailscale nodes access CoreDNS pod directly
- Secure internal-only DNS resolution
- No external network dependencies

### Security
- Non-root user execution
- Dropped capabilities (except NET_ADMIN)
- Security context configuration
- Secret management for sensitive data

### Scalability
- Horizontal pod autoscaler support
- Configurable resource limits
- Pod anti-affinity for high availability
- Multiple replica support

### Monitoring
- Prometheus ServiceMonitor integration
- Health check probes
- Comprehensive logging
- Resource monitoring

### Configuration
- Flexible values structure
- Multiple deployment profiles (minimal, production)
- Custom hosts file support
- Additional CoreDNS plugin configuration

## File Structure

```
chart/
├── Chart.yaml                 # Chart metadata
├── values.yaml               # Default values
├── values-minimal.yaml       # Minimal deployment values
├── values-production.yaml    # Production deployment values
├── values-example.yaml       # Example configuration
├── .helmignore              # Files to exclude from package
├── install.sh               # Interactive installation script
├── validate.sh              # Chart validation script
├── README.md                # Comprehensive documentation
├── CHART_SUMMARY.md         # This summary
└── templates/               # Kubernetes manifests
    ├── _helpers.tpl         # Template helper functions
    ├── deployment.yaml      # Main application deployment
    ├── service.yaml         # DNS service
    ├── secret.yaml          # Tailscale OAuth key secret
    ├── configmap.yaml       # Custom hosts and config
    ├── pvc.yaml            # Persistent volume claim
    ├── serviceaccount.yaml  # Service account
    ├── hpa.yaml            # Horizontal pod autoscaler
    ├── pdb.yaml            # Pod disruption budget
    ├── networkpolicy.yaml   # Network policy
    ├── servicemonitor.yaml # Prometheus service monitor
    ├── ingress.yaml        # Ingress configuration
    └── NOTES.txt           # Post-installation notes
```

## Quick Start

1. **Basic Installation**:
   ```bash
   helm install tailscale-coredns ./chart \
     --set tailscale.authKey="your-oauth-key" \
     --set tailscale.domain="your-domain.com"
   ```

2. **Interactive Installation**:
   ```bash
   ./install.sh
   ```

3. **Production Deployment**:
   ```bash
   helm install tailscale-coredns ./chart \
     -f values-production.yaml \
     --set tailscale.authKey="your-oauth-key" \
     --set tailscale.domain="your-domain.com"
   ```

## Configuration Examples

### Minimal Deployment
```yaml
tailscale:
  authKey: "your-oauth-key"
  domain: "example.com"
  hostname: "coredns"
```

### Production Deployment
```yaml
tailscale:
  authKey: "your-oauth-key"
  domain: "example.com"
  hostname: "coredns"

hpa:
  enabled: true
  minReplicas: 2
  maxReplicas: 5

serviceMonitor:
  enabled: true

networkPolicy:
  enabled: true
```

### Custom Hosts
```yaml
coredns:
  customHosts: |
    192.168.1.100    serviceA.example.com
    192.168.1.101    serviceB.example.com
    10.0.0.100       internal.example.com
```

### Split DNS Configuration
```bash
# Get the Tailscale IP (Tailnet IP) of the CoreDNS device
tailscale ip ts-dns

# Configure this Tailscale IP in your Tailscale DNS settings
# Go to your Tailscale admin console > DNS settings
# Or get the IP from https://login.tailscale.com/admin/machines
```

## Validation

The chart includes comprehensive validation:

```bash
./validate.sh
```

This validates:
- Chart structure
- Template syntax
- Linting rules
- Template rendering with various configurations

## Security Considerations

1. **OAuth Key Management**: Store keys in Kubernetes secrets
2. **Network Policies**: Restrict network access as needed
3. **Pod Security**: Non-root execution with minimal capabilities
4. **Resource Limits**: Prevent resource exhaustion
5. **Monitoring**: Comprehensive health checks and monitoring

## Production Recommendations

1. Use `values-production.yaml` for production deployments
2. Enable horizontal pod autoscaler for scalability
3. Configure network policies for security
4. Set up Prometheus monitoring
5. Use dedicated namespaces
6. Configure pod disruption budgets
7. Set appropriate resource limits
8. Enable persistent storage for state

## Troubleshooting

Common issues and solutions:

1. **Pod not starting**: Check OAuth key validity
2. **DNS resolution failing**: Verify domain configuration
3. **Permission denied**: Ensure NET_ADMIN capability
4. **Storage issues**: Check PVC configuration
5. **Network connectivity**: Verify service and network policies

## Support

For issues and questions:
1. Check the main README.md for detailed documentation
2. Review the NOTES.txt output after installation
3. Check pod logs for error messages
4. Validate configuration with the validation script 