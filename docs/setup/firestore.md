# Firestore Setup Guide

## Overview

Firestore stores repository metadata, branches, targets, and vulnerabilities in Octovy. This enables real-time querying, relationship tracking, and hierarchical data organization.

**Firestore is optional**. Octovy works fine with BigQuery-only storage. Use Firestore when you need real-time metadata access.

## When to Use Firestore

### Use Firestore when:
- You need real-time vulnerability status queries
- You want repository metadata and relationship tracking (repo → branch → target → vulnerability)
- You prefer document-based queries over SQL analytics

### Skip Firestore when:
- You only need historical scan data in BigQuery
- You're doing compliance reporting and analytics
- You want to minimize GCP costs

## Prerequisites

- Google Cloud Platform (GCP) account
- `gcloud` CLI installed
- Appropriate GCP permissions to enable APIs and create databases

## Setup Steps

### Step 1: Enable Firestore API

#### Using gcloud CLI:

```bash
gcloud services enable firestore.googleapis.com --project=YOUR_PROJECT_ID
```

#### Using Google Cloud Console:

1. Go to [Google Cloud Console - APIs & Services](https://console.cloud.google.com/apis/dashboard)
2. Click **"Enable APIs and Services"**
3. Search for **"Firestore"**
4. Click on **"Cloud Firestore"**
5. Click **"Enable"**

### Step 2: Create Firestore Database

#### Using gcloud CLI:

```bash
# Create database in your preferred region
gcloud firestore databases create \
  --location=YOUR_REGION \
  --project=YOUR_PROJECT_ID

# Example regions: us-east1, eu-west1, asia-east1
```

#### Using Google Cloud Console:

1. Go to [Google Cloud Console - Firestore](https://console.cloud.google.com/firestore)
2. Click **"Create Database"**
3. Set the following:
   - **Mode**: Select **"Native mode"** (not Datastore mode)
   - **Location**: Choose your preferred region
4. Click **"Create database"**

### Step 3: Grant Firestore Permissions

The user or service account must have the following Firestore permissions:

- `datastore.databases.get`
- `datastore.entities.create`
- `datastore.entities.get`
- `datastore.entities.update`

You can use the predefined IAM role `roles/datastore.user`:

```bash
# For service account
gcloud projects add-iam-policy-binding YOUR_PROJECT_ID \
  --member=serviceAccount:octovy-scanner@YOUR_PROJECT_ID.iam.gserviceaccount.com \
  --role=roles/datastore.user

# For your user account
gcloud projects add-iam-policy-binding YOUR_PROJECT_ID \
  --member=user:your-email@example.com \
  --role=roles/datastore.user
```

### Step 4: Set Environment Variables

Configure these environment variables to enable Firestore:

```bash
# Firestore configuration
export OCTOVY_FIRESTORE_PROJECT_ID=your-project-id
export OCTOVY_FIRESTORE_DATABASE_ID="(default)"  # or your custom database ID
```

If you don't set these variables, Octovy will skip Firestore and use BigQuery-only storage.

## Environment Variables Reference

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `OCTOVY_FIRESTORE_PROJECT_ID` | ✗ | N/A | GCP Project ID (enables Firestore) |
| `OCTOVY_FIRESTORE_DATABASE_ID` | ✗ | `(default)` | Firestore database ID |

**Note**: Both variables must be set for Firestore to be enabled.

## Data Structure

Octovy stores the following Firestore collections:

### Collections

- **`repositories`**: Repository metadata
  - Document ID: `{owner}/{repo}`
  - Fields: owner, name, scan_count, last_scan_time

- **`branches`**: Branch information per repository
  - Path: `repositories/{repo_id}/branches/{branch_name}`
  - Fields: name, commit, last_scan_time

- **`targets`**: Scan targets (e.g., files, directories)
  - Path: `repositories/{repo_id}/branches/{branch}/targets/{target_id}`
  - Fields: name, type, findings_count

- **`vulnerabilities`**: Individual vulnerabilities
  - Path: `repositories/{repo_id}/branches/{branch}/targets/{target}/vulnerabilities/{vuln_id}`
  - Fields: severity, type, description, fixed_version

## Verify Configuration

Test your Firestore setup:

```bash
# Check that Firestore API is enabled
gcloud services list --enabled --project=YOUR_PROJECT_ID | grep firestore

# List Firestore databases
gcloud firestore databases list --project=YOUR_PROJECT_ID

# Check IAM binding
gcloud projects get-iam-policy YOUR_PROJECT_ID \
  --flatten="bindings[].members" \
  --filter="bindings.role:roles/datastore.user"
```

## Troubleshooting

### "Permission denied" errors

- Verify the service account has `roles/datastore.user` role
- Check that `OCTOVY_FIRESTORE_PROJECT_ID` is set correctly
- Run `gcloud auth application-default login` for local development

### Database not created

- Verify the database exists: `gcloud firestore databases list`
- Check that Firestore API is enabled: `gcloud services list --enabled`
- Database must be in the same project as `OCTOVY_FIRESTORE_PROJECT_ID`

### Collections not appearing

- Firestore collections are created on first use when data is inserted
- Run a scan with `--firestore-project-id` to trigger collection creation
- Check logs for any errors during insertion

### Firestore mode issues

- Octovy requires **Native mode**, not Datastore mode
- If you have an existing Datastore database, create a new Firestore database

## Billing Considerations

Firestore uses the following billing metrics:

- **Read operations**: 1 operation per document read
- **Write operations**: 1 operation per document write
- **Delete operations**: 1 operation per document delete
- **Storage**: Per GB of data stored

Typical Octovy usage:
- 1 repository scan = ~5-10 write operations (repo + branches + targets + vulnerabilities)
- 1 query = ~1 read operation per document

## Next Steps

- [Configure GitHub App](./github-app.md) for webhook scanning
- [Use Firestore in scan command](../commands/scan.md#with-firestore)
- Return to [BigQuery Setup](./bigquery.md)
