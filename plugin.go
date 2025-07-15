// Package tailscale provides a CoreDNS plugin for Tailscale integration.
// This is a compatibility wrapper for the actual plugin implementation.
package tailscale

// Re-export the plugin from internal/plugin for CoreDNS compatibility
import _ "tailscale-coredns/internal/plugin" 