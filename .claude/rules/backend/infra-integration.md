---
paths: pkg/infra/**/*.go
---

# Infrastructure Integration

## Trivy Integration
- Location: [pkg/infra/trivy/](pkg/infra/trivy/)
- Executes Trivy CLI as subprocess with JSON output
- Default path: `trivy` (configurable via `OCTOVY_TRIVY_PATH`)
- Command flags: `fs --exit-code 0 --no-progress --format json --output <file> --list-all-pkgs <dir>`

## GitHub App Authentication
- Location: [pkg/infra/ghapp/](pkg/infra/ghapp/)
- Uses `bradleyfalzon/ghinstallation/v2` for GitHub App authentication
- Installation tokens for API access
- Private key can be file path or PEM content

## BigQuery Client
- Location: [pkg/infra/bq/](pkg/infra/bq/)
- Stores scan results using `cloud.google.com/go/bigquery`
- Schema defined in [pkg/domain/model/result.go](pkg/domain/model/result.go)
- Supports retry logic for schema mismatch errors

## Dependency Injection
The `infra.Clients` struct aggregates all infrastructure dependencies:
```go
clients := infra.New(
    infra.WithGitHubApp(ghApp),
    infra.WithBigQuery(bq),
    infra.WithTrivy(trivy),
    infra.WithHTTPClient(httpClient),
)
```

Functional options pattern for clean dependency injection.
