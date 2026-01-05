---
paths: pkg/**/*.go
---

# Logging Standards

## Structured Logging
Use structured logging via `logging.Default()` and `logging.From(ctx)` from [pkg/utils/logging/](pkg/utils/logging/).

### Framework
- Based on `log/slog` standard library
- Uses `m-mizutani/clog` for colored text output
- Uses `m-mizutani/masq` for automatic secret masking

### Patterns
```go
import "github.com/m-mizutani/octovy/pkg/utils/logging"

// Get logger from context
logger := logging.From(ctx)

// Log with structured fields
logger.Info("processing request",
    "user_id", userID,
    "request_id", reqID,
)

// Log errors with context
logger.Error("operation failed",
    "error", err,
    "attempt", retryCount,
)

// Default logger (when context unavailable)
logging.Default().Warn("fallback logger used")
```

### Configuration
```go
import "github.com/m-mizutani/octovy/pkg/utils/logging"

// Configure logging format, level, and output
logging.Configure(format, level, output)
```

## Secret Masking
Sensitive types must implement `slog.LogValuer` interface:
```go
type Credentials struct {
    APIKey string
}

func (c Credentials) LogValue() slog.Value {
    return slog.StringValue("[REDACTED]")
}
```

See [pkg/domain/types/github.go](pkg/domain/types/github.go) for GitHub App credentials example.
