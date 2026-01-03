package memory_test

import (
	"testing"

	"github.com/m-mizutani/octovy/pkg/repository/memory"
	"github.com/m-mizutani/octovy/pkg/repository/testhelper"
)

func TestMemoryScanRepository(t *testing.T) {
	repo := memory.New()
	testhelper.TestAll(t, repo)
}
