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
	"github.com/m-mizutani/octovy/pkg/usecase"
)

func TestInsertScanResult(t *testing.T) {
	t.Run("insert scan result to BigQuery", func(t *testing.T) {
		mockBQ := &mock.BigQueryMock{}
		uc := usecase.New(infra.New(
			infra.WithBigQuery(mockBQ),
		), usecase.WithBigQueryTableID("scans"))

		ctx := context.Background()

		var insertCalled bool
		mockBQ.InsertFunc = func(ctx context.Context, tableID types.BQTableID, schema bigquery.Schema, data any) error {
			insertCalled = true
			gt.V(t, tableID).Equal(types.BQTableID("scans"))
			return nil
		}

		mockBQ.GetMetadataFunc = func(ctx context.Context, table types.BQTableID) (*bigquery.TableMetadata, error) {
			return nil, nil
		}

		mockBQ.CreateTableFunc = func(ctx context.Context, table types.BQTableID, md *bigquery.TableMetadata) error {
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
		), usecase.WithBigQueryTableID("scans"))

		ctx := context.Background()

		var createTableCalled bool
		mockBQ.CreateTableFunc = func(ctx context.Context, table types.BQTableID, md *bigquery.TableMetadata) error {
			createTableCalled = true
			gt.V(t, table).Equal(types.BQTableID("scans"))
			gt.V(t, md).NotEqual(nil)
			return nil
		}

		mockBQ.GetMetadataFunc = func(ctx context.Context, table types.BQTableID) (*bigquery.TableMetadata, error) {
			return nil, nil // No existing table
		}

		mockBQ.InsertFunc = func(ctx context.Context, tableID types.BQTableID, schema bigquery.Schema, data any) error {
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
		mockBQ.InsertFunc = func(ctx context.Context, tableID types.BQTableID, schema bigquery.Schema, data any) error {
			insertedData = data
			return nil
		}

		mockBQ.GetMetadataFunc = func(ctx context.Context, table types.BQTableID) (*bigquery.TableMetadata, error) {
			return &bigquery.TableMetadata{
				Schema: bigquery.Schema{},
				ETag:   "test-etag",
			}, nil
		}

		mockBQ.CreateTableFunc = func(ctx context.Context, table types.BQTableID, md *bigquery.TableMetadata) error {
			return nil
		}

		mockBQ.UpdateTableFunc = func(ctx context.Context, table types.BQTableID, md bigquery.TableMetadataToUpdate, eTag string) error {
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
}
