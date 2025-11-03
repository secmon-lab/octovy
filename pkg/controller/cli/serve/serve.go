package serve

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
	"github.com/m-mizutani/octovy/pkg/controller/cli/config"
	"github.com/m-mizutani/octovy/pkg/controller/server"
	"github.com/m-mizutani/octovy/pkg/infra"
	"github.com/m-mizutani/octovy/pkg/infra/gh"
	"github.com/m-mizutani/octovy/pkg/infra/trivy"
	"github.com/m-mizutani/octovy/pkg/usecase"
	"github.com/m-mizutani/octovy/pkg/utils"

	"github.com/urfave/cli/v2"

	_ "github.com/lib/pq"
)

func New() *cli.Command {
	var (
		addr                      string
		trivyPath                 string
		disableNoDetectionComment bool

		githubApp    config.GitHubApp
		bigQuery     config.BigQuery
		cloudStorage config.CloudStorage
		sentry       config.Sentry
		policy       config.Policy
	)
	serveFlags := []cli.Flag{
		&cli.StringFlag{
			Name:        "addr",
			Usage:       "Binding address",
			Value:       "127.0.0.1:8000",
			EnvVars:     []string{"OCTOVY_ADDR"},
			Destination: &addr,
		},
		&cli.StringFlag{
			Name:        "trivy-path",
			Usage:       "Path to trivy binary",
			Value:       "trivy",
			EnvVars:     []string{"OCTOVY_TRIVY_PATH"},
			Destination: &trivyPath,
		},
		&cli.BoolFlag{
			Name:        "disable-no-detection-comment",
			Usage:       "Disable comment to PR if no detection",
			EnvVars:     []string{"OCTOVY_DISABLE_NO_DETECTION_COMMENT"},
			Destination: &disableNoDetectionComment,
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
			cloudStorage.Flags(),
			sentry.Flags(),
			policy.Flags(),
		),
		Action: func(c *cli.Context) error {
			utils.Logger().Info("starting serve",
				slog.Any("Addr", addr),
				slog.Any("TrivyPath", trivyPath),
				slog.Any("GitHubApp", githubApp),
				slog.Any("BigQuery", bigQuery),
				slog.Any("CloudStorage", cloudStorage),
				slog.Any("Sentry", sentry),
				slog.Any("Policy", policy),
			)

			if err := sentry.Configure(); err != nil {
				return err
			}

			ghApp, err := gh.New(githubApp.ID, githubApp.PrivateKey(), gh.WithEnableCheckRuns(githubApp.EnableCheckRuns))
			if err != nil {
				return err
			}

			infraOptions := []infra.Option{
				infra.WithGitHubApp(ghApp),
				infra.WithTrivy(trivy.New(trivyPath)),
			}

			if bqClient, err := bigQuery.NewClient(c.Context); err != nil {
				return err
			} else if bqClient != nil {
				infraOptions = append(infraOptions, infra.WithBigQuery(bqClient))
			}

			if csClient, err := cloudStorage.NewClient(c.Context); err != nil {
				return err
			} else if csClient != nil {
				infraOptions = append(infraOptions, infra.WithStorage(csClient))
			}

			if policyClient, err := policy.Configure(); err != nil {
				return err
			} else if policyClient != nil {
				infraOptions = append(infraOptions, infra.WithPolicy(policyClient))
			}

			clients := infra.New(infraOptions...)

			var ucOptions []usecase.Option
			if disableNoDetectionComment {
				ucOptions = append(ucOptions, usecase.WithDisableNoDetectionComment())
			}

			uc := usecase.New(clients, ucOptions...)
			s := server.New(uc, server.WithGitHubSecret(githubApp.Secret))

			serverErr := make(chan error, 1)
			httpServer := &http.Server{
				Addr:    addr,
				Handler: s.Mux(),

				ReadHeaderTimeout: 10 * time.Second,
				ReadTimeout:       30 * time.Second,
				WriteTimeout:      30 * time.Second,
			}

			go func() {
				utils.Logger().Info("starting http server", "addr", addr)
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
				utils.Logger().Info("shutting down server", "signal", sig)

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
