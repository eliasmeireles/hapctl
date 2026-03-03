#!/bin/bash

set -e

VM_NAME="hapctl-dev"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
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
    
    # Substitute SSH key in cloud-init
    envsubst < "$SCRIPT_DIR/cloud-init.yaml" > /tmp/hapctl-cloud-init.yaml
    
    multipass launch \
        --name "$VM_NAME" \
        --cpus 2 \
        --memory 2G \
        --disk 10G \
        --cloud-init /tmp/hapctl-cloud-init.yaml \
        22.04
    
    rm /tmp/hapctl-cloud-init.yaml
    
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

install_hapctl() {
    echo ""
    echo "Building and installing hapctl..."
    
    cd "$PROJECT_ROOT"
    make build
    
    VM_IP=$(multipass info "$VM_NAME" | grep IPv4 | awk '{print $2}')
    
    echo "Copying hapctl binary to VM..."
    multipass transfer "$PROJECT_ROOT/bin/hapctl" "$VM_NAME:/tmp/hapctl"
    
    multipass exec "$VM_NAME" -- sudo mv /tmp/hapctl /usr/local/bin/hapctl
    multipass exec "$VM_NAME" -- sudo chmod +x /usr/local/bin/hapctl
    
    echo "Installing HAProxy..."
    multipass exec "$VM_NAME" -- sudo /usr/local/bin/hapctl install
    
    echo "✅ hapctl installed successfully"
}

setup_haproxy_config() {
    echo ""
    echo "Setting up HAProxy configuration..."
    
    # Create hapctl config directory
    multipass exec "$VM_NAME" -- sudo mkdir -p /etc/hapctl
    multipass exec "$VM_NAME" -- sudo mkdir -p /etc/haproxy/hapctl/resources
    
    # Copy example configs
    multipass transfer "$SCRIPT_DIR/test-bind.yaml" "$VM_NAME:/tmp/test-bind.yaml"
    multipass exec "$VM_NAME" -- sudo mv /tmp/test-bind.yaml /etc/haproxy/hapctl/resources/test-bind.yaml
    
    echo "✅ Configuration files copied"
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
    echo "Inside VM, test hapctl:"
    echo "  sudo hapctl apply -f /etc/haproxy/hapctl/resources/test-bind.yaml"
    echo "  sudo hapctl agent --config /etc/hapctl/config.yaml"
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
    
    create_vm
    wait_for_vm
    install_hapctl
    setup_haproxy_config
    show_info
}

main "$@"
