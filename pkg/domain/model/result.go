package model

import (
	"time"

	"github.com/m-mizutani/octovy/pkg/domain/model/trivy"
	"github.com/m-mizutani/octovy/pkg/domain/types"
)

type Scan struct {
	ID        types.ScanID   `bigquery:"id" json:"id"`
	Timestamp time.Time      `bigquery:"timestamp" json:"timestamp"`
	GitHub    GitHubMetadata `bigquery:"github" json:"github"`
	Report    trivy.Report   `bigquery:"report" json:"report"`
}

type ScanRawRecord struct {
	Scan
	Timestamp int64 `bigquery:"timestamp" json:"timestamp"`
}
