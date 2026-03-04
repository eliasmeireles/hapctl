#!/bin/bash

set -e

VM_NAME="hapctl-dev"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VOLUMES_DIR="${SCRIPT_DIR}/.volumes"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

echo "=== HAProxy Control (hapctl) Development Environment ==="
echo ""

check_multipass() {
    if ! command -v multipass &> /dev/null; then
        echo "❌ Multipass is not installed"
        echo "Install it from: https://multipass.run/"
        exit 1
    fi
    echo "✅ Multipass is installed"
}

get_ssh_key() {
    if [ -f "$HOME/.ssh/id_rsa.pub" ]; then
        export SSH_PUBLIC_KEY=$(cat "$HOME/.ssh/id_rsa.pub")
    elif [ -f "$HOME/.ssh/id_ed25519.pub" ]; then
        export SSH_PUBLIC_KEY=$(cat "$HOME/.ssh/id_ed25519.pub")
    else
        echo "❌ No SSH public key found"
        echo "Generate one with: ssh-keygen -t ed25519"
        exit 1
    fi
    echo "✅ SSH key found"
}

create_vm() {
    echo ""
    echo "Creating VM: $VM_NAME"

    # Create .out directory for cloud-init
    OUT_DIR="$SCRIPT_DIR/.out"
    mkdir -p "$OUT_DIR"
    CLOUD_INIT_FILE="$OUT_DIR/cloud-init-generated.yaml"

    # Substitute SSH key in cloud-init using sed
    sed "s|\${SSH_PUBLIC_KEY}|${SSH_PUBLIC_KEY}|g" "$SCRIPT_DIR/cloud-init.yaml" > "$CLOUD_INIT_FILE"

    # Validate the generated file
    if ! grep -q "ssh_authorized_keys:" "$CLOUD_INIT_FILE"; then
        echo "❌ Failed to generate cloud-init file"
        cat "$CLOUD_INIT_FILE"
        exit 1
    fi

    echo "Cloud-init file generated at: $CLOUD_INIT_FILE"

    # Detect architecture for macOS ARM compatibility
    ARCH=$(uname -m)
    UBUNTU_IMAGE="22.04"

    if [[ "$OSTYPE" == "darwin"* ]] && [[ "$ARCH" == "arm64" ]]; then
        echo "Detected macOS ARM64 - using ARM-compatible Ubuntu image"
    fi

    multipass launch \
        --name "$VM_NAME" \
        --cpus 2 \
        --memory 2G \
        --disk 10G \
        --mount "$VOLUMES_DIR:/home/ubuntu/hapctl" \
        --cloud-init "$CLOUD_INIT_FILE" \
        "$UBUNTU_IMAGE"

    echo "✅ VM created successfully"
}

wait_for_vm() {
    echo ""
    echo "Waiting for VM to be ready..."
    sleep 10

    for i in {1..30}; do
        if multipass exec "$VM_NAME" -- systemctl is-active docker &> /dev/null; then
            echo "✅ VM is ready"
            return 0
        fi
        echo "Waiting... ($i/30)"
        sleep 5
    done

    echo "❌ VM did not become ready in time"
    exit 1
}

prepare_volume() {
    echo ""
    echo "Preparing volume directory..."

    mkdir -p "$VOLUMES_DIR"
    mkdir -p "$VOLUMES_DIR/resources"

    # Build hapctl and webhook-test
    cd "$PROJECT_ROOT"
    make build
    make build-webhook-test

    # Copy files to volume
    cp "$SCRIPT_DIR/config.yaml" "$VOLUMES_DIR/"
    cp "$SCRIPT_DIR/nginx-apps-https.yaml" "$VOLUMES_DIR/resources/"
    cp "$SCRIPT_DIR/docker-compose.yml" "$VOLUMES_DIR/"
    cp -r "$SCRIPT_DIR/html" "$VOLUMES_DIR/"
    cp "$SCRIPT_DIR/generate-ssl.sh" "$VOLUMES_DIR/"
    cp "$SCRIPT_DIR/configure-ssl.sh" "$VOLUMES_DIR/"

    echo "✅ Volume prepared at: $VOLUMES_DIR"
}

install_hapctl() {
    echo ""
    echo "Installing hapctl binary..."

    # Copy hapctl binary to VM
    multipass transfer "$PROJECT_ROOT/bin/hapctl" "$VM_NAME:/tmp/hapctl"

    # Install to /usr/local/bin
    multipass exec "$VM_NAME" -- sudo mv /tmp/hapctl /usr/local/bin/hapctl
    multipass exec "$VM_NAME" -- sudo chmod +x /usr/local/bin/hapctl

    echo "✅ hapctl binary installed to /usr/local/bin"
}

install_webhook_test() {
    echo ""
    echo "Installing webhook-test binary..."

    # Copy webhook-test binary to VM
    multipass transfer "$PROJECT_ROOT/bin/webhook-test" "$VM_NAME:/tmp/webhook-test"

    # Install to /usr/local/bin
    multipass exec "$VM_NAME" -- sudo mv /tmp/webhook-test /usr/local/bin/webhook-test
    multipass exec "$VM_NAME" -- sudo chmod +x /usr/local/bin/webhook-test

    echo "✅ webhook-test binary installed to /usr/local/bin"
}

install_haproxy() {
    echo ""
    echo "Installing HAProxy..."

    # Install HAProxy using hapctl from /usr/local/bin
    multipass exec "$VM_NAME" -- sudo hapctl install

    echo "✅ HAProxy installed"
}

start_docker_containers() {
    echo ""
    echo "Starting Docker containers..."

    # Start containers using docker compose
    multipass exec "$VM_NAME" -- bash -c "cd /home/ubuntu/hapctl && docker compose up -d"

    # Wait for containers to be healthy
    sleep 5

    # Check container status
    multipass exec "$VM_NAME" -- docker ps

    echo "✅ Docker containers started"
}

setup_ssl() {
    echo ""
    echo "Setting up SSL certificate..."

    # Make scripts executable
    multipass exec "$VM_NAME" -- chmod +x /home/ubuntu/hapctl/generate-ssl.sh
    multipass exec "$VM_NAME" -- chmod +x /home/ubuntu/hapctl/configure-ssl.sh

    # Generate self-signed certificate
    multipass exec "$VM_NAME" -- sudo bash /home/ubuntu/hapctl/generate-ssl.sh

    echo "✅ SSL certificate generated"
}

configure_ssl_haproxy() {
    echo ""
    echo "Configuring SSL in HAProxy..."

    # Wait for hapctl to generate configs
    echo "Waiting for hapctl agent to generate configs..."
    sleep 10

    # Configure SSL in HAProxy
    multipass exec "$VM_NAME" -- sudo bash /home/ubuntu/hapctl/configure-ssl.sh

    echo "✅ SSL configured in HAProxy"
}

setup_haproxy_config() {
    echo ""
    echo "Setting up HAProxy configuration..."

    # Create HAProxy services directories (required by agent)
    multipass exec "$VM_NAME" -- sudo mkdir -p /etc/haproxy/services.d/http
    multipass exec "$VM_NAME" -- sudo mkdir -p /etc/haproxy/services.d/tcp

    # Create log directory with proper permissions
    multipass exec "$VM_NAME" -- sudo mkdir -p /var/log/hapctl
    multipass exec "$VM_NAME" -- sudo chmod 755 /var/log/hapctl

    # Symlink entire /etc/hapctl to mounted volume (so all changes sync)
    multipass exec "$VM_NAME" -- sudo ln -sf /home/ubuntu/hapctl /etc/hapctl

    echo "✅ Configuration directory linked from volume"
}

setup_systemd_service() {
    echo ""
    echo "Setting up systemd service..."

    # Copy service file to VM
    multipass transfer "$SCRIPT_DIR/hapctl-agent.service" "$VM_NAME:/tmp/hapctl-agent.service"

    # Install service
    multipass exec "$VM_NAME" -- sudo mv /tmp/hapctl-agent.service /etc/systemd/system/hapctl-agent.service
    multipass exec "$VM_NAME" -- sudo chmod 644 /etc/systemd/system/hapctl-agent.service

    # Reload systemd and enable service
    multipass exec "$VM_NAME" -- sudo systemctl daemon-reload
    multipass exec "$VM_NAME" -- sudo systemctl enable hapctl-agent.service

    echo "✅ Systemd service installed and enabled"
    echo "   Service will NOT start automatically (use: sudo systemctl start hapctl-agent)"
}

show_info() {
    VM_IP=$(multipass info "$VM_NAME" | grep IPv4 | awk '{print $2}')

    echo ""
    echo "=========================================="
    echo "✅ Development environment is ready!"
    echo "=========================================="
    echo ""
    echo "VM Name: $VM_NAME"
    echo "VM IP: $VM_IP"
    echo ""
    echo "Binary installed:"
    echo "  /usr/local/bin/hapctl (available globally)"
    echo ""
    echo "Shared volume (configs & examples):"
    echo "  Host: $VOLUMES_DIR"
    echo "  VM:   /home/ubuntu/hapctl"
    echo "  Symlinked: /etc/hapctl -> /home/ubuntu/hapctl"
    echo ""
    echo "Test applications (Docker containers):"
    echo "  App 1: http://$VM_IP:8080"
    echo "  App 2: http://$VM_IP:8081"
    echo ""
    echo "HAProxy with SSL is running:"
    echo "  HTTP:  http://$VM_IP:80 (redirects to HTTPS)"
    echo "  HTTPS: https://$VM_IP:443 (load balancer with self-signed cert)"
    echo ""
    echo "Note: Self-signed certificate - use 'curl -k' to ignore SSL warnings"
    echo ""
    echo "Useful commands:"
    echo "  multipass shell $VM_NAME              # Access VM shell"
    echo "  multipass exec $VM_NAME -- <command>  # Run command in VM"
    echo "  multipass stop $VM_NAME                # Stop VM"
    echo "  multipass start $VM_NAME               # Start VM"
    echo "  multipass delete $VM_NAME              # Delete VM"
    echo "  multipass purge                        # Remove deleted VMs"
    echo ""
    echo "Inside VM, hapctl is globally available:"
    echo "  hapctl --version                      # Check version"
    echo "  sudo hapctl apply -f /etc/hapctl/resources/test-bind.yaml"
    echo ""
    echo "Systemd service (hapctl-agent):"
    echo "  sudo systemctl start hapctl-agent     # Start the agent"
    echo "  sudo systemctl stop hapctl-agent      # Stop the agent"
    echo "  sudo systemctl status hapctl-agent    # Check status"
    echo "  sudo systemctl restart hapctl-agent   # Restart the agent"
    echo "  sudo journalctl -u hapctl-agent -f    # View logs"
    echo ""
    echo "Edit configs on host in $VOLUMES_DIR and they sync to VM automatically!"
    echo ""
    echo "=========================================="
}

main() {
    check_multipass
    get_ssh_key

    VM_EXISTS=false
    if multipass list | grep -q "$VM_NAME"; then
        VM_EXISTS=true
        echo ""
        echo "⚠️  VM '$VM_NAME' already exists"
        echo ""
        echo "Options:"
        echo "  1) Configure existing VM (update binaries and configs)"
        echo "  2) Delete and recreate VM"
        echo "  3) Abort"
        echo ""
        read -p "Choose option (1/2/3): " -n 1 -r
        echo

        case $REPLY in
            1)
                echo "Configuring existing VM..."
                ;;
            2)
                echo "Deleting and recreating VM..."
                multipass delete "$VM_NAME"
                multipass purge
                VM_EXISTS=false
                ;;
            *)
                echo "Aborted"
                exit 0
                ;;
        esac
    fi

    prepare_volume

    if [ "$VM_EXISTS" = false ]; then
        create_vm
        wait_for_vm
    fi

    install_hapctl
    install_webhook_test

    if [ "$VM_EXISTS" = false ]; then
        install_haproxy
        start_docker_containers
        setup_haproxy_config
        setup_ssl
        setup_systemd_service
    else
        echo ""
        echo "Restarting Docker containers..."
        multipass exec "$VM_NAME" -- bash -c "cd /home/ubuntu/hapctl && docker compose restart"

        echo ""
        echo "Restarting hapctl agent..."
        multipass exec "$VM_NAME" -- sudo systemctl restart hapctl-agent
    fi

    # Start agent to generate configs (for new VMs)
    if [ "$VM_EXISTS" = false ]; then
        echo ""
        echo "Starting hapctl agent to generate initial configs..."
        multipass exec "$VM_NAME" -- sudo systemctl start hapctl-agent

        # Configure SSL after configs are generated
        configure_ssl_haproxy
    fi

    show_info
}

main "$@"
