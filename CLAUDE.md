# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Octovy is a GitHub App that scans repository code for vulnerable dependencies using Trivy. It responds to GitHub webhook events (`push`, `pull_request`) to scan repositories, comment on PRs with vulnerability findings, and optionally store results in BigQuery.

## Architecture

The codebase follows clean architecture with clear separation of concerns:

### Layer Structure
- **Controller** ([pkg/controller/](pkg/controller/)): CLI commands and HTTP server handling
  - `cli/`: CLI configuration and command setup using urfave/cli
  - `server/`: HTTP server with chi router, handles GitHub webhook events at `/webhook/github/app` and `/webhook/github/action`
- **UseCase** ([pkg/usecase/](pkg/usecase/)): Business logic orchestration
  - `ScanGitHubRepo`: Main workflow that downloads repo, scans with Trivy, stores results, and comments on PRs
  - `InsertScanResult`: Exports scan results to BigQuery and Cloud Storage
  - `CommentGitHubPR`: Compares current vs base branch vulnerabilities and posts PR comments
- **Domain** ([pkg/domain/](pkg/domain/)): Core business models and interfaces
  - `interfaces/`: Defines contracts for infrastructure (GitHub, BigQuery, Storage, Policy)
  - `model/`: Business entities (GitHub metadata, Trivy reports, Config with CUE-based ignore lists)
  - `logic/`: Vulnerability filtering and diff logic
  - `types/`: Domain-specific types and constants
- **Infra** ([pkg/infra/](pkg/infra/)): External service implementations
  - `gh/`: GitHub API client using bradleyfalzon/ghinstallation for GitHub App authentication
  - `bq/`: BigQuery client for storing scan results
  - `cs/`: Cloud Storage client for archiving results
  - `trivy/`: Wrapper for Trivy CLI execution
  - `clients.go`: Central dependency injection container

### Key Workflows
1. **GitHub webhook received** → Server validates webhook → UseCase orchestrates scan
2. **Scan workflow**: Download repo archive → Extract to temp dir → Load `.octovy/*.cue` config → Run Trivy → Parse JSON results
3. **PR commenting**: Fetch base branch scan from Storage → Diff vulnerabilities → Post comment only for new findings
4. **Result storage**: Insert to BigQuery with auto-schema updates → Store raw JSON in Cloud Storage

### Configuration System
Uses CUE language for ignore lists. Users place `.cue` files in `.octovy/` directory at repo root. Schema at [pkg/domain/model/schema/ignore.cue](pkg/domain/model/schema/ignore.cue) defines `IgnoreList` with vulnerability IDs, expiration dates (max 90 days), and comments.

### Dependency Injection
The `infra.Clients` struct aggregates all infrastructure dependencies. Tests use mocks generated via `moq` (see Makefile). Interface definitions in [pkg/domain/interfaces/](pkg/domain/interfaces/) enable clean testing boundaries.

## Development Commands

### Testing
```bash
# Run all tests (requires postgres service)
go test --tags github ./...

# Run tests for specific package
go test --tags github ./pkg/usecase

# Run specific test
go test --tags github ./pkg/usecase -run TestScanGitHubRepo

# Vet code
go vet --tags github ./...
```

Note: Tests require PostgreSQL running (see [.github/workflows/test.yml](.github/workflows/test.yml) for service config). Set `TEST_DB_DSN` environment variable.

### Mock Generation
```bash
# Regenerate all mocks (uses moq)
make

# Generate specific mock
make pkg/domain/mock/infra.go
```

### Building
```bash
# Build binary
go build -o octovy .

# Run locally
./octovy serve --addr :8080 [other flags]
```

### Running the Server
```bash
# Via CLI
octovy serve --addr :8080

# Using Docker (production)
docker run -p 8080:8080 -v /path/to/private-key.pem:/key.pem \
  -e OCTOVY_ADDR=:8080 \
  -e OCTOVY_GITHUB_APP_ID=123456 \
  -e OCTOVY_GITHUB_APP_PRIVATE_KEY=/key.pem \
  -e OCTOVY_GITHUB_APP_SECRET=mysecret \
  -e OCTOVY_CLOUD_STORAGE_BUCKET=my-bucket \
  ghcr.io/m-mizutani/octovy
```

## Code Patterns

### Error Handling
Uses `github.com/m-mizutani/goerr/v2` for wrapped errors with context. Patterns:
```go
// Create error with context
return goerr.New("validation failed", goerr.V("user_id", userID))

// Wrap error with context
return goerr.Wrap(err, "operation failed", goerr.V("key", value))

// Multiple context values
return goerr.Wrap(err, "failed to process",
    goerr.V("file", filename),
    goerr.V("line", lineNum),
)
```

Note: goerr v2 requires message as second argument to `Wrap()`. Context values are added as variadic `goerr.V()` or `goerr.Value()` arguments, not via `.With()` method chains.

### Logging
Structured logging via `utils.Logger()` (log/slog) and `utils.CtxLogger(ctx)` for context-aware logging. Supports both text and JSON formats via `OCTOVY_LOG_FORMAT` environment variable.

### Testing
- Uses `github.com/m-mizutani/gt` test framework for assertions
- Common patterns: `gt.V(t, actual).Equal(expected)`, `gt.NoError(t, err)`, `gt.R1(fn()).NoError(t)`
- Mock interfaces generated via `moq` in [pkg/domain/mock/](pkg/domain/mock/)
- Test helpers in [pkg/utils/test.go](pkg/utils/test.go)
- Use `--tags github` build tag for tests requiring GitHub integration

## Important Implementation Details

- **Trivy integration**: Executes trivy CLI as subprocess with JSON output. Default path is `trivy`, configurable via `OCTOVY_TRIVY_PATH`.
- **GitHub App authentication**: Uses installation tokens via `ghinstallation` library. Private key can be file path or PEM content.
- **Temporary file handling**: Repos downloaded to temp dirs with `octovy.<owner>.<repo>.<commit>.*` pattern. Always cleaned up with deferred `utils.SafeRemoveAll()`.
- **Check Runs**: Creates GitHub Check Run at scan start, updates to "completed" with conclusion on finish (success/cancelled).
- **PR comments**: Only comments when new vulnerabilities detected (diff against base branch). Can minimize old comments and optionally disable "no detection" comments via `WithDisableNoDetectionComment()`.

## Environment Variables

Required for server operation (see README.md for full list):
- `OCTOVY_ADDR`: Server bind address
- `OCTOVY_GITHUB_APP_ID`, `OCTOVY_GITHUB_APP_PRIVATE_KEY`, `OCTOVY_GITHUB_APP_SECRET`: GitHub App credentials
- `OCTOVY_CLOUD_STORAGE_BUCKET`: GCS bucket for result storage

Optional BigQuery config: `OCTOVY_BIGQUERY_PROJECT_ID`, `OCTOVY_BIGQUERY_DATASET_ID`, `OCTOVY_BIGQUERY_TABLE_ID`
