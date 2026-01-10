# GitHub App Setup Guide

## Overview

The GitHub App allows Octovy to scan repositories on `push` and `pull_request` events via webhooks. This is required for the `serve` command.

**GitHub App is optional** if you only use the `scan` or `insert` commands locally.

## When to Use GitHub App

### Use GitHub App when:
- You want to automatically scan repositories on every push
- You want to scan pull requests to find vulnerabilities before merge
- You want organization-wide scanning without manual intervention

### Skip GitHub App when:
- You only need manual scanning via `octovy scan`
- You only need to insert existing Trivy results via `octovy insert`

## Prerequisites

- GitHub account (personal or organization)
- Octovy `serve` server running and publicly accessible (with HTTPS)
- DNS pointing to your server

## Setup Steps

### Step 1: Create a GitHub App

1. Go to [GitHub Settings > Developer settings > GitHub Apps](https://github.com/settings/apps)
2. Click **"New GitHub App"**
3. Fill in the app information:

**App name:**
- GitHub App name (e.g., `octovy-scanner`, `my-org-scanner`)

**Homepage URL:**
- Your public server URL (e.g., `https://scanner.example.com`)

**Webhook URL:**
- `https://your-domain.com/webhook/github/app`
- This is where GitHub sends events to Octovy
- Must be publicly accessible and use HTTPS

**Webhook secret:**
- Generate a random secret string (e.g., `openssl rand -hex 32`)
- Octovy uses this to verify webhook authenticity

**Permissions:**

Set the following repository permissions:

- **Contents**: Read-only (to access repository code)
- **Metadata**: Read-only (to access repository metadata)

**Subscribe to events:**

Select these events:

- **Pull request**: Scan PRs before merge
- **Push**: Scan on every push to any branch

**Where can this GitHub App be installed?**

- Select your preference (personal account, organization, or both)

4. Click **"Create GitHub App"**

### Step 2: Generate Private Key

1. Scroll down to **"Private keys"** section
2. Click **"Generate a private key"**
3. A `.pem` file will be downloaded automatically
4. Keep this file secure (it's like a password)

### Step 3: Note Your App Credentials

You'll need these values to configure Octovy:

| Value | Where to find |
|-------|---------------|
| **App ID** | Top of the app's settings page, or visible on GitHub Apps page |
| **Private Key** | The `.pem` file you downloaded |
| **Webhook Secret** | The secret you configured in Step 1 |

### Step 4: Set Environment Variables

Configure these on your Octovy server:

```bash
# Required for GitHub App
export OCTOVY_GITHUB_APP_ID=123456
export OCTOVY_GITHUB_APP_PRIVATE_KEY=/path/to/private-key.pem
export OCTOVY_GITHUB_APP_SECRET=your-webhook-secret

# Also configure BigQuery (required)
export OCTOVY_BIGQUERY_PROJECT_ID=your-project-id

# Optional
export OCTOVY_ADDR=:8080
```

**Alternative**: Set `OCTOVY_GITHUB_APP_PRIVATE_KEY` to the PEM content directly (useful in Docker):

```bash
export OCTOVY_GITHUB_APP_PRIVATE_KEY='-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA...
...
-----END RSA PRIVATE KEY-----'
```

### Step 5: Install the GitHub App

1. Go to the app's page (visible in [GitHub Apps settings](https://github.com/settings/apps))
2. Click **"Install App"**
3. Select which account to install to (personal or organization)
4. Select which repositories to grant access to:
   - **All repositories**: Octovy can scan all repos
   - **Only selected repositories**: Choose specific repos
5. Click **"Install"**

### Step 6: Start the Octovy Server

Once environment variables are set and the app is installed:

```bash
octovy serve --addr :8080
```

Or with Docker:

```bash
docker run -p 8080:8080 \
  -v /path/to/private-key.pem:/key.pem \
  -v $(which trivy):/trivy \
  -e OCTOVY_ADDR=:8080 \
  -e OCTOVY_GITHUB_APP_ID=123456 \
  -e OCTOVY_GITHUB_APP_PRIVATE_KEY=/key.pem \
  -e OCTOVY_GITHUB_APP_SECRET=your-webhook-secret \
  -e OCTOVY_BIGQUERY_PROJECT_ID=my-project \
  -e OCTOVY_TRIVY_PATH=/trivy \
  ghcr.io/secmon-lab/octovy
```

## Environment Variables Reference

| Variable | Required | Description |
|----------|----------|-------------|
| `OCTOVY_GITHUB_APP_ID` | ✓ | GitHub App ID |
| `OCTOVY_GITHUB_APP_PRIVATE_KEY` | ✓ | Path to private key file or PEM content |
| `OCTOVY_GITHUB_APP_SECRET` | ✓ | Webhook secret for verification |

## Webhook Events

Octovy responds to these GitHub webhook events:

### `push` event
- Triggered on every push to any branch
- Octovy scans the pushed commit
- Results stored in BigQuery

### `pull_request` event
- Triggered on PR open, synchronize (new commits), and reopen
- Octovy scans the PR's head commit
- Results stored in BigQuery

## Security Considerations

### Private Key Security

- **Never commit** the `.pem` file to version control
- Use environment variables or secret management (e.g., AWS Secrets Manager, HashiCorp Vault)
- For Docker, use Docker secrets or environment variable injection
- Rotate keys periodically

### Webhook Secret

- Octovy verifies webhook requests using the secret
- GitHub includes the secret in the `X-Hub-Signature-256` header
- Mismatched secret = webhook rejected

### Scope Limitation

- Only grant **Read-only** access to Contents and Metadata
- Octovy doesn't modify repositories
- Minimal permissions reduce security risk

## Troubleshooting

### Webhook not received

- Verify webhook URL is publicly accessible: `curl https://your-domain.com/health`
- Check firewall/security groups allow HTTPS (port 443)
- Verify webhook URL in GitHub App settings is correct

### "Permission denied" on repository access

- GitHub App must be installed on the repository or organization
- Check that Contents and Metadata permissions are set to Read-only
- Repository must grant access to Octovy (check app's access list)

### Signature verification failed

- Verify `OCTOVY_GITHUB_APP_SECRET` matches the secret in GitHub App settings
- Check for leading/trailing whitespace in environment variable
- Regenerate secret if unsure

### Private key errors

- Verify the `.pem` file is readable by the Octovy process
- Check file permissions: `chmod 600 private-key.pem`
- For Docker, verify path is correctly mounted

### No scans happening

- Check Octovy server is running: `curl localhost:8080/health`
- Check server logs for webhook processing
- Verify GitHub App is installed on the repository
- Make a test push to trigger a scan

## Testing the Setup

### Manual webhook test

Use GitHub's webhook delivery logs:

1. Go to GitHub App settings
2. Click **"Advanced"**
3. Scroll to **"Recent Deliveries"**
4. View webhook events and responses
5. Click **"Redeliver"** to test

### Check server endpoints

```bash
# Health check (should return 200)
curl -i https://your-domain.com/health

# Webhook endpoint (will return 401 if accessed directly)
curl -i -X POST https://your-domain.com/webhook/github/app
```

## Next Steps

- [Start the serve command](../commands/serve.md)
- [Configure BigQuery](./bigquery.md)
- [Configure Firestore](./firestore.md) (optional)
