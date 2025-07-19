package main

import (
	"flag"
	"fmt"
	"log"
	"strings"

	"tailscale-coredns/internal/plugin"
	"tailscale-coredns/pkg/api"
)

func main() {
	var (
		action  = flag.String("action", "", "Action to perform: init, cleanup, or status")
		domains = flag.String("domains", "", "Comma-separated list of domains for split DNS")
		domain  = flag.String("domain", "", "Single domain for split DNS (deprecated, use -domains)")
	)
	flag.Parse()

	if *action == "" {
		log.Fatal("Action is required: init, cleanup, or status")
	}

	// Handle domains parameter - check -domains first, fall back to -domain
	domainsStr := *domains
	if domainsStr == "" && *domain != "" {
		domainsStr = *domain
	}

	if domainsStr == "" {
		log.Fatal("Domains are required (use -domains)")
	}

	// Parse domains
	parsedDomains, err := api.ParseDomains(domainsStr)
	if err != nil {
		log.Fatalf("Failed to parse domains: %v", err)
	}

	// Create Tailscale plugin instance
	ts, err := plugin.New(parsedDomains)
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
		enabled, domains := manager.GetSplitDNSStatus()
		fmt.Printf("Split DNS enabled: %t\n", enabled)
		fmt.Printf("Domains: %s\n", strings.Join(domains, ", "))

	default:
		log.Fatalf("Unknown action: %s", *action)
	}
} 