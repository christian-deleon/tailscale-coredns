package main

import (
	"flag"
	"fmt"
	"log"

	"tailscale-coredns/internal/plugin"
)

func main() {
	var (
		action = flag.String("action", "", "Action to perform: init, cleanup, or status")
		domain = flag.String("domain", "", "Domain for split DNS")
	)
	flag.Parse()

	if *action == "" {
		log.Fatal("Action is required: init, cleanup, or status")
	}

	if *domain == "" {
		log.Fatal("Domain is required")
	}

	// Create Tailscale plugin instance
	ts, err := plugin.New(*domain)
	if err != nil {
		log.Fatalf("Failed to create Tailscale plugin: %v", err)
	}

	// Create split DNS manager
	manager := plugin.NewSplitDNSManager(ts)

	switch *action {
	case "init":
		if err := manager.Initialize(); err != nil {
			log.Fatalf("Failed to initialize split DNS: %v", err)
		}
		fmt.Println("Split DNS initialized successfully")

	case "cleanup":
		if err := manager.Cleanup(); err != nil {
			log.Fatalf("Failed to cleanup split DNS: %v", err)
		}
		fmt.Println("Split DNS cleanup completed")

	case "status":
		enabled, domain := manager.GetSplitDNSStatus()
		fmt.Printf("Split DNS enabled: %t\n", enabled)
		fmt.Printf("Domain: %s\n", domain)

	default:
		log.Fatalf("Unknown action: %s", *action)
	}
} 