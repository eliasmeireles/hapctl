# hapctl

[![CI](https://github.com/eliasmeireles/hapctl/actions/workflows/ci.yml/badge.svg)](https://github.com/eliasmeireles/hapctl/actions/workflows/ci.yml)
[![Release](https://github.com/eliasmeireles/hapctl/actions/workflows/release.yml/badge.svg)](https://github.com/eliasmeireles/hapctl/actions/workflows/release.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/eliasmeireles/hapctl)](https://goreportcard.com/report/github.com/eliasmeireles/hapctl)

HAProxy Control - A modern CLI tool for managing HAProxy configurations dynamically.

## Features

- **Declarative Configuration**: Define HAProxy binds using YAML files
- **Automatic Sync**: Watch for configuration changes and apply them automatically
  - Hash-based change detection to avoid unnecessary HAProxy reloads
  - Automatic cleanup of orphaned configurations
  - File watcher with 5-minute fallback scheduler
- **HTTP & TCP Support**: Configure both HTTP and TCP frontends/backends
- **Health Monitoring**: Monitor bind health and send webhook notifications
  - Configurable monitoring interval (default: 30s)
  - Webhook notifications with detailed bind status
  - JSON-formatted health reports
- **CLI Interface**: Manage configurations through command-line interface
- **Logging**: Comprehensive logging with automatic rotation (7-day retention)
- **HAProxy Installation**: Built-in installer for HAProxy on Linux systems

## Installation

### Prerequisites

- Linux system (Ubuntu/Debian recommended)
- Go 1.22+ (for building from source)
- sudo privileges

### Build from Source

```bash
make build
```

### Install HAProxy

```bash
sudo hapctl install
```

This will:
- Install HAProxy using your system's package manager
- Configure HAProxy to include hapctl-managed configs from `/etc/haproxy/services.d/`
- Enable and start the HAProxy service

## Usage

### Install HAProxy (if not already installed)

```bash
# Check if HAProxy is installed
hapctl install --check

# Install HAProxy (requires sudo)
sudo hapctl install
```

### Start the Agent

```bash
# Run directly
sudo hapctl agent --config /etc/hapctl/config.yaml

# With debug logging
sudo hapctl agent --config /etc/hapctl/config.yaml --debug
```

Or use systemd:

```bash
# Start the agent
sudo systemctl start hapctl-agent

# Enable on boot
sudo systemctl enable hapctl-agent

# Check status
sudo systemctl status hapctl-agent

# View logs
sudo journalctl -u hapctl-agent -f
```

### Apply a Bind Configuration

```bash
# Apply a single bind configuration
sudo hapctl apply -f /path/to/bind.yaml

# Validate before applying
sudo hapctl validate -f /path/to/bind.yaml

# With agent running, just create/update YAML files in resource directory
# The agent will automatically detect and apply changes
```

### Validate a configuration

```bash
hapctl validate --file /path/to/bind-config.yaml
```

### Systemd Service Management

```bash
# Install systemd service (with default config)
sudo hapctl service install

# Install with custom config path
sudo hapctl service install --config /path/to/config.yaml

# Install with custom service file
sudo hapctl service install --service-file /path/to/custom.service

# Check service status
sudo hapctl service status

# Uninstall service
sudo hapctl service uninstall

# After installation, manage with systemctl
sudo systemctl start hapctl-agent
sudo systemctl stop hapctl-agent
sudo systemctl restart hapctl-agent
sudo journalctl -u hapctl-agent -f
```

## Configuration

### Agent Configuration

Create `/etc/hapctl/config.yaml`:

```yaml
sync:
  resource-path: /etc/hapctl/resources  # Directory to watch for YAML files
  interval: 5s                            # Sync check interval
  enabled: true                           # Enable automatic sync

monitoring:
  enabled: true                           # Enable health monitoring
  interval: 30s                           # Health check interval
  webhook:
    url: "http://localhost:9090/webhook" # Webhook endpoint for notifications
    timeout: 5s                           # Webhook request timeout
    headers:                              # Optional custom headers
      - name: "X-Source"
        value: "hapctl-agent"
```

### How Sync Works

1. **Hash-based Change Detection**: Calculates SHA256 hash of all YAML files
   - Only triggers HAProxy reload when content actually changes
   - Prevents unnecessary reloads and service disruptions

2. **File Watcher**: Uses `fsnotify` for real-time file change detection
   - Monitors create, write, and delete events
   - Immediate response to configuration changes

3. **Fallback Scheduler**: 5-minute periodic resync
   - Ensures sync even if file watcher fails
   - Performs full reconciliation of all resources

4. **Orphaned Config Cleanup**: Automatically removes stale configurations
   - Detects `hapctl-*.cfg` files without corresponding YAML resources
   - Cleans up on both periodic sync and forced resync

### Bind Configuration Example

```yaml
binds:
  - name: game-server
    override: true
    enabled: true
    description: Game server bind
    type: tcp
    ip: "*"
    port: 7777
    backend:
      servers:
        - name: server1
          address: 127.0.0.1:7777
```

## Development

### Build

```bash
# Build hapctl binary
make build

# Build webhook-test server (for testing monitoring)
make build-webhook-test

# Clean build artifacts
make clean
```

### Test

```bash
# Run all tests with coverage
make test

# Run linters
make lint

# Format code
make fmt
```

### Development Environment (Multipass)

A complete development environment is available using Multipass:

```bash
cd .dev/multipass
./setup.sh
```

This creates a VM with:
- HAProxy installed and configured
- hapctl agent installed as systemd service
- nginx test application on port 8080
- Shared volume for configs at `.dev/multipass/.volumes`

### Webhook Test Server

A simple HTTP server for testing monitoring webhooks:

```bash
# Build and deploy to VM
cd .dev/webhook-test
./deploy.sh

# Run in VM
multipass exec hapctl-dev -- webhook-test

# Or in background
multipass exec hapctl-dev -- bash -c 'nohup webhook-test > /tmp/webhook-test.log 2>&1 &'
```

Endpoints:
- `POST /webhook` - Receives and logs webhook notifications
- `GET /health` - Health check

Default port: 9090 (configurable via `PORT` env var)

## Architecture

### Components

- **Agent** (`internal/cmd/agent.go`): Main entry point, orchestrates all components
- **Sync Watcher** (`internal/sync/watcher.go`):
  - Watches resource directory for YAML changes
  - Calculates content hashes to detect actual changes
  - Registers binds with monitor when applied
  - Cleans up orphaned configurations
- **HAProxy Manager** (`internal/haproxy/manager.go`):
  - Generates HAProxy configuration files
  - Validates and reloads HAProxy
  - Manages service configs in `/etc/haproxy/services.d/`
- **Monitor** (`internal/monitor/monitor.go`):
  - Performs TCP health checks on registered binds
  - Sends webhook notifications with bind status
  - Runs on configurable interval (default: 30s)
- **Config Loader** (`internal/config/loader.go`):
  - Loads and validates YAML configurations
  - Applies defaults and validates settings

### Directory Structure

```
/etc/hapctl/
├── config.yaml              # Agent configuration
└── resources/               # YAML bind definitions
    └── *.yaml

/etc/haproxy/
├── haproxy.cfg             # Main HAProxy config (auto-updated)
└── services.d/             # hapctl-managed configs
    ├── http/               # HTTP service configs
    │   └── hapctl-*.cfg
    └── tcp/                # TCP service configs
        └── hapctl-*.cfg

/var/log/hapctl/
└── agent.log               # Agent logs
```
