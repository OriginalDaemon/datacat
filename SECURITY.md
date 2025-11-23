# Security Features

This document describes the security features available in datacat.

## Table of Contents

- [API Key Authentication](#api-key-authentication)
- [TLS/HTTPS Support](#tlshttps-support)
- [Data Compression](#data-compression)
- [Configuration Examples](#configuration-examples)

---

## API Key Authentication

Protect your datacat server from unauthorized access using API key authentication.

### How It Works

1. **Server** - Configure an API key and enable requirement
2. **Daemon** - Configured with the same API key
3. **All requests** - Daemon includes API key in `Authorization` header
4. **Server validates** - Rejects requests without valid API key

### Server Configuration

```json
{
  "server_port": "9090",
  "api_key": "your-secret-key-here",
  "require_api_key": true
}
```

**Important:**
- Use a strong, randomly generated key
- Keep the API key secret
- Rotate keys periodically
- Use HTTPS to prevent key interception

### Daemon Configuration

```json
{
  "server_url": "http://localhost:9090",
  "api_key": "your-secret-key-here"
}
```

### Generating a Secure API Key

```bash
# Linux/Mac
openssl rand -hex 32

# Or use Python
python3 -c "import secrets; print(secrets.token_hex(32))"

# Windows PowerShell
[Convert]::ToBase64String((1..32 | ForEach-Object { Get-Random -Maximum 256 }))
```

### Backward Compatibility

- If `require_api_key` is `false` (default), server accepts all requests
- This allows gradual migration to authenticated setup
- Health check endpoint (`/health`) never requires authentication

---

## TLS/HTTPS Support

Encrypt data in transit between daemon and server using TLS.

### Server HTTPS Configuration

#### Option 1: Let's Encrypt (Production)

```json
{
  "server_port": "443",
  "tls_cert_file": "/etc/letsencrypt/live/yourdomain.com/fullchain.pem",
  "tls_key_file": "/etc/letsencrypt/live/yourdomain.com/privkey.pem",
  "api_key": "your-secret-key",
  "require_api_key": true
}
```

#### Option 2: Self-Signed Certificate (Development/Internal)

Generate a self-signed certificate:

```bash
# Generate self-signed cert (valid for 365 days)
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365 -nodes \
  -subj "/CN=localhost"
```

Configuration:

```json
{
  "server_port": "9090",
  "tls_cert_file": "./cert.pem",
  "tls_key_file": "./key.pem"
}
```

### Daemon HTTPS Configuration

#### For Valid Certificates

```json
{
  "server_url": "https://your-datacat-server.com",
  "tls_verify": true,
  "api_key": "your-secret-key"
}
```

#### For Self-Signed Certificates

```json
{
  "server_url": "https://localhost:9090",
  "tls_verify": true,
  "tls_insecure_skip_verify": true,
  "api_key": "your-secret-key"
}
```

**Warning:** Only use `tls_insecure_skip_verify: true` for:
- Development/testing
- Internal networks with self-signed certs
- Never in production with external traffic

---

## Data Compression

Reduce bandwidth usage by up to 70% with gzip compression.

### Daemon Configuration

```json
{
  "enable_compression": true
}
```

**Benefits:**
- 60-80% bandwidth reduction for typical workloads
- Faster data transfer over slow networks
- Reduced server bandwidth costs
- No storage overhead (decompressed on server)

**Performance:**
- Compression overhead: ~1-2ms per batch (negligible)
- Network time saved: Often 100ms+ per batch
- Net result: Faster overall performance

### How It Works

1. Daemon marshals data to JSON
2. If `enable_compression: true`, compresses with gzip
3. Adds `Content-Encoding: gzip` header
4. Server automatically decompresses
5. Server processes uncompressed data normally

**Default:** Compression is enabled by default in new configurations.

---

## Configuration Examples

### Development Setup (HTTP, No Auth)

**Server:**
```json
{
  "data_path": "./datacat_data",
  "server_port": "9090",
  "require_api_key": false
}
```

**Daemon:**
```json
{
  "server_url": "http://localhost:9090",
  "enable_compression": true
}
```

### Production Setup (HTTPS + API Key)

**Server:**
```json
{
  "data_path": "/var/datacat/data",
  "server_port": "443",
  "tls_cert_file": "/etc/letsencrypt/live/datacat.example.com/fullchain.pem",
  "tls_key_file": "/etc/letsencrypt/live/datacat.example.com/privkey.pem",
  "api_key": "sk_a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6",
  "require_api_key": true,
  "retention_days": 90
}
```

**Daemon:**
```json
{
  "server_url": "https://datacat.example.com",
  "api_key": "sk_a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6",
  "enable_compression": true,
  "tls_verify": true,
  "batch_interval_seconds": 5
}
```

### Internal Network (Self-Signed Cert + API Key)

**Server:**
```json
{
  "data_path": "./datacat_data",
  "server_port": "9090",
  "tls_cert_file": "./certs/server.crt",
  "tls_key_file": "./certs/server.key",
  "api_key": "internal-network-key-12345",
  "require_api_key": true
}
```

**Daemon:**
```json
{
  "server_url": "https://datacat-internal.local:9090",
  "api_key": "internal-network-key-12345",
  "enable_compression": true,
  "tls_verify": true,
  "tls_insecure_skip_verify": true
}
```

---

## Security Best Practices

### 1. Always Use HTTPS in Production
- Prevents API key interception
- Protects data privacy
- Essential for compliance (GDPR, HIPAA, etc.)

### 2. Generate Strong API Keys
- Use at least 32 bytes of random data
- Never use predictable keys
- Store keys securely (environment variables, key vaults)

### 3. Rotate Keys Regularly
- Change API keys every 90 days (or per your policy)
- Update both server and all daemons
- Keep old key active during transition period

### 4. Network Security
- Run server behind firewall
- Use VPN for remote daemon connections
- Restrict server port access to known IPs

### 5. File Permissions
- Protect config files: `chmod 600 config.json`
- Protect TLS keys: `chmod 600 key.pem`
- Run services as non-root user

### 6. Monitoring
- Log failed authentication attempts
- Alert on suspicious patterns
- Monitor for unusual data volumes

---

## Troubleshooting

### "Unauthorized" Errors

**Problem:** Daemon getting 401 responses

**Solutions:**
1. Check API keys match exactly
2. Verify `require_api_key: true` on server
3. Check for whitespace in key strings
4. Ensure daemon config has `api_key` field

### TLS Certificate Errors

**Problem:** "x509: certificate signed by unknown authority"

**Solutions:**
1. For self-signed certs: Set `tls_insecure_skip_verify: true`
2. For Let's Encrypt: Ensure system has updated CA certificates
3. Verify certificate hasn't expired
4. Check server URL matches certificate CN/SAN

### Compression Issues

**Problem:** "Failed to decompress request"

**Solutions:**
1. Ensure both daemon and server are updated
2. Check daemon logs for compression errors
3. Temporarily disable: `enable_compression: false`
4. Verify network isn't mangling binary data

---

## Migration Guide

### Adding Authentication to Existing Setup

1. **Generate API key**
   ```bash
   openssl rand -hex 32 > api_key.txt
   ```

2. **Update server config** (don't enable requirement yet)
   ```json
   {
     "api_key": "<key-from-file>",
     "require_api_key": false
   }
   ```

3. **Restart server**
   ```bash
   systemctl restart datacat-server
   ```

4. **Update all daemon configs**
   ```json
   {
     "api_key": "<key-from-file>"
   }
   ```

5. **Restart all daemons** (or wait for apps to restart)

6. **Enable requirement** (after all daemons updated)
   ```json
   {
     "require_api_key": true
   }
   ```

7. **Restart server**

### Adding HTTPS to Existing Setup

1. **Obtain TLS certificate** (Let's Encrypt or self-signed)

2. **Update server config**
   ```json
   {
     "tls_cert_file": "./cert.pem",
     "tls_key_file": "./key.pem"
   }
   ```

3. **Restart server**

4. **Update daemon configs** (change http to https)
   ```json
   {
     "server_url": "https://your-server.com"
   }
   ```

5. **Restart daemons**

