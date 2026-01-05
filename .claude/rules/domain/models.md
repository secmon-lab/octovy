---
paths: pkg/domain/model/**/*.go
---

# Domain Model Rules

## Struct Tags Policy
**CRITICAL**: Minimize the use of struct tags. Only add tags when there is an explicit serialization use case.

### Rules
- **DO NOT use `firestore` tags** - Firestore SDK handles field mapping automatically
- **Only use `json` tags when explicitly converting to/from JSON** (e.g., Trivy report parsing, API responses)
- **Domain models that are only used internally should have NO tags**
- **BigQuery models can use `bigquery` tags for schema mapping**
  - However **do not use `bigquery` tags for Trivy Report structures**

### Examples
```go
// CORRECT - Internal domain model (no tags needed)
type Repository struct {
    ID              types.GitHubRepoID
    Owner           string
    DefaultBranch   types.BranchName
}

// CORRECT - Trivy report model (json tags for external JSON parsing)
type TrivyResult struct {
    Target          string          `json:"Target"`
    Vulnerabilities []Vulnerability `json:"Vulnerabilities"`
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

## Test Requirements
Pure data model files (structs with no logic) do NOT require test files. Only files containing methods or functions with logic require tests.
