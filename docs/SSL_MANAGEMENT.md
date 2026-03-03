# SSL Certificate Management

hapctl provides automatic SSL certificate management using Let's Encrypt with support for both regular and wildcard certificates.

## Table of Contents

- [Features](#features)
- [Prerequisites](#prerequisites)
- [Configuration](#configuration)
- [Wildcard Certificates](#wildcard-certificates)
- [DNS Providers](#dns-providers)
- [Certificate Renewal](#certificate-renewal)
- [HAProxy Integration](#haproxy-integration)
- [Troubleshooting](#troubleshooting)

## Features

- ✅ Automatic certificate generation with Let's Encrypt
- ✅ Support for wildcard certificates (`*.example.com`)
- ✅ Automatic renewal (configurable interval, default: twice daily)
- ✅ HAProxy PEM file generation (fullchain + privkey)
- ✅ Multiple DNS providers support (Cloudflare, Route53, DigitalOcean, etc.)
- ✅ Staging mode for testing
- ✅ Configurable key size and type (RSA, ECDSA)
- ✅ Email validation with warnings

## Prerequisites

### 1. Install certbot

```bash
# Ubuntu/Debian
sudo apt-get update
sudo apt-get install certbot

# For wildcard certificates, install DNS plugin
# Example for Cloudflare:
sudo apt-get install python3-certbot-dns-cloudflare
```

### 2. DNS Provider (for wildcard certificates only)

If you plan to use wildcard certificates, you need:
- Access to your DNS provider's API
- API credentials (token or key)

Supported providers:
- Cloudflare (`certbot-dns-cloudflare`)
- AWS Route53 (`certbot-dns-route53`)
- DigitalOcean (`certbot-dns-digitalocean`)
- Google Cloud DNS (`certbot-dns-google`)
- And many more...

## Configuration

### Main Configuration

Edit `/etc/hapctl/config.yaml`:

```yaml
ssl:
  enabled: true                    # Enable SSL management
  config-path: /etc/hapctl/ssl     # Directory for SSL configs
  cert-path: /etc/haproxy/certs    # Where to store HAProxy PEM files
  renewal-check: 12h               # Check for renewal twice daily
  email: admin@example.com         # Required for Let's Encrypt notifications
  
  # Optional: For wildcard certificates
  dns-provider: cloudflare         # DNS provider name
  dns-credentials:
    api-token: "your-api-token"    # Provider-specific credentials
```

### SSL Domain Configuration

Create YAML files in `/etc/hapctl/ssl/` for each set of domains:

**Example: `/etc/hapctl/ssl/my-domains.yaml`**

```yaml
config:
  # Email for Let's Encrypt (REQUIRED)
  mail: admin@example.com
  
  # List of domains (REQUIRED)
  domain:
    - example.com
    - "*.example.com"
    - www.example.com
  
  # Optional: Override global DNS provider
  dns-provider: cloudflare
  dns-credentials:
    api-token: "your-cloudflare-api-token"
  
  # Optional: Use staging server for testing
  staging: false
  
  # Optional: Key configuration
  key-size: 2048        # 2048, 3072, or 4096
  key-type: rsa         # rsa or ecdsa
  
  # Optional: ECDSA curve (when key-type is ecdsa)
  ecdsa-curve: P-256    # P-256, P-384, or P-521
  
  # Optional: Preferred chain
  preferred-chain: "ISRG Root X1"
  
  # Optional: OCSP Must-Staple
  must-staple: false
  
  # Optional: Reuse private key on renewal
  reuse-key: false
```

### Configuration Validation

If email is not configured, hapctl will log a warning and skip the SSL config:

```
[WARN] SSL config /etc/hapctl/ssl/my-domains.yaml ignored: email is required but not set
[WARN] Configure 'mail' in SSL config or 'ssl.email' in main config
```

## Wildcard Certificates

Wildcard certificates (`*.example.com`) require DNS-01 challenge validation.

### Why DNS-01 Challenge?

Let's Encrypt requires proof that you control the DNS for wildcard domains. This is done by:
1. Creating a TXT record at `_acme-challenge.example.com`
2. Let's Encrypt verifies the record
3. Certificate is issued

### Setup for Cloudflare

#### 1. Create API Token

1. Go to https://dash.cloudflare.com/profile/api-tokens
2. Click **"Create Token"**
3. Use template **"Edit zone DNS"** or create custom token with:
   - **Permissions**: `Zone - DNS - Edit`
   - **Zone Resources**: Include your specific zone or all zones
4. Copy the token (shown only once!)

#### 2. Configure hapctl

```yaml
config:
  mail: admin@example.com
  domain:
    - example.com
    - "*.example.com"
  dns-provider: cloudflare
  dns-credentials:
    api-token: "your-cloudflare-api-token-here"
```

#### 3. Alternative: Global API Key (not recommended)

```yaml
dns-credentials:
  email: "your-email@cloudflare.com"
  api-key: "your-global-api-key"
```

**Note**: API Token is more secure as it has limited permissions.

### Setup for AWS Route53

#### 1. Create IAM User with Route53 permissions

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "route53:ListHostedZones",
        "route53:GetChange"
      ],
      "Resource": "*"
    },
    {
      "Effect": "Allow",
      "Action": "route53:ChangeResourceRecordSets",
      "Resource": "arn:aws:route53:::hostedzone/*"
    }
  ]
}
```

#### 2. Configure hapctl

```yaml
config:
  mail: admin@example.com
  domain:
    - example.com
    - "*.example.com"
  dns-provider: route53
  dns-credentials:
    aws-access-key-id: "YOUR_ACCESS_KEY"
    aws-secret-access-key: "YOUR_SECRET_KEY"
```

### Setup for DigitalOcean

#### 1. Create API Token

1. Go to https://cloud.digitalocean.com/account/api/tokens
2. Generate new token with **read and write** permissions
3. Copy the token

#### 2. Configure hapctl

```yaml
config:
  mail: admin@example.com
  domain:
    - example.com
    - "*.example.com"
  dns-provider: digitalocean
  dns-credentials:
    api-token: "your-digitalocean-token"
```

## Certificate Renewal

### Automatic Renewal

hapctl automatically checks for certificate renewal based on `renewal-check` interval (default: 12h = twice daily).

Certificates are renewed when:
- Less than **30 days** remaining until expiration
- Certificate is invalid or missing

### Manual Renewal

Restart the hapctl agent to trigger immediate check:

```bash
sudo systemctl restart hapctl-agent
```

### Renewal Logs

Check renewal activity:

```bash
# View hapctl logs
sudo journalctl -u hapctl-agent -f

# Check certbot logs
sudo tail -f /var/log/letsencrypt/letsencrypt.log
```

## HAProxy Integration

### PEM File Generation

hapctl automatically creates HAProxy-compatible PEM files by combining:
- Certificate fullchain (`fullchain.pem`)
- Private key (`privkey.pem`)

**Output**: `/etc/haproxy/certs/example.com.pem`

### HAProxy Configuration

Configure HAProxy to use the certificates:

```haproxy
frontend https_frontend
    bind *:443 ssl crt /etc/haproxy/certs/

    # Redirect HTTP to HTTPS
    http-request redirect scheme https unless { ssl_fc }
    
    # Your backend configuration
    default_backend web_servers
```

The `crt /etc/haproxy/certs/` directive loads all `.pem` files from the directory.

### Certificate Reload

After certificate renewal, reload HAProxy:

```bash
sudo systemctl reload haproxy
```

**Note**: Future versions of hapctl will automatically reload HAProxy after renewal.

## Troubleshooting

### Certificate Request Failed

**Error**: `certbot failed: timeout`

**Solution**: Check DNS propagation
```bash
# Verify DNS record
dig _acme-challenge.example.com TXT

# Wait for propagation (can take up to 48 hours)
```

### Invalid API Token

**Error**: `Failed to authenticate with DNS provider`

**Solution**:
1. Verify token has correct permissions
2. Check token hasn't expired
3. Ensure token is for the correct zone

### Rate Limits

Let's Encrypt has rate limits:
- 50 certificates per domain per week
- 5 duplicate certificates per week

**Solution**: Use staging mode for testing
```yaml
staging: true
```

### Email Not Configured

**Warning**: `SSL config ignored: email is required but not set`

**Solution**: Configure email in either:
- SSL config file: `mail: admin@example.com`
- Main config: `ssl.email: admin@example.com`

### Certbot Not Found

**Error**: `certbot not installed at /usr/bin/certbot`

**Solution**:
```bash
sudo apt-get update
sudo apt-get install certbot python3-certbot-dns-cloudflare
```

### Permission Denied

**Error**: `failed to create directory /etc/haproxy/certs`

**Solution**:
```bash
sudo mkdir -p /etc/haproxy/certs
sudo chown -R root:root /etc/haproxy/certs
sudo chmod 755 /etc/haproxy/certs
```

## Testing

### Test with Staging Server

Always test with Let's Encrypt staging server first:

```yaml
config:
  mail: admin@example.com
  domain:
    - example.com
  staging: true  # Use staging server
```

### Verify Certificate

```bash
# Check certificate details
openssl x509 -in /etc/haproxy/certs/example.com.pem -text -noout

# Check expiry date
openssl x509 -in /etc/haproxy/certs/example.com.pem -noout -dates

# Test HTTPS connection
curl -vI https://example.com
```

## Best Practices

1. **Use API Tokens** instead of Global API Keys (more secure)
2. **Test with staging** before production
3. **Monitor renewal logs** regularly
4. **Set up email notifications** for Let's Encrypt
5. **Use separate configs** for different domain groups
6. **Keep credentials secure** (use proper file permissions)
7. **Document your DNS provider** setup for team members

## Examples

### Simple Single Domain

```yaml
config:
  mail: admin@example.com
  domain:
    - example.com
```

### Wildcard with Multiple Domains

```yaml
config:
  mail: admin@example.com
  domain:
    - example.com
    - "*.example.com"
    - www.example.com
  dns-provider: cloudflare
  dns-credentials:
    api-token: "your-token"
```

### Multiple Domain Groups

**File: `/etc/hapctl/ssl/production.yaml`**
```yaml
config:
  mail: admin@company.com
  domain:
    - company.com
    - "*.company.com"
  dns-provider: cloudflare
  dns-credentials:
    api-token: "production-token"
```

**File: `/etc/hapctl/ssl/staging.yaml`**
```yaml
config:
  mail: admin@company.com
  domain:
    - staging.company.com
  staging: true
```

## References

- [Let's Encrypt Documentation](https://letsencrypt.org/docs/)
- [Certbot Documentation](https://certbot.eff.org/docs/)
- [Cloudflare API Tokens](https://developers.cloudflare.com/api/tokens/)
- [HAProxy SSL Configuration](https://www.haproxy.com/documentation/hapee/latest/security/tls/)
