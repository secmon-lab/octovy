package interfaces

//go:generate moq -out ../mock/infra.go -pkg mock . BigQuery GitHubApp

import (
	"context"
	"net/http"
	"net/url"

	"cloud.google.com/go/bigquery"

	"github.com/m-mizutani/octovy/pkg/domain/types"
)

type BigQuery interface {
	Insert(ctx context.Context, schema bigquery.Schema, data any) error

	GetMetadata(ctx context.Context) (*bigquery.TableMetadata, error)
	UpdateTable(ctx context.Context, md bigquery.TableMetadataToUpdate, eTag string) error
	CreateTable(ctx context.Context, md *bigquery.TableMetadata) error
}

type GitHubApp interface {
	GetArchiveURL(ctx context.Context, input *GetArchiveURLInput) (*url.URL, error)
	HTTPClient(installID types.GitHubAppInstallID) (*http.Client, error)
}

type GetArchiveURLInput struct {
	Owner     string
	Repo      string
	CommitID  string
	InstallID types.GitHubAppInstallID
}
