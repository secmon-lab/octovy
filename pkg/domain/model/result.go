package model

import (
	"time"

	"github.com/m-mizutani/octovy/pkg/domain/model/trivy"
	"github.com/m-mizutani/octovy/pkg/domain/types"
)

type Scan struct {
	ID        types.ScanID   `bigquery:"id" firestore:"id" json:"id"`
	Timestamp time.Time      `bigquery:"timestamp" firestore:"timestamp" json:"timestamp"`
	GitHub    GitHubMetadata `bigquery:"github" firestore:"github" json:"github"`
	Report    trivy.Report   `bigquery:"report" firestore:"report" json:"report"`
}

type ScanRawRecord struct {
	Scan
	Timestamp int64 `bigquery:"timestamp" firestore:"timestamp" json:"timestamp"`
}
