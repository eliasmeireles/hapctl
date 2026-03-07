# Traefik Development Environment with Multipass

This setup provides a complete Traefik-based load balancer environment using Multipass VMs, as an alternative to the HAProxy setup.

## Features

- **Traefik v2.10** as reverse proxy and load balancer
- **Automatic HTTPS** with self-signed certificates
- **HTTP to HTTPS redirect** on port 80
- **Load balancing** between two nginx applications
- **Traefik Dashboard** for monitoring
- **Docker-based** backend applications
- **File-based configuration** with hot reload

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
                    ┌─────────────────┐
                    │   Multipass VM  │
                    │  traefik-dev    │
                    └────────┬────────┘
                             │
                    ┌────────▼────────┐
                    │     Traefik     │
                    │   Port 80/443   │
                    └────────┬────────┘
                             │
              ┌──────────────┴──────────────┐
              │                             │
      ┌───────▼────────┐          ┌────────▼───────┐
      │  nginx-app1    │          │  nginx-app2    │
      │  Port 8080     │          │  Port 8081     │
      └────────────────┘          └────────────────┘
```

## Access Points

After setup completes, you'll have access to:

- **HTTP**: `http://<VM_IP>:80` (redirects to HTTPS)
- **HTTPS**: `https://<VM_IP>:443` (load balanced)
- **Traefik Dashboard**: `http://<VM_IP>:8888`
- **App 1 Direct**: `http://<VM_IP>:8080`
- **App 2 Direct**: `http://<VM_IP>:8081`

## Configuration Files

### `traefik.yml`
Main Traefik static configuration:
- Entry points (HTTP/HTTPS)
- TLS settings
- Logging configuration
- Provider settings (Docker + File)

### `dynamic-config.yml`
Dynamic configuration for:
- Middlewares (redirects, security headers, rate limiting)
- Default load balancer service
- Health checks

### `docker-compose.yml`
Defines:
- Traefik container with dashboard
- Two nginx backend applications
- Docker labels for automatic service discovery

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

Traefik automatically load balances requests between both nginx applications using:
- **Algorithm**: Round-robin (default)
- **Health checks**: Every 10 seconds on `/health` endpoint
- **Sticky sessions**: Disabled (can be enabled via labels)

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

### Docker Management
```bash
# Inside VM or via exec
docker compose logs traefik -f           # View Traefik logs
docker compose logs nginx-app1 -f        # View app1 logs
docker compose restart traefik           # Restart Traefik
docker compose ps                        # List containers
docker compose down                      # Stop all containers
docker compose up -d                     # Start all containers
```

### Testing Load Balancing
```bash
# Multiple requests to see load balancing
for i in {1..10}; do curl -k https://<VM_IP>; done
```

## Customization

### Add More Backend Services

1. Add service to `docker-compose.yml`:
```yaml
nginx-app3:
  image: nginx:alpine
  networks:
    - traefik-network
  labels:
    - "traefik.enable=true"
    - "traefik.http.routers.app3.rule=PathPrefix(`/app3`)"
    - "traefik.http.services.app3.loadbalancer.server.port=80"
```

2. Update `dynamic-config.yml` to include in load balancer:
```yaml
services:
  load-balancer:
    loadBalancer:
      servers:
        - url: "http://nginx-app3:80"
```

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
multipass exec traefik-dev -- docker compose logs traefik
```

### Verify Certificate
```bash
openssl s_client -connect <VM_IP>:443 -servername localhost
```

### Check Container Health
```bash
multipass exec traefik-dev -- docker ps
```

### Restart Everything
```bash
multipass exec traefik-dev -- bash -c "cd /home/ubuntu/traefik && docker compose restart"
```

## Comparison: Traefik vs HAProxy

| Feature | Traefik | HAProxy |
|---------|---------|---------|
| Configuration | YAML/Labels | Text-based |
| Auto-discovery | Yes (Docker) | No |
| Dashboard | Built-in | External |
| Let's Encrypt | Native | Manual |
| Learning Curve | Medium | Steep |
| Performance | Good | Excellent |

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
