# hapctl

[![CI](https://github.com/eliasmeireles/hapctl/actions/workflows/ci.yml/badge.svg)](https://github.com/eliasmeireles/hapctl/actions/workflows/ci.yml)
[![Release](https://github.com/eliasmeireles/hapctl/actions/workflows/release.yml/badge.svg)](https://github.com/eliasmeireles/hapctl/actions/workflows/release.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/eliasmeireles/hapctl)](https://goreportcard.com/report/github.com/eliasmeireles/hapctl)

HAProxy Control CLI and Agent - A tool to manage HAProxy configurations dynamically through YAML files with monitoring capabilities.

## Features

- **Dynamic Configuration Management**: Monitor and sync HAProxy configurations from YAML files
- **File Watching**: Automatically detect and apply configuration changes
- **Health Monitoring**: Periodic health checks for configured binds with webhook notifications
- **CLI Interface**: Manage configurations through command-line interface
- **Logging**: Comprehensive logging with automatic rotation (7-day retention)
- **HAProxy Installation**: Built-in installer for HAProxy on Linux systems

## Prerequisites

- **HAProxy**: Required for the agent to function
  - Can be installed automatically using `sudo hapctl install`
  - Supports apt (Debian/Ubuntu), yum (CentOS/RHEL), and dnf (Fedora)
  - Or install manually: `sudo apt-get install haproxy` (Debian/Ubuntu)

## Installation

```bash
go install github.com/eliasmeireles/hapctl@latest
```

Or build from source:

```bash
make build
```

## Usage

### Install HAProxy (if not already installed)

```bash
# Check if HAProxy is installed
hapctl install --check

# Install HAProxy (requires sudo)
sudo hapctl install
```

### Start the agent (sync + monitor)

```bash
hapctl agent --config /etc/hapctl/config.yaml
```

### Apply a configuration

```bash
hapctl apply --file /path/to/bind-config.yaml
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

### CLI Configuration (`/etc/hapctl/config.yaml`)

```yaml
sync:
  resource-path: /etc/hapctl/resources
  interval: 5s
  enabled: true

monitoring:
  enabled: true
  interval: 5s
  webhook:
    url: http://localhost:8080/webhook
    headers:
      - name: Content-Type
        value: application/json
      - name: ApiKey
        value: 123456
```

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

### Testing Environment

A complete development environment using Multipass is available in `.dev/multipass/`:

```bash
cd .dev/multipass
./setup.sh
```

This creates a VM with:
- Nginx running on port 8080 (test backend)
- HAProxy installed and ready
- hapctl pre-installed
- Example configurations

See [.dev/README.md](.dev/README.md) for detailed instructions.

## Architecture

- `cmd/hapctl`: CLI entry point
- `internal/config`: Configuration parsing and validation
- `internal/haproxy`: HAProxy configuration generation and management
- `internal/sync`: File watching and synchronization service
- `internal/monitor`: Health monitoring and webhook notifications
- `internal/logger`: Logging system with rotation
