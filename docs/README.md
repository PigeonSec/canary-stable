# Canary - Certificate Transparency Monitor

A high-performance phishing detection tool that monitors Certificate Transparency logs via Certspotter webhooks to identify suspicious domain registrations.

## Overview

Canary uses the Aho-Corasick multi-pattern matching algorithm to efficiently scan certificate domains for phishing indicators. When a new SSL/TLS certificate is issued with domains matching your configured keywords (e.g., "paypal", "login", "bank"), Canary captures and stores the match for investigation.

## Features

- **Fast Pattern Matching**: Uses Aho-Corasick algorithm for efficient multi-keyword matching
- **Partitioned Storage**: SQLite database with 27 partitioned tables (a-z + other) for horizontal scaling
- **Batch Processing**: Configurable batch writes with worker pools for high throughput
- **In-Memory Cache**: 500 most recent matches for quick access
- **Graceful Shutdown**: Proper signal handling and connection draining
- **Health Monitoring**: Built-in metrics and health check endpoints
- **Hot Reload**: Update keywords without restarting the service

## Architecture

```
config.go      - Configuration and global state
models.go      - Data structures
matcher.go     - Aho-Corasick keyword matching
database.go    - SQLite operations and batch processing
handlers.go    - HTTP request handlers
main.go        - Server initialization and routing
```

## Installation

### Build from source

```bash
go build -o canary
```

### Run locally

```bash
./canary
```

By default, the service runs on port 8080. Override with the `PORT` environment variable:

```bash
PORT=3000 ./canary
```

### Deploy as systemd service

1. Create a dedicated user:
```bash
sudo useradd -r -s /bin/false canary
```

2. Create directory and copy files:
```bash
sudo mkdir -p /opt/canary
sudo cp canary /opt/canary/
sudo cp keywords.txt /opt/canary/
sudo chown -R canary:canary /opt/canary
```

3. Install service:
```bash
sudo cp canary.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable canary
sudo systemctl start canary
```

4. Check status:
```bash
sudo systemctl status canary
sudo journalctl -u canary -f
```

## API Endpoints

### POST /hook
Accept Certspotter webhook events. This is the main entry point for certificate data.

**Example webhook configuration:**
```json
{
  "url": "https://your-server.com/hook",
  "method": "POST"
}
```

### GET /matches
Retrieve recent matches from in-memory cache (last 500).

**Response:**
```json
{
  "count": 2,
  "matches": [
    {
      "cert_id": "12345",
      "domains": ["login-paypal-secure.com"],
      "keyword": "paypal",
      "timestamp": "2025-11-06T22:00:00Z",
      "tbs_sha256": "abc123...",
      "cert_sha256": "def456..."
    }
  ]
}
```

### GET /matches/recent?minutes=5
Retrieve matches from database for the last N minutes.

**Parameters:**
- `minutes` - Number of minutes to look back (default: 5)

### POST /matches/clear
Clear the in-memory matches cache.

### POST /keywords
Add new keywords via API.

**Request body:**
```json
{
  "keywords": ["paypal", "stripe", "amazon"]
}
```

### POST /keywords/reload
Reload keywords from the `keywords.txt` file without restarting the service.

### GET /metrics
System metrics and statistics.

**Response:**
```json
{
  "queue_len": 45,
  "total_matches": 1234,
  "total_certs": 567,
  "keyword_count": 42,
  "uptime_seconds": 86400,
  "recent_matches": 500
}
```

### GET /health
Health check endpoint for monitoring.

**Response (healthy):**
```json
{
  "status": "healthy",
  "keywords": 42,
  "uptime": 86400
}
```

**Response (unhealthy):**
```json
{
  "status": "unhealthy",
  "error": "database unreachable"
}
```

## Configuration

### Keywords File

Edit `keywords.txt` to add brand names and phishing terms to monitor:

```
# Financial Services
paypal
stripe
bank

# Social Media
facebook
google
```

- One keyword per line
- Case-insensitive
- Lines starting with `#` are comments
- Blank lines are ignored

Reload keywords without restarting: `curl -X POST http://localhost:8080/keywords/reload`

### Environment Variables

- `PORT` - HTTP server port (default: 8080)

### Database

Canary uses SQLite with Write-Ahead Logging (WAL) mode for concurrent reads/writes. The database file `matches.db` is created automatically in the working directory.

**Connection pool:**
- Max open connections: 25
- Max idle connections: 5
- Busy timeout: 5000ms

**Batch processing:**
- Workers: 4
- Batch size: 200 matches
- Batch timeout: 200ms

## Performance

The Aho-Corasick automaton enables efficient matching of thousands of keywords simultaneously:

- **Time complexity**: O(n + m) where n = text length, m = number of matches
- **Memory**: Linear in the number of keywords
- **Throughput**: Tested with 10,000+ certificates/second

## Security Considerations

1. **Input Validation**: All webhook payloads are validated before processing
2. **SQL Injection**: Uses prepared statements for all database operations
3. **DoS Protection**: Channel buffering and batch processing prevent memory exhaustion
4. **Least Privilege**: systemd service runs as unprivileged user

## Monitoring

Monitor the service with:

```bash
# System metrics
curl http://localhost:8080/metrics

# Health check
curl http://localhost:8080/health

# View logs
journalctl -u canary -f
```

Set up alerting based on the `/health` endpoint (returns 503 when unhealthy).

## Certspotter Integration

Canary can receive certificate data from Certspotter in two ways: via the web service or the CLI tool.

### Option 1: Certspotter Web Service (Recommended)

The easiest way is to use the certspotter.com hosted service:

1. **Sign up** at [https://sslmate.com/certspotter/](https://sslmate.com/certspotter/)
2. **Navigate** to Settings → Webhooks
3. **Add Webhook**:
   - URL: `https://your-canary-server.com/hook`
   - Method: POST
   - Content-Type: application/json
4. **Configure Monitoring**:
   - Option A: Monitor all certificates (high volume)
   - Option B: Monitor specific domains or keywords
5. **Test** the webhook to verify connectivity

### Option 2: Certspotter CLI Tool

For self-hosted monitoring, use the certspotter CLI:

#### Install Certspotter

```bash
# Ubuntu/Debian
sudo apt-get install certspotter

# macOS
brew install certspotter

# From source
git clone https://github.com/SSLMate/certspotter.git
cd certspotter
go build -o certspotter ./cmd/certspotter
```

#### Configure Webhook Script

1. Copy the webhook script:
```bash
sudo cp certspotter-script.sh /usr/local/bin/
sudo chmod +x /usr/local/bin/certspotter-script.sh
```

2. Set your Canary endpoint:
```bash
export CANARY_ENDPOINT='https://your-canary-server.com/hook'
```

3. Create a watchlist (optional):
```bash
# Monitor all domains (high volume!)
echo '.' > /etc/certspotter/watchlist

# Or monitor specific domains
cat > /etc/certspotter/watchlist << EOF
example.com
*.example.com
EOF
```

4. Run certspotter:
```bash
certspotter -watchlist /etc/certspotter/watchlist \
    -script /usr/local/bin/certspotter-script.sh
```

#### Run as Systemd Service

Create `/etc/systemd/system/certspotter.service`:

```ini
[Unit]
Description=Certificate Transparency Log Monitor
After=network.target

[Service]
Type=simple
User=certspotter
Group=certspotter
Environment="CANARY_ENDPOINT=https://your-canary-server.com/hook"
ExecStart=/usr/bin/certspotter \
    -watchlist /etc/certspotter/watchlist \
    -state_dir /var/lib/certspotter \
    -script /usr/local/bin/certspotter-script.sh
Restart=always
RestartSec=60

[Install]
WantedBy=multi-user.target
```

Enable and start:
```bash
sudo systemctl daemon-reload
sudo systemctl enable certspotter
sudo systemctl start certspotter
sudo journalctl -u certspotter -f
```

### Webhook Payload Format

Certspotter sends POST requests with JSON payloads like this:

```json
{
  "id": "1234567890",
  "issuance": {
    "tbs_sha256": "a1b2c3d4...",
    "cert_sha256": "b2c3d4e5...",
    "dns_names": [
      "login-paypal-verify.com",
      "www.login-paypal-verify.com",
      "secure-paypal-update.net"
    ]
  },
  "endpoints": [
    {
      "dns_name": "login-paypal-verify.com"
    }
  ]
}
```

See `certspotter-webhook-example.json` for a complete example.

### How It Works

Once configured, the integration flow is:

1. **Certspotter** monitors Certificate Transparency logs
2. **New certificate discovered** → Certspotter extracts domain names
3. **Webhook triggered** → POST request sent to Canary `/hook` endpoint
4. **Canary processes**:
   - Extracts all domains from certificate
   - Matches against keyword list (e.g., "paypal", "login")
   - If match found → stores in database + logs alert
   - Returns 200 OK response
5. **Alert/investigate** suspicious certificates via `/matches` API

### Testing the Integration

Test your webhook manually:

```bash
curl -X POST https://your-canary-server.com/hook \
  -H "Content-Type: application/json" \
  -d @certspotter-webhook-example.json
```

Expected response:
```json
{
  "status": "ok",
  "matches": 3
}
```

Check Canary logs:
```bash
journalctl -u canary -f
```

You should see:
```
Match found: cert_id=1234567890 keywords=[paypal] domains=[login-paypal-verify.com ...]
```

## License

This project is provided as-is for security research and defensive purposes.

## Contributing

This tool is designed for authorized security monitoring, CTF competitions, and phishing research. Do not use it for malicious purposes.
