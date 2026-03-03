#!/bin/bash

set -e

echo "Generating self-signed SSL certificate for testing..."

CERT_DIR="/etc/haproxy/certs"
CERT_FILE="$CERT_DIR/test-cert.pem"

# Create cert directory
sudo mkdir -p "$CERT_DIR"

# Generate self-signed certificate (valid for 365 days)
sudo openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
    -keyout /tmp/test-key.pem \
    -out /tmp/test-cert.pem \
    -subj "/C=BR/ST=Test/L=Test/O=HapCTL/OU=Dev/CN=hapctl-dev.local"

# Combine cert and key into single PEM file (HAProxy format)
sudo cat /tmp/test-cert.pem /tmp/test-key.pem | sudo tee "$CERT_FILE" > /dev/null

# Set proper permissions
sudo chmod 600 "$CERT_FILE"
sudo chown root:root "$CERT_FILE"

# Cleanup temp files
sudo rm -f /tmp/test-key.pem /tmp/test-cert.pem

echo "✅ SSL certificate generated at: $CERT_FILE"
