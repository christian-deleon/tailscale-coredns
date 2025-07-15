package process

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"tailscale-coredns/internal/config"
)

// Manager handles multiple processes and their lifecycle
type Manager struct {
	config    *config.Config
	processes []*Process
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
}

// Process represents a managed process
type Process struct {
	name    string
	cmd     *exec.Cmd
	running bool
	mu      sync.RWMutex
}

// NewManager creates a new process manager
func NewManager(cfg *config.Config) *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	return &Manager{
		config:    cfg,
		processes: make([]*Process, 0),
		ctx:       ctx,
		cancel:    cancel,
	}
}

// StartTailscaled starts the tailscaled daemon
func (m *Manager) StartTailscaled() error {
	args := []string{
		"--tun=userspace-networking",
		"--state=/state/tailscaled.state",
		"--socket=/run/tailscale/tailscaled.sock",
	}

	cmd := exec.CommandContext(m.ctx, "tailscaled", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start tailscaled: %w", err)
	}

	process := &Process{
		name:    "tailscaled",
		cmd:     cmd,
		running: true,
	}

	m.mu.Lock()
	m.processes = append(m.processes, process)
	m.mu.Unlock()

	log.Printf("Started tailscaled with PID: %d", cmd.Process.Pid)

	// Monitor the process
	go m.monitorProcess(process)

	return nil
}

// WaitForTailscaledSocket waits for the tailscaled socket to be available
func (m *Manager) WaitForTailscaledSocket() error {
	socketPath := "/run/tailscale/tailscaled.sock"
	timeout := 30 * time.Second
	deadline := time.Now().Add(timeout)

	log.Println("Waiting for tailscaled socket...")

	for time.Now().Before(deadline) {
		if _, err := os.Stat(socketPath); err == nil {
			log.Println("tailscaled socket ready")
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	return fmt.Errorf("timeout waiting for tailscaled socket")
}

// AuthenticateTailscale authenticates with Tailscale using OAuth
func (m *Manager) AuthenticateTailscale() error {
	log.Println("Authenticating with Tailscale...")

	// Build authkey with ephemeral parameter
	authkey := m.config.ClientSecret
	if m.config.Ephemeral {
		authkey += "?ephemeral=true"
	}

	args := []string{
		"--socket=/run/tailscale/tailscaled.sock",
		"up",
		"--authkey=" + authkey,
		"--advertise-tags=tag:ts-dns",
		"--hostname=" + m.config.Hostname,
	}

	cmd := exec.CommandContext(m.ctx, "tailscale", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to authenticate with Tailscale: %w", err)
	}

	log.Println("Tailscale authentication completed")
	return nil
}

// WaitForTailscaleConnection waits for Tailscale connection to be established
func (m *Manager) WaitForTailscaleConnection() error {
	log.Println("Waiting for Tailscale connection to be established...")

	timeout := 60 * time.Second
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		cmd := exec.CommandContext(m.ctx, "tailscale", "--socket=/run/tailscale/tailscaled.sock", "status")
		if err := cmd.Run(); err == nil {
			log.Println("Tailscale connection established")
			return nil
		}
		time.Sleep(1 * time.Second)
	}

	return fmt.Errorf("timeout waiting for Tailscale connection")
}

// StartCoreDNS starts the CoreDNS server with the specified config file
func (m *Manager) StartCoreDNS(corefilePath string) error {
	cmd := exec.CommandContext(m.ctx, "coredns", "-conf", corefilePath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start CoreDNS: %w", err)
	}

	process := &Process{
		name:    "coredns",
		cmd:     cmd,
		running: true,
	}

	m.mu.Lock()
	m.processes = append(m.processes, process)
	m.mu.Unlock()

	log.Printf("Started CoreDNS with PID: %d", cmd.Process.Pid)

	// Monitor the process
	go m.monitorProcess(process)

	return nil
}

// monitorProcess monitors a process and handles its lifecycle
func (m *Manager) monitorProcess(process *Process) {
	err := process.cmd.Wait()

	process.mu.Lock()
	process.running = false
	process.mu.Unlock()

	if err != nil && m.ctx.Err() == nil {
		log.Printf("Process %s exited unexpectedly: %v", process.name, err)
		// Trigger shutdown
		m.cancel()
	}
}

// LogoutTailscale logs out from Tailscale
func (m *Manager) LogoutTailscale() error {
	log.Println("Starting Tailscale logout process...")

	cmd := exec.Command("tailscale", "--socket=/run/tailscale/tailscaled.sock", "logout")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		log.Printf("Error during Tailscale logout: %v", err)
		// Don't return error here to continue with other cleanup
	} else {
		log.Println("Tailscale logout completed successfully")
	}

	return nil
}

// Stop gracefully stops all managed processes
func (m *Manager) Stop() error {
	log.Println("Initiating graceful shutdown of all managed processes...")

	// Cancel context to stop all processes
	log.Println("Cancelling process manager context...")
	m.cancel()

	// Logout from Tailscale first
	log.Println("Step 1: Logging out from Tailscale...")
	m.LogoutTailscale()

	m.mu.RLock()
	processes := make([]*Process, len(m.processes))
	copy(processes, m.processes)
	m.mu.RUnlock()

	log.Printf("Step 2: Sending TERM signals to %d managed processes...", len(processes))

	// Send TERM signal to all running processes
	for _, process := range processes {
		process.mu.RLock()
		if process.running && process.cmd.Process != nil {
			log.Printf("Sending TERM signal to %s (PID: %d)", process.name, process.cmd.Process.Pid)
			if err := process.cmd.Process.Signal(syscall.SIGTERM); err != nil {
				log.Printf("Failed to send TERM signal to %s: %v", process.name, err)
			}
		} else {
			log.Printf("Process %s is not running or has no PID", process.name)
		}
		process.mu.RUnlock()
	}

	// Wait for graceful shutdown with timeout
	log.Println("Step 3: Waiting for processes to shut down gracefully...")
	timeout := 10 * time.Second
	deadline := time.Now().Add(timeout)

	gracefulShutdown := false
	for time.Now().Before(deadline) {
		allStopped := true
		runningProcesses := []string{}
		for _, process := range processes {
			process.mu.RLock()
			if process.running {
				allStopped = false
				runningProcesses = append(runningProcesses, process.name)
			}
			process.mu.RUnlock()
		}
		if allStopped {
			log.Println("All processes shut down gracefully")
			gracefulShutdown = true
			break
		}
		if len(runningProcesses) > 0 {
			log.Printf("Still waiting for processes: %v", runningProcesses)
		}
		time.Sleep(1 * time.Second)
	}

	if !gracefulShutdown {
		log.Printf("Step 4: Graceful shutdown timeout reached, force killing remaining processes...")
		// Force kill any remaining processes
		for _, process := range processes {
			process.mu.RLock()
			if process.running && process.cmd.Process != nil {
				log.Printf("Force killing %s (PID: %d)", process.name, process.cmd.Process.Pid)
				if err := process.cmd.Process.Kill(); err != nil {
					log.Printf("Failed to force kill %s: %v", process.name, err)
				}
			}
			process.mu.RUnlock()
		}
	}

	log.Println("Process shutdown sequence completed")
	return nil
}

// RunWithSignalHandling runs the process manager with signal handling
func (m *Manager) RunWithSignalHandling() error {
	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	log.Println("Process manager is now waiting for signals...")

	// Wait for either termination signal or context cancellation
	select {
	case sig := <-sigChan:
		log.Printf("Received termination signal: %v - initiating graceful shutdown", sig)
	case <-m.ctx.Done():
		log.Println("Process manager context cancelled - initiating shutdown")
	}

	log.Println("Beginning shutdown sequence...")

	// Stop all processes
	if err := m.Stop(); err != nil {
		log.Printf("Error during shutdown: %v", err)
		return err
	}

	log.Println("Shutdown sequence completed successfully")
	return nil
}

// GetRunningProcesses returns a list of currently running process names
func (m *Manager) GetRunningProcesses() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var running []string
	for _, process := range m.processes {
		process.mu.RLock()
		if process.running {
			running = append(running, process.name)
		}
		process.mu.RUnlock()
	}

	return running
}