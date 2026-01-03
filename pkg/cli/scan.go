package cli

import (
	"context"
	"os/exec"
	"strings"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gots/slice"
	"github.com/m-mizutani/octovy/pkg/cli/config"
	"github.com/m-mizutani/octovy/pkg/domain/model"
	"github.com/m-mizutani/octovy/pkg/infra"
	trivyInfra "github.com/m-mizutani/octovy/pkg/infra/trivy"
	"github.com/m-mizutani/octovy/pkg/usecase"
	"github.com/urfave/cli/v3"
)

func scanCommand() *cli.Command {
	var (
		bigQuery  config.BigQuery
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
		}, bigQuery.Flags()),
		Action: func(ctx context.Context, c *cli.Command) error {

			// Auto-detect GitHub metadata from git if not specified
			if err := autoDetectGitMetadata(ctx, &meta); err != nil {
				return err
			}

			return runScan(ctx, dir, trivyPath, meta, &bigQuery)
		},
	}
}

// autoDetectGitMetadata auto-detects GitHub metadata from git command if not already set
func autoDetectGitMetadata(ctx context.Context, meta *model.GitHubMetadata) error {
	if meta.CommitID == "" {
		cmd := exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
		output, err := cmd.Output()
		if err != nil {
			return goerr.Wrap(err, "failed to get commit ID from git")
		}
		meta.CommitID = strings.TrimSpace(string(output))
	}

	if meta.Owner == "" || meta.RepoName == "" {
		cmd := exec.CommandContext(ctx, "git", "remote", "get-url", "origin")
		output, err := cmd.Output()
		if err != nil {
			return goerr.Wrap(err, "failed to get remote URL from git")
		}

		// Parse git remote URL (e.g., git@github.com:owner/repo.git or https://github.com/owner/repo.git)
		url := strings.TrimSpace(string(output))
		var owner, repo string

		if strings.HasPrefix(url, "git@github.com:") {
			// SSH format: git@github.com:owner/repo.git
			parts := strings.TrimPrefix(url, "git@github.com:")
			parts = strings.TrimSuffix(parts, ".git")
			ownerRepo := strings.Split(parts, "/")
			if len(ownerRepo) == 2 {
				owner = ownerRepo[0]
				repo = ownerRepo[1]
			}
		} else if strings.Contains(url, "github.com/") {
			// HTTPS format: https://github.com/owner/repo.git
			parts := strings.Split(url, "github.com/")
			if len(parts) == 2 {
				parts[1] = strings.TrimSuffix(parts[1], ".git")
				ownerRepo := strings.Split(parts[1], "/")
				if len(ownerRepo) == 2 {
					owner = ownerRepo[0]
					repo = ownerRepo[1]
				}
			}
		}

		if owner == "" || repo == "" {
			return goerr.New("failed to parse GitHub owner/repo from git remote URL", goerr.V("url", url))
		}

		if meta.Owner == "" {
			meta.Owner = owner
		}
		if meta.RepoName == "" {
			meta.RepoName = repo
		}
	}

	return nil
}

func runScan(ctx context.Context, dir, trivyPath string, meta model.GitHubMetadata, bigQuery *config.BigQuery) error {
	// Create BigQuery client if configured
	bqClient, err := bigQuery.NewClient(ctx)
	if err != nil {
		return goerr.Wrap(err, "failed to create BigQuery client")
	}

	// Create clients and usecase
	trivyClient := trivyInfra.New(trivyPath)
	clients := infra.New(
		infra.WithTrivy(trivyClient),
		infra.WithBigQuery(bqClient),
	)

	uc := usecase.New(clients, usecase.WithBigQueryTableID(bigQuery.TableID()))

	// Scan directory and insert to BigQuery
	if err := uc.ScanAndInsert(ctx, dir, meta); err != nil {
		return goerr.Wrap(err, "failed to scan local directory")
	}

	return nil
}
