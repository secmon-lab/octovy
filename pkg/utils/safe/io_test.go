package safe_test

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/octovy/pkg/utils/safe"
)

func TestClose(t *testing.T) {
	t.Run("close valid reader", func(t *testing.T) {
		reader := io.NopCloser(bytes.NewReader([]byte("test")))
		safe.Close(reader) // Should not panic
	})

	t.Run("close nil reader", func(t *testing.T) {
		safe.Close(nil) // Should not panic
	})
}

func TestRemove(t *testing.T) {
	t.Run("remove existing file", func(t *testing.T) {
		tmpFile := gt.R1(os.CreateTemp("", "test-*.txt")).NoError(t)
		path := tmpFile.Name()
		gt.NoError(t, tmpFile.Close())

		safe.Remove(path) // Should not panic

		_, err := os.Stat(path)
		gt.True(t, os.IsNotExist(err))
	})

	t.Run("remove non-existing file", func(t *testing.T) {
		safe.Remove("/nonexistent/path/file.txt") // Should not panic
	})
}

func TestRemoveAll(t *testing.T) {
	t.Run("remove existing directory", func(t *testing.T) {
		tmpDir := gt.R1(os.MkdirTemp("", "test-dir-*")).NoError(t)

		// Create some files inside
		gt.NoError(t, os.WriteFile(filepath.Join(tmpDir, "file.txt"), []byte("test"), 0644))

		safe.RemoveAll(tmpDir) // Should not panic

		_, err := os.Stat(tmpDir)
		gt.True(t, os.IsNotExist(err))
	})

	t.Run("remove non-existing directory", func(t *testing.T) {
		safe.RemoveAll("/nonexistent/directory") // Should not panic
	})
}

func TestRollback(t *testing.T) {
	t.Run("rollback with nil transaction", func(t *testing.T) {
		safe.Rollback(nil) // Should not panic
	})
}

func TestCloseWithError(t *testing.T) {
	t.Run("close reader that returns error", func(t *testing.T) {
		reader := &errorCloser{}
		safe.Close(reader) // Should not panic, should log
	})

	t.Run("close reader that returns EOF", func(t *testing.T) {
		reader := &eofCloser{}
		safe.Close(reader) // Should not panic, should not log
	})
}

type errorCloser struct{}

func (e *errorCloser) Close() error {
	return io.ErrUnexpectedEOF
}

type eofCloser struct{}

func (e *eofCloser) Close() error {
	return io.EOF
}
