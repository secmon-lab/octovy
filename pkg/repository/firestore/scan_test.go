package firestore_test

import (
	"context"
	"os"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/octovy/pkg/repository/firestore"
	"github.com/m-mizutani/octovy/pkg/repository/testhelper"
)

func TestFirestoreScanRepository(t *testing.T) {
	projectID := os.Getenv("TEST_FIRESTORE_PROJECT_ID")
	databaseID := os.Getenv("TEST_FIRESTORE_DATABASE_ID")

	if projectID == "" || databaseID == "" {
		t.Skip("Firestore credentials not configured (TEST_FIRESTORE_PROJECT_ID, TEST_FIRESTORE_DATABASE_ID)")
	}

	ctx := context.Background()
	repo, err := firestore.New(ctx, projectID, databaseID)
	gt.NoError(t, err)

	testhelper.TestAll(t, repo)
}

func TestToFirestoreID(t *testing.T) {
	// Valid cases
	id, err := firestore.ToFirestoreID("owner1", "repo1")
	gt.NoError(t, err)
	gt.V(t, id).Equal("owner1:repo1")

	id, err = firestore.ToFirestoreID("my-org", "my-repo")
	gt.NoError(t, err)
	gt.V(t, id).Equal("my-org:my-repo")

	// Invalid cases
	_, err = firestore.ToFirestoreID("", "repo1")
	gt.Error(t, err)

	_, err = firestore.ToFirestoreID("owner1", "")
	gt.Error(t, err)

	_, err = firestore.ToFirestoreID("owner:1", "repo1")
	gt.Error(t, err)

	_, err = firestore.ToFirestoreID("owner1", "repo:1")
	gt.Error(t, err)
}
