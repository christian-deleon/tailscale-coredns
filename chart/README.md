# Tailscale CoreDNS Helm Chart

This Helm chart deploys the Tailscale CoreDNS plugin in Kubernetes, providing DNS resolution for Tailscale hostnames and custom domains. The primary use case is to configure Tailscale's Split DNS settings to point to this CoreDNS instance, allowing your Tailscale network to use custom DNS resolution.

## Prerequisites

- Kubernetes 1.19+
- Helm 3.0+
- Tailscale OAuth key with appropriate permissions
- A domain name for DNS resolution

## Installation

### 1. Add the Helm repository (if published)

```bash
helm repo add tailscale-coredns https://your-repo-url
helm repo update
```

### 2. Create a values file

Create a `values.yaml` file with your configuration:

```yaml
# Required: Tailscale configuration
tailscale:
  authKey: "tskey-auth-xxxxx-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
  domain: "mydomain.com"
  hostname: "coredns"
  forwardTo: "8.8.8.8"
  ephemeral: true

# Optional: Custom hosts file
coredns:
  customHosts: |
    # Custom DNS entries
    192.168.1.100    serviceA.mydomain.com
    192.168.1.101    serviceB.mydomain.com

# Optional: Additional CoreDNS configuration
coredns:
  additionalConfig: |
    # Example: Route53 plugin
    example.private. {
        route53 example.private.:Z0123456789ABCDEF
        fallthrough
        log
        errors
    }

# Service configuration (disabled by default for Split DNS)
service:
  enabled: false  # Set to true only if you need external access
  type: ClusterIP
  port: 53

# Resources
resources:
  limits:
    cpu: 500m
    memory: 512Mi
  requests:
    cpu: 100m
    memory: 128Mi
```

### 3. Install the chart

```bash
# Install with custom values
helm install tailscale-coredns ./chart -f values.yaml

# Or install with inline values
helm install tailscale-coredns ./chart \
  --set tailscale.authKey="tskey-auth-xxxxx-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx" \
  --set tailscale.domain="mydomain.com" \
  --set tailscale.hostname="coredns"
```

## Configuration

### Required Parameters

| Parameter | Description | Example |
|-----------|-------------|---------|
| `tailscale.authKey` | Tailscale OAuth authentication key | `tskey-auth-xxxxx-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx` |
| `tailscale.domain` | Your domain name for DNS resolution | `mydomain.com` |
| `tailscale.hostname` | Hostname for this CoreDNS instance | `coredns` |

### Optional Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `tailscale.forwardTo` | Forward server for unresolved queries | `8.8.8.8` |
| `tailscale.ephemeral` | Enable ephemeral mode for Tailscale | `true` |
| `tailscale.refreshInterval` | Refresh interval in seconds | `30` |
| `coredns.customHosts` | Custom hosts file content | `""` |
| `coredns.additionalConfig` | Additional CoreDNS configuration | `""` |
| `service.enabled` | Enable Kubernetes service (not needed for Split DNS) | `false` |
| `service.type` | Kubernetes service type | `ClusterIP` |
| `persistence.enabled` | Enable persistent storage | `true` |
| `resources.limits.cpu` | CPU limit | `500m` |
| `resources.limits.memory` | Memory limit | `512Mi` |

### Advanced Configuration

#### Horizontal Pod Autoscaler

```yaml
hpa:
  enabled: true
  minReplicas: 1
  maxReplicas: 3
  targetCPUUtilizationPercentage: 80
  targetMemoryUtilizationPercentage: 80
```

#### Pod Disruption Budget

```yaml
podDisruptionBudget:
  enabled: true
  minAvailable: 1
```

#### Network Policy

```yaml
networkPolicy:
  enabled: true
  ingressRules:
    - from:
        - namespaceSelector:
            matchLabels:
              name: allowed-namespace
      ports:
        - protocol: UDP
          port: 53
```

#### Service (for external access)

```yaml
service:
  enabled: true  # Only enable if you need external access
  type: LoadBalancer
  port: 53
```

#### Service Monitor (Prometheus)

```yaml
serviceMonitor:
  enabled: true
  interval: 30s
  scrapeTimeout: 10s
```

## Usage

### Testing DNS Resolution

For Split DNS with Tailscale, you typically don't need to expose the service externally. Instead, configure Tailscale's DNS settings to point to this CoreDNS instance.

If you need to test locally:

```bash
# Port forward to access the service locally (only if service.enabled=true)
kubectl port-forward svc/tailscale-coredns 5353:53

# Test DNS resolution
nslookup mydomain.com 127.0.0.1 -port=5353
nslookup coredns.mydomain.com 127.0.0.1 -port=5353
```

### Configuring Tailscale Split DNS

1. Get the Tailscale IP (Tailnet IP) of the CoreDNS device:
   ```bash
   # From any Tailscale device
   tailscale ip ts-dns
   
   # Or get it from the Tailscale admin console
   # Go to https://login.tailscale.com/admin/machines
   ```

2. Configure Tailscale DNS settings in your admin console:
   - Go to your Tailscale admin console
   - Navigate to DNS settings
   - Add the **Tailscale IP** (not the pod IP) as a DNS server
   - Configure Split DNS for your domain

**Important**: Use the Tailscale IP (Tailnet IP) of the CoreDNS device, not the Kubernetes pod IP address.

### Checking Logs

```bash
# Get pod logs
kubectl logs -l app.kubernetes.io/name=tailscale-coredns

# Follow logs
kubectl logs -f -l app.kubernetes.io/name=tailscale-coredns
```

### Scaling

```bash
# Scale the deployment
kubectl scale deployment tailscale-coredns --replicas=3
```

## Security

### Tailscale Authentication

The chart creates a Kubernetes Secret to store the Tailscale OAuth key securely. Make sure to:

1. Use a dedicated OAuth key for this deployment
2. Set appropriate permissions on the OAuth key
3. Rotate the key regularly

### Network Access

For Split DNS deployments:
- The service is disabled by default since Tailscale nodes access the CoreDNS pod directly
- No external network exposure is required
- The CoreDNS instance is only accessible within the Tailscale network

### Network Security

The deployment includes:
- Non-root user execution
- Dropped capabilities (except NET_ADMIN)
- Read-only root filesystem (disabled for Tailscale requirements)
- Security context configuration

### RBAC

The chart creates a ServiceAccount with minimal required permissions. For production deployments, consider:

1. Creating a dedicated namespace
2. Implementing network policies
3. Using pod security standards

## Troubleshooting

### Common Issues

1. **Pod not starting**: Check if the Tailscale OAuth key is valid
2. **DNS resolution failing**: Verify the domain configuration
3. **Permission denied**: Ensure the pod has NET_ADMIN capability
4. **Storage issues**: Check PVC configuration
5. **Network connectivity**: Verify Tailscale network connectivity
6. **Split DNS not working**: Ensure Tailscale DNS settings are configured correctly

### Split DNS Troubleshooting

1. **Verify Tailscale connectivity**:
   ```bash
   kubectl exec -it <pod-name> -- tailscale status
   ```

2. **Check DNS resolution from Tailscale**:
   ```bash
   kubectl exec -it <pod-name> -- nslookup your-domain.com
   ```

3. **Verify Tailscale DNS settings**:
   - Check that the CoreDNS pod IP is configured in Tailscale DNS settings
   - Ensure Split DNS is enabled for your domain

### Debugging Commands

```bash
# Check pod status
kubectl get pods -l app.kubernetes.io/name=tailscale-coredns

# Describe pod for events
kubectl describe pod -l app.kubernetes.io/name=tailscale-coredns

# Check service
kubectl get svc tailscale-coredns

# Check configmaps
kubectl get configmaps -l app.kubernetes.io/name=tailscale-coredns

# Check secrets
kubectl get secrets -l app.kubernetes.io/name=tailscale-coredns
```

## Upgrading

```bash
# Upgrade the release
helm upgrade tailscale-coredns ./chart -f values.yaml

# Check upgrade status
helm status tailscale-coredns
```

## Uninstalling

```bash
# Uninstall the release
helm uninstall tailscale-coredns

# Clean up persistent volumes (optional)
kubectl delete pvc -l app.kubernetes.io/name=tailscale-coredns
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Test the chart
5. Submit a pull request

## License

This chart is licensed under the same license as the main project.