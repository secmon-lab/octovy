# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Octovy is a GitHub App that scans repository code for vulnerable dependencies using Trivy. It responds to GitHub webhook events (`push`, `pull_request`) to scan repositories and store results in BigQuery. It also provides a CLI command to scan local directories.

## Architecture

The codebase follows clean architecture with clear separation of concerns.

### Key Dependencies
- **CLI Framework**: urfave/cli/v3 for command-line interface
- **HTTP Router**: go-chi/chi/v5 for HTTP routing
- **Testing**: m-mizutani/gt for assertions, matryer/moq for mock generation
- **Logging**: log/slog with m-mizutani/clog (colored output), m-mizutani/masq (secret masking)
- **Error Handling**: m-mizutani/goerr/v2 with context values
- **GitHub Integration**: bradleyfalzon/ghinstallation/v2 for GitHub App authentication
- **BigQuery**: cloud.google.com/go/bigquery for scan result storage
- **Firestore**: cloud.google.com/go/firestore for repository metadata and vulnerability tracking (optional)
- **Trivy**: External CLI tool (aquasecurity/trivy) executed as subprocess

### Layer Structure
- **CLI** ([pkg/cli/](pkg/cli/)): CLI commands using urfave/cli/v3
  - `scan`: Scans local directory with Trivy and inserts results to BigQuery
  - `insert`: Inserts existing Trivy scan result JSON to BigQuery (and optionally Firestore)
  - `serve`: Starts HTTP server for GitHub webhook handling with graceful shutdown
  - Configuration via [pkg/cli/config/](pkg/cli/config/): GitHubApp, BigQuery, Sentry configs with flag/env binding
- **Controller** ([pkg/controller/](pkg/controller/)): HTTP server handling
  - `server/`: HTTP server with chi/v5 router, handles GitHub webhook events at `/webhook/github/app` and `/webhook/github/action`
  - Health check endpoint at `/health`
  - Graceful shutdown with configurable timeouts
- **UseCase** ([pkg/usecase/](pkg/usecase/)): Business logic orchestration
  - `ScanGitHubRepo`: Downloads GitHub repo archive, extracts to temp dir, scans with Trivy, inserts to BigQuery
  - `ScanAndInsert`: Scans local directory with Trivy and inserts results to BigQuery (used by both CLI and webhook)
  - `InsertScanResult`: Exports scan results to BigQuery with metadata
- **Domain** ([pkg/domain/](pkg/domain/)): Core business models and interfaces
  - `interfaces/`: Defines contracts for infrastructure (GitHub, BigQuery) and UseCase
  - `model/`: Business entities (GitHub metadata, Trivy reports, Scan records for BigQuery)
  - `types/`: Domain-specific types (GitHub App credentials with slog.LogValuer for safe logging)
  - `mock/`: Generated mock implementations for testing
- **Infra** ([pkg/infra/](pkg/infra/)): External service implementations
  - `ghapp/`: GitHub API client using bradleyfalzon/ghinstallation for GitHub App authentication
  - `bq/`: BigQuery client for storing scan results
  - `trivy/`: Wrapper for Trivy CLI execution
  - `clients.go`: Central dependency injection container
- **Repository** ([pkg/repository/](pkg/repository/)): Data persistence layer
  - `memory/`: In-memory implementation for testing and development
  - `firestore/`: Firestore implementation for production use (optional)
  - `testhelper/`: Shared test functions ensuring identical behavior across implementations
  - Manages scan results, repository metadata, branch information, targets, and vulnerabilities
  - Firestore collection structure: `repo/{owner:repo}/branch/{name}/target/{id}/vulnerability/{id}`
  - Document ID format: Uses `:` separator (e.g., `owner:repo`) since GitHub names cannot contain colons
- **Utils** ([pkg/utils/](pkg/utils/)): Shared utilities
  - `logging/`: Structured logging with slog/clog, secret masking, context support
  - `safe/`: Safe I/O operations (Close, Remove, RemoveAll)
  - `errutil/`: Error handling utilities (HandleError for Sentry integration)
  - `testutil/`: Test helpers for environment variable management

### Key Workflows
1. **GitHub webhook received** → Server validates webhook → UseCase orchestrates scan
2. **GitHub repo scan workflow** (`ScanGitHubRepo`): Download repo archive via GitHub API → Extract to temp dir → Scan with Trivy → Insert to BigQuery → Cleanup temp files
3. **Local scan workflow** (`ScanAndInsert`): Scan directory with Trivy → Parse JSON results → Insert to BigQuery
4. **CLI scan command**: Auto-detect git metadata (owner/repo/commit from git commands) → Create clients → Call `ScanAndInsert` usecase

### Configuration System
Configuration is handled through CLI flags and environment variables. See [pkg/cli/config/](pkg/cli/config/) for configuration structures including GitHub App, BigQuery, Firestore, and Sentry settings. Firestore is completely optional - the application works normally with BigQuery-only storage when Firestore is not configured.

### Dependency Injection
The `infra.Clients` struct aggregates all infrastructure dependencies (GitHubApp, BigQuery, Trivy, HTTPClient). Created via `infra.New()` with functional options pattern (`WithGitHubApp`, `WithBigQuery`, `WithTrivy`, `WithHTTPClient`). Tests use mocks generated via `moq` in [pkg/domain/mock/](pkg/domain/mock/). Interface definitions in [pkg/domain/interfaces/](pkg/domain/interfaces/) enable clean testing boundaries.

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
**NOTE**: Building is mentioned here for reference only. Per project policy, do NOT run `go build` commands during development.

```bash
# Build binary (for reference only - do not run)
go build -o octovy .
```

### Running Commands
```bash
# Scan local directory (auto-detects GitHub metadata from git)
octovy scan

# Scan specific directory
octovy scan --dir /path/to/repo

# Scan with explicit GitHub metadata
octovy scan --github-owner myorg --github-repo myrepo --github-commit-id abc123

# Scan with BigQuery configuration
octovy scan --bigquery-project-id my-project --bigquery-dataset-id my-dataset

# Insert existing Trivy scan result to BigQuery (auto-detects GitHub metadata)
octovy insert -f scan-result.json

# Insert with explicit GitHub metadata
octovy insert -f scan-result.json --github-owner myorg --github-repo myrepo --github-commit-id abc123

# Insert with BigQuery and Firestore configuration
octovy insert -f scan-result.json --bigquery-project-id my-project --firestore-project-id my-project

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

### CRITICAL - No TODO or Future Comments
**NEVER leave TODO, FIXME, XXX, HACK, or "in future" comments in code unless explicitly instructed by the user.**

Rules:
- **Implement features completely** - Do not add placeholder comments for future work
- **If a feature cannot be implemented now**, ask the user for clarification instead of leaving a TODO
- **No "Note: will be used in future"** comments - Either implement it now or don't add it at all
- **No deferring implementation** with comments like "this will be added later"
- **Complete all integration work** - Don't create infrastructure that isn't used immediately

Examples of PROHIBITED patterns:
```go
// BAD - TODO comment without user instruction
// TODO: Add validation here

// BAD - Future placeholder
// Note: This will be used in future when X supports Y

// BAD - Deferred implementation
// This feature will be implemented later

// BAD - Unused variable with future comment
firestoreRepo := createRepo()
// Note: Firestore repository is created but not yet integrated
_ = firestoreRepo
```

GOOD pattern - Complete implementation:
```go
// GOOD - Feature is fully implemented
if firestoreConfig.Enabled() {
    repo := createRepo()
    defer repo.Close()
    clients = append(clients, infra.WithScanRepository(repo))
}
```

### Struct Tags
**CRITICAL**: Minimize the use of struct tags. Only add tags when there is an explicit serialization use case.

Rules:
- **DO NOT use `firestore` tags** - Firestore SDK handles field mapping automatically
- **Only use `json` tags when explicitly converting to/from JSON** (e.g., API responses, external file formats)
- **Domain models that are only used internally should have NO tags**
- **BigQuery models can use `bigquery` tags for schema mapping**

Examples:
```go
// CORRECT - Internal domain model (no tags needed)
type Repository struct {
    ID              types.GitHubRepoID
    Owner           string
    DefaultBranch   types.BranchName
}

// CORRECT - API response model (json tags for explicit JSON conversion)
type APIResponse struct {
    Status string `json:"status"`
    Data   []Item `json:"data"`
}

// CORRECT - BigQuery model (bigquery tags for schema mapping)
type ScanRecord struct {
    ID        string `bigquery:"id"`
    Timestamp int64  `bigquery:"timestamp"`
}

// INCORRECT - Unnecessary tags
type Model struct {
    ID   string `json:"id"`
    Name string `json:"name"`
}

// INCORRECT - Firestore tags (prohibited)
type Model struct {
    ID   string `firestore:"id"`
    Name string `firestore:"name"`
}
```

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
Structured logging via `logging.Default()` and `logging.From(ctx)` from [pkg/utils/logging/](pkg/utils/logging/) (using log/slog with clog for colored text output). Supports both text and JSON formats. Configure with `logging.Configure(format, level, output)` function. Secrets are automatically masked using masq library.

### Testing - Mock Usage Policy
**CRITICAL**: Do NOT create and use mock repositories (e.g. `mock.ScanRepositoryMock`) in tests. Use actual implementations instead.

Rules:
- **ALWAYS use `memory.New()`** for repository testing instead of mocks
- **Mock only external services** that cannot be run locally (BigQuery, external APIs)
- **Memory implementation provides real behavior** - use it to verify actual data persistence and retrieval
- Mocks hide bugs and don't test real integration - avoid them for internal components

Example:
```go
// BAD - Using mock repository
mockRepo := &mock.ScanRepositoryMock{}
mockRepo.ListVulnerabilitiesFunc = func(...) {...}

// GOOD - Using memory repository
memRepo := memory.New()
// Test against actual implementation
vulns, err := memRepo.ListVulnerabilities(ctx, repoID, branchName, targetID)
```

### Testing
- Uses `github.com/m-mizutani/gt` test framework for assertions
- Common patterns: `gt.V(t, actual).Equal(expected)`, `gt.NoError(t, err)`, `gt.R1(fn()).NoError(t)`
- Mock interfaces generated via `moq` (github.com/matryer/moq) in [pkg/domain/mock/](pkg/domain/mock/)
- Test helpers in [pkg/utils/testutil/](pkg/utils/testutil/)

**IMPORTANT - Test Coverage Requirements**:
- **Every Go source file MUST have a corresponding test file**: If `xxx.go` exists, `xxx_test.go` MUST exist
  - **Exception**: Pure data model files (structs with no logic) do NOT require test files
  - Model files are in `pkg/domain/model/` and contain only struct definitions with `json`/`bigquery` tags
  - If a file contains any methods or functions with logic, it MUST have tests
- **Unit tests are mandatory**: Each function and method requires unit tests covering normal cases, edge cases, and error scenarios
- **Integration tests are required**: End-to-end workflows must have integration tests validating the complete flow
- **Test-Driven Development**: When adding new features or fixing bugs, write tests first before implementation
- Do not merge code without proper test coverage
- **Mock generation**: Use `task gen` or `go generate ./...` to regenerate mocks after interface changes

**Firestore Testing Pattern**:
When implementing repository layers with both Memory and Firestore implementations:

1. **Common Test Helper Approach**: Memory and Firestore implementations MUST use the same test cases
   - Create test helper functions in `pkg/repository/testhelper/` that accept the interface
   - Both implementations call the same test helper functions
   - This ensures identical behavior between Memory and Firestore implementations

2. **Environment-Based Firestore Testing**:
   - Firestore tests connect to real Firestore ONLY when `TEST_FIRESTORE_PROJECT_ID` and `TEST_FIRESTORE_DATABASE_ID` are set
   - If these environment variables are not set, skip Firestore tests with `t.Skip()`
   - This allows local development without Firestore while ensuring CI tests against real Firestore

3. **CRITICAL - Test ID Randomization**:
   - **ALWAYS randomize test IDs using UUID** to ensure test isolation and prevent conflicts
   - Generate unique IDs at the start of each test function using `uuid.New()`
   - Use short UUIDs (first 8 characters) for readability: `uuid.New().String()[:8]`
   - Apply to ALL entity IDs: repository IDs, branch names, target IDs, etc.
   - This prevents test failures when tests run in parallel or when data persists between runs
   - Example:
     ```go
     import "github.com/google/uuid"

     func TestRepositoryCRUD(t *testing.T, repo interfaces.ScanRepository) {
         // Generate unique IDs for this test run
         owner := fmt.Sprintf("owner-%s", uuid.New().String()[:8])
         repoName := fmt.Sprintf("repo-%s", uuid.New().String()[:8])
         repoID := types.GitHubRepoID(fmt.Sprintf("%s/%s", owner, repoName))

         // Use these randomized IDs throughout the test
         testRepo := &model.Repository{
             ID:    repoID,
             Owner: owner,
             Name:  repoName,
             // ...
         }
     }
     ```

Example pattern:
```go
// pkg/repository/testhelper/scan_repository_test.go
func TestRepositoryCRUD(t *testing.T, repo interfaces.ScanRepository) {
    // Common test logic for both Memory and Firestore
}

// pkg/repository/memory/scan_test.go
func TestMemoryScanRepository(t *testing.T) {
    repo := New()
    testhelper.TestRepositoryCRUD(t, repo)
}

// pkg/repository/firestore/scan_test.go
func TestFirestoreScanRepository(t *testing.T) {
    projectID := os.Getenv("TEST_FIRESTORE_PROJECT_ID")
    databaseID := os.Getenv("TEST_FIRESTORE_DATABASE_ID")

    if projectID == "" || databaseID == "" {
        t.Skip("Firestore credentials not configured")
    }

    repo, err := New(context.Background(), projectID, databaseID)
    gt.NoError(t, err)
    defer repo.Close()

    // Same tests as Memory implementation
    testhelper.TestRepositoryCRUD(t, repo)
}
```

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

- **Trivy integration**: Executes trivy CLI as subprocess (`pkg/infra/trivy/`) with JSON output. Default path is `trivy`, configurable via `OCTOVY_TRIVY_PATH`. Runs with flags: `fs --exit-code 0 --no-progress --format json --output <file> --list-all-pkgs <dir>`.
- **GitHub App authentication**: Uses installation tokens via `ghinstallation` library (`pkg/infra/ghapp/`). Private key can be file path or PEM content.
- **Temporary file handling**:
  - Repo archives downloaded to temp files: `octovy_code.<owner>.<repo>.<commit>.*.zip`
  - Extracted to temp dirs: `octovy.<owner>.<repo>.<commit>.*`
  - Scan results saved to: `octovy_result.*.json`
  - Always cleaned up with deferred `safe.Remove()` and `safe.RemoveAll()` from [pkg/utils/safe/](pkg/utils/safe/)
- **Auto-detection**: The `scan` command automatically detects GitHub metadata from git commands:
  - Commit ID: `git rev-parse HEAD`
  - Owner/Repo: Parsed from `git remote get-url origin` (supports both SSH and HTTPS formats)
- **BigQuery schema**: See `model.Scan` struct in [pkg/domain/model/result.go](pkg/domain/model/result.go). Uses `bigquery` and `json` tags for schema mapping.
- **Error handling**: All errors use `goerr.Wrap()` with context values. Sentry integration available via `errutil.HandleError()`.
- **Secret masking**: GitHub App credentials (Secret, PrivateKey) implement `slog.LogValuer` interface in [pkg/domain/types/github.go](pkg/domain/types/github.go) to prevent logging sensitive data.

## Environment Variables

Required for server operation:
- `OCTOVY_ADDR`: Server bind address (e.g., `:8080`)
- `OCTOVY_GITHUB_APP_ID`: GitHub App ID
- `OCTOVY_GITHUB_APP_PRIVATE_KEY`: Path to GitHub App private key file
- `OCTOVY_GITHUB_APP_SECRET`: Webhook secret for verifying GitHub requests

Optional configuration:
- `OCTOVY_RESULT_FILE`: Path to Trivy scan result JSON file (used by `insert` command)
- `OCTOVY_TRIVY_PATH`: Path to trivy binary (default: `trivy`)
- `OCTOVY_BIGQUERY_PROJECT_ID`: BigQuery project ID for storing scan results
- `OCTOVY_BIGQUERY_DATASET_ID`: BigQuery dataset ID (default: `octovy`)
- `OCTOVY_BIGQUERY_TABLE_ID`: BigQuery table ID (default: `scans`)
- `OCTOVY_BIGQUERY_IMPERSONATE_SERVICE_ACCOUNT`: Service account to impersonate for BigQuery access
- `OCTOVY_FIRESTORE_PROJECT_ID`: Firestore project ID for repository metadata (optional)
- `OCTOVY_FIRESTORE_DATABASE_ID`: Firestore database ID (optional, default: `(default)`)
- `OCTOVY_LOG_FORMAT`: Log format (`text` or `json`, default: `text`)
- `OCTOVY_SENTRY_DSN`: Sentry DSN for error tracking
- `OCTOVY_SENTRY_ENV`: Sentry environment name

Testing configuration:
- `TEST_FIRESTORE_PROJECT_ID`: Firestore project ID for integration tests
- `TEST_FIRESTORE_DATABASE_ID`: Firestore database ID for integration tests
