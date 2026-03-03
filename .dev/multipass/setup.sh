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

    # Validate YAML syntax
    if command -v python3 &> /dev/null; then
        if ! python3 -c "import yaml; yaml.safe_load(open('$CLOUD_INIT_FILE'))" 2>/dev/null; then
            echo "❌ Invalid YAML syntax in cloud-init file"
            cat "$CLOUD_INIT_FILE"
            exit 1
        fi
    fi

    echo "Cloud-init file generated at: $CLOUD_INIT_FILE"

    multipass launch \
        --name "$VM_NAME" \
        --cpus 2 \
        --memory 2G \
        --disk 10G \
        --cloud-init "$CLOUD_INIT_FILE" \
        22.04

    echo "✅ VM created successfully"
}

wait_for_vm() {
    echo ""
    echo "Waiting for VM to be ready..."
    sleep 10

    for i in {1..30}; do
        if multipass exec "$VM_NAME" -- systemctl is-active nginx &> /dev/null; then
            echo "✅ VM is ready"
            return 0
        fi
        echo "Waiting... ($i/30)"
        sleep 2
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

    # Copy files to volume (configs and examples only)
    cp "$SCRIPT_DIR/config.yaml" "$VOLUMES_DIR/"
    cp "$SCRIPT_DIR/test-bind.yaml" "$VOLUMES_DIR/resources/"

    echo "✅ Volume prepared at: $VOLUMES_DIR"
}

mount_volume() {
    echo ""
    echo "Mounting volume to VM..."

    # Mount the volume
    multipass mount "$VOLUMES_DIR" "$VM_NAME:/home/ubuntu/hapctl"

    echo "✅ Volume mounted"
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
    echo "Test application (nginx):"
    echo "  http://$VM_IP:8080"
    echo ""
    echo "After starting hapctl agent, HAProxy will be available at:"
    echo "  http://$VM_IP:80"
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

    if multipass list | grep -q "$VM_NAME"; then
        echo ""
        echo "⚠️  VM '$VM_NAME' already exists"
        read -p "Do you want to delete and recreate it? (y/N): " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            multipass delete "$VM_NAME"
            multipass purge
        else
            echo "Aborted"
            exit 0
        fi
    fi

    prepare_volume
    create_vm
    wait_for_vm
    mount_volume
    install_hapctl
    install_webhook_test
    install_haproxy
    setup_haproxy_config
    setup_systemd_service
    show_info
}

main "$@"
