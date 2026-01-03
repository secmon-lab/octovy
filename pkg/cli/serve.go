package cli

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gots/slice"
	"github.com/m-mizutani/octovy/pkg/cli/config"
	"github.com/m-mizutani/octovy/pkg/controller/server"
	"github.com/m-mizutani/octovy/pkg/infra"
	"github.com/m-mizutani/octovy/pkg/infra/trivy"
	"github.com/m-mizutani/octovy/pkg/usecase"
	"github.com/m-mizutani/octovy/pkg/utils/logging"

	"github.com/urfave/cli/v3"

	_ "github.com/lib/pq"
)

func serveCommand() *cli.Command {
	var (
		addr      string
		trivyPath string

		githubApp config.GitHubApp
		bigQuery  config.BigQuery
		sentry    config.Sentry
	)
	serveFlags := []cli.Flag{
		&cli.StringFlag{
			Name:        "addr",
			Usage:       "Binding address",
			Value:       "127.0.0.1:8000",
			Sources:     cli.EnvVars("OCTOVY_ADDR"),
			Destination: &addr,
		},
		&cli.StringFlag{
			Name:        "trivy-path",
			Usage:       "Path to trivy binary",
			Value:       "trivy",
			Sources:     cli.EnvVars("OCTOVY_TRIVY_PATH"),
			Destination: &trivyPath,
		},
	}

	return &cli.Command{
		Name:    "serve",
		Aliases: []string{"s"},
		Usage:   "Server mode",
		Flags: slice.Flatten(
			serveFlags,
			githubApp.Flags(),
			bigQuery.Flags(),
			sentry.Flags(),
		),
		Action: func(ctx context.Context, c *cli.Command) error {
			logging.Default().Info("starting serve",
				slog.Any("Addr", addr),
				slog.Any("TrivyPath", trivyPath),
				slog.Any("GitHubApp", githubApp),
				slog.Any("BigQuery", bigQuery),
				slog.Any("Sentry", sentry),
			)

			if err := sentry.Configure(ctx); err != nil {
				return err
			}

			ghApp, err := githubApp.New()
			if err != nil {
				return err
			}

			infraOptions := []infra.Option{
				infra.WithGitHubApp(ghApp),
				infra.WithTrivy(trivy.New(trivyPath)),
			}

			if bqClient, err := bigQuery.NewClient(ctx); err != nil {
				return err
			} else if bqClient != nil {
				infraOptions = append(infraOptions, infra.WithBigQuery(bqClient))
			}

			clients := infra.New(infraOptions...)

			uc := usecase.New(clients)
			s := server.New(uc, server.WithGitHubSecret(githubApp.Secret()))

			serverErr := make(chan error, 1)
			httpServer := &http.Server{
				Addr:    addr,
				Handler: s.Mux(),

				ReadHeaderTimeout: 10 * time.Second,
				ReadTimeout:       30 * time.Second,
				WriteTimeout:      30 * time.Second,
			}

			go func() {
				logging.Default().Info("starting http server", "addr", addr)
				if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
					serverErr <- goerr.Wrap(err, "failed to listen and serve")
				}
			}()

			quit := make(chan os.Signal, 1)
			signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

			select {
			case err := <-serverErr:
				return err

			case sig := <-quit:
				logging.Default().Info("shutting down server", "signal", sig)

				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()

				if err := httpServer.Shutdown(ctx); err != nil {
					return goerr.Wrap(err, "failed to shutdown server")
				}
			}

			return nil
		},
	}
}
