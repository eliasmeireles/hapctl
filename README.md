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

## Installation

```bash
go install github.com/eliasmeireles/hapctl@latest
```

Or build from source:

```bash
make build
```

## Usage

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

## Configuration

### CLI Configuration (`/etc/hapctl/config.yaml`)

```yaml
sync:
  resource-path: /etc/haproxy/hapctl/resources
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

## Architecture

- `cmd/hapctl`: CLI entry point
- `internal/config`: Configuration parsing and validation
- `internal/haproxy`: HAProxy configuration generation and management
- `internal/sync`: File watching and synchronization service
- `internal/monitor`: Health monitoring and webhook notifications
- `internal/logger`: Logging system with rotation

