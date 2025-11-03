package trivy

import (
	"bytes"
	"context"
	"os/exec"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/octovy/pkg/utils"
)

type Client interface {
	Run(ctx context.Context, args []string) error
}

type clientImpl struct {
	path string
}

func New(path string) Client {
	return &clientImpl{
		path: path,
	}
}

func (x *clientImpl) Run(ctx context.Context, args []string) error {
	// Why: The arguments are not from user input
	// nosemgrep: go.lang.security.audit.dangerous-exec-command.dangerous-exec-command
	// #nosec: G204
	cmd := exec.CommandContext(ctx, x.path, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		utils.CtxLogger(ctx).With("stderr", stderr.String()).With("stdout", stdout.String()).Error("trivy failed")
		return goerr.Wrap(err, "executing trivy", goerr.V("stderr", stderr.String()), goerr.V("stdout", stdout.String()))
	}

	return nil
}
