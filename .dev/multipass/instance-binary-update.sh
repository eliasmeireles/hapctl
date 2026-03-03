#!/bin/bash

set -e

echo "Updating hapctl binary..."
systemctl stop hapctl-agent

echo "Copying binary to /usr/local/bin/hapctl..."
cp /home/ubuntu/hapctl/hapctl /usr/local/bin/hapctl

echo "Clearing log..."
rm -rf /var/log/hapctl/hapctl.log || true

echo "Starting hapctl-agent..."
systemctl start hapctl-agent

sleep 5

tail -f /var/log/hapctl/hapctl.log