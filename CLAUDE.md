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

### Syntax Check (IMPORTANT)
**CRITICAL**: During development, use `go vet` for syntax checking. Do NOT use `go build` until all implementation is complete.

```bash
# Syntax check (USE THIS during development)
go vet ./...

# Build (ONLY when all implementation is complete and ready for final verification)
go build ./...
```

### Testing
```bash
# Run all tests
go test ./...

# Run tests for specific package
go test ./pkg/usecase

# Run specific test
go test ./pkg/usecase -run TestScanGitHubRepo
```

### Mock Generation (IMPORTANT)
**CRITICAL**: Always use `task gen` for mock generation. Do NOT use `go generate` directly.

```bash
# Regenerate all mocks (ALWAYS use this)
task gen
```

## Coding Rules

Detailed coding rules are organized in [.claude/rules/](.claude/rules/):

### Backend Rules (`backend/`)
Go-specific rules for backend development (applies to `pkg/**/*.go`):

- **[coding-standards.md](.claude/rules/backend/coding-standards.md)**: No TODO/future comments policy
- **[error-handling.md](.claude/rules/backend/error-handling.md)**: goerr/v2 patterns and Sentry integration
- **[logging.md](.claude/rules/backend/logging.md)**: Structured logging with slog/clog/masq
- **[testing.md](.claude/rules/backend/testing.md)**: Test coverage requirements, mock usage policy, Firestore testing patterns
- **[repository-pattern.md](.claude/rules/backend/repository-pattern.md)**: Memory/Firestore dual implementation strategy
- **[infra-integration.md](.claude/rules/backend/infra-integration.md)**: Trivy, GitHub App, BigQuery integration details
- **[temporary-files.md](.claude/rules/backend/temporary-files.md)**: Safe file cleanup patterns

### Domain Rules (`domain/`)
Domain model specific rules:

- **[models.md](.claude/rules/domain/models.md)**: Struct tag policy for domain models (applies to `pkg/domain/model/**/*.go`)

Many rules use YAML frontmatter with `paths` to apply only to specific file patterns, ensuring rules are enforced only when working with relevant files.
