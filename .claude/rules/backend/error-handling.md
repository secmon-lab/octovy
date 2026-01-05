---
paths: pkg/**/*.go
---

# Error Handling

## goerr/v2 Pattern
All errors must use `github.com/m-mizutani/goerr/v2` for wrapped errors with context.

### Patterns
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

**Note**: goerr v2 requires message as second argument to `Wrap()`. Context values are added as variadic `goerr.V()` or `goerr.Value()` arguments, not via `.With()` method chains.

## Sentry Integration
For critical errors, use Sentry integration:
```go
import "github.com/m-mizutani/octovy/pkg/utils/errutil"

if err := criticalOperation(); err != nil {
    errutil.HandleError(ctx, err)
    return err
}
```
