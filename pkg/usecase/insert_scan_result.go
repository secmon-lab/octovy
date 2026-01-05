package usecase

import (
	"context"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/m-mizutani/bqs"
	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/octovy/pkg/domain/interfaces"
	"github.com/m-mizutani/octovy/pkg/domain/model"
	"github.com/m-mizutani/octovy/pkg/domain/model/trivy"
	"github.com/m-mizutani/octovy/pkg/domain/types"
)

func (x *UseCase) InsertScanResult(ctx context.Context, meta model.GitHubMetadata, report trivy.Report) error {
	if err := report.Validate(); err != nil {
		return goerr.Wrap(err, "invalid trivy report")
	}

	scan := &model.Scan{
		ID:        types.NewScanID(),
		Timestamp: time.Now().UTC(),
		GitHub:    meta,
		Report:    report,
	}

	// Insert to BigQuery
	if x.clients.BigQuery() != nil {
		schema, schemaUpdated, err := createOrUpdateBigQueryTable(ctx, x.clients.BigQuery(), scan)
		if err != nil {
			return err
		}

		rawRecord := &model.ScanRawRecord{
			Scan:      *scan,
			Timestamp: scan.Timestamp.UnixMicro(),
		}

		// Set schemaUpdated flag in context for Insert method to determine if retry is needed
		insertCtx := context.WithValue(ctx, "schema_updated", schemaUpdated)
		if err := x.clients.BigQuery().Insert(insertCtx, schema, rawRecord); err != nil {
			return goerr.Wrap(err, "failed to insert scan data to BigQuery")
		}
	}

	// Insert to Firestore
	if x.clients.ScanRepository() != nil {
		if err := x.insertToFirestore(ctx, meta, scan, report); err != nil {
			return goerr.Wrap(err, "failed to insert scan data to Firestore")
		}
	}

	return nil
}

func createOrUpdateBigQueryTable(ctx context.Context, bq interfaces.BigQuery, scan *model.Scan) (schema bigquery.Schema, schemaUpdated bool, err error) {
	schema, err = bqs.Infer(scan)
	if err != nil {
		return nil, false, goerr.Wrap(err, "failed to infer scan schema")
	}

	metaData, err := bq.GetMetadata(ctx)
	if err != nil {
		return nil, false, goerr.Wrap(err, "failed to create BigQuery table")
	}
	if metaData == nil {
		if err := bq.CreateTable(ctx, &bigquery.TableMetadata{
			Schema: schema,
		}); err != nil {
			return nil, false, goerr.Wrap(err, "failed to create BigQuery table")
		}

		return schema, false, nil
	}

	if bqs.Equal(metaData.Schema, schema) {
		return schema, false, nil
	}

	mergedSchema, err := bqs.Merge(metaData.Schema, schema)
	if err != nil {
		return nil, false, goerr.Wrap(err, "failed to merge BigQuery schema")
	}
	if err := bq.UpdateTable(ctx, bigquery.TableMetadataToUpdate{
		Schema: mergedSchema,
	}, metaData.ETag); err != nil {
		return nil, false, goerr.Wrap(err, "failed to update BigQuery table")
	}

	return mergedSchema, true, nil
}

func (x *UseCase) insertToFirestore(ctx context.Context, meta model.GitHubMetadata, scan *model.Scan, report trivy.Report) error {
	repo := x.clients.ScanRepository()

	// Create or update repository
	repoID := types.GitHubRepoID(meta.Owner + "/" + meta.RepoName)
	repository := &model.Repository{
		ID:             repoID,
		Owner:          meta.Owner,
		Name:           meta.RepoName,
		DefaultBranch:  types.BranchName(meta.DefaultBranch),
		InstallationID: meta.InstallationID,
		CreatedAt:      scan.Timestamp,
		UpdatedAt:      scan.Timestamp,
	}
	if err := repo.CreateOrUpdateRepository(ctx, repository); err != nil {
		return goerr.Wrap(err, "failed to create or update repository")
	}

	// Create or update branch
	branch := &model.Branch{
		Name:          types.BranchName(meta.Branch),
		LastScanID:    scan.ID,
		LastScanAt:    scan.Timestamp,
		LastCommitSHA: types.CommitSHA(meta.CommitID),
		Status:        types.ScanStatusSuccess,
		CreatedAt:     scan.Timestamp,
		UpdatedAt:     scan.Timestamp,
	}
	if err := repo.CreateOrUpdateBranch(ctx, repoID, branch); err != nil {
		return goerr.Wrap(err, "failed to create or update branch")
	}

	// Process each target (Result) in the report
	for _, result := range report.Results {
		// Create or update target
		targetID := model.ToTargetID(result.Target)
		target := &model.Target{
			ID:        targetID,
			Target:    result.Target,
			Class:     string(result.Class),
			Type:      result.Type,
			CreatedAt: scan.Timestamp,
			UpdatedAt: scan.Timestamp,
		}
		if err := repo.CreateOrUpdateTarget(ctx, repoID, branch.Name, target); err != nil {
			return goerr.Wrap(err, "failed to create or update target")
		}

		// Process vulnerabilities with status management
		if err := x.processVulnerabilities(ctx, repo, repoID, branch.Name, targetID, result.Vulnerabilities, scan.Timestamp); err != nil {
			return goerr.Wrap(err, "failed to process vulnerabilities")
		}
	}

	return nil
}

func (x *UseCase) processVulnerabilities(ctx context.Context, repo interfaces.ScanRepository, repoID types.GitHubRepoID, branchName types.BranchName, targetID types.TargetID, detectedVulns []trivy.DetectedVulnerability, timestamp time.Time) error {
	// Get existing vulnerabilities
	existing, err := repo.ListVulnerabilities(ctx, repoID, branchName, targetID)
	if err != nil {
		return goerr.Wrap(err, "failed to list existing vulnerabilities")
	}

	existingMap := make(map[string]*model.Vulnerability)
	for _, v := range existing {
		existingMap[v.ID] = v
	}

	// Build detected vulnerability map and new vulnerabilities list
	detectedMap := make(map[string]bool)
	var newVulns []*model.Vulnerability
	statusUpdates := make(map[string]types.VulnStatus)

	for i := range detectedVulns {
		vuln := model.NewVulnerability(&detectedVulns[i])
		detectedMap[vuln.ID] = true

		if existingVuln, exists := existingMap[vuln.ID]; exists {
			// Existing vulnerability detected
			if existingVuln.Status == types.VulnStatusFixed {
				// Fixed → Active (re-detection)
				statusUpdates[vuln.ID] = types.VulnStatusActive
			}
			// Continuous detection → keep status (no update needed)
		} else {
			// New detection → Active
			vuln.Status = types.VulnStatusActive
			vuln.CreatedAt = timestamp
			vuln.UpdatedAt = timestamp
			newVulns = append(newVulns, vuln)
		}
	}

	// Mark vulnerabilities not detected as Fixed
	for id, existingVuln := range existingMap {
		if !detectedMap[id] && existingVuln.Status == types.VulnStatusActive {
			statusUpdates[id] = types.VulnStatusFixed
		}
	}

	// Batch create new vulnerabilities
	if len(newVulns) > 0 {
		if err := repo.BatchCreateVulnerabilities(ctx, repoID, branchName, targetID, newVulns); err != nil {
			return goerr.Wrap(err, "failed to batch create vulnerabilities")
		}
	}

	// Batch update statuses
	if len(statusUpdates) > 0 {
		if err := repo.BatchUpdateVulnerabilityStatus(ctx, repoID, branchName, targetID, statusUpdates); err != nil {
			return goerr.Wrap(err, "failed to batch update vulnerability status")
		}
	}

	return nil
}
