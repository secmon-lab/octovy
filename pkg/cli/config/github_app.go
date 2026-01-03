package config

import (
	"log/slog"

	"github.com/m-mizutani/octovy/pkg/domain/types"
	"github.com/m-mizutani/octovy/pkg/infra/ghapp"
	"github.com/urfave/cli/v3"
)

type GitHubApp struct {
	id         types.GitHubAppID
	secret     types.GitHubAppSecret     `masq:"secret"`
	privateKey types.GitHubAppPrivateKey `masq:"secret"`
}

func (x *GitHubApp) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.Int64Flag{
			Name:        "github-app-id",
			Usage:       "GitHub App ID",
			Category:    "GitHub App",
			Destination: (*int64)(&x.id),
			Sources:     cli.EnvVars("OCTOVY_GITHUB_APP_ID"),
			Required:    true,
		},
		&cli.StringFlag{
			Name:        "github-app-private-key",
			Usage:       "GitHub App Private Key",
			Category:    "GitHub App",
			Destination: (*string)(&x.privateKey),
			Sources:     cli.EnvVars("OCTOVY_GITHUB_APP_PRIVATE_KEY"),
			Required:    true,
		},
		&cli.StringFlag{
			Name:        "github-app-secret",
			Usage:       "GitHub App Webhook Secret",
			Category:    "GitHub App",
			Destination: (*string)(&x.secret),
			Sources:     cli.EnvVars("OCTOVY_GITHUB_APP_SECRET"),
		},
	}
}

func (x GitHubApp) New() (*ghapp.Client, error) {
	return ghapp.New(x.id, x.privateKey)
}

func (x GitHubApp) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Int64("ID", int64(x.id)),
		slog.Int("Secret.len", len(x.secret)),
		slog.Int("privateKey.len", len(x.privateKey)),
	)
}

func (x GitHubApp) Secret() types.GitHubAppSecret {
	return x.secret
}
