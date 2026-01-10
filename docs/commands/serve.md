# Serve Command

## Overview

The `serve` command runs Octovy as an HTTP server to scan repositories via GitHub App webhooks. It automatically scans repositories on `push` and `pull_request` events.

**Requirements:**
- GitHub App configured ([setup guide](../setup/github-app.md))
- BigQuery configured ([setup guide](../setup/bigquery.md))
- Trivy installed on the server
- Publicly accessible HTTPS server

## Use Cases

- **Organization-wide scanning**: Automatically scan all repositories on push
- **PR vulnerability checks**: Scan pull requests before merge
- **Continuous monitoring**: Maintain vulnerability status across repositories
- **Webhook automation**: Integrate with existing GitHub workflows

## Basic Usage

### Start Server

```bash
octovy serve --addr :8080
```

### With Environment Variables

```bash
export OCTOVY_ADDR=:8080
export OCTOVY_GITHUB_APP_ID=123456
export OCTOVY_GITHUB_APP_PRIVATE_KEY="$(cat /path/to/private-key.pem)"
export OCTOVY_GITHUB_APP_SECRET=your-webhook-secret
export OCTOVY_BIGQUERY_PROJECT_ID=my-project

octovy serve
```

### With Docker

The official Octovy image does not include Trivy. Create a custom image that includes Trivy using a multi-stage build:

**Dockerfile:**

```dockerfile
FROM aquasecurity/trivy:latest AS trivy
FROM ghcr.io/secmon-lab/octovy:latest

COPY --from=trivy /usr/local/bin/trivy /usr/local/bin/trivy
```

**Build and run:**

```bash
# Build custom image
docker build -t octovy-with-trivy .

# Run
docker run -p 8080:8080 \
  -e OCTOVY_ADDR=:8080 \
  -e OCTOVY_GITHUB_APP_ID=123456 \
  -e OCTOVY_GITHUB_APP_PRIVATE_KEY="$(cat /path/to/private-key.pem)" \
  -e OCTOVY_GITHUB_APP_SECRET=your-webhook-secret \
  -e OCTOVY_BIGQUERY_PROJECT_ID=my-project \
  octovy-with-trivy
```

## Command Flags Reference

| Flag | Env Variable | Required | Default | Description |
|------|--------------|----------|---------|-------------|
| `--addr` | `OCTOVY_ADDR` | ✓ | N/A | Server bind address (e.g., `:8080`, `127.0.0.1:8080`) |
| `--github-app-id` | `OCTOVY_GITHUB_APP_ID` | ✓ | N/A | GitHub App ID |
| `--github-app-private-key` | `OCTOVY_GITHUB_APP_PRIVATE_KEY` | ✓ | N/A | Private key PEM content |
| `--github-app-secret` | `OCTOVY_GITHUB_APP_SECRET` | ✓ | N/A | Webhook secret |
| `--bigquery-project-id` | `OCTOVY_BIGQUERY_PROJECT_ID` | ✓ | N/A | GCP Project ID |
| `--bigquery-dataset-id` | `OCTOVY_BIGQUERY_DATASET_ID` | ✗ | `octovy` | BigQuery dataset name |
| `--bigquery-table-id` | `OCTOVY_BIGQUERY_TABLE_ID` | ✗ | `scans` | BigQuery table name |
| `--firestore-project-id` | `OCTOVY_FIRESTORE_PROJECT_ID` | ✗ | N/A | Firestore project ID (enables Firestore) |
| `--firestore-database-id` | `OCTOVY_FIRESTORE_DATABASE_ID` | ✗ | `(default)` | Firestore database ID |
| `--trivy-path` | `OCTOVY_TRIVY_PATH` | ✗ | `trivy` | Path to Trivy binary |
| `--log-format` | `OCTOVY_LOG_FORMAT` | ✗ | `text` | Log format: `text` or `json` |
| `--shutdown-timeout` | `OCTOVY_SHUTDOWN_TIMEOUT` | ✗ | `30s` | Graceful shutdown timeout |

## Examples

### Local Development

```bash
# Set up environment
export OCTOVY_GITHUB_APP_ID=123456
export OCTOVY_GITHUB_APP_PRIVATE_KEY=$(cat private-key.pem)
export OCTOVY_GITHUB_APP_SECRET=your-webhook-secret
export OCTOVY_BIGQUERY_PROJECT_ID=my-project

# Start server
octovy serve --addr :8080

# Test health endpoint
curl http://localhost:8080/health
```

### Production Deployment (Docker Compose)

Create the same `Dockerfile` as above, then use Docker Compose:

```yaml
services:
  octovy:
    build: .
    ports:
      - "8080:8080"
    environment:
      OCTOVY_ADDR: :8080
      OCTOVY_GITHUB_APP_ID: ${GITHUB_APP_ID}
      OCTOVY_GITHUB_APP_PRIVATE_KEY: ${GITHUB_APP_PRIVATE_KEY}
      OCTOVY_GITHUB_APP_SECRET: ${GITHUB_APP_SECRET}
      OCTOVY_BIGQUERY_PROJECT_ID: ${GCP_PROJECT_ID}
      OCTOVY_LOG_FORMAT: json
    restart: unless-stopped
```

## Endpoints

### POST /webhook/github/app

GitHub webhook endpoint. Receives `push` and `pull_request` events.

### GET /health

Health check endpoint.

## How It Works

1. **Receive webhook**: GitHub sends `push` or `pull_request` event
2. **Verify signature**: Validates webhook authenticity using secret
3. **Download repository**: Downloads repository code as archive from GitHub API
4. **Extract to temp directory**: Extracts archive to temporary location
5. **Run Trivy**: Scans code with Trivy
6. **Insert results**: Stores findings in BigQuery
7. **Store metadata**: Optionally stores in Firestore
8. **Cleanup**: Deletes temporary files

## Environment Variables Reference

### Required

| Variable | Default | Description |
|----------|---------|-------------|
| `OCTOVY_ADDR` | N/A | Server bind address |
| `OCTOVY_GITHUB_APP_ID` | N/A | GitHub App ID |
| `OCTOVY_GITHUB_APP_PRIVATE_KEY` | N/A | Private key PEM content |
| `OCTOVY_GITHUB_APP_SECRET` | N/A | Webhook secret |
| `OCTOVY_BIGQUERY_PROJECT_ID` | N/A | GCP Project ID |

### Optional

| Variable | Default | Description |
|----------|---------|-------------|
| `OCTOVY_BIGQUERY_DATASET_ID` | `octovy` | BigQuery dataset |
| `OCTOVY_BIGQUERY_TABLE_ID` | `scans` | BigQuery table |
| `OCTOVY_FIRESTORE_PROJECT_ID` | N/A | Firestore project (enables Firestore) |
| `OCTOVY_FIRESTORE_DATABASE_ID` | `(default)` | Firestore database |
| `OCTOVY_TRIVY_PATH` | `trivy` | Trivy binary path |
| `OCTOVY_LOG_FORMAT` | `text` | Log format (`text` or `json`) |
| `OCTOVY_SHUTDOWN_TIMEOUT` | `30s` | Graceful shutdown timeout |

## Graceful Shutdown

The server handles graceful shutdown:

```bash
# Send SIGTERM
kill -TERM <pid>

# Server waits for in-flight scans (default: 30 seconds)
# Then exits cleanly
```

Configure timeout:

```bash
export OCTOVY_SHUTDOWN_TIMEOUT=60s
octovy serve --addr :8080
```

## Monitoring and Logging

### Health Checks

```bash
# Liveness check
curl http://localhost:8080/health

# From within container
curl http://localhost:8080/health && echo "OK" || echo "FAILED"
```

### JSON Logging

For structured logging:

```bash
export OCTOVY_LOG_FORMAT=json
octovy serve --addr :8080 | jq .
```

### Log Analysis

```bash
# View errors only
octovy serve --addr :8080 --log-format json | jq 'select(.level=="error")'

# View scan events
octovy serve --addr :8080 --log-format json | jq 'select(.type=="scan")'
```

## Troubleshooting

### Port already in use

```bash
# Find process using port 8080
lsof -i :8080

# Use different port
octovy serve --addr :8081
```

### Webhook not received

1. Verify webhook URL in GitHub App settings
2. Check firewall allows HTTPS (port 443)
3. Verify GitHub App is installed on repository
4. Check webhook delivery logs in GitHub

### Signature verification failed

- Verify webhook secret in both GitHub and Octovy
- Check for leading/trailing whitespace
- Regenerate secret if unsure

### Trivy not found

```bash
# Install Trivy
curl -sfL https://raw.githubusercontent.com/aquasecurity/trivy/main/contrib/install.sh | sh -s -- -b /usr/local/bin

# Or specify path
export OCTOVY_TRIVY_PATH=/path/to/trivy
```

### Memory issues during scans

Large repositories may use significant memory. Increase limits:

```bash
docker run -m 2g ...
```

## Next Steps

- [Configure GitHub App](../setup/github-app.md)
- [Configure BigQuery](../setup/bigquery.md)
- [Configure Firestore](../setup/firestore.md) (optional)
- [Monitor webhook deliveries](https://github.com/settings/apps)
