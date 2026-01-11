package interfaces

//go:generate moq -out ../mock/usecase.go -pkg mock . UseCase

import (
	"context"

	"github.com/m-mizutani/octovy/pkg/domain/model"
	"github.com/m-mizutani/octovy/pkg/domain/model/trivy"
	"github.com/m-mizutani/octovy/pkg/domain/types"
)

type UseCase interface {
	InsertScanResult(ctx context.Context, meta model.GitHubMetadata, report trivy.Report) (types.ScanID, error)
	ScanGitHubRepo(ctx context.Context, input *model.ScanGitHubRepoInput) error
}
