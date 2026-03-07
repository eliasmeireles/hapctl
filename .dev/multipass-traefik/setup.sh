#!/bin/bash

set -e

VM_NAME="traefik-dev"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VOLUMES_DIR="${SCRIPT_DIR}/.volumes"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

echo "=== Traefik Development Environment ==="
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

    OUT_DIR="$SCRIPT_DIR/.out"
    mkdir -p "$OUT_DIR"
    CLOUD_INIT_FILE="$OUT_DIR/cloud-init-generated.yaml"

    sed "s|\${SSH_PUBLIC_KEY}|${SSH_PUBLIC_KEY}|g" "$SCRIPT_DIR/cloud-init.yaml" > "$CLOUD_INIT_FILE"

    if ! grep -q "ssh_authorized_keys:" "$CLOUD_INIT_FILE"; then
        echo "❌ Failed to generate cloud-init file"
        cat "$CLOUD_INIT_FILE"
        exit 1
    fi

    echo "Cloud-init file generated at: $CLOUD_INIT_FILE"

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
        --mount "$VOLUMES_DIR:/home/ubuntu/traefik" \
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
    mkdir -p "$VOLUMES_DIR/traefik/config"
    mkdir -p "$VOLUMES_DIR/traefik/certs"
    mkdir -p "$VOLUMES_DIR/traefik/logs"

    cp "$SCRIPT_DIR/docker-compose.yml" "$VOLUMES_DIR/"
    cp "$SCRIPT_DIR/traefik.yml" "$VOLUMES_DIR/traefik/"
    cp "$SCRIPT_DIR/dynamic-config.yml" "$VOLUMES_DIR/traefik/config/"
    cp -r "$SCRIPT_DIR/html" "$VOLUMES_DIR/"
    cp "$SCRIPT_DIR/generate-ssl.sh" "$VOLUMES_DIR/"

    echo "✅ Volume prepared at: $VOLUMES_DIR"
}

setup_ssl() {
    echo ""
    echo "Setting up SSL certificate..."

    multipass exec "$VM_NAME" -- chmod +x /home/ubuntu/traefik/generate-ssl.sh
    multipass exec "$VM_NAME" -- sudo bash /home/ubuntu/traefik/generate-ssl.sh

    echo "✅ SSL certificate generated"
}

start_docker_containers() {
    echo ""
    echo "Starting Docker containers with Traefik..."

    multipass exec "$VM_NAME" -- bash -c "cd /home/ubuntu/traefik && docker compose up -d"

    sleep 5

    multipass exec "$VM_NAME" -- docker ps

    echo "✅ Docker containers started"
}

show_info() {
    VM_IP=$(multipass info "$VM_NAME" | grep IPv4 | awk '{print $2}')

    echo ""
    echo "=========================================="
    echo "✅ Traefik development environment is ready!"
    echo "=========================================="
    echo ""
    echo "VM Name: $VM_NAME"
    echo "VM IP: $VM_IP"
    echo ""
    echo "Shared volume:"
    echo "  Host: $VOLUMES_DIR"
    echo "  VM:   /home/ubuntu/traefik"
    echo ""
    echo "Test applications (Docker containers):"
    echo "  App 1: http://$VM_IP:8080"
    echo "  App 2: http://$VM_IP:8081"
    echo ""
    echo "Traefik with SSL is running:"
    echo "  HTTP:  http://$VM_IP:80 (redirects to HTTPS)"
    echo "  HTTPS: https://$VM_IP:443 (load balancer with self-signed cert)"
    echo "  Dashboard: http://$VM_IP:8888"
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
    echo "Traefik management:"
    echo "  docker compose logs traefik -f        # View Traefik logs"
    echo "  docker compose restart traefik        # Restart Traefik"
    echo ""
    echo "Edit configs on host in $VOLUMES_DIR and they sync to VM automatically!"
    echo ""
    echo "=========================================="
}

main() {
    check_multipass
    get_ssh_key

    if multipass list | grep -q "$VM_NAME"; then
        echo "⚠️  VM $VM_NAME already exists"
        read -p "Do you want to delete and recreate it? (y/N) " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            multipass delete "$VM_NAME"
            multipass purge
        else
            echo "Aborting..."
            exit 1
        fi
    fi

    prepare_volume
    create_vm
    wait_for_vm
    setup_ssl
    start_docker_containers
    show_info
}

main
