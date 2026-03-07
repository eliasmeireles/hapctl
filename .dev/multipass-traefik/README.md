# Traefik Development Environment with Multipass

This setup provides a complete Traefik-based load balancer environment using Multipass VMs, as an alternative to the HAProxy setup.

## Features

- **Traefik v2.10.7** installed as native binary (systemd service)
- **Automatic HTTPS** with self-signed certificates
- **HTTP to HTTPS redirect** on port 80
- **Load balancing** between two HTTP test applications
- **Traefik Dashboard** for monitoring
- **Native installation** (no Docker required)
- **File-based configuration** with hot reload
- **Systemd integration** for service management

## Prerequisites

- [Multipass](https://multipass.run/) installed
- SSH key pair (`~/.ssh/id_rsa.pub` or `~/.ssh/id_ed25519.pub`)

## Quick Start

```bash
cd .dev/multipass-traefik
chmod +x setup.sh
./setup.sh
```

The script will:
1. Create a Ubuntu 22.04 VM with Docker
2. Generate self-signed SSL certificates
3. Start Traefik and two nginx applications
4. Configure automatic HTTPS redirect

## Architecture

```
                    ┌─────────────────────────────┐
                    │      Multipass VM           │
                    │      traefik-dev            │
                    │                             │
                    │  ┌──────────────────────┐   │
                    │  │  Traefik (systemd)   │   │
                    │  │  /usr/local/bin      │   │
                    │  │  Port 80/443         │   │
                    │  └──────────┬───────────┘   │
                    │             │               │
                    │  ┌──────────┴──────────┐    │
                    │  │                     │    │
                    │  ▼                     ▼    │
                    │ ┌──────────┐   ┌──────────┐│
                    │ │ HTTP App1│   │ HTTP App2││
                    │ │ Port 8080│   │ Port 8081││
                    │ └──────────┘   └──────────┘│
                    └─────────────────────────────┘
```

## Access Points

After setup completes, you'll have access to:

- **HTTP**: `http://<VM_IP>:80` (redirects to HTTPS)
- **HTTPS**: `https://<VM_IP>:443` (load balanced)
- **Traefik Dashboard**: `http://<VM_IP>:8080` (Traefik API)
- **App 1 Direct**: `http://<VM_IP>:8080` (Python HTTP server)
- **App 2 Direct**: `http://<VM_IP>:8081` (Python HTTP server)

## Configuration Files

### `traefik.yml`
Main Traefik static configuration:
- Entry points (HTTP/HTTPS)
- TLS settings
- Logging configuration
- File provider settings (no Docker)

### `dynamic-config.yml`
Dynamic configuration for:
- Middlewares (redirects, security headers, rate limiting)
- Default load balancer service
- Backend servers (127.0.0.1:8080, 127.0.0.1:8081)
- Health checks

### `traefik.service`
Systemd service unit file:
- Runs Traefik as a system service
- Auto-restart on failure
- Security hardening options

## SSL Certificates

Self-signed certificates are generated automatically:
- **Location**: `.volumes/traefik/certs/`
- **Validity**: 365 days
- **Domains**: `*.localhost`, `app1.localhost`, `app2.localhost`

To use with curl:
```bash
curl -k https://<VM_IP>
```

## Traefik Dashboard

Access the dashboard at `http://<VM_IP>:8888` to view:
- Active routers and services
- Middleware configuration
- Real-time metrics
- Health status

## Load Balancing

Traefik load balances requests between both HTTP applications using:
- **Algorithm**: Round-robin (default)
- **Health checks**: Every 10 seconds on `/health` endpoint
- **Backends**: 127.0.0.1:8080 and 127.0.0.1:8081
- **Sticky sessions**: Disabled (can be enabled via configuration)

## Dynamic Configuration

Traefik watches for configuration changes in real-time:

1. Edit files in `.volumes/traefik/config/`
2. Changes are detected automatically
3. No restart required

## Useful Commands

### VM Management
```bash
multipass shell traefik-dev              # Access VM
multipass stop traefik-dev               # Stop VM
multipass start traefik-dev              # Start VM
multipass delete traefik-dev             # Delete VM
multipass purge                          # Remove deleted VMs
```

### Traefik Service Management
```bash
# Inside VM or via multipass exec
sudo systemctl status traefik            # Check Traefik status
sudo systemctl restart traefik           # Restart Traefik
sudo systemctl stop traefik              # Stop Traefik
sudo systemctl start traefik             # Start Traefik
sudo journalctl -u traefik -f            # View Traefik logs (follow)
sudo journalctl -u traefik --since today # View today's logs
traefik version                          # Check Traefik version
```

### Testing Load Balancing
```bash
# Multiple requests to see load balancing
for i in {1..10}; do curl -k https://<VM_IP>; done
```

## Customization

### Add More Backend Services

1. Start a new HTTP server on a different port (e.g., 8082):
```bash
cd /home/ubuntu/traefik/html/app3
python3 -m http.server 8082 &
```

2. Update `dynamic-config.yml` to include in load balancer:
```yaml
services:
  load-balancer:
    loadBalancer:
      servers:
        - url: "http://127.0.0.1:8080"
        - url: "http://127.0.0.1:8081"
        - url: "http://127.0.0.1:8082"
```

3. Traefik will automatically reload the configuration

### Enable Let's Encrypt (Production)

Edit `traefik.yml`:
```yaml
certificatesResolvers:
  letsencrypt:
    acme:
      email: your-email@example.com
      storage: /etc/traefik/acme.json
      httpChallenge:
        entryPoint: web
```

### Add Custom Middleware

Edit `dynamic-config.yml`:
```yaml
http:
  middlewares:
    custom-headers:
      headers:
        customRequestHeaders:
          X-Custom-Header: "value"
```

## Troubleshooting

### Check Traefik Logs
```bash
multipass exec traefik-dev -- sudo journalctl -u traefik -n 100
multipass exec traefik-dev -- sudo tail -f /var/log/traefik/traefik.log
```

### Check Traefik Status
```bash
multipass exec traefik-dev -- sudo systemctl status traefik
```

### Verify Certificate
```bash
openssl s_client -connect <VM_IP>:443 -servername localhost
```

### Check Backend Applications
```bash
multipass exec traefik-dev -- curl http://127.0.0.1:8080
multipass exec traefik-dev -- curl http://127.0.0.1:8081
```

### Restart Traefik
```bash
multipass exec traefik-dev -- sudo systemctl restart traefik
```

### Validate Configuration
```bash
multipass exec traefik-dev -- traefik version
multipass exec traefik-dev -- cat /etc/traefik/traefik.yml
```

## Comparison: Traefik vs HAProxy

| Feature        | Traefik        | HAProxy         |
| -------------- | -------------- | --------------- |
| Configuration  | YAML           | Text-based      |
| Installation   | Single binary  | Package manager |
| Dashboard      | Built-in       | External        |
| Let's Encrypt  | Native support | Manual setup    |
| Hot reload     | Automatic      | Manual reload   |
| Learning Curve | Medium         | Steep           |
| Performance    | Good           | Excellent       |
| File watching  | Yes            | No              |

## Clean Up

```bash
multipass delete traefik-dev
multipass purge
rm -rf .volumes .out
```

## References

- [Traefik Documentation](https://doc.traefik.io/traefik/)
- [Traefik Docker Provider](https://doc.traefik.io/traefik/providers/docker/)
- [Multipass Documentation](https://multipass.run/docs)
