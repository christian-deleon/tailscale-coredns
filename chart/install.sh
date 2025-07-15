#!/bin/bash

# Tailscale CoreDNS Helm Chart Installation Script
# This script helps you install the Tailscale CoreDNS chart with proper configuration
# The primary use case is Split DNS with Tailscale - no external service exposure needed

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

# Check if required tools are installed
check_prerequisites() {
    print_status "Checking prerequisites..."

    if ! command -v kubectl &> /dev/null; then
        print_error "kubectl is not installed. Please install kubectl first."
        exit 1
    fi

    if ! command -v helm &> /dev/null; then
        print_error "helm is not installed. Please install helm first."
        exit 1
    fi

    print_success "Prerequisites check passed"
}

# Get user input
get_configuration() {
    print_status "Please provide the required configuration:"

    # Get Tailscale OAuth key
    read -p "Enter your Tailscale OAuth key: " TS_AUTHKEY
    if [ -z "$TS_AUTHKEY" ]; then
        print_error "Tailscale OAuth key is required"
        exit 1
    fi

    # Get domain
    read -p "Enter your domain (e.g., mydomain.com): " TS_DOMAIN
    if [ -z "$TS_DOMAIN" ]; then
        print_error "Domain is required"
        exit 1
    fi

    # Get hostname
    read -p "Enter hostname for this CoreDNS instance [coredns]: " TS_HOSTNAME
    TS_HOSTNAME=${TS_HOSTNAME:-coredns}

    # Get forward server
    read -p "Enter forward server for unresolved queries [8.8.8.8]: " TS_FORWARD_TO
    TS_FORWARD_TO=${TS_FORWARD_TO:-8.8.8.8}

    # Get namespace
    read -p "Enter namespace for deployment [default]: " NAMESPACE
    NAMESPACE=${NAMESPACE:-default}

    # Get release name
    read -p "Enter release name [tailscale-coredns]: " RELEASE_NAME
    RELEASE_NAME=${RELEASE_NAME:-tailscale-coredns}

    # Get values file
    read -p "Enter values file to use [minimal]: " VALUES_FILE
    VALUES_FILE=${VALUES_FILE:-minimal}
}

# Create namespace if it doesn't exist
create_namespace() {
    if [ "$NAMESPACE" != "default" ]; then
        print_status "Creating namespace $NAMESPACE..."
        kubectl create namespace "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -
        print_success "Namespace $NAMESPACE created/verified"
    fi
}

# Install the chart
install_chart() {
    print_status "Installing Tailscale CoreDNS chart..."

    # Build helm command
    HELM_CMD="helm install $RELEASE_NAME . --namespace $NAMESPACE"

    # Add values file if specified
    if [ "$VALUES_FILE" != "minimal" ]; then
        if [ -f "values-$VALUES_FILE.yaml" ]; then
            HELM_CMD="$HELM_CMD -f values-$VALUES_FILE.yaml"
        else
            print_warning "Values file values-$VALUES_FILE.yaml not found, using default values"
        fi
    fi

    # Add set parameters
    HELM_CMD="$HELM_CMD --set tailscale.authKey=$TS_AUTHKEY"
    HELM_CMD="$HELM_CMD --set tailscale.domain=$TS_DOMAIN"
    HELM_CMD="$HELM_CMD --set tailscale.hostname=$TS_HOSTNAME"
    HELM_CMD="$HELM_CMD --set tailscale.forwardTo=$TS_FORWARD_TO"

    print_status "Running: $HELM_CMD"
    eval $HELM_CMD

    print_success "Chart installed successfully"
}

# Verify installation
verify_installation() {
    print_status "Verifying installation..."

    # Wait for pods to be ready
    print_status "Waiting for pods to be ready..."
    kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=tailscale-coredns -n "$NAMESPACE" --timeout=300s

    # Check service
    print_status "Checking service..."
    kubectl get svc -n "$NAMESPACE" -l app.kubernetes.io/name=tailscale-coredns

    # Check pods
    print_status "Checking pods..."
    kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/name=tailscale-coredns

    print_success "Installation verified"
}

# Show usage instructions
show_usage() {
    print_status "Installation complete! Here's how to use it:"
    echo ""
    echo "1. Configure Tailscale Split DNS:"
    echo "   # Get the Tailscale IP (Tailnet IP) of the CoreDNS device:"
    echo "   tailscale ip ts-dns"
    echo "   # Configure this Tailscale IP in your Tailscale DNS settings"
    echo "   # Go to your Tailscale admin console > DNS settings"
    echo "   # Or get the IP from https://login.tailscale.com/admin/machines"
    echo ""
    echo "2. Check logs:"
    echo "   kubectl logs -l app.kubernetes.io/name=tailscale-coredns -n $NAMESPACE"
    echo ""
    echo "3. Check status:"
    echo "   helm status $RELEASE_NAME -n $NAMESPACE"
    echo ""
    echo "4. Uninstall:"
    echo "   helm uninstall $RELEASE_NAME -n $NAMESPACE"
    echo ""
    echo "Note: Service is disabled by default for Split DNS deployment."
    echo "      If you need external access, set service.enabled=true in values."
    echo ""
}

# Main function
main() {
    echo "=========================================="
    echo "Tailscale CoreDNS Helm Chart Installer"
    echo "=========================================="
    echo ""

    check_prerequisites
    get_configuration
    create_namespace
    install_chart
    verify_installation
    show_usage
}

# Run main function
main "$@"