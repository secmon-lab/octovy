package interfaces

import (
	"context"

	"github.com/m-mizutani/octovy/pkg/domain/model"
	"github.com/m-mizutani/octovy/pkg/domain/types"
)

//go:generate moq -out ../mock/scan_repository_mock.go -pkg mock . ScanRepository

// ScanRepository manages scan information for GitHub repositories
type ScanRepository interface {
	// Repository operations
	CreateOrUpdateRepository(ctx context.Context, repo *model.Repository) error
	GetRepository(ctx context.Context, repoID types.GitHubRepoID) (*model.Repository, error)
	ListRepositories(ctx context.Context, installationID int64) ([]*model.Repository, error)

	// Branch operations
	CreateOrUpdateBranch(ctx context.Context, repoID types.GitHubRepoID, branch *model.Branch) error
	GetBranch(ctx context.Context, repoID types.GitHubRepoID, branchName types.BranchName) (*model.Branch, error)
	ListBranches(ctx context.Context, repoID types.GitHubRepoID) ([]*model.Branch, error)

	// Target operations
	CreateOrUpdateTarget(ctx context.Context, repoID types.GitHubRepoID, branchName types.BranchName, target *model.Target) error
	GetTarget(ctx context.Context, repoID types.GitHubRepoID, branchName types.BranchName, targetID types.TargetID) (*model.Target, error)
	ListTargets(ctx context.Context, repoID types.GitHubRepoID, branchName types.BranchName) ([]*model.Target, error)

	// Vulnerability operations (batch only)
	ListVulnerabilities(ctx context.Context, repoID types.GitHubRepoID, branchName types.BranchName, targetID types.TargetID) ([]*model.Vulnerability, error)
	BatchCreateVulnerabilities(ctx context.Context, repoID types.GitHubRepoID, branchName types.BranchName, targetID types.TargetID, vulns []*model.Vulnerability) error
	BatchUpdateVulnerabilityStatus(ctx context.Context, repoID types.GitHubRepoID, branchName types.BranchName, targetID types.TargetID, updates map[string]types.VulnStatus) error
}
