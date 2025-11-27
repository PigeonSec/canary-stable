# Canary

**Real-time Phishing & Brand Impersonation Detection**

Canary monitors Certificate Transparency (CT) logs in real-time to detect phishing domains and brand impersonation attempts the moment they are issued. It uses high-performance pattern matching to evaluate millions of certificates against your custom rules.

![Canary Dashboard](web/canary.webp)

## Features

- **Real-time Monitoring**: Scans 40+ CT logs instantly via Certspotter.
- **High Performance**: Evaluates thousands of certificates per second using Aho-Corasick.
- **Flexible Rules**: Boolean logic (AND, OR, NOT), priority levels, and regex-like matching.
- **Web Dashboard**: Live view of matches, rule management, and system metrics.
- **API First**: Full REST API for integration with SOAR, Slack, or custom tooling.
- **Secure**: Built-in authentication, CSRF protection, and secure session management.
- **Docker Ready**: One-line deployment with Docker Compose.

## Quick Start

### 1. Deploy with Docker

```bash
# Clone and start
git clone https://github.com/yourusername/canary.git
cd canary
cd deployments/docker
docker-compose up -d
```

Access the dashboard at **http://localhost:8080**.

### 2. Default Credentials

- **Username**: `admin`
- **Password**: (Check container logs on first run)

```bash
docker-compose logs canary | grep "INITIAL USER"
```

### 3. Configure Rules

Edit `data/rules.yaml` to add your monitoring rules:

```yaml
- name: paypal-phishing
  keywords: paypal AND (login OR secure OR update)
  priority: critical
  comment: "Detects PayPal phishing attempts"

- name: brand-monitor
  keywords: mycompanyname
  priority: medium
```

Reload rules via the dashboard or API:
```bash
curl -X POST http://localhost:8080/rules/reload
```

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP listening port |
| `DEBUG` | `false` | Enable debug logging |
| `PUBLIC_DASHBOARD` | `false` | Allow read-only access without login |
| `DOMAIN` | - | Set for HTTPS/Cookie security (e.g., `canary.example.com`) |
| `PARTITION_RETENTION_DAYS` | `30` | Days to keep match history |

### Docker Compose

For production, set the `DOMAIN` variable to enable secure cookies and strict CORS.

```yaml
services:
  canary:
    environment:
      - DOMAIN=canary.yourdomain.com
      - PUBLIC_DASHBOARD=false
```

## API Documentation

Canary provides a comprehensive REST API.

- **Interactive Docs**: Visit `/docs` (e.g., `http://localhost:8080/docs`) for Swagger UI.
- **Authentication**: Most endpoints require a session cookie.
- **Endpoints**:
  - `GET /matches/recent`: Retrieve latest matches.
  - `GET /rules`: List active rules.
  - `POST /rules/create`: Add new rules dynamically.

## License

**Canary** is for authorized security research and defensive purposes only.

### Third-Party Acknowledgments

Canary is built on the shoulders of giants. We gratefully acknowledge:

- **Certspotter** (MPL-2.0) by SSLMate for CT log monitoring.
- **go-sqlite3** (MIT) for database storage.
- **ahocorasick** (MIT) for efficient string matching.
- **gopsutil** (BSD-3) for system metrics.
- **minify** (MIT) for frontend optimization.

Full license texts are available in the respective repositories.
