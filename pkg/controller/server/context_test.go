package server_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/octovy/pkg/controller/server"
	"github.com/m-mizutani/octovy/pkg/utils/logging"
)

func TestDetachContext(t *testing.T) {
	t.Run("inherits logger from original context", func(t *testing.T) {
		originalCtx := context.Background()
		customLogger := slog.Default().With("test", "value")
		originalCtx = logging.With(originalCtx, customLogger)

		bgCtx := server.DetachContext(originalCtx)

		inheritedLogger := logging.From(bgCtx)
		gt.V(t, inheritedLogger).Equal(customLogger)
	})

	t.Run("inherits request ID from original context", func(t *testing.T) {
		originalCtx := context.Background()
		reqID, originalCtx := logging.CtxRequestID(originalCtx)

		bgCtx := server.DetachContext(originalCtx)

		inheritedReqID, _ := logging.CtxRequestID(bgCtx)
		gt.V(t, inheritedReqID).Equal(reqID)
	})

	t.Run("inherits time function from original context", func(t *testing.T) {
		originalCtx := context.Background()
		fixedTime := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
		originalCtx = logging.CtxWithTime(originalCtx, func() time.Time {
			return fixedTime
		})

		bgCtx := server.DetachContext(originalCtx)

		inheritedTime := logging.CtxTime(bgCtx)
		gt.V(t, inheritedTime).Equal(fixedTime)
	})

	t.Run("detached context is not cancelled when original is cancelled", func(t *testing.T) {
		originalCtx, cancel := context.WithCancel(context.Background())

		bgCtx := server.DetachContext(originalCtx)

		// Cancel the original context
		cancel()

		// The original context should be cancelled
		gt.V(t, originalCtx.Err()).Equal(context.Canceled)

		// The detached context should NOT be cancelled
		gt.V(t, bgCtx.Err()).Equal(nil)
	})

	t.Run("inherits all values together", func(t *testing.T) {
		originalCtx := context.Background()

		// Set up original context with all values
		customLogger := slog.Default().With("component", "test")
		originalCtx = logging.With(originalCtx, customLogger)

		reqID, originalCtx := logging.CtxRequestID(originalCtx)

		fixedTime := time.Date(2024, 12, 25, 10, 30, 0, 0, time.UTC)
		originalCtx = logging.CtxWithTime(originalCtx, func() time.Time {
			return fixedTime
		})

		// Detach and verify all values are inherited
		bgCtx := server.DetachContext(originalCtx)

		inheritedLogger := logging.From(bgCtx)
		gt.V(t, inheritedLogger).Equal(customLogger)

		inheritedReqID, _ := logging.CtxRequestID(bgCtx)
		gt.V(t, inheritedReqID).Equal(reqID)

		inheritedTime := logging.CtxTime(bgCtx)
		gt.V(t, inheritedTime).Equal(fixedTime)
	})
}
