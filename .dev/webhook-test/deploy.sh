#!/bin/bash

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
VM_NAME="${VM_NAME:-hapctl-dev}"

echo "Building webhook-test..."
cd "$PROJECT_ROOT"
make build-webhook-test

echo ""
echo "Deploying webhook-test to VM: $VM_NAME"

if ! multipass list | grep -q "$VM_NAME"; then
    echo "Error: VM '$VM_NAME' not found"
    echo "Available VMs:"
    multipass list
    exit 1
fi

echo "Transferring binary to VM..."
multipass transfer "$PROJECT_ROOT/bin/webhook-test" "$VM_NAME:/tmp/webhook-test"

echo "Installing to /usr/local/bin..."
multipass exec "$VM_NAME" -- sudo mv /tmp/webhook-test /usr/local/bin/webhook-test
multipass exec "$VM_NAME" -- sudo chmod +x /usr/local/bin/webhook-test

echo ""
echo "✅ webhook-test deployed successfully!"
echo ""
echo "To run the webhook server:"
echo "  multipass exec $VM_NAME -- webhook-test"
echo ""
echo "To run in background:"
echo "  multipass exec $VM_NAME -- bash -c 'nohup webhook-test > /tmp/webhook-test.log 2>&1 &'"
echo ""
echo "To view logs:"
echo "  multipass exec $VM_NAME -- tail -f /tmp/webhook-test.log"
echo ""
