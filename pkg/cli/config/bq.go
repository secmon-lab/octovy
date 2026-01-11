package config

import (
	"context"
	"log/slog"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/octovy/pkg/domain/interfaces"
	"github.com/m-mizutani/octovy/pkg/domain/types"
	"github.com/m-mizutani/octovy/pkg/infra/bq"
	"github.com/urfave/cli/v3"
	"google.golang.org/api/impersonate"
	"google.golang.org/api/option"
)

type BigQuery struct {
	projectID                 types.GoogleProjectID
	datasetID                 types.BQDatasetID
	tableID                   types.BQTableID
	impersonateServiceAccount string
}

func (x *BigQuery) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "bigquery-project-id",
			Usage:       "BigQuery project ID",
			Category:    "BigQuery",
			Destination: (*string)(&x.projectID),
			Sources:     cli.EnvVars("OCTOVY_BIGQUERY_PROJECT_ID"),
		},
		&cli.StringFlag{
			Name:        "bigquery-dataset-id",
			Usage:       "BigQuery dataset ID",
			Category:    "BigQuery",
			Destination: (*string)(&x.datasetID),
			Sources:     cli.EnvVars("OCTOVY_BIGQUERY_DATASET_ID"),
			Value:       "octovy",
		},
		&cli.StringFlag{
			Name:        "bigquery-table-id",
			Usage:       "BigQuery table ID",
			Category:    "BigQuery",
			Destination: (*string)(&x.tableID),
			Sources:     cli.EnvVars("OCTOVY_BIGQUERY_TABLE_ID"),
			Value:       "scans",
		},
		&cli.StringFlag{
			Name:        "bq-impersonate-service-account",
			Usage:       "Impersonate service account for BigQuery",
			Destination: &x.impersonateServiceAccount,
			Sources:     cli.EnvVars("OCTOVY_BIGQUERY_IMPERSONATE_SERVICE_ACCOUNT"),
		},
	}
}

func (x *BigQuery) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Any("ProjectID", x.projectID),
		slog.Any("DatasetID", x.datasetID),
		slog.Any("TableID", x.tableID),
		slog.Any("ImpersonateServiceAccount", x.impersonateServiceAccount),
	)
}

func (x *BigQuery) NewClient(ctx context.Context) (interfaces.BigQuery, error) {
	if x.projectID == "" && x.datasetID == "" {
		return nil, nil
	}
	var options []option.ClientOption
	if x.impersonateServiceAccount != "" {
		ts, err := impersonate.CredentialsTokenSource(ctx, impersonate.CredentialsConfig{
			TargetPrincipal: x.impersonateServiceAccount,
			Scopes: []string{
				"https://www.googleapis.com/auth/bigquery",
				"https://www.googleapis.com/auth/cloud-platform",
			},
		})
		if err != nil {
			return nil, goerr.Wrap(err, "failed to create token source for impersonate")
		}

		options = append(options, option.WithTokenSource(ts))
	}

	return bq.New(ctx, x.projectID, x.datasetID, x.tableID, options...)
}
