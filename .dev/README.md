# Development Environment

This directory contains development and testing environments for hapctl.

## Multipass Environment

The Multipass environment provides a complete VM setup for testing hapctl with a real HAProxy and nginx backend.

### Prerequisites

- [Multipass](https://multipass.run/) installed
- SSH key pair (`~/.ssh/id_rsa.pub` or `~/.ssh/id_ed25519.pub`)
- `envsubst` command (usually available via `gettext` package)

### Quick Start

```bash
# Create and setup the development VM
cd .dev/multipass
./setup.sh
```

This will:
1. Create a Ubuntu 22.04 VM named `hapctl-dev`
2. Install nginx on port 8080 with a test application
3. Build and install hapctl in the VM
4. Install HAProxy
5. Copy example configuration files

### What's Inside

The VM includes:

- **Nginx Test App** (port 8080)
  - Simple HTML page showing it's running
  - Health check endpoint at `/health`
  
- **HAProxy** (managed by hapctl)
  - Will proxy port 80 → 8080 when configured
  
- **hapctl** (pre-installed)
  - Ready to manage HAProxy configurations

### Testing hapctl

#### 1. Access the VM

```bash
multipass shell hapctl-dev
```

#### 2. Test the backend directly

```bash
curl http://localhost:8080
curl http://localhost:8080/health
```

#### 3. Apply HAProxy configuration

```bash
sudo hapctl apply -f /etc/haproxy/hapctl/resources/test-bind.yaml
```

#### 4. Test HAProxy proxy

```bash
curl http://localhost:80
```

You should see the same content as port 8080, but now proxied through HAProxy!

#### 5. Start the agent (optional)

```bash
# Copy config to expected location
sudo cp /etc/haproxy/hapctl/resources/config.yaml /etc/hapctl/config.yaml

# Start agent
sudo hapctl agent --config /etc/hapctl/config.yaml
```

The agent will:
- Monitor `/etc/haproxy/hapctl/resources/` for YAML changes
- Automatically apply configuration changes
- Perform health checks every 10 seconds

### Testing Configuration Changes

1. Create a new bind configuration:

```bash
sudo nano /etc/haproxy/hapctl/resources/another-service.yaml
```

2. Add content:

```yaml
binds:
  - name: another-service
    override: true
    enabled: true
    description: Another test service
    type: tcp
    port: 9000
    backend:
      servers:
        - name: backend1
          address: 127.0.0.1:8080
```

3. If agent is running, it will auto-apply. Otherwise:

```bash
sudo hapctl apply -f /etc/haproxy/hapctl/resources/another-service.yaml
```

### Useful Commands

```bash
# VM Management
multipass list                    # List all VMs
multipass info hapctl-dev         # Show VM details
multipass shell hapctl-dev        # Access VM shell
multipass stop hapctl-dev         # Stop VM
multipass start hapctl-dev        # Start VM
multipass delete hapctl-dev       # Delete VM
multipass purge                   # Remove deleted VMs

# Inside VM
sudo systemctl status haproxy     # Check HAProxy status
sudo systemctl status nginx       # Check nginx status
sudo hapctl install --check       # Verify HAProxy installation
sudo journalctl -u haproxy -f     # Follow HAProxy logs
tail -f /var/log/hapctl/hapctl.log  # Follow hapctl logs (or ~/.hapctl/log/hapctl.log)
```

### Troubleshooting

#### Port 80 already in use

If HAProxy fails to start because port 80 is in use:

```bash
# Check what's using port 80
sudo netstat -tlnp | grep :80

# If it's nginx on port 80, stop it
sudo systemctl stop nginx
```

#### Logs location

Depending on permissions, logs may be in:
- `/var/log/hapctl/` (if running as root)
- `~/.hapctl/log/` (fallback location)

#### HAProxy validation fails

Check HAProxy configuration:

```bash
sudo haproxy -c -f /etc/haproxy/haproxy.cfg
```

View generated configurations:

```bash
ls -la /etc/haproxy/services.d/http/
ls -la /etc/haproxy/services.d/tcp/
cat /etc/haproxy/services.d/http/test-app.cfg
```

### Cleanup

To completely remove the development environment:

```bash
multipass delete hapctl-dev
multipass purge
```

## Architecture

```
┌─────────────────────────────────────┐
│         hapctl-dev VM               │
│                                     │
│  ┌──────────┐      ┌─────────────┐ │
│  │  hapctl  │─────▶│   HAProxy   │ │
│  │  agent   │      │   (port 80) │ │
│  └──────────┘      └──────┬──────┘ │
│       │                   │        │
│       │ monitors          │ proxy  │
│       ▼                   ▼        │
│  ┌──────────┐      ┌─────────────┐ │
│  │  YAML    │      │    nginx    │ │
│  │  configs │      │  (port 8080)│ │
│  └──────────┘      └─────────────┘ │
│                                     │
└─────────────────────────────────────┘
```

## Next Steps

- Modify `test-bind.yaml` to test different configurations
- Add multiple backend servers to test load balancing
- Test monitoring and webhook notifications
- Experiment with TCP binds
- Test configuration validation and error handling
