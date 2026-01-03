# Octovy

Octovy is a GitHub App that scans your repository's code for potentially vulnerable dependencies. It utilizes [trivy](https://github.com/aquasecurity/trivy) to detect software vulnerabilities. When triggered by events like `push` and `pull_request` from GitHub, Octovy scans the repository for dependency vulnerabilities and inserts the scan results into BigQuery.

![architecture](https://github.com/m-mizutani/octovy/assets/605953/4366161f-a4ff-4abb-9766-0fb4df818cb1)

Octovy can also be used as a CLI tool to scan local directories and insert results into BigQuery.

## Setup

### 1. Creating a GitHub App

Start by creating a GitHub App [here](https://github.com/settings/apps). You can use any name and description you like. However, ensure you set the following configurations:

- **General**
  - **Webhook URL**: `https://<your domain>/webhook/github/app`
  - **Webhook secret**: A string of your choosing (e.g. `mysecret_XOIJPOIFEA`)

- **Permissions & events**
  - Repository Permissions
    - **Contents**: Set to Read-only
    - **Metadata**: Set to Read-only
  - Subscribe to events
    - **Pull request**
    - **Push**

Once you have completed the setup, make sure to take note of the following information from the **General** section for future reference:

- **App ID** (e.g. `123456`)
- **Private Key**: Click `Generate a private key` and download the key file (e.g. `your-app-name.2023-08-14.private-key.pem`)

### 2. Setting Up Cloud Resources

- **BigQuery** (Optional): Create a BigQuery dataset for storing the scan results. Octovy will automatically create and update the table schema. The default dataset name is `octovy` and the default table name is `scans`.

### 3. Deploying Octovy

The recommended method of deploying Octovy is via a container image, available at `ghcr.io/m-mizutani/octovy`. This image is built using GitHub Actions and published to the GitHub Container Registry.

To run Octovy, set the following environment variables:

#### Required Environment Variables
- `OCTOVY_ADDR`: The address to bind the server to (e.g. `:8080`)
- `OCTOVY_GITHUB_APP_ID`: The GitHub App ID
- `OCTOVY_GITHUB_APP_PRIVATE_KEY`: The path to the private key file
- `OCTOVY_GITHUB_APP_SECRET`: The secret string used to verify the webhook request from GitHub

#### Optional Environment Variables
- `OCTOVY_TRIVY_PATH`: The path to the trivy binary (default: `trivy`). If you use the container image, you don't need to set this variable.
- `OCTOVY_BIGQUERY_PROJECT_ID`: The BigQuery project ID
- `OCTOVY_BIGQUERY_DATASET_ID`: The BigQuery dataset ID (default: `octovy`)
- `OCTOVY_BIGQUERY_TABLE_ID`: The BigQuery table ID (default: `scans`)
- `OCTOVY_BIGQUERY_IMPERSONATE_SERVICE_ACCOUNT`: The service account to impersonate when accessing BigQuery
- `OCTOVY_LOG_FORMAT`: Log format (`text` or `json`, default: `text`)
- `OCTOVY_SENTRY_DSN`: The DSN for Sentry
- `OCTOVY_SENTRY_ENV`: The environment for Sentry

## Usage

### CLI: Scan Local Directory

Octovy can be used as a CLI tool to scan local directories and insert results into BigQuery:

```bash
# Scan current directory (auto-detects GitHub metadata from git)
octovy scan

# Scan specific directory
octovy scan --dir /path/to/repo

# Specify GitHub metadata explicitly
octovy scan --github-owner myorg --github-repo myrepo --github-commit-id abc123

# With BigQuery configuration
octovy scan \
  --bigquery-project-id my-project \
  --bigquery-dataset-id my-dataset \
  --bigquery-table-id scans
```

The `scan` command will:
1. Auto-detect GitHub metadata (owner, repo, commit ID) from git if not explicitly provided
2. Run Trivy on the specified directory
3. Insert scan results into BigQuery (if BigQuery is configured)

### Server: GitHub App Webhook

Run the server to receive GitHub webhook events:

```bash
octovy serve --addr :8080
```

The server will:
1. Receive webhook events from GitHub (`push` and `pull_request`)
2. Download the repository code
3. Run Trivy scan
4. Insert results into BigQuery (if configured)

## Configuration

### Ignore list

The developer can ignore specific vulnerabilities by adding them to the ignore list. The config file is written in CUE. See CUE definition in [pkg/domain/model/schema/ignore.cue](pkg/domain/model/schema/ignore.cue).

The config file should be placed in `.octovy` directory at the root of the repository. Octovy checks all files in the `.octovy` directory recursively and loads them. (e.g. `.octovy/ignore.cue`)

The following is an example of the ignore list configuration:

```cue
package octovy

IgnoreList: [
  {
    Target: "Gemfile.lock"
    Vulns: [
      {
        ID:        "CVE-2020-8130"
        ExpiresAt: "2024-08-01T00:00:00Z"
        Comment:   "This is not used"
      },
    ]
  },
]
```

`package` name should be `octovy`. `IgnoreList` is a list of `Ignore` struct.

- `Target` is the file path to ignore. That should be matched `Target` of trivy
- `Vulns` is a list of `IgnoreVuln` struct.
  - `ID` (required):  the vulnerability ID to ignore. (e.g. `CVE-2022-2202`)
  - `ExpiresAt` (required): The expiration date of the ignore. It should be in RFC3339 format. (e.g. `2023-08-01T00:00:00`). The date must be in 90 days and if it's over 90 days, Octovy will ignore it.
  - `Comment` (optional): The developer's comment


## License

Octovy is licensed under the Apache License 2.0. Copyright 2023 Masayoshi Mizutani <mizutani@hey.com>