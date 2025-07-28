package template

import (
	"fmt"
	"os"
	"strings"
	"text/template"

	"tailscale-coredns/internal/config"
)

const corefileTemplate = `. {
    tailscale {{ .DomainsString }}
{{- if .HostsFile }}
    hosts {{ .HostsFile }} {
        fallthrough
    }
{{- end }}
{{- if .RewriteRules }}
	{{ .RewriteRules }}
{{- end }}
{{- if .ForwardTo }}
    forward . {{ .ForwardTo }}
{{- end }}
    log
    errors
}
{{- if .AdditionalConfig }}

{{ .AdditionalConfig }}
{{- end }}`

// CorefileData represents the data used to generate the Corefile
type CorefileData struct {
	DomainsString    string
	HostsFile        string
	ForwardTo        string
	RewriteRules     string
	AdditionalConfig string
}

// Generator handles Corefile generation
type Generator struct {
	template *template.Template
}

// NewGenerator creates a new Corefile generator
func NewGenerator() (*Generator, error) {
	tmpl, err := template.New("corefile").Parse(corefileTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse corefile template: %w", err)
	}

	return &Generator{
		template: tmpl,
	}, nil
}

// GenerateCorefile generates a Corefile based on the provided configuration
func (g *Generator) GenerateCorefile(cfg *config.Config) (string, error) {
	// Join domains with spaces for the tailscale plugin line
	domainsString := strings.Join(cfg.Domains, " ")

	// Load rewrite rules if rewrite file is specified
	var rewriteRules string
	if cfg.RewriteFile != "" {
		loadedRules, err := LoadRewriteRules(cfg.RewriteFile)
		if err != nil {
			return "", fmt.Errorf("failed to load rewrite rules: %w", err)
		}
		rewriteRules = loadedRules
	}

	data := CorefileData{
		DomainsString:    domainsString,
		HostsFile:        cfg.HostsFile,
		ForwardTo:        cfg.ForwardTo,
		RewriteRules:     strings.TrimSpace(rewriteRules),
		AdditionalConfig: strings.TrimSpace(cfg.AdditionalConfig),
	}

	var buf strings.Builder
	if err := g.template.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute corefile template: %w", err)
	}

	return buf.String(), nil
}

// WriteCorefile generates and writes a Corefile to the specified path
func (g *Generator) WriteCorefile(cfg *config.Config, outputPath string) error {
	content, err := g.GenerateCorefile(cfg)
	if err != nil {
		return fmt.Errorf("failed to generate corefile: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write corefile to %s: %w", outputPath, err)
	}

	return nil
}

// LoadAdditionalConfig loads additional configuration from file if it exists
func LoadAdditionalConfig(path string) (string, error) {
	if path == "" {
		path = "/etc/ts-dns/additional/additional.conf"
	}

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return "", nil // File doesn't exist, return empty string
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read additional config file %s: %w", path, err)
	}

	// Filter out commented lines and empty lines
	lines := strings.Split(string(content), "\n")
	var filteredLines []string

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		// Skip empty lines and lines that start with #
		if trimmedLine != "" && !strings.HasPrefix(trimmedLine, "#") {
			filteredLines = append(filteredLines, line)
		}
	}

	// If no non-commented content, return empty string
	if len(filteredLines) == 0 {
		return "", nil
	}

	return strings.Join(filteredLines, "\n"), nil
}

// LoadRewriteRules loads rewrite rules from file if it exists
func LoadRewriteRules(path string) (string, error) {
	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return "", nil // File doesn't exist, return empty string
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read rewrite rules file %s: %w", path, err)
	}

	// Filter out commented lines and empty lines
	lines := strings.Split(string(content), "\n")
	var filteredLines []string

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		// Skip empty lines and lines that start with #
		if trimmedLine != "" && !strings.HasPrefix(trimmedLine, "#") {
			filteredLines = append(filteredLines, line)
		}
	}

	// If no non-commented content, return empty string
	if len(filteredLines) == 0 {
		return "", nil
	}

	// Convert each line to a rewrite plugin directive
	var rewriteLines []string
	for _, line := range filteredLines {
		rewriteLines = append(rewriteLines, "    rewrite "+line)
	}

	return strings.Join(rewriteLines, "\n"), nil
}