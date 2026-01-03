package config

import (
	"log/slog"

	"github.com/getsentry/sentry-go"
	"github.com/m-mizutani/goerr/v2"
	"github.com/urfave/cli/v3"
)

type Sentry struct {
	dsn         string
	environment string
}

func (x *Sentry) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "sentry-dsn",
			Usage:       "Sentry DSN",
			Category:    "Sentry",
			Destination: &x.dsn,
			Sources:     cli.EnvVars("OCTOVY_SENTRY_DSN"),
		},
		&cli.StringFlag{
			Name:        "sentry-env",
			Usage:       "Sentry environment",
			Category:    "Sentry",
			Destination: &x.environment,
			Sources:     cli.EnvVars("OCTOVY_SENTRY_ENV"),
		},
	}
}

func (x *Sentry) Configure() error {
	if x.dsn == "" {
		return nil
	}

	if err := sentry.Init(sentry.ClientOptions{
		Dsn:         x.dsn,
		Environment: x.environment,
	}); err != nil {
		return goerr.Wrap(err, "failed to initialize sentry")
	}

	return nil
}

func (x *Sentry) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Any("DSN.len", len(x.dsn)),
		slog.Any("Environment", x.environment),
	)
}
