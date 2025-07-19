package plugin

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	clog "github.com/coredns/coredns/plugin/pkg/log"
)

// SplitDNSManager handles the lifecycle of split DNS configuration
type SplitDNSManager struct {
	ts *Tailscale
}

// NewSplitDNSManager creates a new split DNS manager
func NewSplitDNSManager(ts *Tailscale) *SplitDNSManager {
	return &SplitDNSManager{
		ts: ts,
	}
}

// Initialize sets up split DNS and adds the current node's IP
func (m *SplitDNSManager) Initialize() error {
	if !m.ts.enableSplitDNS {
		clog.Info("Split DNS is disabled")
		return nil
	}

	clog.Info("Initializing split DNS...")

	// Add current node to split DNS
	// Note: Tailscale connection should already be established by the main process
	if err := m.ts.AddToSplitDNS(); err != nil {
		return fmt.Errorf("failed to add to split DNS: %w", err)
	}

	clog.Info("Split DNS initialization completed")
	return nil
}

// Cleanup removes the current node's IP from split DNS
func (m *SplitDNSManager) Cleanup() error {
	if !m.ts.enableSplitDNS {
		return nil
	}

	clog.Info("Cleaning up split DNS...")

	if err := m.ts.RemoveFromSplitDNS(); err != nil {
		return fmt.Errorf("failed to remove from split DNS: %w", err)
	}

	clog.Info("Split DNS cleanup completed")
	return nil
}

// waitForTailscale waits for Tailscale to be ready and have an IP
func (m *SplitDNSManager) waitForTailscale() error {
	clog.Info("Waiting for Tailscale to be ready...")

	timeout := 60 * time.Second
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if _, err := m.ts.GetOwnIP(); err == nil {
			clog.Info("Tailscale is ready")
			return nil
		}

		time.Sleep(1 * time.Second)
	}

	return fmt.Errorf("timeout waiting for Tailscale to be ready")
}

// RunWithSignalHandling runs the split DNS manager with proper signal handling
func (m *SplitDNSManager) RunWithSignalHandling() error {
	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	// Initialize split DNS
	if err := m.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize split DNS: %w", err)
	}

	// Wait for termination signal
	<-sigChan
	clog.Info("Received termination signal")

	// Cleanup
	if err := m.Cleanup(); err != nil {
		return fmt.Errorf("failed to cleanup split DNS: %w", err)
	}

	return nil
}

// GetSplitDNSStatus returns the current split DNS status
func (m *SplitDNSManager) GetSplitDNSStatus() (bool, []string) {
	return m.ts.enableSplitDNS, m.ts.splitDNSDomains
}