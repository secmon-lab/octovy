package cli

import (
	"context"
	"log/slog"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gots/slice"
	"github.com/m-mizutani/octovy/pkg/cli/config"
	"github.com/m-mizutani/octovy/pkg/domain/interfaces"
	"github.com/m-mizutani/octovy/pkg/domain/model"
	"github.com/m-mizutani/octovy/pkg/infra"
	"github.com/m-mizutani/octovy/pkg/usecase"
	"github.com/m-mizutani/octovy/pkg/utils/logging"
	"github.com/urfave/cli/v3"
)

func insertCommand() *cli.Command {
	var (
		bigQuery   config.BigQuery
		firestore  config.Firestore
		resultFile string
		meta       model.GitHubMetadata
	)

	return &cli.Command{
		Name:    "insert",
		Aliases: []string{"i", "ins"},
		Usage:   "Insert Trivy scan result to BigQuery (and optionally Firestore)",
		Flags: slice.Flatten([]cli.Flag{
			&cli.StringFlag{
				Name:        "result-file",
				Aliases:     []string{"f"},
				Usage:       "Path to Trivy scan result JSON file (required)",
				Sources:     cli.EnvVars("OCTOVY_RESULT_FILE"),
				Destination: &resultFile,
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
			&cli.StringFlag{
				Name:        "github-branch",
				Usage:       "GitHub branch name (optional, auto-detect from git if not specified)",
				Sources:     cli.EnvVars("OCTOVY_GITHUB_BRANCH"),
				Destination: &meta.Branch,
			},
			&cli.StringFlag{
				Name:        "github-default-branch",
				Usage:       "GitHub default branch name (optional)",
				Sources:     cli.EnvVars("OCTOVY_GITHUB_DEFAULT_BRANCH"),
				Destination: &meta.DefaultBranch,
			},
			&cli.Int64Flag{
				Name:        "github-installation-id",
				Usage:       "GitHub App installation ID (optional)",
				Sources:     cli.EnvVars("OCTOVY_GITHUB_INSTALLATION_ID"),
				Destination: &meta.InstallationID,
			},
		}, bigQuery.Flags(), firestore.Flags()),
		Action: func(ctx context.Context, c *cli.Command) error {
			if resultFile == "" {
				return goerr.New("result file is required")
			}

			// Auto-detect GitHub metadata from git if not specified
			if err := AutoDetectGitMetadata(ctx, &meta); err != nil {
				return err
			}

			// Validate required GitHub metadata
			if err := meta.ValidateBasic(); err != nil {
				return err
			}

			return runInsert(ctx, resultFile, meta, &bigQuery, &firestore)
		},
	}
}

func runInsert(ctx context.Context, resultFile string, meta model.GitHubMetadata, bigQuery *config.BigQuery, firestoreConfig *config.Firestore) error {
	// Log insert configuration
	logging.Default().Info("Starting insert",
		slog.String("result_file", resultFile),
		slog.String("github_owner", meta.Owner),
		slog.String("github_repo", meta.RepoName),
		slog.String("github_branch", meta.Branch),
		slog.String("github_commit", meta.CommitID),
		slog.Any("bigquery", bigQuery),
		slog.Bool("firestore_enabled", firestoreConfig.Enabled()),
	)

	// Load Trivy report from file
	report, err := usecase.LoadTrivyReportFromFile(ctx, resultFile)
	if err != nil {
		return goerr.Wrap(err, "failed to load trivy report")
	}

	// Create BigQuery client
	bqClient, err := bigQuery.NewClient(ctx)
	if err != nil {
		return goerr.Wrap(err, "failed to create BigQuery client")
	}
	if err := requireBigQuery(bqClient); err != nil {
		return err
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
	clientOpts := []infra.Option{
		infra.WithBigQuery(bqClient),
	}
	if firestoreRepo != nil {
		clientOpts = append(clientOpts, infra.WithScanRepository(firestoreRepo))
	}
	clients := infra.New(clientOpts...)

	uc := usecase.New(clients)

	// Insert scan result to BigQuery and Firestore
	scanID, err := uc.InsertScanResult(ctx, meta, *report)
	if err != nil {
		return goerr.Wrap(err, "failed to insert scan result")
	}

	logging.Default().Info("Insert completed successfully", slog.String("scan_id", scanID.String()))

	return nil
}
