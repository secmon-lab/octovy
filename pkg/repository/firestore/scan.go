package firestore

import (
	"context"
	"strings"

	"cloud.google.com/go/firestore"
	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/octovy/pkg/domain/model"
	"github.com/m-mizutani/octovy/pkg/domain/types"
	"github.com/m-mizutani/octovy/pkg/repository"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	collectionRepo          = "repo"
	collectionBranch        = "branch"
	collectionTarget        = "target"
	collectionVulnerability = "vulnerability"
	batchSize               = 500
)

type scanRepository struct {
	client *firestore.Client
}

// ToFirestoreID converts owner and repo to a Firestore-safe document ID
// Uses colon (:) as separator since GitHub owner names cannot contain colons
func ToFirestoreID(owner, repo string) (string, error) {
	if owner == "" || repo == "" {
		return "", goerr.Wrap(repository.ErrInvalidInput, "owner or repo is empty",
			goerr.V("owner", owner),
			goerr.V("repo", repo),
		)
	}

	if strings.Contains(owner, ":") || strings.Contains(repo, ":") {
		return "", goerr.Wrap(repository.ErrInvalidInput, "owner or repo contains invalid character ':'",
			goerr.V("owner", owner),
			goerr.V("repo", repo),
		)
	}

	return owner + ":" + repo, nil
}

// toBranchDocID converts a branch name to a Firestore-safe document ID
// Replaces "/" with ":" since Git branch names can contain "/" (e.g., "feature/foo")
// but Git ref names cannot contain ":", so this replacement is safe and reversible
// Note: This conversion is only for Firestore document IDs. The actual branch name
// stored in the document remains unchanged.
func toBranchDocID(branchName string) string {
	return strings.ReplaceAll(branchName, "/", ":")
}

// Repository operations

func (r *scanRepository) CreateOrUpdateRepository(ctx context.Context, repo *model.Repository) error {
	firestoreID, err := ToFirestoreID(repo.Owner, repo.Name)
	if err != nil {
		return err
	}

	docRef := r.client.Collection(collectionRepo).Doc(firestoreID)

	// Set the document (creates or updates)
	_, err = docRef.Set(ctx, repo)
	if err != nil {
		return goerr.Wrap(err, "failed to create or update repository",
			goerr.V("repoID", repo.ID),
		)
	}

	return nil
}

func (r *scanRepository) GetRepository(ctx context.Context, repoID types.GitHubRepoID) (*model.Repository, error) {
	// Parse repoID to get owner and repo
	parts := strings.Split(string(repoID), "/")
	if len(parts) != 2 {
		return nil, goerr.Wrap(repository.ErrInvalidInput, "invalid repoID format",
			goerr.V("repoID", repoID),
		)
	}

	firestoreID, err := ToFirestoreID(parts[0], parts[1])
	if err != nil {
		return nil, err
	}

	docRef := r.client.Collection(collectionRepo).Doc(firestoreID)
	snap, err := docRef.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, goerr.Wrap(repository.ErrNotFound, "repository not found",
				goerr.V("repoID", repoID),
			)
		}
		return nil, goerr.Wrap(err, "failed to get repository",
			goerr.V("repoID", repoID),
		)
	}

	var repo model.Repository
	if err := snap.DataTo(&repo); err != nil {
		return nil, goerr.Wrap(err, "failed to decode repository",
			goerr.V("repoID", repoID),
		)
	}

	return &repo, nil
}

func (r *scanRepository) ListRepositories(ctx context.Context, installationID int64) ([]*model.Repository, error) {
	query := r.client.Collection(collectionRepo).Where("InstallationID", "==", installationID)

	iter := query.Documents(ctx)
	defer iter.Stop()

	var repos []*model.Repository
	for {
		snap, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, goerr.Wrap(err, "failed to iterate repositories",
				goerr.V("installationID", installationID),
			)
		}

		var repo model.Repository
		if err := snap.DataTo(&repo); err != nil {
			return nil, goerr.Wrap(err, "failed to decode repository")
		}

		repos = append(repos, &repo)
	}

	return repos, nil
}

// Branch operations

func (r *scanRepository) CreateOrUpdateBranch(ctx context.Context, repoID types.GitHubRepoID, branch *model.Branch) error {
	parts := strings.Split(string(repoID), "/")
	if len(parts) != 2 {
		return goerr.Wrap(repository.ErrInvalidInput, "invalid repoID format",
			goerr.V("repoID", repoID),
		)
	}

	firestoreID, err := ToFirestoreID(parts[0], parts[1])
	if err != nil {
		return err
	}

	docRef := r.client.Collection(collectionRepo).Doc(firestoreID).
		Collection(collectionBranch).Doc(toBranchDocID(string(branch.Name)))

	_, err = docRef.Set(ctx, branch)
	if err != nil {
		return goerr.Wrap(err, "failed to create or update branch",
			goerr.V("repoID", repoID),
			goerr.V("branchName", branch.Name),
		)
	}

	return nil
}

func (r *scanRepository) GetBranch(ctx context.Context, repoID types.GitHubRepoID, branchName types.BranchName) (*model.Branch, error) {
	parts := strings.Split(string(repoID), "/")
	if len(parts) != 2 {
		return nil, goerr.Wrap(repository.ErrInvalidInput, "invalid repoID format",
			goerr.V("repoID", repoID),
		)
	}

	firestoreID, err := ToFirestoreID(parts[0], parts[1])
	if err != nil {
		return nil, err
	}

	docRef := r.client.Collection(collectionRepo).Doc(firestoreID).
		Collection(collectionBranch).Doc(toBranchDocID(string(branchName)))

	snap, err := docRef.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, goerr.Wrap(repository.ErrNotFound, "branch not found",
				goerr.V("repoID", repoID),
				goerr.V("branchName", branchName),
			)
		}
		return nil, goerr.Wrap(err, "failed to get branch",
			goerr.V("repoID", repoID),
			goerr.V("branchName", branchName),
		)
	}

	var branch model.Branch
	if err := snap.DataTo(&branch); err != nil {
		return nil, goerr.Wrap(err, "failed to decode branch",
			goerr.V("repoID", repoID),
			goerr.V("branchName", branchName),
		)
	}

	return &branch, nil
}

func (r *scanRepository) ListBranches(ctx context.Context, repoID types.GitHubRepoID) ([]*model.Branch, error) {
	parts := strings.Split(string(repoID), "/")
	if len(parts) != 2 {
		return nil, goerr.Wrap(repository.ErrInvalidInput, "invalid repoID format",
			goerr.V("repoID", repoID),
		)
	}

	firestoreID, err := ToFirestoreID(parts[0], parts[1])
	if err != nil {
		return nil, err
	}

	iter := r.client.Collection(collectionRepo).Doc(firestoreID).
		Collection(collectionBranch).Documents(ctx)
	defer iter.Stop()

	var branches []*model.Branch
	for {
		snap, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, goerr.Wrap(err, "failed to iterate branches",
				goerr.V("repoID", repoID),
			)
		}

		var branch model.Branch
		if err := snap.DataTo(&branch); err != nil {
			return nil, goerr.Wrap(err, "failed to decode branch")
		}

		branches = append(branches, &branch)
	}

	return branches, nil
}

// Target operations

func (r *scanRepository) CreateOrUpdateTarget(ctx context.Context, repoID types.GitHubRepoID, branchName types.BranchName, target *model.Target) error {
	parts := strings.Split(string(repoID), "/")
	if len(parts) != 2 {
		return goerr.Wrap(repository.ErrInvalidInput, "invalid repoID format",
			goerr.V("repoID", repoID),
		)
	}

	firestoreID, err := ToFirestoreID(parts[0], parts[1])
	if err != nil {
		return err
	}

	docRef := r.client.Collection(collectionRepo).Doc(firestoreID).
		Collection(collectionBranch).Doc(toBranchDocID(string(branchName))).
		Collection(collectionTarget).Doc(string(target.ID))

	_, err = docRef.Set(ctx, target)
	if err != nil {
		return goerr.Wrap(err, "failed to create or update target",
			goerr.V("repoID", repoID),
			goerr.V("branchName", branchName),
			goerr.V("targetID", target.ID),
		)
	}

	return nil
}

func (r *scanRepository) GetTarget(ctx context.Context, repoID types.GitHubRepoID, branchName types.BranchName, targetID types.TargetID) (*model.Target, error) {
	parts := strings.Split(string(repoID), "/")
	if len(parts) != 2 {
		return nil, goerr.Wrap(repository.ErrInvalidInput, "invalid repoID format",
			goerr.V("repoID", repoID),
		)
	}

	firestoreID, err := ToFirestoreID(parts[0], parts[1])
	if err != nil {
		return nil, err
	}

	docRef := r.client.Collection(collectionRepo).Doc(firestoreID).
		Collection(collectionBranch).Doc(toBranchDocID(string(branchName))).
		Collection(collectionTarget).Doc(string(targetID))

	snap, err := docRef.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, goerr.Wrap(repository.ErrNotFound, "target not found",
				goerr.V("repoID", repoID),
				goerr.V("branchName", branchName),
				goerr.V("targetID", targetID),
			)
		}
		return nil, goerr.Wrap(err, "failed to get target",
			goerr.V("repoID", repoID),
			goerr.V("branchName", branchName),
			goerr.V("targetID", targetID),
		)
	}

	var target model.Target
	if err := snap.DataTo(&target); err != nil {
		return nil, goerr.Wrap(err, "failed to decode target",
			goerr.V("repoID", repoID),
			goerr.V("branchName", branchName),
			goerr.V("targetID", targetID),
		)
	}

	return &target, nil
}

func (r *scanRepository) ListTargets(ctx context.Context, repoID types.GitHubRepoID, branchName types.BranchName) ([]*model.Target, error) {
	parts := strings.Split(string(repoID), "/")
	if len(parts) != 2 {
		return nil, goerr.Wrap(repository.ErrInvalidInput, "invalid repoID format",
			goerr.V("repoID", repoID),
		)
	}

	firestoreID, err := ToFirestoreID(parts[0], parts[1])
	if err != nil {
		return nil, err
	}

	iter := r.client.Collection(collectionRepo).Doc(firestoreID).
		Collection(collectionBranch).Doc(toBranchDocID(string(branchName))).
		Collection(collectionTarget).Documents(ctx)
	defer iter.Stop()

	var targets []*model.Target
	for {
		snap, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, goerr.Wrap(err, "failed to iterate targets",
				goerr.V("repoID", repoID),
				goerr.V("branchName", branchName),
			)
		}

		var target model.Target
		if err := snap.DataTo(&target); err != nil {
			return nil, goerr.Wrap(err, "failed to decode target")
		}

		targets = append(targets, &target)
	}

	return targets, nil
}

// Vulnerability operations

func (r *scanRepository) ListVulnerabilities(ctx context.Context, repoID types.GitHubRepoID, branchName types.BranchName, targetID types.TargetID) ([]*model.Vulnerability, error) {
	parts := strings.Split(string(repoID), "/")
	if len(parts) != 2 {
		return nil, goerr.Wrap(repository.ErrInvalidInput, "invalid repoID format",
			goerr.V("repoID", repoID),
		)
	}

	firestoreID, err := ToFirestoreID(parts[0], parts[1])
	if err != nil {
		return nil, err
	}

	iter := r.client.Collection(collectionRepo).Doc(firestoreID).
		Collection(collectionBranch).Doc(toBranchDocID(string(branchName))).
		Collection(collectionTarget).Doc(string(targetID)).
		Collection(collectionVulnerability).Documents(ctx)
	defer iter.Stop()

	var vulns []*model.Vulnerability
	for {
		snap, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, goerr.Wrap(err, "failed to iterate vulnerabilities",
				goerr.V("repoID", repoID),
				goerr.V("branchName", branchName),
				goerr.V("targetID", targetID),
			)
		}

		var vuln model.Vulnerability
		if err := snap.DataTo(&vuln); err != nil {
			return nil, goerr.Wrap(err, "failed to decode vulnerability")
		}

		vulns = append(vulns, &vuln)
	}

	return vulns, nil
}

func (r *scanRepository) BatchCreateVulnerabilities(ctx context.Context, repoID types.GitHubRepoID, branchName types.BranchName, targetID types.TargetID, vulns []*model.Vulnerability) error {
	parts := strings.Split(string(repoID), "/")
	if len(parts) != 2 {
		return goerr.Wrap(repository.ErrInvalidInput, "invalid repoID format",
			goerr.V("repoID", repoID),
		)
	}

	firestoreID, err := ToFirestoreID(parts[0], parts[1])
	if err != nil {
		return err
	}

	vulnCollection := r.client.Collection(collectionRepo).Doc(firestoreID).
		Collection(collectionBranch).Doc(toBranchDocID(string(branchName))).
		Collection(collectionTarget).Doc(string(targetID)).
		Collection(collectionVulnerability)

	// Process in batches of 500 (Firestore limit)
	for i := 0; i < len(vulns); i += batchSize {
		end := i + batchSize
		if end > len(vulns) {
			end = len(vulns)
		}

		batch := r.client.Batch()
		for _, vuln := range vulns[i:end] {
			docRef := vulnCollection.Doc(vuln.ID)
			batch.Set(docRef, vuln)
		}

		if _, err := batch.Commit(ctx); err != nil {
			return goerr.Wrap(err, "failed to batch create vulnerabilities",
				goerr.V("repoID", repoID),
				goerr.V("branchName", branchName),
				goerr.V("targetID", targetID),
				goerr.V("batchStart", i),
				goerr.V("batchEnd", end),
			)
		}
	}

	return nil
}

func (r *scanRepository) BatchUpdateVulnerabilityStatus(ctx context.Context, repoID types.GitHubRepoID, branchName types.BranchName, targetID types.TargetID, updates map[string]types.VulnStatus) error {
	parts := strings.Split(string(repoID), "/")
	if len(parts) != 2 {
		return goerr.Wrap(repository.ErrInvalidInput, "invalid repoID format",
			goerr.V("repoID", repoID),
		)
	}

	firestoreID, err := ToFirestoreID(parts[0], parts[1])
	if err != nil {
		return err
	}

	vulnCollection := r.client.Collection(collectionRepo).Doc(firestoreID).
		Collection(collectionBranch).Doc(toBranchDocID(string(branchName))).
		Collection(collectionTarget).Doc(string(targetID)).
		Collection(collectionVulnerability)

	// Convert map to slice for batching
	type update struct {
		id     string
		status types.VulnStatus
	}
	var updateList []update
	for id, status := range updates {
		updateList = append(updateList, update{id: id, status: status})
	}

	// Process in batches of 500 (Firestore limit)
	for i := 0; i < len(updateList); i += batchSize {
		end := i + batchSize
		if end > len(updateList) {
			end = len(updateList)
		}

		batch := r.client.Batch()
		for _, u := range updateList[i:end] {
			docRef := vulnCollection.Doc(u.id)
			batch.Update(docRef, []firestore.Update{
				{Path: "Status", Value: u.status},
				{Path: "UpdatedAt", Value: firestore.ServerTimestamp},
			})
		}

		if _, err := batch.Commit(ctx); err != nil {
			return goerr.Wrap(err, "failed to batch update vulnerability status",
				goerr.V("repoID", repoID),
				goerr.V("branchName", branchName),
				goerr.V("targetID", targetID),
				goerr.V("batchStart", i),
				goerr.V("batchEnd", end),
			)
		}
	}

	return nil
}
