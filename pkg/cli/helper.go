package cli

import (
	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/octovy/pkg/domain/interfaces"
)

func requireBigQuery(client interfaces.BigQuery) error {
	if client == nil {
		return goerr.New("BigQuery client is required (project ID and dataset ID must be set)")
	}
	return nil
}
