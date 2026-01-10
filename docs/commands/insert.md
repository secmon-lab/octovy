# Insert Command

## Overview

The `insert` command inserts Trivy scan result JSON files into BigQuery. Use it to integrate existing Trivy workflows or to decouple scanning and result insertion.

**Requirements:**
- BigQuery configured ([setup guide](../setup/bigquery.md))
- Trivy scan result JSON file

## Use Cases

- **Existing Trivy workflows**: Integrate Octovy with existing scanning infrastructure
- **Decouple scanning and insertion**: Run Trivy separately, insert results later
- **Multiple Trivy configurations**: Insert results from different Trivy configs
- **Batch processing**: Insert multiple scan results
- **Legacy data migration**: Import historical Trivy results into BigQuery

## Basic Usage

### Insert Trivy Results

Auto-detects git metadata (owner, repo, commit) from the local repository:

```bash
octovy insert -f scan-result.json
```

### Insert with Explicit Metadata

```bash
octovy insert -f scan-result.json \
  --github-owner myorg \
  --github-repo myrepo \
  --github-commit-id abc123def456
```

### With BigQuery Configuration

```bash
octovy insert -f scan-result.json \
  --bigquery-project-id my-project \
  --bigquery-dataset-id octovy \
  --bigquery-table-id scans
```

## Generate Trivy Results

Before using `octovy insert`, generate a Trivy scan result:

```bash
# Scan with Trivy in JSON format
trivy fs --format json --output scan-result.json /path/to/code

# Scan with additional options
trivy fs --format json --severity HIGH,CRITICAL --output scan-result.json /path/to/code
```

Trivy output options:

```bash
# With severity filter
trivy fs --severity HIGH,CRITICAL --format json --output results.json .

# With license scanning
trivy fs --format json --scanners license --output results.json .

# With secret scanning
trivy fs --format json --scanners secret --output results.json .

# Combined scan
trivy fs --format json --scanners vuln,license,secret --output results.json .
```

## Command Flags Reference

| Flag | Env Variable | Required | Default | Description |
|------|--------------|----------|---------|-------------|
| `-f, --file` | N/A | ✓ | N/A | Path to Trivy JSON result file |
| `--bigquery-project-id` | `OCTOVY_BIGQUERY_PROJECT_ID` | ✓ | N/A | GCP Project ID |
| `--bigquery-dataset-id` | `OCTOVY_BIGQUERY_DATASET_ID` | ✗ | `octovy` | BigQuery dataset name |
| `--bigquery-table-id` | `OCTOVY_BIGQUERY_TABLE_ID` | ✗ | `scans` | BigQuery table name |
| `--firestore-project-id` | `OCTOVY_FIRESTORE_PROJECT_ID` | ✗ | N/A | Firestore project ID (enables Firestore) |
| `--firestore-database-id` | `OCTOVY_FIRESTORE_DATABASE_ID` | ✗ | `(default)` | Firestore database ID |
| `--github-owner` | `OCTOVY_GITHUB_OWNER` | ✗ | Auto-detected | Repository owner |
| `--github-repo` | `OCTOVY_GITHUB_REPO` | ✗ | Auto-detected | Repository name |
| `--github-commit-id` | `OCTOVY_GITHUB_COMMIT_ID` | ✗ | Auto-detected | Commit hash |
| `--log-format` | `OCTOVY_LOG_FORMAT` | ✗ | `text` | Log format: `text` or `json` |

## Examples

### Generate and Insert Trivy Results

```bash
# Step 1: Generate Trivy scan result
trivy fs --format json --output scan-result.json /path/to/code

# Step 2: Insert into BigQuery
export OCTOVY_BIGQUERY_PROJECT_ID=my-project
octovy insert -f scan-result.json
```

### Insert with Explicit Metadata

```bash
trivy fs --format json --output scan-result.json .

octovy insert -f scan-result.json \
  --github-owner myorg \
  --github-repo myrepo \
  --github-commit-id abc123def456 \
  --bigquery-project-id my-project
```

### Insert with Firestore

```bash
octovy insert -f scan-result.json \
  --bigquery-project-id my-project \
  --firestore-project-id my-project \
  --firestore-database-id "(default)"
```

### Batch Insert Multiple Results

```bash
#!/bin/bash
# Insert multiple scan results

PROJECT_ID="my-project"

for result in results/*.json; do
  echo "Inserting $result..."
  octovy insert -f "$result" \
    --bigquery-project-id "$PROJECT_ID" \
    --github-owner myorg \
    --github-repo myrepo
done
```

### In CI/CD Pipeline (GitHub Actions)

```yaml
name: Vulnerability Scan and Insert

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

      - name: Scan with Trivy
        run: |
          trivy fs --format json --output scan-result.json .

      - name: Install Octovy
        run: go install github.com/secmon-lab/octovy/cmd/octovy@latest

      - name: Insert Results into BigQuery
        run: |
          octovy insert -f scan-result.json \
            --bigquery-project-id ${{ secrets.GCP_PROJECT_ID }} \
            --github-owner ${{ github.repository_owner }} \
            --github-repo ${{ github.event.repository.name }} \
            --github-commit-id ${{ github.sha }}
```

### In CI/CD Pipeline (GitLab CI)

```yaml
scan_and_insert:
  image: alpine:latest
  script:
    - apk add --no-cache curl git
    - curl -sfL https://raw.githubusercontent.com/aquasecurity/trivy/main/contrib/install.sh | sh -s -- -b /usr/local/bin
    - trivy fs --format json --output scan-result.json .
    - go install github.com/secmon-lab/octovy/cmd/octovy@latest
    - |
      octovy insert -f scan-result.json \
        --bigquery-project-id $GCP_PROJECT_ID \
        --github-owner $CI_PROJECT_NAMESPACE \
        --github-repo $CI_PROJECT_NAME \
        --github-commit-id $CI_COMMIT_SHA
```

### Separate Scanning and Insertion (Different Machines)

Machine 1 (Scanning):
```bash
# Run Trivy on a machine with the code
trivy fs --format json --output scan-result.json /path/to/code

# Copy or upload scan-result.json to another machine
scp scan-result.json user@gcp-machine:/tmp/
```

Machine 2 (Insertion):
```bash
# Insert on a machine with GCP credentials
octovy insert -f /tmp/scan-result.json \
  --bigquery-project-id my-project \
  --github-owner myorg \
  --github-repo myrepo \
  --github-commit-id abc123def456
```

### Insert with Different Trivy Configurations

```bash
# High-severity only
trivy fs --severity HIGH,CRITICAL --format json --output high.json .
octovy insert -f high.json --bigquery-project-id my-project

# All vulnerabilities
trivy fs --format json --output all.json .
octovy insert -f all.json --bigquery-project-id my-project
```

## How It Works

1. **Validate JSON file**:
   - Parses the Trivy JSON result file
   - Checks format compatibility

2. **Auto-detect metadata** (if not specified):
   - Uses `git` commands to find owner, repo, and commit hash
   - Falls back to specified flags

3. **Insert into BigQuery**:
   - Transforms Trivy results into BigQuery schema
   - Stores findings in BigQuery table

4. **Store in Firestore** (if enabled):
   - Creates repository metadata records
   - Records branch and target information
   - Stores vulnerability data

## Trivy JSON Format

The JSON file must be a valid Trivy filesystem scan result. Example:

```json
{
  "SchemaVersion": 2,
  "ArtifactName": ".",
  "ArtifactType": "filesystem",
  "Metadata": {
    "ImageConfig": {
      "architecture": "amd64"
    }
  },
  "Results": [
    {
      "Target": "package.json",
      "Class": "os-pkgs",
      "Type": "npm",
      "Vulnerabilities": [
        {
          "VulnerabilityID": "CVE-2021-12345",
          "PkgName": "lodash",
          "PkgVer": "4.17.20",
          "Severity": "HIGH",
          "Title": "Prototype pollution"
        }
      ]
    }
  ]
}
```

## Error Handling

### File Not Found

```bash
# Error: cannot read file
octovy insert -f scan-result.json

# Solution: Check file path
ls -la scan-result.json
```

### Invalid JSON Format

```bash
# Error: invalid JSON format
octovy insert -f scan-result.json

# Solution: Validate with Trivy
trivy --version  # Ensure compatible Trivy version
```

### Authentication Failures

See [BigQuery troubleshooting](../setup/bigquery.md#troubleshooting)

### Git Metadata Not Detected

```bash
# If not in a git repository, specify metadata explicitly
octovy insert -f scan-result.json \
  --github-owner myorg \
  --github-repo myrepo \
  --github-commit-id abc123def456
```

## Troubleshooting

### "File not found" error

- Verify file path is correct: `ls -la /path/to/file.json`
- Use absolute paths when possible
- Check file permissions are readable

### "Invalid Trivy result" error

- Ensure file is from Trivy filesystem scan (not image/container scan)
- Regenerate with: `trivy fs --format json --output scan-result.json .`
- Check Trivy version compatibility

### BigQuery insert fails

- Check BigQuery configuration: `bq ls`
- Verify authentication: `gcloud auth application-default print-access-token`
- See [BigQuery troubleshooting](../setup/bigquery.md#troubleshooting)

### Metadata detection fails

- Works only if in a git repository
- Use explicit flags: `--github-owner`, `--github-repo`, `--github-commit-id`

## Next Steps

- [Use scan command for automated scanning](./scan.md)
- [Run serve command for webhook integration](./serve.md)
- [Configure Firestore for metadata tracking](../setup/firestore.md)
