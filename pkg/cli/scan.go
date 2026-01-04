package cli

import (
	"context"
	"log/slog"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gots/slice"
	"github.com/m-mizutani/octovy/pkg/cli/config"
	"github.com/m-mizutani/octovy/pkg/domain/interfaces"
	"github.com/m-mizutani/octovy/pkg/domain/model"
	"github.com/m-mizutani/octovy/pkg/infra"
	trivyInfra "github.com/m-mizutani/octovy/pkg/infra/trivy"
	"github.com/m-mizutani/octovy/pkg/usecase"
	"github.com/m-mizutani/octovy/pkg/utils/logging"
	"github.com/urfave/cli/v3"
)

func scanCommand() *cli.Command {
	var (
		bigQuery  config.BigQuery
		firestore config.Firestore
		dir       string
		trivyPath string
		meta      model.GitHubMetadata
	)

	return &cli.Command{
		Name:    "scan",
		Aliases: []string{"sc"},
		Usage:   "Scan local directory with Trivy and insert results to BigQuery",
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
			if err := autoDetectGitMetadata(ctx, &meta); err != nil {
				return err
			}

			return runScan(ctx, dir, trivyPath, meta, &bigQuery, &firestore)
		},
	}
}

// autoDetectGitMetadata auto-detects GitHub metadata from git repository if not already set
func autoDetectGitMetadata(ctx context.Context, meta *model.GitHubMetadata) error {
	repo, err := git.PlainOpen(".")
	if err != nil {
		return goerr.Wrap(err, "failed to open git repository")
	}

	if meta.CommitID == "" {
		head, err := repo.Head()
		if err != nil {
			return goerr.Wrap(err, "failed to get HEAD")
		}
		meta.CommitID = head.Hash().String()
	}

	if meta.Branch == "" {
		head, err := repo.Head()
		if err != nil {
			return goerr.Wrap(err, "failed to get HEAD for branch")
		}
		if head.Name().IsBranch() {
			meta.Branch = head.Name().Short()
		}
	}

	if meta.Owner == "" || meta.RepoName == "" {
		remote, err := repo.Remote("origin")
		if err != nil {
			return goerr.Wrap(err, "failed to get remote origin")
		}

		if len(remote.Config().URLs) == 0 {
			return goerr.New("no remote URL found")
		}

		// Parse git remote URL (e.g., git@github.com:owner/repo.git or https://github.com/owner/repo.git)
		url := remote.Config().URLs[0]
		var owner, repoName string

		if strings.HasPrefix(url, "git@github.com:") {
			// SSH format: git@github.com:owner/repo.git
			parts := strings.TrimPrefix(url, "git@github.com:")
			parts = strings.TrimSuffix(parts, ".git")
			ownerRepo := strings.Split(parts, "/")
			if len(ownerRepo) == 2 {
				owner = ownerRepo[0]
				repoName = ownerRepo[1]
			}
		} else if strings.Contains(url, "github.com/") {
			// HTTPS format: https://github.com/owner/repo.git
			parts := strings.Split(url, "github.com/")
			if len(parts) == 2 {
				parts[1] = strings.TrimSuffix(parts[1], ".git")
				ownerRepo := strings.Split(parts[1], "/")
				if len(ownerRepo) == 2 {
					owner = ownerRepo[0]
					repoName = ownerRepo[1]
				}
			}
		}

		if owner == "" || repoName == "" {
			return goerr.New("failed to parse GitHub owner/repo from git remote URL", goerr.V("url", url))
		}

		if meta.Owner == "" {
			meta.Owner = owner
		}
		if meta.RepoName == "" {
			meta.RepoName = repoName
		}
	}

	return nil
}

func runScan(ctx context.Context, dir, trivyPath string, meta model.GitHubMetadata, bigQuery *config.BigQuery, firestoreConfig *config.Firestore) error {
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
