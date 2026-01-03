package logging_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/octovy/pkg/utils/logging"
)

func TestWith(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()

	newCtx := logging.With(ctx, logger)
	// Verify the logger can be retrieved from the context
	retrieved := logging.From(newCtx)
	gt.V(t, retrieved).Equal(logger)
}

func TestFrom(t *testing.T) {
	t.Run("get logger from context with logger", func(t *testing.T) {
		ctx := context.Background()
		logger := slog.Default()
		ctx = logging.With(ctx, logger)

		retrieved := logging.From(ctx)
		gt.V(t, retrieved).Equal(logger)
	})

	t.Run("get logger from context without logger", func(t *testing.T) {
		ctx := context.Background()
		retrieved := logging.From(ctx)
		// Should return default logger, verify it's the same instance when called again
		retrieved2 := logging.From(ctx)
		gt.V(t, retrieved).Equal(retrieved2)
		// Verify it's actually a logger instance by checking it can be used
		gt.V(t, retrieved.Handler()).Equal(logging.Default().Handler())
	})
}

func TestCtxRequestID(t *testing.T) {
	t.Run("get new request ID from context", func(t *testing.T) {
		ctx := context.Background()

		reqID, newCtx := logging.CtxRequestID(ctx)
		gt.V(t, reqID).NotEqual("")
		// Verify the context contains the request ID
		retrievedID, _ := logging.CtxRequestID(newCtx)
		gt.V(t, retrievedID).Equal(reqID)
	})

	t.Run("get existing request ID from context", func(t *testing.T) {
		ctx := context.Background()

		reqID1, ctx1 := logging.CtxRequestID(ctx)
		reqID2, ctx2 := logging.CtxRequestID(ctx1)

		gt.V(t, reqID1).Equal(reqID2)
		// Verify both contexts return the same request ID
		retrievedID1, _ := logging.CtxRequestID(ctx1)
		retrievedID2, _ := logging.CtxRequestID(ctx2)
		gt.V(t, retrievedID1).Equal(reqID1)
		gt.V(t, retrievedID2).Equal(reqID1)
	})
}

func TestCtxTime(t *testing.T) {
	t.Run("get current time from context", func(t *testing.T) {
		ctx := context.Background()

		tm := logging.CtxTime(ctx)
		gt.V(t, tm.IsZero()).Equal(false)
	})
}

func TestCtxWithTime(t *testing.T) {
	t.Run("set and get custom time from context", func(t *testing.T) {
		ctx := context.Background()

		called := false
		ctx = logging.CtxWithTime(ctx, func() time.Time {
			called = true
			return time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		})

		tm := logging.CtxTime(ctx)
		gt.True(t, called)
		gt.V(t, tm.Year()).Equal(2024)
	})
}
