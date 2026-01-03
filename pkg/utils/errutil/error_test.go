package errutil_test

import (
	"context"
	"errors"
	"testing"

	"github.com/m-mizutani/octovy/pkg/utils/errutil"
)

func TestHandleError(t *testing.T) {
	t.Run("handle error with context", func(t *testing.T) {
		ctx := context.Background()
		err := errors.New("test error")

		// Should not panic
		errutil.HandleError(ctx, "test message", err)
	})

	t.Run("handle nil error", func(t *testing.T) {
		ctx := context.Background()

		// Should not panic
		errutil.HandleError(ctx, "test message", nil)
	})
}
