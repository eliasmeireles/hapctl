# Webhook Test Server

Simple HTTP server to test webhook notifications from hapctl monitoring.

## Usage

### Build
```bash
make build-webhook-test
```

### Run locally
```bash
./bin/webhook-test
```

Or with custom port:
```bash
PORT=8888 ./bin/webhook-test
```

### Run in VM
The binary is automatically installed to `/usr/local/bin/webhook-test` during setup.

```bash
multipass exec hapctl-dev -- webhook-test
```

Or run as background service:
```bash
multipass exec hapctl-dev -- nohup webhook-test > /var/log/webhook-test.log 2>&1 &
```

## Endpoints

- `POST /webhook` - Receives and logs webhook notifications
- `GET /health` - Health check endpoint

## Configuration

Set the `PORT` environment variable to change the listening port (default: 9090).

## Example webhook configuration

In your hapctl config.yaml:
```yaml
monitoring:
  enabled: true
  interval: 30s
  webhook:
    url: "http://localhost:9090/webhook"
    timeout: 5s
```
