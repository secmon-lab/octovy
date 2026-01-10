# Scan Command

## Overview

The `scan` command scans repositories with Trivy and inserts results into BigQuery. It has two subcommands:

- **`scan local`**: Scans a local directory on your machine
- **`scan remote`**: Scans a GitHub repository remotely via GitHub App API

**Requirements:**
- BigQuery configured ([setup guide](../setup/bigquery.md))
- Trivy installed on the system
- For remote scanning: GitHub App configured ([setup guide](../setup/github-app.md))

## Use Cases

- **Local development**: Scan your project before committing (`scan local`)
- **CI/CD pipelines**: Automated scanning on every push (`scan local`)
- **Remote repository scans**: Scan GitHub repositories without cloning (`scan remote`)
- **Organization-wide scans**: Scan all repositories for a GitHub owner (`scan remote`)
- **Scheduled scans**: Regular vulnerability checks via cron or GitLab CI

---

## Scan Local

Scans a local directory with Trivy and inserts results into BigQuery.

### Basic Usage

```bash
# Scan current directory (auto-detects git metadata)
octovy scan local

# Scan specific directory
octovy scan local --dir /path/to/repository

# With BigQuery configuration
octovy scan local \
  --bigquery-project-id my-project \
  --bigquery-dataset-id octovy \
  --bigquery-table-id scans
```

### Command Flags

| Flag | Env Variable | Required | Default | Description |
|------|--------------|----------|---------|-------------|
| `--dir`, `-d` | `OCTOVY_SCAN_DIR` | No | `.` (current dir) | Directory to scan |
| `--bigquery-project-id` | `OCTOVY_BIGQUERY_PROJECT_ID` | Yes | N/A | GCP Project ID |
| `--bigquery-dataset-id` | `OCTOVY_BIGQUERY_DATASET_ID` | No | `octovy` | BigQuery dataset name |
| `--bigquery-table-id` | `OCTOVY_BIGQUERY_TABLE_ID` | No | `scans` | BigQuery table name |
| `--firestore-project-id` | `OCTOVY_FIRESTORE_PROJECT_ID` | No | N/A | Firestore project ID (enables Firestore) |
| `--firestore-database-id` | `OCTOVY_FIRESTORE_DATABASE_ID` | No | `(default)` | Firestore database ID |
| `--github-owner` | `OCTOVY_GITHUB_OWNER` | No | Auto-detected | Repository owner |
| `--github-repo` | `OCTOVY_GITHUB_REPO` | No | Auto-detected | Repository name |
| `--github-commit-id` | `OCTOVY_GITHUB_COMMIT_ID` | No | Auto-detected | Commit hash |
| `--trivy-path` | `OCTOVY_TRIVY_PATH` | No | `trivy` | Path to Trivy binary |

### Examples

#### Minimal Scan

```bash
export OCTOVY_BIGQUERY_PROJECT_ID=my-project
octovy scan local
```

This scans the current directory, auto-detecting git metadata.

#### With Explicit GitHub Metadata

Useful when git metadata is not available:

```bash
octovy scan local \
  --dir /path/to/code \
  --github-owner myorg \
  --github-repo myrepo \
  --github-commit-id abc123def456 \
  --bigquery-project-id my-project
```

#### With Firestore Metadata

Also store repository metadata in Firestore:

```bash
octovy scan local \
  --dir /path/to/code \
  --bigquery-project-id my-project \
  --firestore-project-id my-project \
  --firestore-database-id "(default)"
```

### How It Works

1. **Auto-detect metadata** (if not specified):
   - Uses `git` commands to find owner, repo, and commit hash
   - Works only if the directory is a git repository

2. **Run Trivy scan**:
   - Executes Trivy on the specified directory
   - Outputs results in JSON format

3. **Insert into BigQuery**:
   - Parses Trivy results
   - Stores findings in BigQuery table
   - Creates table schema if not exists

4. **Store in Firestore** (if enabled):
   - Stores repository metadata
   - Creates or updates branch and target records
   - Records vulnerability data

---

## Scan Remote

Scans a GitHub repository remotely using the GitHub App API. This allows scanning without cloning the repository locally.

**Requirements:**
- GitHub App must be configured and installed on the target repository/organization
- See [GitHub App setup guide](../setup/github-app.md) for configuration

### Basic Usage

```bash
# Scan a specific repository (latest commit on default branch)
octovy scan remote \
  --github-owner myorg \
  --github-repo myrepo \
  --github-app-id 12345 \
  --github-app-private-key "$(cat private-key.pem)" \
  --bigquery-project-id my-project

# Scan a specific branch
octovy scan remote \
  --github-owner myorg \
  --github-repo myrepo \
  --github-branch main \
  --github-app-id 12345 \
  --github-app-private-key "$(cat private-key.pem)" \
  --bigquery-project-id my-project

# Scan a specific commit
octovy scan remote \
  --github-owner myorg \
  --github-repo myrepo \
  --github-commit abc123def456 \
  --github-app-id 12345 \
  --github-app-private-key "$(cat private-key.pem)" \
  --bigquery-project-id my-project

# Scan ALL repositories for an owner using GitHub API (--all mode)
octovy scan remote \
  --github-owner myorg \
  --all \
  --github-app-id 12345 \
  --github-app-private-key "$(cat private-key.pem)" \
  --bigquery-project-id my-project

# Scan repositories for an owner from Firestore (requires Firestore setup)
octovy scan remote \
  --github-owner myorg \
  --github-app-id 12345 \
  --github-app-private-key "$(cat private-key.pem)" \
  --bigquery-project-id my-project \
  --firestore-project-id my-project
```

### Command Flags

| Flag | Env Variable | Required | Default | Description |
|------|--------------|----------|---------|-------------|
| `--github-owner` | `OCTOVY_GITHUB_OWNER` | Yes | N/A | GitHub repository owner |
| `--github-repo` | `OCTOVY_GITHUB_REPO` | No | N/A | Repository name (if omitted, scans repos for owner) |
| `--github-commit` | `OCTOVY_GITHUB_COMMIT` | No | N/A | Commit ID to scan (mutually exclusive with `--github-branch`) |
| `--github-branch` | `OCTOVY_GITHUB_BRANCH` | No | N/A | Branch name to scan (mutually exclusive with `--github-commit`) |
| `--github-app-installation-id` | `OCTOVY_GITHUB_APP_INSTALLATION_ID` | No | N/A | GitHub App Installation ID |
| `--github-app-id` | `OCTOVY_GITHUB_APP_ID` | Yes | N/A | GitHub App ID |
| `--github-app-private-key` | `OCTOVY_GITHUB_APP_PRIVATE_KEY` | Yes | N/A | GitHub App Private Key (PEM format) |
| `--all`, `-a` | `OCTOVY_SCAN_ALL` | No | `false` | Scan all repos for owner using GitHub API (no Firestore needed) |
| `--bigquery-project-id` | `OCTOVY_BIGQUERY_PROJECT_ID` | Yes | N/A | GCP Project ID |
| `--bigquery-dataset-id` | `OCTOVY_BIGQUERY_DATASET_ID` | No | `octovy` | BigQuery dataset name |
| `--bigquery-table-id` | `OCTOVY_BIGQUERY_TABLE_ID` | No | `scans` | BigQuery table name |
| `--firestore-project-id` | `OCTOVY_FIRESTORE_PROJECT_ID` | No | N/A | Firestore project ID |
| `--firestore-database-id` | `OCTOVY_FIRESTORE_DATABASE_ID` | No | `(default)` | Firestore database ID |
| `--trivy-path` | `OCTOVY_TRIVY_PATH` | No | `trivy` | Path to Trivy binary |

### Examples

#### Single Repository Scan

```bash
export OCTOVY_GITHUB_APP_ID=12345
export OCTOVY_GITHUB_APP_PRIVATE_KEY="$(cat /path/to/private-key.pem)"
export OCTOVY_BIGQUERY_PROJECT_ID=my-project

octovy scan remote \
  --github-owner myorg \
  --github-repo myrepo
```

#### Organization-Wide Scan

There are two modes for scanning all repositories of an owner:

**1. Using GitHub API (`--all` mode) - Recommended**

Scan all repositories that the GitHub App has access to, without requiring Firestore:

```bash
octovy scan remote \
  --github-owner myorg \
  --all \
  --github-app-id 12345 \
  --github-app-private-key "$(cat private-key.pem)" \
  --bigquery-project-id my-project
```

This mode:
- Fetches the repository list directly from GitHub API
- Does not require Firestore configuration
- Automatically excludes archived and disabled repositories
- Scans the default branch of each repository

**2. Using Firestore (legacy mode)**

Scan repositories registered in Firestore:

```bash
octovy scan remote \
  --github-owner myorg \
  --github-app-id 12345 \
  --github-app-private-key "$(cat private-key.pem)" \
  --bigquery-project-id my-project \
  --firestore-project-id my-project
```

This mode:
- Requires Firestore to be configured
- Only scans repositories that have been previously registered in Firestore
- Useful when you want to scan only a specific subset of repositories

#### Branch-Specific Scan

```bash
octovy scan remote \
  --github-owner myorg \
  --github-repo myrepo \
  --github-branch feature/new-feature \
  --github-app-id 12345 \
  --github-app-private-key "$(cat private-key.pem)" \
  --bigquery-project-id my-project
```

### How It Works

1. **Authenticate with GitHub**:
   - Uses GitHub App credentials to authenticate
   - Obtains installation access token for the target repository

2. **Download repository archive**:
   - Downloads the repository as a tarball via GitHub API
   - Extracts to a temporary directory

3. **Run Trivy scan**:
   - Scans the extracted repository with Trivy
   - Generates JSON report

4. **Insert into BigQuery**:
   - Parses scan results
   - Stores findings in BigQuery table

5. **Cleanup**:
   - Removes temporary files

---

## CI/CD Integration

### GitHub Actions (Local Scan)

```yaml
name: Vulnerability Scan

on: [push, pull_request]

jobs:
  scan:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Setup Google Cloud
        uses: google-github-actions/auth@v1
        with:
          credentials_json: ${{ secrets.GCP_SA_KEY }}

      - name: Install Trivy
        run: |
          curl -sfL https://raw.githubusercontent.com/aquasecurity/trivy/main/contrib/install.sh | sh -s -- -b /usr/local/bin

      - name: Install Octovy
        run: go install github.com/secmon-lab/octovy/cmd/octovy@latest

      - name: Scan with Octovy
        run: |
          octovy scan local \
            --bigquery-project-id ${{ secrets.GCP_PROJECT_ID }} \
            --github-owner ${{ github.repository_owner }} \
            --github-repo ${{ github.event.repository.name }} \
            --github-commit-id ${{ github.sha }}
```

### GitLab CI (Local Scan)

```yaml
vulnerability_scan:
  image: alpine:latest
  script:
    - apk add --no-cache curl git go
    - curl -sfL https://raw.githubusercontent.com/aquasecurity/trivy/main/contrib/install.sh | sh -s -- -b /usr/local/bin
    - go install github.com/secmon-lab/octovy/cmd/octovy@latest
    - |
      octovy scan local \
        --bigquery-project-id $GCP_PROJECT_ID \
        --github-owner $CI_PROJECT_NAMESPACE \
        --github-repo $CI_PROJECT_NAME \
        --github-commit-id $CI_COMMIT_SHA
```

### Scheduled Scan (Cron)

```bash
#!/bin/bash
# Scheduled daily scan via cron
0 2 * * * /usr/local/bin/octovy scan local \
  --dir /srv/app \
  --bigquery-project-id my-project \
  --github-owner myorg \
  --github-repo myapp >> /var/log/octovy-scan.log 2>&1
```

---

## Error Handling

### Authentication Failures

If you get `Permission denied` errors:

```bash
# For local development
gcloud auth application-default login

# Or for service account
export GOOGLE_APPLICATION_CREDENTIALS=/path/to/service-account-key.json
```

See [BigQuery setup](../setup/bigquery.md#troubleshooting) for more details.

### Trivy Not Found

If you get `trivy: command not found`:

```bash
# Install Trivy
curl -sfL https://raw.githubusercontent.com/aquasecurity/trivy/main/contrib/install.sh | sh -s -- -b /usr/local/bin

# Or specify path
octovy scan local --trivy-path /path/to/trivy
```

### GitHub App Authentication Errors (Remote Scan)

If you get authentication errors with `scan remote`:

1. Verify GitHub App ID is correct
2. Ensure private key is in PEM format
3. Check that the GitHub App is installed on the target repository
4. Verify the installation has access to the repository

See [GitHub App setup](../setup/github-app.md#troubleshooting) for more details.

### Git Metadata Not Detected

If in a non-git directory or git commands fail:

```bash
# Specify metadata explicitly
octovy scan local \
  --dir /path/to/scan \
  --github-owner myorg \
  --github-repo myrepo \
  --github-commit-id $(date +%s)  # Use timestamp if commit unknown
```

## Troubleshooting

### "Dataset not found"

- Verify BigQuery dataset exists: `bq ls`
- Check `--bigquery-dataset-id` matches your dataset
- See [BigQuery troubleshooting](../setup/bigquery.md#troubleshooting)

### "No vulnerabilities found"

This is normal for clean code. Results are still inserted into BigQuery with 0 findings.

### Slow scans

- Large directories take longer to scan
- Trivy caches results; first run is slower
- Check system resources (disk, memory)

## Next Steps

- [Use insert command for existing results](./insert.md)
- [Run serve command for webhooks](./serve.md)
- [Configure GitHub App for remote scanning](../setup/github-app.md)
- [Configure Firestore for metadata tracking](../setup/firestore.md)
