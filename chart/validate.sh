#!/bin/bash

# Tailscale CoreDNS Helm Chart Validation Script
# This script validates the Helm chart templates

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if helm is installed
check_helm() {
    if ! command -v helm &> /dev/null; then
        print_error "helm is not installed. Please install helm first."
        exit 1
    fi
    print_success "Helm is installed"
}

# Validate chart structure
validate_chart_structure() {
    print_status "Validating chart structure..."

    # Check required files
    required_files=("Chart.yaml" "values.yaml" "templates/deployment.yaml" "templates/service.yaml")

    for file in "${required_files[@]}"; do
        if [ ! -f "$file" ]; then
            print_error "Required file $file is missing"
            exit 1
        fi
    done

    print_success "Chart structure is valid"
}

# Lint the chart
lint_chart() {
    print_status "Linting Helm chart..."

    if helm lint .; then
        print_success "Chart linting passed"
    else
        print_error "Chart linting failed"
        exit 1
    fi
}

# Test template rendering
test_templates() {
    print_status "Testing template rendering..."

    # Test with default values
    if helm template . --debug > /dev/null 2>&1; then
        print_success "Template rendering with default values passed"
    else
        print_error "Template rendering with default values failed"
        exit 1
    fi

    # Test with minimal values
    if helm template . -f values-minimal.yaml --debug > /dev/null 2>&1; then
        print_success "Template rendering with minimal values passed"
    else
        print_error "Template rendering with minimal values failed"
        exit 1
    fi

    # Test with production values
    if helm template . -f values-production.yaml --debug > /dev/null 2>&1; then
        print_success "Template rendering with production values passed"
    else
        print_error "Template rendering with production values failed"
        exit 1
    fi
}

# Test with custom values
test_custom_values() {
    print_status "Testing with custom values..."

    # Create a temporary values file
    cat > temp-values.yaml << EOF
tailscale:
  authKey: "test-key"
  domain: "test.com"
  hostname: "test"
  forwardTo: "1.1.1.1"
  ephemeral: false
  refreshInterval: 60

coredns:
  customHosts: |
    192.168.1.100    test1.test.com
    192.168.1.101    test2.test.com

service:
  type: LoadBalancer

persistence:
  enabled: false

hpa:
  enabled: true
  minReplicas: 2
  maxReplicas: 5

serviceMonitor:
  enabled: true
EOF

    if helm template . -f temp-values.yaml --debug > /dev/null 2>&1; then
        print_success "Template rendering with custom values passed"
    else
        print_error "Template rendering with custom values failed"
        exit 1
    fi

    # Clean up
    rm -f temp-values.yaml
}

# Check for common issues
check_common_issues() {
    print_status "Checking for common issues..."

    # Check for hardcoded values
    if grep -r "hardcoded" templates/ 2>/dev/null; then
        print_warning "Found potential hardcoded values"
    fi

    # Check for missing labels
    if ! grep -r "app.kubernetes.io/name" templates/ > /dev/null 2>&1; then
        print_warning "Missing app.kubernetes.io/name labels"
    fi

    # Check for security context
    if ! grep -r "securityContext" templates/ > /dev/null 2>&1; then
        print_warning "Missing security context"
    fi

    print_success "Common issues check completed"
}

# Main function
main() {
    echo "=========================================="
    echo "Tailscale CoreDNS Helm Chart Validator"
    echo "=========================================="
    echo ""

    check_helm
    validate_chart_structure
    lint_chart
    test_templates
    test_custom_values
    check_common_issues

    echo ""
    print_success "All validation checks passed!"
    echo ""
    echo "The Helm chart is ready for deployment."
    echo "You can now install it using:"
    echo "  helm install tailscale-coredns ."
    echo ""
}

# Run main function
main "$@"