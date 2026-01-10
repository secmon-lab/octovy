# Documentation

Welcome to Octovy documentation! This guide covers setup and usage of all Octovy commands.

## Getting Started

Start with [Quick Start](../README.md#quick-start) in the main README.

## Commands

Octovy provides three commands for different use cases:

### [scan](./commands/scan.md)

Scans repositories with Trivy and inserts results into BigQuery. Has two subcommands:

- **`scan local`**: Scans a local directory on your machine
- **`scan remote`**: Scans a GitHub repository remotely via GitHub App API

**Use when:**
- You want to scan code locally (`scan local`)
- Integrating with CI/CD pipelines (`scan local`)
- Scanning GitHub repositories without cloning (`scan remote`)
- Running organization-wide scans (`scan remote`)

**Quick examples:**
```bash
# Local scan
octovy scan local --bigquery-project-id my-project

# Remote scan (requires GitHub App)
octovy scan remote \
  --github-owner myorg \
  --github-repo myrepo \
  --github-app-id 12345 \
  --github-app-private-key "$(cat private-key.pem)" \
  --bigquery-project-id my-project
```

[Full documentation →](./commands/scan.md)

### [insert](./commands/insert.md)

Inserts existing Trivy scan result JSON files into BigQuery.

**Use when:**
- Integrating with existing Trivy workflows
- Separating scanning and insertion steps
- Importing historical scan results

**Quick example:**
```bash
trivy fs --format json --output scan-result.json .
octovy insert -f scan-result.json --bigquery-project-id my-project
```

[Full documentation →](./commands/insert.md)

### [serve](./commands/serve.md)

Runs as an HTTP server to scan repositories via GitHub App webhooks.

**Use when:**
- You want automatic scanning on every push
- Scanning pull requests before merge
- Organization-wide vulnerability monitoring

**Quick example:**
```bash
export OCTOVY_GITHUB_APP_ID=123456
export OCTOVY_GITHUB_APP_PRIVATE_KEY=/path/to/private-key.pem
octovy serve --addr :8080
```

[Full documentation →](./commands/serve.md)

## Setup Guides

### Required Setup

#### [BigQuery Setup](./setup/bigquery.md)

**Required for all commands**

BigQuery is the primary storage backend. All Trivy scan results are stored here.

**What you'll do:**
- Create a BigQuery dataset
- Configure authentication
- Grant permissions

[Full setup guide →](./setup/bigquery.md)

### Optional Setup

#### [GitHub App Setup](./setup/github-app.md)

**Required for `serve` and `scan remote` commands**

Configure a GitHub App to automatically scan repositories on push and pull request events, or to scan repositories remotely via CLI.

**What you'll do:**
- Create a GitHub App
- Generate private key
- Install on repositories

[Full setup guide →](./setup/github-app.md)

#### [Firestore Setup](./setup/firestore.md)

**Optional for all commands**

Firestore stores repository metadata for real-time querying and relationship tracking. Use only if you need real-time metadata access.

**What you'll do:**
- Enable Firestore API
- Create a database
- Grant permissions

[Full setup guide →](./setup/firestore.md)

## Quick Reference

### Command Comparison

| Feature | scan local | scan remote | insert | serve |
|---------|------------|-------------|--------|-------|
| **Purpose** | Local scanning | Remote GitHub scanning | Insert existing results | Webhook scanning |
| **BigQuery** | ✓ Required | ✓ Required | ✓ Required | ✓ Required |
| **GitHub App** | — | ✓ Required | — | ✓ Required |
| **Firestore** | ✓ Optional | ✓ Optional | ✓ Optional | ✓ Optional |
| **Setup time** | 5 min | 15 min | 5 min | 15 min |

### Environment Variables Quick Lookup

**Required for all commands:**
```bash
OCTOVY_BIGQUERY_PROJECT_ID=your-project-id
```

**For scan local command:**
```bash
OCTOVY_SCAN_DIR=/path/to/code
```

**For scan remote command:**
```bash
OCTOVY_GITHUB_OWNER=myorg
OCTOVY_GITHUB_REPO=myrepo           # optional; omit to scan all repos
OCTOVY_GITHUB_APP_ID=123456
OCTOVY_GITHUB_APP_PRIVATE_KEY=/path/to/private-key.pem
```

**For insert command:**
```bash
# Specify file via -f flag or JSON_FILE env var
```

**For serve command:**
```bash
OCTOVY_ADDR=:8080
OCTOVY_GITHUB_APP_ID=123456
OCTOVY_GITHUB_APP_PRIVATE_KEY=/path/to/private-key.pem
OCTOVY_GITHUB_APP_SECRET=your-secret
```

**Optional for all:**
```bash
OCTOVY_BIGQUERY_DATASET_ID=octovy      # default: octovy
OCTOVY_BIGQUERY_TABLE_ID=scans          # default: scans
OCTOVY_FIRESTORE_PROJECT_ID=...         # enables Firestore
OCTOVY_FIRESTORE_DATABASE_ID="(default)" # default database
OCTOVY_TRIVY_PATH=/path/to/trivy        # default: trivy
OCTOVY_LOG_FORMAT=text|json             # default: text
```

## Common Workflows

### Workflow 1: Manual Local Scanning

```bash
# Setup (one time)
gcloud auth application-default login
export OCTOVY_BIGQUERY_PROJECT_ID=my-project

# Scan (anytime)
cd /path/to/project
octovy scan local
```

[See scan command →](./commands/scan.md)

### Workflow 2: CI/CD Integration

```bash
# In CI/CD pipeline (GitHub Actions, GitLab CI, etc.)
octovy scan local \
  --bigquery-project-id $PROJECT_ID \
  --github-owner $OWNER \
  --github-repo $REPO \
  --github-commit-id $COMMIT_SHA
```

[See scan command →](./commands/scan.md)

### Workflow 3: Remote Repository Scanning

```bash
# Scan a specific GitHub repository without cloning
octovy scan remote \
  --github-owner myorg \
  --github-repo myrepo \
  --github-app-id $APP_ID \
  --github-app-private-key "$(cat private-key.pem)" \
  --bigquery-project-id $PROJECT_ID

# Scan all repositories for an organization
octovy scan remote \
  --github-owner myorg \
  --github-app-id $APP_ID \
  --github-app-private-key "$(cat private-key.pem)" \
  --bigquery-project-id $PROJECT_ID
```

[See scan command →](./commands/scan.md)

### Workflow 4: Separate Trivy from Insertion

```bash
# On scanning machine
trivy fs --format json --output results.json /code

# On insertion machine
octovy insert -f results.json --bigquery-project-id my-project
```

[See insert command →](./commands/insert.md)

### Workflow 5: Automatic Webhook Scanning

```bash
# Setup (one time)
# 1. Create GitHub App
# 2. Install on repositories
# 3. Configure server

# Run server
export OCTOVY_GITHUB_APP_ID=123456
export OCTOVY_GITHUB_APP_PRIVATE_KEY=/path/to/key.pem
export OCTOVY_BIGQUERY_PROJECT_ID=my-project
octovy serve --addr :8080

# Now pushing to GitHub automatically triggers scans
```

[See serve command →](./commands/serve.md)

## FAQ

### Q: Do I need to set up GitHub App?

**A:** It depends on your use case:
- **Not needed**: `scan local` (local directory scanning) and `insert` (inserting existing results)
- **Required**: `scan remote` (remote GitHub repository scanning) and `serve` (automatic webhook scanning)

### Q: Is Firestore required?

**A:** No. Firestore is optional. All scan results are always stored in BigQuery. Use Firestore only if you need real-time metadata tracking.

### Q: How long does setup take?

**A:**
- BigQuery setup: 5 minutes
- GitHub App setup: 10 minutes
- Firestore setup: 5 minutes
- Total: 15-20 minutes

### Q: Can I use Octovy without Google Cloud?

**A:** No. BigQuery and optionally Firestore are Google Cloud services. You need a GCP account and project.

### Q: What are the costs?

**A:** Depends on usage:
- **BigQuery**: Pay per query and storage (first 1 TB free per month)
- **Firestore**: Pay per read/write/delete operations (free tier available)
- **Trivy**: Free, open source

Typical costs are very low for small-scale usage.

## Troubleshooting

### Common Issues

- **Authentication failed**: [BigQuery troubleshooting](./setup/bigquery.md#troubleshooting)
- **Webhook not received**: [GitHub App troubleshooting](./setup/github-app.md#troubleshooting)
- **Trivy not found**: [scan command troubleshooting](./commands/scan.md#troubleshooting)

### Getting Help

1. Check the command's troubleshooting section
2. Check the setup guide for your backend (BigQuery, Firestore, GitHub App)
3. Review server logs: `octovy serve --log-format json`

## Related Resources

- [Octovy Repository](https://github.com/secmon-lab/octovy)
- [Trivy Documentation](https://aquasecurity.github.io/trivy/)
- [BigQuery Documentation](https://cloud.google.com/bigquery/docs)
- [GitHub App Documentation](https://docs.github.com/en/developers/apps/building-github-apps)
