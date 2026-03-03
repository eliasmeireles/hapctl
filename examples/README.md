# hapctl Examples

This directory contains example configuration files for hapctl.

## Configuration Files

### `config.yaml`
Main configuration file for the hapctl agent. This configures:
- Sync settings (resource path, interval)
- Monitoring settings (health checks, webhook notifications)

### `tcp-bind.yaml`
Example TCP bind configurations:
- Game server on port 7777
- MySQL proxy on port 3306

### `http-bind.yaml`
Example HTTP bind configurations:
- Web API on port 8080
- Vault UI on port 80

## Usage

### Validate a configuration
```bash
hapctl validate -f examples/tcp-bind.yaml
```

### Apply a configuration
```bash
sudo hapctl apply -f examples/tcp-bind.yaml
```

### Run the agent
```bash
sudo hapctl agent --config examples/config.yaml
```

## Directory Structure

When running the agent, create the resource directory:
```bash
sudo mkdir -p /etc/hapctl/resources
```

Place your bind configuration files in this directory, and the agent will automatically detect and apply them.

## HAProxy Configuration

Ensure your main HAProxy configuration (`/etc/haproxy/haproxy.cfg`) includes the services directory:

```
global
    log /dev/log local0
    log /dev/log local1 notice
    chroot /var/lib/haproxy
    stats socket /run/haproxy/admin.sock mode 660 level admin
    stats timeout 30s
    user haproxy
    group haproxy
    daemon

defaults
    log     global
    mode    http
    option  httplog
    option  dontlognull
    timeout connect 5000
    timeout client  50000
    timeout server  50000

# Include dynamically managed services
include /etc/haproxy/services.d/http/*.cfg
include /etc/haproxy/services.d/tcp/*.cfg
```

## Monitoring Webhook

If you configure a webhook, hapctl will send health check reports in this format:

```json
{
  "timestamp": "2026-03-03T12:00:00Z",
  "binds": [
    {
      "name": "game-server",
      "type": "tcp",
      "ip": "*",
      "port": 7777,
      "status": "healthy",
      "timestamp": "2026-03-03T12:00:00Z"
    },
    {
      "name": "mysql-proxy",
      "type": "tcp",
      "ip": "10.99.0.243",
      "port": 3306,
      "status": "unhealthy",
      "error": "connection refused",
      "timestamp": "2026-03-03T12:00:00Z"
    }
  ]
}
```
