package usecase

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/m-mizutani/bqs"
	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/octovy/pkg/domain/interfaces"
	"github.com/m-mizutani/octovy/pkg/domain/model"
	"github.com/m-mizutani/octovy/pkg/domain/model/trivy"
	"github.com/m-mizutani/octovy/pkg/domain/types"
)

func (x *UseCase) InsertScanResult(ctx context.Context, meta model.GitHubMetadata, report trivy.Report, cfg model.Config) error {
	if err := report.Validate(); err != nil {
		return goerr.Wrap(err, "invalid trivy report")
	}

	cfgData, err := json.Marshal(cfg)
	if err != nil {
		return goerr.Wrap(err, "failed to marshal config")
	}

	scan := &model.Scan{
		ID:        types.NewScanID(),
		Timestamp: time.Now().UTC(),
		GitHub:    meta,
		Report:    report,
		Config:    string(cfgData),
	}

	if x.clients.BigQuery() != nil {
		schema, err := createOrUpdateBigQueryTable(ctx, x.clients.BigQuery(), x.tableID, scan)
		if err != nil {
			return err
		}

		rawRecord := &model.ScanRawRecord{
			Scan:      *scan,
			Timestamp: scan.Timestamp.UnixMicro(),
		}
		if err := x.clients.BigQuery().Insert(ctx, x.tableID, schema, rawRecord); err != nil {
			return goerr.Wrap(err, "failed to insert scan data to BigQuery")
		}
	}

	if x.clients.BigQuery() != nil {
		raw, err := json.Marshal(scan)
		if err != nil {
			return goerr.Wrap(err, "failed to marshal scan data")
		}

		commitKey := toStorageCommitKey(scan.GitHub)
		branchKey := toStorageBranchKey(scan.GitHub)

		for _, key := range []string{commitKey, branchKey} {
			buf := strings.NewReader(string(raw))
			reader := io.NopCloser(buf)
			if err := x.clients.Storage().Put(ctx, key, reader); err != nil {
				return err
			}
		}
	}
	return nil
}

func toStorageCommitKey(meta model.GitHubMetadata) string {
	return strings.Join([]string{
		meta.Owner,
		meta.RepoName,
		"commit",
		meta.CommitID,
		"scan.json.gz",
	}, "/")
}

func toStorageBranchKey(meta model.GitHubMetadata) string {
	return strings.Join([]string{
		meta.Owner,
		meta.RepoName,
		"branch",
		meta.Branch,
		"scan.json.gz",
	}, "/")
}

func createOrUpdateBigQueryTable(ctx context.Context, bq interfaces.BigQuery, tableID types.BQTableID, scan *model.Scan) (bigquery.Schema, error) {
	schema, err := bqs.Infer(scan)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to infer scan schema")
	}

	metaData, err := bq.GetMetadata(ctx, tableID)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create BigQuery table")
	}
	if metaData == nil {
		if err := bq.CreateTable(ctx, tableID, &bigquery.TableMetadata{
			Schema: schema,
		}); err != nil {
			return nil, goerr.Wrap(err, "failed to create BigQuery table")
		}

		return schema, nil
	}

	if bqs.Equal(metaData.Schema, schema) {
		return schema, nil
	}

	mergedSchema, err := bqs.Merge(metaData.Schema, schema)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to merge BigQuery schema")
	}
	if err := bq.UpdateTable(ctx, tableID, bigquery.TableMetadataToUpdate{
		Schema: mergedSchema,
	}, metaData.ETag); err != nil {
		return nil, goerr.Wrap(err, "failed to update BigQuery table")
	}

	return mergedSchema, nil
}
