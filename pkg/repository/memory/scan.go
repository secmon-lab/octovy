package memory

import (
	"context"
	"sync"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/octovy/pkg/domain/model"
	"github.com/m-mizutani/octovy/pkg/domain/types"
	"github.com/m-mizutani/octovy/pkg/repository"
)

type repoData struct {
	repo     *model.Repository
	branches map[string]*branchData
}

type branchData struct {
	branch  *model.Branch
	targets map[string]*targetData
}

type targetData struct {
	target *model.Target
	vulns  map[string]*model.Vulnerability
}

type scanRepository struct {
	mu    sync.RWMutex
	repos map[string]*repoData
}

// Repository operations

func (r *scanRepository) CreateOrUpdateRepository(ctx context.Context, repo *model.Repository) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.repos[string(repo.ID)]; !exists {
		r.repos[string(repo.ID)] = &repoData{
			repo:     copyRepository(repo),
			branches: make(map[string]*branchData),
		}
	} else {
		r.repos[string(repo.ID)].repo = copyRepository(repo)
	}

	return nil
}

func (r *scanRepository) GetRepository(ctx context.Context, repoID types.GitHubRepoID) (*model.Repository, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	data, exists := r.repos[string(repoID)]
	if !exists {
		return nil, goerr.Wrap(repository.ErrNotFound, "repository not found",
			goerr.V("repoID", repoID),
		)
	}

	return copyRepository(data.repo), nil
}

func (r *scanRepository) ListRepositories(ctx context.Context, installationID int64) ([]*model.Repository, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var repos []*model.Repository
	for _, data := range r.repos {
		if data.repo.InstallationID == installationID {
			repos = append(repos, copyRepository(data.repo))
		}
	}

	return repos, nil
}

// Branch operations

func (r *scanRepository) CreateOrUpdateBranch(ctx context.Context, repoID types.GitHubRepoID, branch *model.Branch) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, exists := r.repos[string(repoID)]
	if !exists {
		return goerr.Wrap(repository.ErrNotFound, "repository not found",
			goerr.V("repoID", repoID),
		)
	}

	branchName := string(branch.Name)
	if _, exists := data.branches[branchName]; !exists {
		data.branches[branchName] = &branchData{
			branch:  copyBranch(branch),
			targets: make(map[string]*targetData),
		}
	} else {
		data.branches[branchName].branch = copyBranch(branch)
	}

	return nil
}

func (r *scanRepository) GetBranch(ctx context.Context, repoID types.GitHubRepoID, branchName types.BranchName) (*model.Branch, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	data, exists := r.repos[string(repoID)]
	if !exists {
		return nil, goerr.Wrap(repository.ErrNotFound, "repository not found",
			goerr.V("repoID", repoID),
		)
	}

	branchData, exists := data.branches[string(branchName)]
	if !exists {
		return nil, goerr.Wrap(repository.ErrNotFound, "branch not found",
			goerr.V("repoID", repoID),
			goerr.V("branchName", branchName),
		)
	}

	return copyBranch(branchData.branch), nil
}

func (r *scanRepository) ListBranches(ctx context.Context, repoID types.GitHubRepoID) ([]*model.Branch, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	data, exists := r.repos[string(repoID)]
	if !exists {
		return nil, goerr.Wrap(repository.ErrNotFound, "repository not found",
			goerr.V("repoID", repoID),
		)
	}

	var branches []*model.Branch
	for _, branchData := range data.branches {
		branches = append(branches, copyBranch(branchData.branch))
	}

	return branches, nil
}

// Target operations

func (r *scanRepository) CreateOrUpdateTarget(ctx context.Context, repoID types.GitHubRepoID, branchName types.BranchName, target *model.Target) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, exists := r.repos[string(repoID)]
	if !exists {
		return goerr.Wrap(repository.ErrNotFound, "repository not found",
			goerr.V("repoID", repoID),
		)
	}

	branchData, exists := data.branches[string(branchName)]
	if !exists {
		return goerr.Wrap(repository.ErrNotFound, "branch not found",
			goerr.V("repoID", repoID),
			goerr.V("branchName", branchName),
		)
	}

	targetID := string(target.ID)
	if _, exists := branchData.targets[targetID]; !exists {
		branchData.targets[targetID] = &targetData{
			target: copyTarget(target),
			vulns:  make(map[string]*model.Vulnerability),
		}
	} else {
		branchData.targets[targetID].target = copyTarget(target)
	}

	return nil
}

func (r *scanRepository) GetTarget(ctx context.Context, repoID types.GitHubRepoID, branchName types.BranchName, targetID types.TargetID) (*model.Target, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	data, exists := r.repos[string(repoID)]
	if !exists {
		return nil, goerr.Wrap(repository.ErrNotFound, "repository not found",
			goerr.V("repoID", repoID),
		)
	}

	branchData, exists := data.branches[string(branchName)]
	if !exists {
		return nil, goerr.Wrap(repository.ErrNotFound, "branch not found",
			goerr.V("repoID", repoID),
			goerr.V("branchName", branchName),
		)
	}

	targetData, exists := branchData.targets[string(targetID)]
	if !exists {
		return nil, goerr.Wrap(repository.ErrNotFound, "target not found",
			goerr.V("repoID", repoID),
			goerr.V("branchName", branchName),
			goerr.V("targetID", targetID),
		)
	}

	return copyTarget(targetData.target), nil
}

func (r *scanRepository) ListTargets(ctx context.Context, repoID types.GitHubRepoID, branchName types.BranchName) ([]*model.Target, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	data, exists := r.repos[string(repoID)]
	if !exists {
		return nil, goerr.Wrap(repository.ErrNotFound, "repository not found",
			goerr.V("repoID", repoID),
		)
	}

	branchData, exists := data.branches[string(branchName)]
	if !exists {
		return nil, goerr.Wrap(repository.ErrNotFound, "branch not found",
			goerr.V("repoID", repoID),
			goerr.V("branchName", branchName),
		)
	}

	var targets []*model.Target
	for _, targetData := range branchData.targets {
		targets = append(targets, copyTarget(targetData.target))
	}

	return targets, nil
}

// Vulnerability operations

func (r *scanRepository) ListVulnerabilities(ctx context.Context, repoID types.GitHubRepoID, branchName types.BranchName, targetID types.TargetID) ([]*model.Vulnerability, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	data, exists := r.repos[string(repoID)]
	if !exists {
		return nil, goerr.Wrap(repository.ErrNotFound, "repository not found",
			goerr.V("repoID", repoID),
		)
	}

	branchData, exists := data.branches[string(branchName)]
	if !exists {
		return nil, goerr.Wrap(repository.ErrNotFound, "branch not found",
			goerr.V("repoID", repoID),
			goerr.V("branchName", branchName),
		)
	}

	targetData, exists := branchData.targets[string(targetID)]
	if !exists {
		return nil, goerr.Wrap(repository.ErrNotFound, "target not found",
			goerr.V("repoID", repoID),
			goerr.V("branchName", branchName),
			goerr.V("targetID", targetID),
		)
	}

	var vulns []*model.Vulnerability
	for _, vuln := range targetData.vulns {
		vulns = append(vulns, copyVulnerability(vuln))
	}

	return vulns, nil
}

func (r *scanRepository) BatchCreateVulnerabilities(ctx context.Context, repoID types.GitHubRepoID, branchName types.BranchName, targetID types.TargetID, vulns []*model.Vulnerability) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, exists := r.repos[string(repoID)]
	if !exists {
		return goerr.Wrap(repository.ErrNotFound, "repository not found",
			goerr.V("repoID", repoID),
		)
	}

	branchData, exists := data.branches[string(branchName)]
	if !exists {
		return goerr.Wrap(repository.ErrNotFound, "branch not found",
			goerr.V("repoID", repoID),
			goerr.V("branchName", branchName),
		)
	}

	targetData, exists := branchData.targets[string(targetID)]
	if !exists {
		return goerr.Wrap(repository.ErrNotFound, "target not found",
			goerr.V("repoID", repoID),
			goerr.V("branchName", branchName),
			goerr.V("targetID", targetID),
		)
	}

	for _, vuln := range vulns {
		targetData.vulns[vuln.ID] = copyVulnerability(vuln)
	}

	return nil
}

func (r *scanRepository) BatchUpdateVulnerabilityStatus(ctx context.Context, repoID types.GitHubRepoID, branchName types.BranchName, targetID types.TargetID, updates map[string]types.VulnStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, exists := r.repos[string(repoID)]
	if !exists {
		return goerr.Wrap(repository.ErrNotFound, "repository not found",
			goerr.V("repoID", repoID),
		)
	}

	branchData, exists := data.branches[string(branchName)]
	if !exists {
		return goerr.Wrap(repository.ErrNotFound, "branch not found",
			goerr.V("repoID", repoID),
			goerr.V("branchName", branchName),
		)
	}

	targetData, exists := branchData.targets[string(targetID)]
	if !exists {
		return goerr.Wrap(repository.ErrNotFound, "target not found",
			goerr.V("repoID", repoID),
			goerr.V("branchName", branchName),
			goerr.V("targetID", targetID),
		)
	}

	for vulnID, status := range updates {
		if vuln, exists := targetData.vulns[vulnID]; exists {
			vuln.Status = status
		}
	}

	return nil
}

// Helper functions for deep copy

func copyRepository(repo *model.Repository) *model.Repository {
	if repo == nil {
		return nil
	}
	cpy := *repo
	return &cpy
}

func copyBranch(branch *model.Branch) *model.Branch {
	if branch == nil {
		return nil
	}
	cpy := *branch
	return &cpy
}

func copyTarget(target *model.Target) *model.Target {
	if target == nil {
		return nil
	}
	cpy := *target
	return &cpy
}

func copyVulnerability(vuln *model.Vulnerability) *model.Vulnerability {
	if vuln == nil {
		return nil
	}
	cpy := *vuln

	// Deep copy slices and maps
	if vuln.References != nil {
		cpy.References = make([]string, len(vuln.References))
		copy(cpy.References, vuln.References)
	}

	if vuln.CweIDs != nil {
		cpy.CweIDs = make([]string, len(vuln.CweIDs))
		copy(cpy.CweIDs, vuln.CweIDs)
	}

	if vuln.CVSS != nil {
		cpy.CVSS = make(map[string]model.CVSS)
		for k, v := range vuln.CVSS {
			cpy.CVSS[k] = v
		}
	}

	return &cpy
}
