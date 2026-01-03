package safe

import (
	"database/sql"
	"io"
	"log/slog"
	"os"

	"github.com/m-mizutani/octovy/pkg/utils/logging"
)

// Close safely closes the resource and logs error if any
func Close(closer io.Closer) {
	if closer != nil {
		if err := closer.Close(); err != nil {
			if err == io.EOF {
				return
			}
			logging.Default().Warn("Fail to close resource", slog.Any("error", err))
		}
	}
}

// Remove safely removes the file and logs error if any
func Remove(path string) {
	if err := os.Remove(path); err != nil {
		logging.Default().Warn("Fail to remove file", slog.Any("error", err))
	}
}

// RemoveAll safely removes the directory and logs error if any
func RemoveAll(path string) {
	if err := os.RemoveAll(path); err != nil {
		logging.Default().Warn("Fail to remove file", slog.Any("error", err))
	}
}

// Rollback safely rolls back the transaction and logs error if any
func Rollback(tx *sql.Tx) {
	if tx != nil {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			logging.Default().Warn("Fail to rollback transaction", slog.Any("error", err))
		}
	}
}
