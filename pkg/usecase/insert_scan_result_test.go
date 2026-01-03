package usecase_test

import (
	"context"
	"testing"

	"cloud.google.com/go/bigquery"
	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/octovy/pkg/domain/mock"
	"github.com/m-mizutani/octovy/pkg/domain/model"
	"github.com/m-mizutani/octovy/pkg/domain/model/trivy"
	"github.com/m-mizutani/octovy/pkg/domain/types"
	"github.com/m-mizutani/octovy/pkg/infra"
	"github.com/m-mizutani/octovy/pkg/repository/memory"
	"github.com/m-mizutani/octovy/pkg/usecase"
)

func TestInsertScanResult(t *testing.T) {
	t.Run("insert scan result to BigQuery and Firestore", func(t *testing.T) {
		mockBQ := &mock.BigQueryMock{}
		memRepo := memory.New()
		uc := usecase.New(infra.New(
			infra.WithBigQuery(mockBQ),
			infra.WithScanRepository(memRepo),
		))

		ctx := context.Background()

		var bqInsertCalled bool
		mockBQ.InsertFunc = func(ctx context.Context, schema bigquery.Schema, data any) error {
			bqInsertCalled = true
			return nil
		}

		mockBQ.GetMetadataFunc = func(ctx context.Context) (*bigquery.TableMetadata, error) {
			return nil, nil
		}

		mockBQ.CreateTableFunc = func(ctx context.Context, md *bigquery.TableMetadata) error {
			return nil
		}

		meta := model.GitHubMetadata{
			GitHubCommit: model.GitHubCommit{
				GitHubRepo: model.GitHubRepo{
					Owner:    "test-owner",
					RepoName: "test-repo",
					RepoID:   123,
				},
				Branch:   "main",
				CommitID: "0000000000000000000000000000000000000000",
			},
			InstallationID: 456,
		}
		report := trivy.Report{
			SchemaVersion: 2,
			Results: []trivy.Result{
				{
					Target: "test-target",
					Class:  "os-pkgs",
					Type:   "alpine",
					Vulnerabilities: []trivy.DetectedVulnerability{
						{
							VulnerabilityID:  "CVE-2024-0001",
							PkgName:          "test-pkg",
							InstalledVersion: "1.0.0",
							Vulnerability: trivy.Vulnerability{
								Severity: "HIGH",
							},
						},
					},
				},
			},
		}

		gt.NoError(t, uc.InsertScanResult(ctx, meta, report))

		gt.True(t, bqInsertCalled)

		// Verify Firestore data
		repoID := types.GitHubRepoID("test-owner/test-repo")
		repo, err := memRepo.GetRepository(ctx, repoID)
		gt.NoError(t, err)
		gt.V(t, repo.Owner).Equal("test-owner")
		gt.V(t, repo.Name).Equal("test-repo")

		branch, err := memRepo.GetBranch(ctx, repoID, "main")
		gt.NoError(t, err)
		gt.V(t, string(branch.Name)).Equal("main")

		target, err := memRepo.GetTarget(ctx, repoID, "main", "test-target")
		gt.NoError(t, err)
		gt.V(t, string(target.ID)).Equal("test-target")

		vulns, err := memRepo.ListVulnerabilities(ctx, repoID, "main", "test-target")
		gt.NoError(t, err)
		gt.V(t, len(vulns)).Equal(1)
		gt.V(t, vulns[0].ID).Equal("CVE-2024-0001")
		gt.V(t, vulns[0].Status).Equal(types.VulnStatusActive)
	})

	t.Run("insert scan result to BigQuery", func(t *testing.T) {
		mockBQ := &mock.BigQueryMock{}
		uc := usecase.New(infra.New(
			infra.WithBigQuery(mockBQ),
		))

		ctx := context.Background()

		var insertCalled bool
		mockBQ.InsertFunc = func(ctx context.Context, schema bigquery.Schema, data any) error {
			insertCalled = true
			return nil
		}

		mockBQ.GetMetadataFunc = func(ctx context.Context) (*bigquery.TableMetadata, error) {
			return nil, nil
		}

		mockBQ.CreateTableFunc = func(ctx context.Context, md *bigquery.TableMetadata) error {
			return nil
		}

		meta := model.GitHubMetadata{
			GitHubCommit: model.GitHubCommit{
				GitHubRepo: model.GitHubRepo{
					Owner:    "test-owner",
					RepoName: "test-repo",
					RepoID:   123,
				},
				Branch:   "main",
				CommitID: "0000000000000000000000000000000000000000",
			},
			InstallationID: 456,
		}
		report := trivy.Report{
			SchemaVersion: 2,
		}

		gt.NoError(t, uc.InsertScanResult(ctx, meta, report))

		gt.True(t, insertCalled)
	})

	t.Run("table creation when metadata is nil", func(t *testing.T) {
		mockBQ := &mock.BigQueryMock{}
		uc := usecase.New(infra.New(
			infra.WithBigQuery(mockBQ),
		))

		ctx := context.Background()

		var createTableCalled bool
		mockBQ.CreateTableFunc = func(ctx context.Context, md *bigquery.TableMetadata) error {
			createTableCalled = true
			gt.V(t, md).NotEqual(nil)
			return nil
		}

		mockBQ.GetMetadataFunc = func(ctx context.Context) (*bigquery.TableMetadata, error) {
			return nil, nil // No existing table
		}

		mockBQ.InsertFunc = func(ctx context.Context, schema bigquery.Schema, data any) error {
			return nil
		}

		meta := model.GitHubMetadata{
			GitHubCommit: model.GitHubCommit{
				GitHubRepo: model.GitHubRepo{
					Owner:    "test-owner",
					RepoName: "test-repo",
					RepoID:   123,
				},
				Branch:   "main",
				CommitID: "0000000000000000000000000000000000000000",
			},
			InstallationID: 456,
		}
		report := trivy.Report{SchemaVersion: 2}

		gt.NoError(t, uc.InsertScanResult(ctx, meta, report))
		gt.True(t, createTableCalled)
	})

	t.Run("insert with vulnerabilities and packages", func(t *testing.T) {
		mockBQ := &mock.BigQueryMock{}
		uc := usecase.New(infra.New(
			infra.WithBigQuery(mockBQ),
		))

		ctx := context.Background()

		var insertedData any
		mockBQ.InsertFunc = func(ctx context.Context, schema bigquery.Schema, data any) error {
			insertedData = data
			return nil
		}

		mockBQ.GetMetadataFunc = func(ctx context.Context) (*bigquery.TableMetadata, error) {
			return &bigquery.TableMetadata{
				Schema: bigquery.Schema{},
				ETag:   "test-etag",
			}, nil
		}

		mockBQ.CreateTableFunc = func(ctx context.Context, md *bigquery.TableMetadata) error {
			return nil
		}

		mockBQ.UpdateTableFunc = func(ctx context.Context, md bigquery.TableMetadataToUpdate, eTag string) error {
			return nil
		}

		meta := model.GitHubMetadata{
			GitHubCommit: model.GitHubCommit{
				GitHubRepo: model.GitHubRepo{
					Owner:    "test-owner",
					RepoName: "test-repo",
					RepoID:   123,
				},
				Branch:   "main",
				CommitID: "abc1234567890123456789012345678901234567",
			},
			InstallationID: 456,
		}
		report := trivy.Report{
			SchemaVersion: 2,
			Results: []trivy.Result{
				{
					Target: "test-target",
					Vulnerabilities: []trivy.DetectedVulnerability{
						{
							VulnerabilityID:  "CVE-2024-0001",
							PkgName:          "test-pkg",
							InstalledVersion: "1.0.0",
							Vulnerability: trivy.Vulnerability{
								Severity: "HIGH",
							},
						},
					},
				},
			},
		}

		gt.NoError(t, uc.InsertScanResult(ctx, meta, report))
		gt.V(t, insertedData).NotEqual(nil)
	})

	t.Run("vulnerability status transitions", func(t *testing.T) {
		mockBQ := &mock.BigQueryMock{}
		memRepo := memory.New()
		uc := usecase.New(infra.New(
			infra.WithBigQuery(mockBQ),
			infra.WithScanRepository(memRepo),
		))

		ctx := context.Background()

		mockBQ.InsertFunc = func(ctx context.Context, schema bigquery.Schema, data any) error {
			return nil
		}
		mockBQ.GetMetadataFunc = func(ctx context.Context) (*bigquery.TableMetadata, error) {
			return nil, nil
		}
		mockBQ.CreateTableFunc = func(ctx context.Context, md *bigquery.TableMetadata) error {
			return nil
		}

		meta := model.GitHubMetadata{
			GitHubCommit: model.GitHubCommit{
				GitHubRepo: model.GitHubRepo{
					Owner:    "test-owner",
					RepoName: "test-repo",
					RepoID:   123,
				},
				Branch:   "main",
				CommitID: "0000000000000000000000000000000000000000",
			},
			InstallationID: 456,
		}

		repoID := types.GitHubRepoID("test-owner/test-repo")
		branchName := types.BranchName("main")
		targetID := types.TargetID("go.mod")

		// Scenario 1: First scan - new vulnerabilities should be Active
		report1 := trivy.Report{
			SchemaVersion: 2,
			Results: []trivy.Result{
				{
					Target: "go.mod",
					Class:  "lang-pkgs",
					Type:   "gomod",
					Vulnerabilities: []trivy.DetectedVulnerability{
						{
							VulnerabilityID:  "CVE-2024-0001",
							PkgName:          "pkg1",
							InstalledVersion: "1.0.0",
							Vulnerability: trivy.Vulnerability{
								Severity: "HIGH",
							},
						},
					},
				},
			},
		}

		gt.NoError(t, uc.InsertScanResult(ctx, meta, report1))

		vulns, err := memRepo.ListVulnerabilities(ctx, repoID, branchName, targetID)
		gt.NoError(t, err)
		gt.V(t, len(vulns)).Equal(1)
		gt.V(t, vulns[0].ID).Equal("CVE-2024-0001")
		gt.V(t, vulns[0].Status).Equal(types.VulnStatusActive)

		// Scenario 2: Second scan - continuous detection should keep status
		gt.NoError(t, uc.InsertScanResult(ctx, meta, report1))

		vulns, err = memRepo.ListVulnerabilities(ctx, repoID, branchName, targetID)
		gt.NoError(t, err)
		gt.V(t, len(vulns)).Equal(1)
		gt.V(t, vulns[0].Status).Equal(types.VulnStatusActive)

		// Scenario 3: Third scan - vulnerability fixed (not detected)
		report2 := trivy.Report{
			SchemaVersion: 2,
			Results: []trivy.Result{
				{
					Target: "go.mod",
					Class:  "lang-pkgs",
					Type:   "gomod",
					Vulnerabilities: nil,
				},
			},
		}

		gt.NoError(t, uc.InsertScanResult(ctx, meta, report2))

		vulns, err = memRepo.ListVulnerabilities(ctx, repoID, branchName, targetID)
		gt.NoError(t, err)
		gt.V(t, len(vulns)).Equal(1)
		gt.V(t, vulns[0].Status).Equal(types.VulnStatusFixed)

		// Scenario 4: Fourth scan - re-detection (Fixed → Active)
		gt.NoError(t, uc.InsertScanResult(ctx, meta, report1))

		vulns, err = memRepo.ListVulnerabilities(ctx, repoID, branchName, targetID)
		gt.NoError(t, err)
		gt.V(t, len(vulns)).Equal(1)
		gt.V(t, vulns[0].Status).Equal(types.VulnStatusActive)
	})
}
