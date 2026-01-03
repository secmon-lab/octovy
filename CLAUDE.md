# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Octovy is a GitHub App that scans repository code for vulnerable dependencies using Trivy. It responds to GitHub webhook events (`push`, `pull_request`) to scan repositories and store results in BigQuery. It also provides a CLI command to scan local directories.

## Architecture

The codebase follows clean architecture with clear separation of concerns:

### Layer Structure
- **CLI** ([pkg/cli/](pkg/cli/)): CLI commands using urfave/cli
  - `scan`: Scans local directory with Trivy and inserts results to BigQuery
  - `serve`: Starts HTTP server for GitHub webhook handling
- **Controller** ([pkg/controller/](pkg/controller/)): HTTP server handling
  - `server/`: HTTP server with chi router, handles GitHub webhook events at `/webhook/github/app` and `/webhook/github/action`
- **UseCase** ([pkg/usecase/](pkg/usecase/)): Business logic orchestration
  - `ScanGitHubRepo`: Main workflow that downloads repo, scans with Trivy, and stores results
  - `InsertScanResult`: Exports scan results to BigQuery
- **Domain** ([pkg/domain/](pkg/domain/)): Core business models and interfaces
  - `interfaces/`: Defines contracts for infrastructure (GitHub, BigQuery, Trivy)
  - `model/`: Business entities (GitHub metadata, Trivy reports, Config with CUE-based ignore lists)
  - `types/`: Domain-specific types and constants
- **Infra** ([pkg/infra/](pkg/infra/)): External service implementations
  - `ghapp/`: GitHub API client using bradleyfalzon/ghinstallation for GitHub App authentication
  - `bq/`: BigQuery client for storing scan results
  - `trivy/`: Wrapper for Trivy CLI execution
  - `clients.go`: Central dependency injection container
- **Utils** ([pkg/utils/](pkg/utils/)): Shared utilities
  - `logging/`: Structured logging with slog
  - `safe/`: Safe I/O operations
  - `testutil/`: Test helpers and utilities

### Key Workflows
1. **GitHub webhook received** → Server validates webhook → UseCase orchestrates scan
2. **Scan workflow**: Download repo archive → Extract to temp dir → Run Trivy → Parse JSON results
3. **Result storage**: Insert to BigQuery with auto-schema updates
4. **Local scan workflow**: Run `scan` command → Auto-detect git metadata → Run Trivy on local directory → Insert results to BigQuery

### Configuration System
Uses CUE language for ignore lists. Users place `.cue` files in `.octovy/` directory at repo root. Schema at [pkg/domain/model/schema/ignore.cue](pkg/domain/model/schema/ignore.cue) defines `IgnoreList` with vulnerability IDs, expiration dates (max 90 days), and comments.

### Dependency Injection
The `infra.Clients` struct aggregates all infrastructure dependencies. Tests use mocks generated via `moq` (see Taskfile.yaml). Interface definitions in [pkg/domain/interfaces/](pkg/domain/interfaces/) enable clean testing boundaries.

## Development Commands

**IMPORTANT**: Do NOT run `go build` commands. Build verification is not permitted.

### Testing
```bash
# Run all tests
go test ./...

# Run tests for specific package
go test ./pkg/usecase

# Run specific test
go test ./pkg/usecase -run TestScanGitHubRepo

# Vet code
go vet ./...
```

### Mock Generation
```bash
# Regenerate all mocks (uses moq via go generate)
task gen

# Alternative: run go generate directly
go generate ./...
```

### Building
```bash
# Build binary
go build -o octovy .

# Run local scan
./octovy scan --dir /path/to/repo

# Run server
./octovy serve --addr :8080
```

### Running Commands
```bash
# Scan local directory
octovy scan --dir . --github-owner myorg --github-repo myrepo

# Start webhook server
octovy serve --addr :8080

# Using Docker (production)
docker run -p 8080:8080 -v /path/to/private-key.pem:/key.pem \
  -e OCTOVY_ADDR=:8080 \
  -e OCTOVY_GITHUB_APP_ID=123456 \
  -e OCTOVY_GITHUB_APP_PRIVATE_KEY=/key.pem \
  -e OCTOVY_GITHUB_APP_SECRET=mysecret \
  -e OCTOVY_BIGQUERY_PROJECT_ID=my-project \
  -e OCTOVY_BIGQUERY_DATASET_ID=my-dataset \
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
Structured logging via `logging.Default()` and `logging.FromContext(ctx)` from [pkg/utils/logging/](pkg/utils/logging/) (using log/slog). Supports both text and JSON formats via `OCTOVY_LOG_FORMAT` environment variable.

### Testing
- Uses `github.com/m-mizutani/gt` test framework for assertions
- Common patterns: `gt.V(t, actual).Equal(expected)`, `gt.NoError(t, err)`, `gt.R1(fn()).NoError(t)`
- Mock interfaces generated via `moq` (github.com/matryer/moq) in [pkg/domain/mock/](pkg/domain/mock/)
- Test helpers in [pkg/utils/testutil/](pkg/utils/testutil/)

**IMPORTANT - Test Coverage Requirements**:
- **Every Go source file MUST have a corresponding test file**: If `xxx.go` exists, `xxx_test.go` MUST exist
  - **Exception**: Pure data model files (structs with no logic) do NOT require test files
  - Model files are in `pkg/domain/model/` and contain only struct definitions with JSON tags
  - If a file contains any methods or functions with logic, it MUST have tests
- **Unit tests are mandatory**: Each function and method requires unit tests covering normal cases, edge cases, and error scenarios
- **Integration tests are required**: End-to-end workflows must have integration tests validating the complete flow
- **Test-Driven Development**: When adding new features or fixing bugs, write tests first before implementation
- Do not merge code without proper test coverage

**CRITICAL - Test Quality Standards**:
- **PROHIBITED test patterns** - These are NOT acceptable tests:
  - Checking only `!= nil` or `== nil` without verifying actual values
  - Checking only the count/length of items without verifying content
  - Tests that only verify no error occurred without checking actual behavior
  - Tests without meaningful assertions about the actual output
- **REQUIRED test patterns**:
  - Verify actual values match expected values (use `gt.V(t, actual).Equal(expected)`)
  - Check specific fields and their content, not just presence
  - Validate complete behavior including side effects
  - Test error messages and error types, not just error presence
  - For slices/arrays: verify both length AND actual content of each item
  - For structs: verify specific field values, not just non-nil
- **Example violations**:
  ```go
  // BAD - Only checks not nil
  gt.V(t, result).NotEqual(nil)

  // BAD - Only checks count
  gt.V(t, len(items)).Equal(3)

  // BAD - Only checks no error
  gt.NoError(t, err)  // without checking actual result

  // GOOD - Checks actual values
  gt.V(t, result.Name).Equal("expected-name")
  gt.V(t, result.Status).Equal(StatusActive)

  // GOOD - Checks both count and content
  gt.V(t, len(items)).Equal(3)
  gt.V(t, items[0].ID).Equal("item-1")
  gt.V(t, items[1].Value).Equal(42)
  ```

## Important Implementation Details

- **Trivy integration**: Executes trivy CLI as subprocess with JSON output. Default path is `trivy`, configurable via `OCTOVY_TRIVY_PATH`.
- **GitHub App authentication**: Uses installation tokens via `ghinstallation` library. Private key can be file path or PEM content.
- **Temporary file handling**: Repos downloaded to temp dirs with `octovy.<owner>.<repo>.<commit>.*` pattern. Always cleaned up with deferred `safe.RemoveAll()` from [pkg/utils/safe/](pkg/utils/safe/).
- **Auto-detection**: The `scan` command automatically detects GitHub metadata (owner, repo, commit ID) from git commands if not explicitly provided.

## Environment Variables

Required for server operation:
- `OCTOVY_ADDR`: Server bind address (e.g., `:8080`)
- `OCTOVY_GITHUB_APP_ID`: GitHub App ID
- `OCTOVY_GITHUB_APP_PRIVATE_KEY`: Path to GitHub App private key file
- `OCTOVY_GITHUB_APP_SECRET`: Webhook secret for verifying GitHub requests

Optional configuration:
- `OCTOVY_TRIVY_PATH`: Path to trivy binary (default: `trivy`)
- `OCTOVY_BIGQUERY_PROJECT_ID`: BigQuery project ID for storing scan results
- `OCTOVY_BIGQUERY_DATASET_ID`: BigQuery dataset ID (default: `octovy`)
- `OCTOVY_BIGQUERY_TABLE_ID`: BigQuery table ID (default: `scans`)
- `OCTOVY_BIGQUERY_IMPERSONATE_SERVICE_ACCOUNT`: Service account to impersonate for BigQuery access
- `OCTOVY_LOG_FORMAT`: Log format (`text` or `json`, default: `text`)
- `OCTOVY_SENTRY_DSN`: Sentry DSN for error tracking
- `OCTOVY_SENTRY_ENV`: Sentry environment name
