package cli

import (
	"context"
	"log/slog"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gots/slice"
	"github.com/m-mizutani/octovy/pkg/cli/config"
	"github.com/m-mizutani/octovy/pkg/domain/interfaces"
	"github.com/m-mizutani/octovy/pkg/domain/model"
	"github.com/m-mizutani/octovy/pkg/domain/types"
	"github.com/m-mizutani/octovy/pkg/infra"
	trivyInfra "github.com/m-mizutani/octovy/pkg/infra/trivy"
	"github.com/m-mizutani/octovy/pkg/usecase"
	"github.com/m-mizutani/octovy/pkg/utils/logging"
	"github.com/urfave/cli/v3"
)

func scanCommand() *cli.Command {
	return &cli.Command{
		Name:  "scan",
		Usage: "Scan repository with Trivy and insert results to BigQuery",
		Commands: []*cli.Command{
			scanLocalCommand(),
			scanRemoteCommand(),
		},
	}
}

func scanLocalCommand() *cli.Command {
	var (
		bigQuery  config.BigQuery
		firestore config.Firestore
		dir       string
		trivyPath string
		meta      model.GitHubMetadata
	)

	return &cli.Command{
		Name:  "local",
		Usage: "Scan local directory with Trivy and insert results to BigQuery",
		Flags: slice.Flatten([]cli.Flag{
			&cli.StringFlag{
				Name:        "dir",
				Aliases:     []string{"d"},
				Usage:       "Path to directory to scan",
				Value:       ".",
				Destination: &dir,
			},
			&cli.StringFlag{
				Name:        "trivy-path",
				Usage:       "Path to trivy binary",
				Value:       "trivy",
				Sources:     cli.EnvVars("OCTOVY_TRIVY_PATH"),
				Destination: &trivyPath,
			},
			&cli.StringFlag{
				Name:        "github-owner",
				Usage:       "GitHub repository owner (auto-detect from git if not specified)",
				Sources:     cli.EnvVars("OCTOVY_GITHUB_OWNER"),
				Destination: &meta.Owner,
			},
			&cli.StringFlag{
				Name:        "github-repo",
				Usage:       "GitHub repository name (auto-detect from git if not specified)",
				Sources:     cli.EnvVars("OCTOVY_GITHUB_REPO"),
				Destination: &meta.RepoName,
			},
			&cli.StringFlag{
				Name:        "github-commit-id",
				Usage:       "GitHub commit ID (auto-detect from git if not specified)",
				Sources:     cli.EnvVars("OCTOVY_GITHUB_COMMIT_ID"),
				Destination: &meta.CommitID,
			},
		}, bigQuery.Flags(), firestore.Flags()),
		Action: func(ctx context.Context, c *cli.Command) error {
			// Auto-detect GitHub metadata from git if not specified
			if err := AutoDetectGitMetadata(ctx, &meta); err != nil {
				return err
			}

			// Validate required GitHub metadata
			if err := meta.ValidateBasic(); err != nil {
				return err
			}

			return runScanLocal(ctx, dir, trivyPath, meta, &bigQuery, &firestore)
		},
	}
}

func scanRemoteCommand() *cli.Command {
	var (
		bigQuery     config.BigQuery
		firestore    config.Firestore
		githubApp    config.GitHubApp
		trivyPath    string
		owner        string
		repo         string
		commit       string
		branch       string
		installIDRaw int64
	)

	return &cli.Command{
		Name:  "remote",
		Usage: "Scan GitHub repository with Trivy and insert results to BigQuery",
		Flags: slice.Flatten([]cli.Flag{
			&cli.StringFlag{
				Name:        "github-owner",
				Usage:       "GitHub repository owner (required)",
				Sources:     cli.EnvVars("OCTOVY_GITHUB_OWNER"),
				Destination: &owner,
				Required:    true,
			},
			&cli.StringFlag{
				Name:        "github-repo",
				Usage:       "GitHub repository name (required)",
				Sources:     cli.EnvVars("OCTOVY_GITHUB_REPO"),
				Destination: &repo,
				Required:    true,
			},
			&cli.StringFlag{
				Name:        "github-commit",
				Usage:       "GitHub commit ID (mutually exclusive with --github-branch)",
				Sources:     cli.EnvVars("OCTOVY_GITHUB_COMMIT"),
				Destination: &commit,
			},
			&cli.StringFlag{
				Name:        "github-branch",
				Usage:       "GitHub branch name (mutually exclusive with --github-commit)",
				Sources:     cli.EnvVars("OCTOVY_GITHUB_BRANCH"),
				Destination: &branch,
			},
			&cli.Int64Flag{
				Name:        "github-app-installation-id",
				Usage:       "GitHub App Installation ID (required for full specification mode)",
				Sources:     cli.EnvVars("OCTOVY_GITHUB_APP_INSTALLATION_ID"),
				Destination: &installIDRaw,
			},
			&cli.StringFlag{
				Name:        "trivy-path",
				Usage:       "Path to trivy binary",
				Value:       "trivy",
				Sources:     cli.EnvVars("OCTOVY_TRIVY_PATH"),
				Destination: &trivyPath,
			},
		}, bigQuery.Flags(), firestore.Flags(), githubApp.Flags()),
		Action: func(ctx context.Context, c *cli.Command) error {
			return runScanRemote(ctx, &scanRemoteParams{
				owner:        owner,
				repo:         repo,
				commit:       commit,
				branch:       branch,
				installIDRaw: installIDRaw,
				trivyPath:    trivyPath,
				bigQuery:     &bigQuery,
				firestore:    &firestore,
				githubApp:    &githubApp,
			})
		},
	}
}

type scanRemoteParams struct {
	owner        string
	repo         string
	commit       string
	branch       string
	installIDRaw int64
	trivyPath    string
	bigQuery     *config.BigQuery
	firestore    *config.Firestore
	githubApp    *config.GitHubApp
}

func runScanRemote(ctx context.Context, params *scanRemoteParams) error {
	// Create GitHub App client
	ghClient, err := params.githubApp.New()
	if err != nil {
		return goerr.Wrap(err, "failed to create GitHub App client")
	}

	// Create BigQuery client
	bqClient, err := params.bigQuery.NewClient(ctx)
	if err != nil {
		return goerr.Wrap(err, "failed to create BigQuery client")
	}

	// Create Firestore repository if configured
	var firestoreRepo interfaces.ScanRepository
	if params.firestore.Enabled() {
		repo, err := params.firestore.NewRepository(ctx)
		if err != nil {
			return goerr.Wrap(err, "failed to create Firestore repository")
		}
		firestoreRepo = repo
	}

	// Create clients
	trivyClient := trivyInfra.New(params.trivyPath)
	clientOpts := []infra.Option{
		infra.WithGitHubApp(ghClient),
		infra.WithTrivy(trivyClient),
		infra.WithBigQuery(bqClient),
	}
	if firestoreRepo != nil {
		clientOpts = append(clientOpts, infra.WithScanRepository(firestoreRepo))
	}
	clients := infra.New(clientOpts...)

	// Execute scan using usecase
	uc := usecase.New(clients)
	input := &model.ScanGitHubRepoRemoteInput{
		Owner:     params.owner,
		Repo:      params.repo,
		Commit:    params.commit,
		Branch:    params.branch,
		InstallID: types.GitHubAppInstallID(params.installIDRaw),
	}

	if err := uc.ScanGitHubRepoRemote(ctx, input); err != nil {
		return goerr.Wrap(err, "failed to scan GitHub repository")
	}

	return nil
}

func runScanLocal(ctx context.Context, dir, trivyPath string, meta model.GitHubMetadata, bigQuery *config.BigQuery, firestoreConfig *config.Firestore) error {
	// Log scan configuration
	logging.Default().Info("Starting scan",
		slog.String("dir", dir),
		slog.String("trivy_path", trivyPath),
		slog.String("github_owner", meta.Owner),
		slog.String("github_repo", meta.RepoName),
		slog.String("github_branch", meta.Branch),
		slog.String("github_commit", meta.CommitID),
		slog.Any("bigquery", bigQuery),
		slog.Bool("firestore_enabled", firestoreConfig.Enabled()),
	)

	// Create BigQuery client if configured
	bqClient, err := bigQuery.NewClient(ctx)
	if err != nil {
		return goerr.Wrap(err, "failed to create BigQuery client")
	}

	// Create Firestore repository if configured
	var firestoreRepo interfaces.ScanRepository
	if firestoreConfig.Enabled() {
		repo, err := firestoreConfig.NewRepository(ctx)
		if err != nil {
			return goerr.Wrap(err, "failed to create Firestore repository")
		}
		firestoreRepo = repo
	}

	// Create clients and usecase
	trivyClient := trivyInfra.New(trivyPath)
	clientOpts := []infra.Option{
		infra.WithTrivy(trivyClient),
		infra.WithBigQuery(bqClient),
	}
	if firestoreRepo != nil {
		clientOpts = append(clientOpts, infra.WithScanRepository(firestoreRepo))
	}
	clients := infra.New(clientOpts...)

	uc := usecase.New(clients)

	// Scan directory and insert to BigQuery
	if err := uc.ScanAndInsert(ctx, dir, meta); err != nil {
		return goerr.Wrap(err, "failed to scan local directory")
	}

	return nil
}
