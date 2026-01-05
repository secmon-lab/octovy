---
paths: pkg/**/*.go
---

# Temporary File Handling

## File Naming Patterns
- Repo archives: `octovy_code.<owner>.<repo>.<commit>.*.zip`
- Extracted directories: `octovy.<owner>.<repo>.<commit>.*`
- Scan results: `octovy_result.*.json`

## Cleanup Policy
**CRITICAL**: Always clean up temporary files and directories using deferred cleanup.

### Patterns
```go
import "github.com/m-mizutani/octovy/pkg/utils/safe"

// File cleanup
tmpFile, err := os.CreateTemp("", "octovy_*.json")
if err != nil {
    return err
}
defer safe.Remove(tmpFile.Name())

// Directory cleanup
tmpDir, err := os.MkdirTemp("", "octovy.*")
if err != nil {
    return err
}
defer safe.RemoveAll(tmpDir)
```

## Safe I/O Operations
Use [pkg/utils/safe/](pkg/utils/safe/) for error-safe cleanup:
- `safe.Remove()`: Safely remove files
- `safe.RemoveAll()`: Safely remove directories
- `safe.Close()`: Safely close I/O resources

These functions log errors but don't fail the operation, preventing deferred cleanup from masking primary errors.
