# Octovy

[![test](https://github.com/secmon-lab/octovy/actions/workflows/test.yml/badge.svg)](https://github.com/secmon-lab/octovy/actions/workflows/test.yml)
[![gosec](https://github.com/secmon-lab/octovy/actions/workflows/gosec.yml/badge.svg)](https://github.com/secmon-lab/octovy/actions/workflows/gosec.yml)
[![trivy](https://github.com/secmon-lab/octovy/actions/workflows/trivy.yml/badge.svg)](https://github.com/secmon-lab/octovy/actions/workflows/trivy.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/secmon-lab/octovy.svg)](https://pkg.go.dev/github.com/secmon-lab/octovy)
[![Go Report Card](https://goreportcard.com/badge/github.com/secmon-lab/octovy)](https://goreportcard.com/report/github.com/secmon-lab/octovy)

**A standalone [Trivy](https://github.com/aquasecurity/trivy)-to-BigQuery export tool**

![image](https://github.com/user-attachments/assets/52a87974-bec8-4f13-ab48-2ef2c24400ea)

## Overview

[Trivy](https://github.com/aquasecurity/trivy) is a powerful open-source vulnerability scanner and SBOM generator with comprehensive detection capabilities across multiple ecosystems. Trivy as a CLI tool focuses on scanning functionality; for organizations that want to integrate scan results into their existing data infrastructure (such as BigQuery), Octovy provides a lightweight solution.

Octovy exports Trivy scan results to BigQuery, making them searchable via SQL. It provides three core functions:

- **Insert existing Trivy results** (`insert`): Import Trivy JSON output files into BigQuery
- **Scan and insert** (`scan`): Run Trivy on a local directory and insert results into BigQuery
- **GitHub App webhook server** (`serve`): Scan repositories automatically on `push` and `pull_request` events

These functions can be used with GitHub Actions or deployed as a GitHub App. Storing results in BigQuery enables organization-wide vulnerability management:

- **Measure vulnerability exposure**: Query how many packages with known vulnerabilities exist across all repositories in your organization
- **Rapid incident response**: When a critical vulnerability is announced, search for affected packages by name or version across your organization immediately—before vulnerability databases or scanners are updated
- **Continuous monitoring**: Set up scheduled queries to check for specific critical vulnerabilities periodically

## Prerequisites

Before using Octovy, you need to set up BigQuery and configure Google Cloud authentication.

### 1. Create BigQuery Dataset

```bash
bq mk --dataset your-project-id:octovy
```

### 2. Configure Authentication

```bash
gcloud auth application-default login
```

For detailed setup instructions (service accounts, IAM permissions, etc.), see [BigQuery Setup Guide](./docs/setup/bigquery.md).

## Commands

### `scan` - Scan and Insert

Scans repositories with Trivy and inserts results into BigQuery. Has two subcommands:

#### `scan local` - Scan Local Directory

Scans a local directory. Auto-detects git metadata (owner, repo, commit) from the local repository.

```bash
# Scan current directory
octovy scan local --bigquery-project-id your-project-id

# Scan specific directory
octovy scan local --dir /path/to/code --bigquery-project-id your-project-id

# With explicit metadata
octovy scan local \
  --bigquery-project-id your-project-id \
  --github-owner myorg \
  --github-repo myrepo \
  --github-commit-id abc123
```

#### `scan remote` - Scan GitHub Repository

Scans a GitHub repository remotely via GitHub App API. Requires GitHub App configuration.

```bash
# Scan a specific repository
octovy scan remote \
  --bigquery-project-id your-project-id \
  --github-owner myorg \
  --github-repo myrepo \
  --github-app-id 12345 \
  --github-app-private-key "$(cat private-key.pem)"

# Scan all repositories for an organization
octovy scan remote \
  --bigquery-project-id your-project-id \
  --github-owner myorg \
  --all \
  --github-app-id 12345 \
  --github-app-private-key "$(cat private-key.pem)"
```

[Full documentation →](./docs/commands/scan.md)

### `insert` - Insert Existing Results

Inserts Trivy scan result JSON files into BigQuery. Useful when you already have Trivy workflows or want to decouple scanning from insertion.

```bash
# Generate Trivy result and insert
trivy fs --format json --output results.json .
octovy insert -f results.json --bigquery-project-id your-project-id

# Insert with explicit metadata
octovy insert -f results.json \
  --bigquery-project-id your-project-id \
  --github-owner myorg \
  --github-repo myrepo
```

[Full documentation →](./docs/commands/insert.md)

### `serve` - GitHub App Server

Runs an HTTP server that receives GitHub webhooks and automatically scans repositories on `push` and `pull_request` events.

```bash
octovy serve \
  --addr :8080 \
  --bigquery-project-id your-project-id \
  --github-app-id 12345 \
  --github-app-private-key "$(cat private-key.pem)" \
  --github-app-secret your-webhook-secret
```

[Full documentation →](./docs/commands/serve.md)

## BigQuery Queries

Once scan results are in BigQuery, you can run powerful queries for vulnerability management.

### Find All Critical Vulnerabilities

```sql
SELECT
  github.owner,
  github.repo_name,
  vuln.VulnerabilityID,
  vuln.PkgName,
  vuln.Severity
FROM `your-project.octovy.scans`,
  UNNEST(report.Results) AS result,
  UNNEST(result.Vulnerabilities) AS vuln
WHERE vuln.Severity = 'CRITICAL'
ORDER BY timestamp DESC
```

### Search for a Specific Package (e.g., Log4j)

When a critical vulnerability like Log4Shell is announced, immediately find all affected repositories:

```sql
SELECT DISTINCT
  github.owner,
  github.repo_name,
  pkg.Name,
  pkg.Version
FROM `your-project.octovy.scans`,
  UNNEST(report.Results) AS result,
  UNNEST(result.Packages) AS pkg
WHERE LOWER(pkg.Name) LIKE '%log4j%'
ORDER BY github.owner, github.repo_name
```

For more query examples and detailed schema documentation, see [BigQuery Schema Reference](./docs/schema/scans.md).

## Setup Guides

### Required Setup

- **[BigQuery Setup](./docs/setup/bigquery.md)** - Required for all commands

### Optional Setup

- **[GitHub App Setup](./docs/setup/github-app.md)** - Required for `serve` and `scan remote` commands
- **[Firestore Setup](./docs/setup/firestore.md)** - Optional for real-time metadata tracking

## Documentation

See [docs/README.md](./docs/README.md) for:
- Detailed command documentation
- Setup guides for all services
- Common workflows and examples
- Troubleshooting guides
- FAQ

## License

Octovy is licensed under the Apache License 2.0. Copyright 2023 Masayoshi Mizutani <mizutani@hey.com>
