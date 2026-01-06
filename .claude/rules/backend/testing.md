# Testing Standards

## Mock Usage Policy

### Repository Mock Prohibition
**CRITICAL - STRICTLY PROHIBITED**: Do NOT create or use mock repositories (`mock.ScanRepositoryMock`) in tests under ANY circumstances.

**Rules**:
- **ALWAYS use `memory.New()`** for repository testing instead of mocks
- **Mock only external services** that cannot be run locally (BigQuery, GitHub API, external APIs)
- **NEVER mock internal repository implementations** - this is a critical rule violation
- **Memory implementation provides real behavior** - use it to verify actual data persistence and retrieval
- Mocks hide bugs and don't test real integration - avoid them for internal components

**Why this rule exists**:
- Repository mocks allow tests to pass without verifying actual data persistence logic
- Memory repository is fast, requires no external dependencies, and tests real behavior
- Using mocks for repositories defeats the purpose of the dual implementation strategy (memory/firestore)
- Mock repositories create maintenance burden and hide integration bugs

**Example**:
```go
// BAD - STRICTLY PROHIBITED - Using mock repository
mockRepo := &mock.ScanRepositoryMock{}
mockRepo.GetRepositoryFunc = func(...) {...}
mockRepo.ListVulnerabilitiesFunc = func(...) {...}

// GOOD - Using memory repository
import "github.com/m-mizutani/octovy/pkg/repository/memory"

memRepo := memory.New()
// Test against actual implementation with real data persistence
repo, err := memRepo.GetRepository(ctx, repoID)
vulns, err := memRepo.ListVulnerabilities(ctx, repoID, branchName, targetID)
```

**What to mock instead**:
- External APIs: `mock.GitHubAppMock`, HTTP clients
- Cloud services: `mock.BigQueryMock`
- External tools: Trivy client mock

## Testing Framework
- Uses `github.com/m-mizutani/gt` test framework for assertions
- Common patterns: `gt.V(t, actual).Equal(expected)`, `gt.NoError(t, err)`, `gt.R1(fn()).NoError(t)`
- Mock interfaces generated via `moq` (github.com/matryer/moq) in [pkg/domain/mock/](pkg/domain/mock/)
- Test helpers in [pkg/utils/testutil/](pkg/utils/testutil/)

## Test Coverage Requirements
- **Every Go source file MUST have a corresponding test file**: If `xxx.go` exists, `xxx_test.go` MUST exist
  - **Exception**: Pure data model files (structs with no logic) do NOT require test files
  - Model files are in `pkg/domain/model/` and contain only struct definitions with `json`/`bigquery` tags
  - If a file contains any methods or functions with logic, it MUST have tests
- **Unit tests are mandatory**: Each function and method requires unit tests covering normal cases, edge cases, and error scenarios
- **Integration tests are required**: End-to-end workflows must have integration tests validating the complete flow
- **Test-Driven Development**: When adding new features or fixing bugs, write tests first before implementation
- Do not merge code without proper test coverage
- **Mock generation**: Use `task gen` or `go generate ./...` to regenerate mocks after interface changes

## Firestore Testing Pattern
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

## Test Quality Standards
**CRITICAL**: Tests must verify actual behavior, not just absence of errors.

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
