# Octovy

[![test](https://github.com/secmon-lab/octovy/actions/workflows/test.yml/badge.svg)](https://github.com/secmon-lab/octovy/actions/workflows/test.yml)
[![gosec](https://github.com/secmon-lab/octovy/actions/workflows/gosec.yml/badge.svg)](https://github.com/secmon-lab/octovy/actions/workflows/gosec.yml)
[![trivy](https://github.com/secmon-lab/octovy/actions/workflows/trivy.yml/badge.svg)](https://github.com/secmon-lab/octovy/actions/workflows/trivy.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/secmon-lab/octovy.svg)](https://pkg.go.dev/github.com/secmon-lab/octovy)
[![Go Report Card](https://goreportcard.com/badge/github.com/secmon-lab/octovy)](https://goreportcard.com/report/github.com/secmon-lab/octovy)

**A standalone [Trivy](https://github.com/aquasecurity/trivy)-to-BigQuery export tool**

![image](https://github.com/user-attachments/assets/52a87974-bec8-4f13-ab48-2ef2c24400ea)

## Overview

Octovy stores [Trivy](https://github.com/aquasecurity/trivy) vulnerability scan results in BigQuery. This enables historical analysis, vulnerability trend tracking, and compliance auditing.

**What it does:**
- **BigQuery Storage**: Stores Trivy scan results in BigQuery for SQL-based querying
- **Three entry points**: Scan local directories, insert existing results, or automate via GitHub webhooks
- **Firestore Support** (optional): Stores metadata in Firestore for real-time querying

## Key Features

Octovy provides three ways to insert scan results into BigQuery:

### 1. `scan` - Scan Local Directory
Scans a local directory with Trivy and inserts results into BigQuery. Use for local development and CI/CD pipelines.

### 2. `insert` - Insert Existing Results
Inserts Trivy scan result JSON files into BigQuery. Use to integrate with existing Trivy workflows.

### 3. `serve` - GitHub Webhook Server
Runs as a GitHub App to scan repositories on `push` and `pull_request` events. Use for organization-wide scanning.

## Setup

### Prerequisites: BigQuery (Required)

BigQuery is the primary storage backend for Octovy. You must configure BigQuery before using any Octovy command.

#### 1. Create a BigQuery Dataset

```bash
# Using gcloud CLI
gcloud config set project YOUR_PROJECT_ID
bq mk --dataset YOUR_PROJECT_ID:octovy
```

Or create via [Google Cloud Console](https://console.cloud.google.com/bigquery):
- Navigate to BigQuery
- Click "Create Dataset"
- Dataset ID: `octovy` (or your preferred name)
- Data location: Choose your preferred region

#### 2. Configure Authentication

Octovy uses Google Cloud Application Default Credentials (ADC):

```bash
# For local development
gcloud auth application-default login

# For production (service account)
export GOOGLE_APPLICATION_CREDENTIALS=/path/to/service-account-key.json
```

#### 3. Grant Permissions

The service account or user must have:
- `bigquery.datasets.get`
- `bigquery.tables.create`
- `bigquery.tables.updateData`
- `bigquery.tables.get`

You can use the predefined role `roles/bigquery.dataEditor`.

#### 4. Set Environment Variables

```bash
export OCTOVY_BIGQUERY_PROJECT_ID=your-project-id
export OCTOVY_BIGQUERY_DATASET_ID=octovy  # default: octovy
export OCTOVY_BIGQUERY_TABLE_ID=scans     # default: scans
```

Octovy will automatically create and update the table schema as needed.

### Optional: Firestore

Firestore stores repository metadata, branches, targets, and vulnerabilities. This enables real-time querying and relationship tracking.

**When to use Firestore:**
- Real-time vulnerability status queries
- Repository metadata and relationship tracking
- Hierarchical data organization (repo → branch → target → vulnerability)

**When to skip Firestore:**
- Only historical scan data needed
- Compliance reporting and analytics use cases

#### 1. Enable Firestore

```bash
# Enable Firestore API
gcloud services enable firestore.googleapis.com --project=YOUR_PROJECT_ID

# Create Firestore database
gcloud firestore databases create --location=YOUR_REGION --project=YOUR_PROJECT_ID
```

Or via [Google Cloud Console](https://console.cloud.google.com/firestore):
- Navigate to Firestore
- Click "Create Database"
- Choose Native mode
- Select your region

#### 2. Grant Permissions

The service account must have:
- `datastore.databases.get`
- `datastore.entities.create`
- `datastore.entities.get`
- `datastore.entities.update`

You can use the predefined role `roles/datastore.user`.

#### 3. Set Environment Variables

```bash
export OCTOVY_FIRESTORE_PROJECT_ID=your-project-id
export OCTOVY_FIRESTORE_DATABASE_ID="(default)"  # or your database ID
```

### Optional: GitHub App (for `serve` command only)

To use the `serve` command to scan repositories via GitHub webhooks, create a GitHub App.

#### 1. Create a GitHub App

Go to [GitHub Settings > Developer settings > GitHub Apps](https://github.com/settings/apps) and create a new app:

**General Settings:**
- **Webhook URL**: `https://your-domain.com/webhook/github/app`
- **Webhook secret**: Generate a random string (e.g., `openssl rand -hex 32`)

**Permissions:**
- Repository permissions:
  - **Contents**: Read-only
  - **Metadata**: Read-only

**Subscribe to events:**
- **Pull request**
- **Push**

#### 2. Generate Private Key

After creating the app:
- Scroll to "Private keys" section
- Click "Generate a private key"
- Download the `.pem` file

#### 3. Note Your App Credentials

You'll need:
- **App ID** (shown on the app's general settings page)
- **Private Key** (the `.pem` file you downloaded)
- **Webhook Secret** (the secret you configured)

#### 4. Install the App

Install the GitHub App on repositories to scan:
- Go to the app's page
- Click "Install App"
- Select repositories

## Usage

### 1. `scan` - Scan Local Directory

Scans a local directory with Trivy and inserts results into BigQuery.

**Basic Usage:**

```bash
# Scan current directory (auto-detects git metadata)
octovy scan

# Scan specific directory
octovy scan --dir /path/to/repository
```

**With BigQuery Configuration:**

```bash
octovy scan \
  --bigquery-project-id my-project \
  --bigquery-dataset-id octovy \
  --bigquery-table-id scans
```

**With Firestore:**

```bash
octovy scan \
  --bigquery-project-id my-project \
  --firestore-project-id my-project \
  --firestore-database-id "(default)"
```

**With Explicit GitHub Metadata:**

```bash
octovy scan \
  --github-owner myorg \
  --github-repo myrepo \
  --github-commit-id abc123def456
```

**How it works:**
1. Auto-detects GitHub metadata from local git repository (owner, repo, commit ID)
2. Runs Trivy scan on the directory
3. Inserts results into BigQuery
4. Optionally stores metadata in Firestore

### 2. `insert` - Insert Existing Trivy Results

Inserts a Trivy scan result JSON file into BigQuery.

**Basic Usage:**

```bash
# Insert existing Trivy result (auto-detects git metadata)
octovy insert -f scan-result.json

# Insert with explicit metadata
octovy insert -f scan-result.json \
  --github-owner myorg \
  --github-repo myrepo \
  --github-commit-id abc123def456
```

**Generate Trivy Results:**

```bash
# First, run Trivy to generate results
trivy fs --format json --output scan-result.json /path/to/code

# Then insert into BigQuery
octovy insert -f scan-result.json
```

**With BigQuery and Firestore:**

```bash
octovy insert -f scan-result.json \
  --bigquery-project-id my-project \
  --bigquery-dataset-id octovy \
  --firestore-project-id my-project
```

**How it works:**
1. Reads the Trivy JSON result file
2. Auto-detects GitHub metadata from local git repository (if not specified)
3. Inserts results into BigQuery
4. Optionally stores metadata in Firestore

**Use Cases:**
- Integrate with existing Trivy workflows
- Separate scanning and insertion steps
- Insert results from different Trivy configurations

### 3. `serve` - GitHub Webhook Server

Runs Octovy as a server to scan repositories on GitHub events.

**Basic Usage:**

```bash
octovy serve --addr :8080
```

**With Environment Variables:**

```bash
export OCTOVY_ADDR=:8080
export OCTOVY_GITHUB_APP_ID=123456
export OCTOVY_GITHUB_APP_PRIVATE_KEY=/path/to/private-key.pem
export OCTOVY_GITHUB_APP_SECRET=your-webhook-secret
export OCTOVY_BIGQUERY_PROJECT_ID=my-project
export OCTOVY_BIGQUERY_DATASET_ID=octovy
export OCTOVY_FIRESTORE_PROJECT_ID=my-project

octovy serve
```

**Using Docker:**

The Docker image does not include Trivy. You must provide Trivy binary by mounting it from the host:

```bash
# Install Trivy on your host system first
# See: https://aquasecurity.github.io/trivy/latest/getting-started/installation/

# Run Octovy with Trivy mounted from host
docker run -p 8080:8080 \
  -v /path/to/private-key.pem:/key.pem \
  -v $(which trivy):/trivy \
  -e OCTOVY_ADDR=:8080 \
  -e OCTOVY_GITHUB_APP_ID=123456 \
  -e OCTOVY_GITHUB_APP_PRIVATE_KEY=/key.pem \
  -e OCTOVY_GITHUB_APP_SECRET=your-webhook-secret \
  -e OCTOVY_BIGQUERY_PROJECT_ID=my-project \
  -e OCTOVY_BIGQUERY_DATASET_ID=octovy \
  -e OCTOVY_TRIVY_PATH=/trivy \
  ghcr.io/secmon-lab/octovy
```

Alternatively, you can build a custom image with Trivy included. See [examples/Dockerfile](examples/Dockerfile) for a multi-stage build example.

**How it works:**
1. Receives webhook events from GitHub (`push` and `pull_request`)
2. Downloads repository code as archive
3. Extracts and scans with Trivy
4. Inserts results into BigQuery
5. Optionally stores metadata in Firestore
6. Cleans up temporary files

**Required Environment Variables:**
- `OCTOVY_ADDR`: Server bind address (e.g., `:8080`)
- `OCTOVY_GITHUB_APP_ID`: GitHub App ID
- `OCTOVY_GITHUB_APP_PRIVATE_KEY`: Path to private key file or PEM content
- `OCTOVY_GITHUB_APP_SECRET`: Webhook secret

**Optional Environment Variables:**
- `OCTOVY_BIGQUERY_PROJECT_ID`: BigQuery project ID
- `OCTOVY_BIGQUERY_DATASET_ID`: BigQuery dataset ID (default: `octovy`)
- `OCTOVY_BIGQUERY_TABLE_ID`: BigQuery table ID (default: `scans`)
- `OCTOVY_FIRESTORE_PROJECT_ID`: Firestore project ID
- `OCTOVY_FIRESTORE_DATABASE_ID`: Firestore database ID (default: `(default)`)
- `OCTOVY_TRIVY_PATH`: Path to Trivy binary (default: `trivy`)
- `OCTOVY_LOG_FORMAT`: Log format - `text` or `json` (default: `text`)
- `OCTOVY_SENTRY_DSN`: Sentry DSN for error tracking
- `OCTOVY_SENTRY_ENV`: Sentry environment name

**Endpoints:**
- `POST /webhook/github/app` - GitHub App webhook endpoint
- `GET /health` - Health check endpoint

## License

Octovy is licensed under the Apache License 2.0. Copyright 2023 Masayoshi Mizutani <mizutani@hey.com>
