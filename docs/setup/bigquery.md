# BigQuery Setup Guide

## Overview

BigQuery is the primary storage backend for Octovy. All Trivy scan results are stored in BigQuery for SQL-based querying, historical analysis, and compliance auditing.

BigQuery is **required** to use any Octovy command (scan, insert, or serve).

## Prerequisites

- Google Cloud Platform (GCP) account
- `gcloud` CLI installed
- Appropriate GCP permissions to create datasets and grant IAM roles

## Setup Steps

### Step 1: Create a BigQuery Dataset

#### Using gcloud CLI:

```bash
# Set your project
gcloud config set project YOUR_PROJECT_ID

# Create dataset
bq mk --dataset YOUR_PROJECT_ID:octovy
```

#### Using Google Cloud Console:

1. Go to [Google Cloud Console - BigQuery](https://console.cloud.google.com/bigquery)
2. Click **"Create Dataset"**
3. Set the following:
   - **Dataset ID**: `octovy` (or your preferred name)
   - **Data location**: Choose your preferred region (e.g., `us`, `eu`, `asia-east1`)
   - **Default table expiration**: (Optional) Set if you want automatic cleanup
4. Click **"Create dataset"**

### Step 2: Configure Authentication

Octovy uses Google Cloud Application Default Credentials (ADC). Choose one of the following:

#### Option A: Local Development (using your Google Account)

```bash
gcloud auth application-default login
```

This will open a browser to authenticate with your Google account.

#### Option B: Production (using Service Account)

1. Create a service account:
```bash
gcloud iam service-accounts create octovy-scanner \
  --display-name="Octovy Scanner Service Account" \
  --project=YOUR_PROJECT_ID
```

2. Create and download a key:
```bash
gcloud iam service-accounts keys create octovy-key.json \
  --iam-account=octovy-scanner@YOUR_PROJECT_ID.iam.gserviceaccount.com
```

3. Set environment variable:
```bash
export GOOGLE_APPLICATION_CREDENTIALS=/path/to/octovy-key.json
```

### Step 3: Grant BigQuery Permissions

The user or service account must have the following BigQuery permissions:

- `bigquery.datasets.get`
- `bigquery.tables.create`
- `bigquery.tables.updateData`
- `bigquery.tables.get`

You can use the predefined IAM role `roles/bigquery.dataEditor`:

```bash
# For service account
gcloud projects add-iam-policy-binding YOUR_PROJECT_ID \
  --member=serviceAccount:octovy-scanner@YOUR_PROJECT_ID.iam.gserviceaccount.com \
  --role=roles/bigquery.dataEditor
```

Or grant more granular permissions using a custom role.

### Step 4: Set Environment Variables

Configure these environment variables before running Octovy:

```bash
# Required
export OCTOVY_BIGQUERY_PROJECT_ID=your-project-id

# Optional (with defaults)
export OCTOVY_BIGQUERY_DATASET_ID=octovy     # default: octovy
export OCTOVY_BIGQUERY_TABLE_ID=scans        # default: scans
```

Octovy will automatically create and update the table schema as needed.

## Environment Variables Reference

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `OCTOVY_BIGQUERY_PROJECT_ID` | ✓ | N/A | GCP Project ID |
| `OCTOVY_BIGQUERY_DATASET_ID` | ✗ | `octovy` | BigQuery dataset name |
| `OCTOVY_BIGQUERY_TABLE_ID` | ✗ | `scans` | BigQuery table name |

## Verify Configuration

Test your BigQuery setup:

```bash
# Check authentication
gcloud auth application-default print-access-token

# List datasets (should see your octovy dataset)
bq ls

# Check table (after first scan)
bq show octovy.scans
```

## Table Schema

Octovy automatically creates the BigQuery table with the following columns:

- `scan_id`: Unique scan identifier
- `timestamp`: Scan timestamp
- `github_owner`: Repository owner
- `github_repo`: Repository name
- `github_commit`: Commit hash
- `findings`: Array of vulnerability findings
- `scan_status`: Scan status (success/error)

The schema is automatically updated as new vulnerability types are detected.

## Troubleshooting

### "Permission denied" errors

- Verify the service account has `roles/bigquery.dataEditor` role
- Check that `OCTOVY_BIGQUERY_PROJECT_ID` is set correctly
- Run `gcloud auth application-default login` for local development

### Dataset not found

- Verify the dataset exists: `bq ls`
- Check `OCTOVY_BIGQUERY_DATASET_ID` matches your dataset name
- Dataset must be in the same project as `OCTOVY_BIGQUERY_PROJECT_ID`

### Table not created

- Octovy creates the table automatically on first use
- Run a scan command to trigger table creation
- Check logs for any errors during insertion

## Next Steps

- [Configure GitHub App](./github-app.md) for webhook scanning
- [Configure Firestore](./firestore.md) (optional) for real-time metadata tracking
- [Scan a local directory](../commands/scan.md)
