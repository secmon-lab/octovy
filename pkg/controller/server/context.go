package server

import (
	"context"

	"github.com/m-mizutani/octovy/pkg/utils/logging"
)

// DetachContext creates a new context.Background() based context that inherits
// logger, request ID, and time function from the original context.
// This is useful when running background goroutines from HTTP request handlers,
// as the original request context will be cancelled when the HTTP request completes.
func DetachContext(ctx context.Context) context.Context {
	bgCtx := context.Background()

	// Inherit logger from the original context
	bgCtx = logging.With(bgCtx, logging.From(ctx))

	// Inherit request ID and time function from the original context
	bgCtx = logging.InheritContextValues(bgCtx, ctx)

	return bgCtx
}
