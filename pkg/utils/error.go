package utils

import (
	"context"
	"fmt"

	"github.com/getsentry/sentry-go"
	"github.com/m-mizutani/goerr/v2"
)

func HandleError(ctx context.Context, msg string, err error) {
	// Sending error to Sentry
	hub := sentry.CurrentHub().Clone()
	hub.ConfigureScope(func(scope *sentry.Scope) {
		if goErr := goerr.Unwrap(err); goErr != nil {
			for k, v := range goErr.Values() {
				scope.SetExtra(fmt.Sprintf("%v", k), v)
			}
		}
	})
	evID := hub.CaptureException(err)

	CtxLogger(ctx).Error(msg,
		"error", err,
		"sentry.EventID", evID,
	)
}
