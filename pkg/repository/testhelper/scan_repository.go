package testhelper

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/octovy/pkg/domain/interfaces"
	"github.com/m-mizutani/octovy/pkg/domain/model"
	"github.com/m-mizutani/octovy/pkg/domain/types"
	"github.com/m-mizutani/octovy/pkg/repository"
)

// TestAll runs all test cases for ScanRepository
// This is the main entry point for testing any ScanRepository implementation
func TestAll(t *testing.T, repo interfaces.ScanRepository) {
	t.Run("RepositoryCRUD", func(t *testing.T) {
		TestRepositoryCRUD(t, repo)
	})
	t.Run("BranchCRUD", func(t *testing.T) {
		TestBranchCRUD(t, repo)
	})
	t.Run("BranchWithSlash", func(t *testing.T) {
		TestBranchWithSlash(t, repo)
	})
	t.Run("TargetCRUD", func(t *testing.T) {
		TestTargetCRUD(t, repo)
	})
	t.Run("VulnerabilityBatchOps", func(t *testing.T) {
		TestVulnerabilityBatchOps(t, repo)
	})
	t.Run("VulnerabilityStatusUpdate", func(t *testing.T) {
		TestVulnerabilityStatusUpdate(t, repo)
	})
}

// TestRepositoryCRUD tests basic CRUD operations for Repository
func TestRepositoryCRUD(t *testing.T, repo interfaces.ScanRepository) {
	ctx := context.Background()

	// Generate unique IDs for this test
	owner := fmt.Sprintf("owner-%s", uuid.New().String()[:8])
	repoName := fmt.Sprintf("repo-%s", uuid.New().String()[:8])
	repoID := types.GitHubRepoID(fmt.Sprintf("%s/%s", owner, repoName))

	// Create a repository
	now := time.Now()
	testRepo := &model.Repository{
		ID:             repoID,
		Owner:          owner,
		Name:           repoName,
		DefaultBranch:  "main",
		InstallationID: 12345,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	err := repo.CreateOrUpdateRepository(ctx, testRepo)
	gt.NoError(t, err)

	// Get the repository
	retrieved, err := repo.GetRepository(ctx, repoID)
	gt.NoError(t, err)
	gt.V(t, retrieved.ID).Equal(testRepo.ID)
	gt.V(t, retrieved.Owner).Equal(testRepo.Owner)
	gt.V(t, retrieved.Name).Equal(testRepo.Name)
	gt.V(t, retrieved.DefaultBranch).Equal(testRepo.DefaultBranch)
	gt.V(t, retrieved.InstallationID).Equal(testRepo.InstallationID)

	// Update the repository
	testRepo.DefaultBranch = "develop"
	testRepo.UpdatedAt = time.Now()
	err = repo.CreateOrUpdateRepository(ctx, testRepo)
	gt.NoError(t, err)

	// Verify update
	retrieved, err = repo.GetRepository(ctx, repoID)
	gt.NoError(t, err)
	gt.V(t, retrieved.DefaultBranch).Equal(types.BranchName("develop"))

	// List repositories by installation ID
	repos, err := repo.ListRepositories(ctx, 12345)
	gt.NoError(t, err)
	gt.V(t, len(repos)).Equal(1)
	gt.V(t, repos[0].ID).Equal(testRepo.ID)

	// Test not found
	nonExistentID := types.GitHubRepoID(fmt.Sprintf("nonexistent-%s/repo-%s", uuid.New().String()[:8], uuid.New().String()[:8]))
	_, err = repo.GetRepository(ctx, nonExistentID)
	gt.Error(t, err)
	gt.True(t, errors.Is(err, repository.ErrNotFound))
}

// TestBranchCRUD tests basic CRUD operations for Branch
func TestBranchCRUD(t *testing.T, repo interfaces.ScanRepository) {
	ctx := context.Background()

	// Generate unique IDs for this test
	owner := fmt.Sprintf("owner-%s", uuid.New().String()[:8])
	repoName := fmt.Sprintf("repo-%s", uuid.New().String()[:8])
	repoID := types.GitHubRepoID(fmt.Sprintf("%s/%s", owner, repoName))

	// First create a repository
	now := time.Now()
	testRepo := &model.Repository{
		ID:             repoID,
		Owner:          owner,
		Name:           repoName,
		DefaultBranch:  "main",
		InstallationID: 12345,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	err := repo.CreateOrUpdateRepository(ctx, testRepo)
	gt.NoError(t, err)

	// Create a branch
	testBranch := &model.Branch{
		Name:          "main",
		LastScanID:    "scan-123",
		LastScanAt:    now,
		LastCommitSHA: "abc123",
		Status:        types.ScanStatusSuccess,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	err = repo.CreateOrUpdateBranch(ctx, repoID, testBranch)
	gt.NoError(t, err)

	// Get the branch
	retrieved, err := repo.GetBranch(ctx, repoID, "main")
	gt.NoError(t, err)
	gt.V(t, retrieved.Name).Equal(testBranch.Name)
	gt.V(t, retrieved.LastScanID).Equal(testBranch.LastScanID)
	gt.V(t, retrieved.LastCommitSHA).Equal(testBranch.LastCommitSHA)
	gt.V(t, retrieved.Status).Equal(testBranch.Status)

	// Update the branch
	testBranch.LastScanID = "scan-456"
	testBranch.LastCommitSHA = "def456"
	testBranch.Status = "failed"
	testBranch.UpdatedAt = time.Now()

	err = repo.CreateOrUpdateBranch(ctx, repoID, testBranch)
	gt.NoError(t, err)

	// Verify update
	retrieved, err = repo.GetBranch(ctx, repoID, "main")
	gt.NoError(t, err)
	gt.V(t, retrieved.LastScanID).Equal(types.ScanID("scan-456"))
	gt.V(t, retrieved.LastCommitSHA).Equal("def456")
	gt.V(t, retrieved.Status).Equal("failed")

	// List branches
	branches, err := repo.ListBranches(ctx, repoID)
	gt.NoError(t, err)
	gt.V(t, len(branches)).Equal(1)
	gt.V(t, branches[0].Name).Equal(testBranch.Name)

	// Test not found
	nonExistentBranch := types.BranchName(fmt.Sprintf("branch-%s", uuid.New().String()[:8]))
	_, err = repo.GetBranch(ctx, repoID, nonExistentBranch)
	gt.Error(t, err)
	gt.True(t, errors.Is(err, repository.ErrNotFound))
}

// TestTargetCRUD tests basic CRUD operations for Target
func TestTargetCRUD(t *testing.T, repo interfaces.ScanRepository) {
	ctx := context.Background()

	// Generate unique IDs for this test
	owner := fmt.Sprintf("owner-%s", uuid.New().String()[:8])
	repoName := fmt.Sprintf("repo-%s", uuid.New().String()[:8])
	repoID := types.GitHubRepoID(fmt.Sprintf("%s/%s", owner, repoName))
	targetID := types.TargetID(fmt.Sprintf("target-%s", uuid.New().String()[:8]))

	// Setup: create repository and branch
	now := time.Now()
	testRepo := &model.Repository{
		ID:             repoID,
		Owner:          owner,
		Name:           repoName,
		DefaultBranch:  "main",
		InstallationID: 12345,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	err := repo.CreateOrUpdateRepository(ctx, testRepo)
	gt.NoError(t, err)

	testBranch := &model.Branch{
		Name:          "main",
		LastScanID:    "scan-123",
		LastScanAt:    now,
		LastCommitSHA: "abc123",
		Status:        types.ScanStatusSuccess,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	err = repo.CreateOrUpdateBranch(ctx, repoID, testBranch)
	gt.NoError(t, err)

	// Create a target
	testTarget := &model.Target{
		ID:        targetID,
		Target:    "go.mod",
		Class:     "lang-pkgs",
		Type:      "gomod",
		CreatedAt: now,
		UpdatedAt: now,
	}

	err = repo.CreateOrUpdateTarget(ctx, repoID, "main", testTarget)
	gt.NoError(t, err)

	// Get the target
	retrieved, err := repo.GetTarget(ctx, repoID, "main", targetID)
	gt.NoError(t, err)
	gt.V(t, retrieved.ID).Equal(testTarget.ID)
	gt.V(t, retrieved.Target).Equal(testTarget.Target)
	gt.V(t, retrieved.Class).Equal(testTarget.Class)
	gt.V(t, retrieved.Type).Equal(testTarget.Type)

	// Update the target
	testTarget.Target = "package.json"
	testTarget.Type = "npm"
	testTarget.UpdatedAt = time.Now()

	err = repo.CreateOrUpdateTarget(ctx, repoID, "main", testTarget)
	gt.NoError(t, err)

	// Verify update
	retrieved, err = repo.GetTarget(ctx, repoID, "main", targetID)
	gt.NoError(t, err)
	gt.V(t, retrieved.Target).Equal("package.json")
	gt.V(t, retrieved.Type).Equal("npm")

	// List targets
	targets, err := repo.ListTargets(ctx, repoID, "main")
	gt.NoError(t, err)
	gt.V(t, len(targets)).Equal(1)
	gt.V(t, targets[0].ID).Equal(testTarget.ID)

	// Test not found
	_, err = repo.GetTarget(ctx, repoID, "main", "nonexistent")
	gt.Error(t, err)
	gt.True(t, errors.Is(err, repository.ErrNotFound))
}

// TestVulnerabilityBatchOps tests batch operations for vulnerabilities
func TestVulnerabilityBatchOps(t *testing.T, repo interfaces.ScanRepository) {
	ctx := context.Background()

	// Generate unique IDs for this test
	owner := fmt.Sprintf("owner-%s", uuid.New().String()[:8])
	repoName := fmt.Sprintf("repo-%s", uuid.New().String()[:8])
	repoID := types.GitHubRepoID(fmt.Sprintf("%s/%s", owner, repoName))
	targetID := types.TargetID(fmt.Sprintf("target-%s", uuid.New().String()[:8]))

	// Setup: create repository, branch, and target
	now := time.Now()
	testRepo := &model.Repository{
		ID:             repoID,
		Owner:          owner,
		Name:           repoName,
		DefaultBranch:  "main",
		InstallationID: 12345,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	err := repo.CreateOrUpdateRepository(ctx, testRepo)
	gt.NoError(t, err)

	testBranch := &model.Branch{
		Name:          "main",
		LastScanID:    "scan-123",
		LastScanAt:    now,
		LastCommitSHA: "abc123",
		Status:        types.ScanStatusSuccess,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	err = repo.CreateOrUpdateBranch(ctx, repoID, testBranch)
	gt.NoError(t, err)

	testTarget := &model.Target{
		ID:        targetID,
		Target:    "go.mod",
		Class:     "lang-pkgs",
		Type:      "gomod",
		CreatedAt: now,
		UpdatedAt: now,
	}
	err = repo.CreateOrUpdateTarget(ctx, repoID, "main", testTarget)
	gt.NoError(t, err)

	// Create multiple vulnerabilities
	vulns := []*model.Vulnerability{
		{
			ID:               "CVE-2021-0001",
			PkgName:          "package1",
			PkgPath:          "/path/to/package1",
			InstalledVersion: "1.0.0",
			FixedVersion:     "1.0.1",
			Severity:         "HIGH",
			Title:            "Test Vulnerability 1",
			Description:      "Test description 1",
			References:       []string{"https://example.com/1"},
			PrimaryURL:       "https://example.com/1",
			CweIDs:           []string{"CWE-79"},
			CVSS: map[string]model.CVSS{
				"nvd": {V3Score: 7.5, V3Vector: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:N/A:N"},
			},
			PublishedDate:    "2021-01-01",
			LastModifiedDate: "2021-01-02",
			Status:           types.VulnStatusActive,
			CreatedAt:        now,
			UpdatedAt:        now,
		},
		{
			ID:               "CVE-2021-0002",
			PkgName:          "package2",
			PkgPath:          "/path/to/package2",
			InstalledVersion: "2.0.0",
			FixedVersion:     "2.0.1",
			Severity:         "CRITICAL",
			Title:            "Test Vulnerability 2",
			Description:      "Test description 2",
			References:       []string{"https://example.com/2"},
			PrimaryURL:       "https://example.com/2",
			CweIDs:           []string{"CWE-89"},
			CVSS: map[string]model.CVSS{
				"nvd": {V3Score: 9.8, V3Vector: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H"},
			},
			PublishedDate:    "2021-02-01",
			LastModifiedDate: "2021-02-02",
			Status:           types.VulnStatusActive,
			CreatedAt:        now,
			UpdatedAt:        now,
		},
	}

	err = repo.BatchCreateVulnerabilities(ctx, repoID, "main", targetID, vulns)
	gt.NoError(t, err)

	// List vulnerabilities
	retrieved, err := repo.ListVulnerabilities(ctx, repoID, "main", targetID)
	gt.NoError(t, err)
	gt.V(t, len(retrieved)).Equal(2)

	// Verify content
	vulnMap := make(map[string]*model.Vulnerability)
	for _, v := range retrieved {
		vulnMap[v.ID] = v
	}

	v1 := vulnMap["CVE-2021-0001"]
	gt.V(t, v1).NotEqual(nil)
	gt.V(t, v1.PkgName).Equal("package1")
	gt.V(t, v1.Severity).Equal("HIGH")
	gt.V(t, v1.Status).Equal(types.VulnStatusActive)
	gt.V(t, v1.CVSS["nvd"].V3Score).Equal(7.5)

	v2 := vulnMap["CVE-2021-0002"]
	gt.V(t, v2).NotEqual(nil)
	gt.V(t, v2.PkgName).Equal("package2")
	gt.V(t, v2.Severity).Equal("CRITICAL")
	gt.V(t, v2.Status).Equal(types.VulnStatusActive)
	gt.V(t, v2.CVSS["nvd"].V3Score).Equal(9.8)
}

// TestVulnerabilityStatusUpdate tests batch status update
func TestVulnerabilityStatusUpdate(t *testing.T, repo interfaces.ScanRepository) {
	ctx := context.Background()

	// Generate unique IDs for this test
	owner := fmt.Sprintf("owner-%s", uuid.New().String()[:8])
	repoName := fmt.Sprintf("repo-%s", uuid.New().String()[:8])
	repoID := types.GitHubRepoID(fmt.Sprintf("%s/%s", owner, repoName))
	targetID := types.TargetID(fmt.Sprintf("target-%s", uuid.New().String()[:8]))

	// Setup: create repository, branch, target, and vulnerabilities
	now := time.Now()
	testRepo := &model.Repository{
		ID:             repoID,
		Owner:          owner,
		Name:           repoName,
		DefaultBranch:  "main",
		InstallationID: 12345,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	err := repo.CreateOrUpdateRepository(ctx, testRepo)
	gt.NoError(t, err)

	testBranch := &model.Branch{
		Name:          "main",
		LastScanID:    "scan-123",
		LastScanAt:    now,
		LastCommitSHA: "abc123",
		Status:        types.ScanStatusSuccess,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	err = repo.CreateOrUpdateBranch(ctx, repoID, testBranch)
	gt.NoError(t, err)

	testTarget := &model.Target{
		ID:        targetID,
		Target:    "go.mod",
		Class:     "lang-pkgs",
		Type:      "gomod",
		CreatedAt: now,
		UpdatedAt: now,
	}
	err = repo.CreateOrUpdateTarget(ctx, repoID, "main", testTarget)
	gt.NoError(t, err)

	// Create vulnerabilities
	vulns := []*model.Vulnerability{
		{
			ID:               "CVE-2021-0001",
			PkgName:          "package1",
			InstalledVersion: "1.0.0",
			FixedVersion:     "1.0.1",
			Severity:         "HIGH",
			Status:           types.VulnStatusActive,
			CreatedAt:        now,
			UpdatedAt:        now,
		},
		{
			ID:               "CVE-2021-0002",
			PkgName:          "package2",
			InstalledVersion: "2.0.0",
			FixedVersion:     "2.0.1",
			Severity:         "CRITICAL",
			Status:           types.VulnStatusActive,
			CreatedAt:        now,
			UpdatedAt:        now,
		},
	}

	err = repo.BatchCreateVulnerabilities(ctx, repoID, "main", targetID, vulns)
	gt.NoError(t, err)

	// Update status to fixed
	updates := map[string]types.VulnStatus{
		"CVE-2021-0001": types.VulnStatusFixed,
	}

	err = repo.BatchUpdateVulnerabilityStatus(ctx, repoID, "main", targetID, updates)
	gt.NoError(t, err)

	// Verify status update
	retrieved, err := repo.ListVulnerabilities(ctx, repoID, "main", targetID)
	gt.NoError(t, err)

	vulnMap := make(map[string]*model.Vulnerability)
	for _, v := range retrieved {
		vulnMap[v.ID] = v
	}

	gt.V(t, vulnMap["CVE-2021-0001"].Status).Equal(types.VulnStatusFixed)
	gt.V(t, vulnMap["CVE-2021-0002"].Status).Equal(types.VulnStatusActive)
}

// TestBranchWithSlash tests branch names containing "/" which must be safely converted for Firestore
func TestBranchWithSlash(t *testing.T, repo interfaces.ScanRepository) {
	ctx := context.Background()

	// Generate unique IDs for this test
	owner := fmt.Sprintf("owner-%s", uuid.New().String()[:8])
	repoName := fmt.Sprintf("repo-%s", uuid.New().String()[:8])
	repoID := types.GitHubRepoID(fmt.Sprintf("%s/%s", owner, repoName))

	// First create a repository
	now := time.Now()
	testRepo := &model.Repository{
		ID:             repoID,
		Owner:          owner,
		Name:           repoName,
		DefaultBranch:  "main",
		InstallationID: 12345,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	err := repo.CreateOrUpdateRepository(ctx, testRepo)
	gt.NoError(t, err)

	// Test various branch names with "/"
	testCases := []struct {
		name       string
		branchName types.BranchName
	}{
		{
			name:       "feature branch",
			branchName: "feature/foo",
		},
		{
			name:       "multi-level feature branch",
			branchName: "feature/firestore-scan-repository",
		},
		{
			name:       "deeply nested branch",
			branchName: "hotfix/bug/urgent/fix",
		},
		{
			name:       "release branch",
			branchName: "release/v1.0.0",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a branch with "/" in name
			testBranch := &model.Branch{
				Name:          tc.branchName,
				LastScanID:    "scan-123",
				LastScanAt:    now,
				LastCommitSHA: "abc123",
				Status:        types.ScanStatusSuccess,
				CreatedAt:     now,
				UpdatedAt:     now,
			}

			err = repo.CreateOrUpdateBranch(ctx, repoID, testBranch)
			gt.NoError(t, err)

			// Get the branch back
			retrieved, err := repo.GetBranch(ctx, repoID, tc.branchName)
			gt.NoError(t, err)
			gt.V(t, retrieved.Name).Equal(tc.branchName)
			gt.V(t, retrieved.LastScanID).Equal(testBranch.LastScanID)
			gt.V(t, retrieved.Status).Equal(testBranch.Status)

			// Update the branch
			testBranch.LastScanID = "scan-456"
			testBranch.Status = types.ScanStatusFailure
			err = repo.CreateOrUpdateBranch(ctx, repoID, testBranch)
			gt.NoError(t, err)

			// Verify update
			retrieved, err = repo.GetBranch(ctx, repoID, tc.branchName)
			gt.NoError(t, err)
			gt.V(t, retrieved.LastScanID).Equal(types.ScanID("scan-456"))
			gt.V(t, retrieved.Status).Equal(types.ScanStatusFailure)
		})
	}

	// List all branches and verify they all exist
	branches, err := repo.ListBranches(ctx, repoID)
	gt.NoError(t, err)
	gt.V(t, len(branches)).Equal(len(testCases))

	// Verify all branch names are preserved correctly
	branchNames := make(map[types.BranchName]bool)
	for _, b := range branches {
		branchNames[b.Name] = true
	}

	for _, tc := range testCases {
		gt.True(t, branchNames[tc.branchName])
	}
}
