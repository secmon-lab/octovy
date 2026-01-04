package cli

import (
	"context"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/octovy/pkg/domain/model"
)

// AutoDetectGitMetadata auto-detects GitHub metadata from git repository if not already set
func AutoDetectGitMetadata(ctx context.Context, meta *model.GitHubMetadata) error {
	repo, err := git.PlainOpen(".")
	if err != nil {
		return goerr.Wrap(err, "failed to open git repository")
	}

	if meta.CommitID == "" || meta.Branch == "" {
		head, err := repo.Head()
		if err != nil {
			return goerr.Wrap(err, "failed to get HEAD")
		}

		if meta.CommitID == "" {
			meta.CommitID = head.Hash().String()
		}

		if meta.Branch == "" && head.Name().IsBranch() {
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
