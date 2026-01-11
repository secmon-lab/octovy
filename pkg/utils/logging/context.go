package logging

import (
	"context"
	"log/slog"
	"time"

	"github.com/m-mizutani/octovy/pkg/domain/types"
)

type ctxRequestIDKey struct{}

// CtxRequestID returns request ID from context. If request ID is not set, return new request ID and context with it
func CtxRequestID(ctx context.Context) (types.RequestID, context.Context) {
	if id, ok := ctx.Value(ctxRequestIDKey{}).(types.RequestID); ok {
		return id, ctx
	}

	newID := types.NewRequestID()
	return newID, context.WithValue(ctx, ctxRequestIDKey{}, newID)
}

type ctxLoggerKey struct{}

// With returns a new context with logger
func With(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxLoggerKey{}, logger)
}

// From returns logger from context. If logger is not set, return default logger
func From(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(ctxLoggerKey{}).(*slog.Logger); ok {
		return l
	}
	return defaultLogger
}

type ctxTimeKey struct{}
type TimeFunc func() time.Time

// CtxTime returns time from context. If time is not set, return current time and context with it
func CtxTime(ctx context.Context) time.Time {
	if t, ok := ctx.Value(ctxTimeKey{}).(TimeFunc); ok {
		return t()
	}
	return time.Now()
}

// CtxWithTime returns a new context with time function
func CtxWithTime(ctx context.Context, timeFunc TimeFunc) context.Context {
	return context.WithValue(ctx, ctxTimeKey{}, timeFunc)
}

// InheritContextValues copies request ID and time function from src context to dst context.
// This is useful when creating a background context that should inherit values from
// the original HTTP request context. Note that logger is NOT copied by this function;
// use With() separately to copy the logger.
func InheritContextValues(dst, src context.Context) context.Context {
	// Copy request ID if exists
	if reqID, ok := src.Value(ctxRequestIDKey{}).(types.RequestID); ok {
		dst = context.WithValue(dst, ctxRequestIDKey{}, reqID)
	}

	// Copy time function if exists
	if timeFunc, ok := src.Value(ctxTimeKey{}).(TimeFunc); ok {
		dst = context.WithValue(dst, ctxTimeKey{}, timeFunc)
	}

	return dst
}
