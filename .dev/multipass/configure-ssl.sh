#!/bin/bash

set -e

echo "Configuring SSL for HAProxy..."

# Generate self-signed certificate
bash /home/ubuntu/hapctl/generate-ssl.sh

# Stop hapctl agent to prevent config regeneration
echo "Stopping hapctl agent..."
sudo systemctl stop hapctl-agent

# Patch the main HAProxy config to add SSL
HAPROXY_CONFIG="/etc/haproxy/haproxy.cfg"

if [ -f "$HAPROXY_CONFIG" ]; then
    echo "Patching $HAPROXY_CONFIG to add SSL..."

    # Backup original
    sudo cp "$HAPROXY_CONFIG" "${HAPROXY_CONFIG}.bak"

    # Replace "bind *:443" with "bind *:443 ssl crt /etc/haproxy/certs/test-cert.pem"
    # Only replace if not already patched
    if grep -q "bind \*:443 ssl" "$HAPROXY_CONFIG"; then
        echo "⚠️  SSL already configured in HAProxy config"
    else
        sudo sed -i 's|bind \*:443$|bind *:443 ssl crt /etc/haproxy/certs/test-cert.pem|g' "$HAPROXY_CONFIG"
        echo "✅ SSL configured in HAProxy config"
    fi

    # Reload HAProxy
    sudo systemctl reload haproxy
    echo "✅ HAProxy reloaded"

    echo ""
    echo "⚠️  Note: hapctl-agent is STOPPED to prevent config regeneration without SSL"
    echo "   To restart: sudo systemctl start hapctl-agent"
    echo "   (Agent will regenerate config without SSL if restarted)"
else
    echo "❌ Config file not found: $HAPROXY_CONFIG"
    exit 1
fi
