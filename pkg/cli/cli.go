package cli

import (
	"context"

	"github.com/m-mizutani/octovy/pkg/utils/logging"
	"github.com/urfave/cli/v3"
)

// ConfigureLogging is exported for testing purposes
var ConfigureLogging = logging.Configure

type CLI struct {
}

func New() *CLI {
	return &CLI{}
}

func (x *CLI) Run(argv []string) error {
	var (
		logLevel  string
		logFormat string
		logOutput string
	)

	app := &cli.Command{
		Name:  "octovy",
		Usage: "Vulnerability management system with Trivy",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "log-level",
				Usage:       "Log level [trace|debug|info|warn|error]",
				Aliases:     []string{"l"},
				Sources:     cli.EnvVars("OCTOVY_LOG_LEVEL"),
				Destination: &logLevel,
				Value:       "info",
			},
			&cli.StringFlag{
				Name:        "log-format",
				Usage:       "Log format [text|json]",
				Aliases:     []string{"f"},
				Sources:     cli.EnvVars("OCTOVY_LOG_FORMAT"),
				Destination: &logFormat,
				Value:       "text",
			},
			&cli.StringFlag{
				Name:        "log-output",
				Usage:       "Log output [-|stdout|stderr|<file>]",
				Aliases:     []string{"o"},
				Sources:     cli.EnvVars("OCTOVY_LOG_OUTPUT"),
				Destination: &logOutput,
				Value:       "-",
			},
		},
		Commands: []*cli.Command{
			serveCommand(),
			scanCommand(),
		},
		Before: func(ctx context.Context, c *cli.Command) (context.Context, error) {
			if err := ConfigureLogging(logFormat, logLevel, logOutput); err != nil {
				return ctx, err
			}
			return ctx, nil
		},
	}

	if err := app.Run(context.Background(), argv); err != nil {
		logging.Default().Error("fatal error", "error", err)
		return err
	}

	return nil
}
