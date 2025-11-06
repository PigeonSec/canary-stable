# Canary - Phishing Detection via Certificate Transparency

Monitor Certificate Transparency logs to detect phishing domains as they're registered.

Uses Aho-Corasick algorithm instead of regex for efficient multi-keyword matching - searches for thousands of keywords simultaneously in O(n+m) time vs O(n*k) with regex, making it ideal for high-volume CT log monitoring.

## Quick Start

### 1. Build

```bash
go build -o canary ./cmd/canary
```

### 2. Configure Keywords

Edit `data/keywords.txt` with brands/terms to monitor:

```
paypal
stripe
login
secure
banking
```

### 3. Run

```bash
./canary
```

Service runs on port 8080 (override with `PORT` env var).

## Certspotter Integration

### Option A: Web Service (Recommended)

1. Sign up at https://sslmate.com/certspotter/
2. Add webhook: `https://your-server.com/hook`
3. Done - certificates will be forwarded automatically

### Option B: Self-Hosted CLI

Install certspotter:
```bash
# Ubuntu/Debian
sudo apt-get install certspotter

# macOS
brew install certspotter
```

Run with webhook:
```bash
export CANARY_ENDPOINT='https://your-server.com/hook'
certspotter -watchlist <(echo .) -script scripts/certspotter-webhook.sh
```

## API Endpoints

### POST /hook
Accept Certspotter webhooks (certificate data in JSON).

**Example:**
```bash
curl -X POST http://localhost:8080/hook \
  -H "Content-Type: application/json" \
  -d @scripts/certspotter-example.json
```

### GET /matches
Get recent matches from memory (last 500).

```bash
curl http://localhost:8080/matches
```

### GET /matches/recent?minutes=5
Query database for matches in last N minutes.

```bash
curl "http://localhost:8080/matches/recent?minutes=60"
```

### POST /keywords
Add new keywords dynamically.

```bash
curl -X POST http://localhost:8080/keywords \
  -H "Content-Type: application/json" \
  -d '{"keywords": ["amazon", "microsoft"]}'
```

### POST /keywords/reload
Reload keywords from `data/keywords.txt` without restarting.

```bash
curl -X POST http://localhost:8080/keywords/reload
```

### GET /metrics
System statistics.

```bash
curl http://localhost:8080/metrics
```

### GET /health
Health check (returns 200 if healthy, 503 if unhealthy).

```bash
curl http://localhost:8080/health
```

## Production Deployment

### Deploy as Systemd Service

```bash
# Build
go build -o canary ./cmd/canary

# Create user
sudo useradd -r -s /bin/false canary

# Install
sudo mkdir -p /opt/canary/data
sudo cp canary /opt/canary/
sudo cp -r data /opt/canary/
sudo chown -R canary:canary /opt/canary

# Setup service
sudo cp deployments/canary.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable canary
sudo systemctl start canary
sudo systemctl status canary
```

View logs:
```bash
journalctl -u canary -f
```

## How It Works

1. Certspotter monitors CT logs → discovers new certificates
2. Sends webhook to `/hook` with domain names
3. Canary matches domains against keywords (Aho-Corasick algorithm)
4. Matches stored in SQLite (partitioned by keyword first letter)
5. Query via API or view logs for suspicious certificates

## Project Structure

```
canary/
├── cmd/canary/          # Main application entry point
├── internal/
│   ├── config/          # Global configuration
│   ├── models/          # Data structures
│   ├── database/        # SQLite operations
│   ├── matcher/         # Keyword matching (Aho-Corasick)
│   └── handlers/        # HTTP handlers
├── data/                # Runtime data (keywords, database)
├── scripts/             # Helper scripts
├── deployments/         # Systemd service files
└── docs/                # Documentation
```

## License

For authorized security research and defensive purposes only.
