package ghapp_test

import (
	"context"
	"os"
	"strconv"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/octovy/pkg/domain/types"
	"github.com/m-mizutani/octovy/pkg/infra/ghapp"
)

func TestNew(t *testing.T) {
	t.Run("create new GitHub App client with valid inputs", func(t *testing.T) {
		appID := types.GitHubAppID(12345)
		privateKey := types.GitHubAppPrivateKey("test-key")

		_, err := ghapp.New(appID, privateKey)
		gt.NoError(t, err)
	})

	t.Run("create with empty private key fails", func(t *testing.T) {
		appID := types.GitHubAppID(12345)
		privateKey := types.GitHubAppPrivateKey("")

		client, err := ghapp.New(appID, privateKey)
		gt.Error(t, err)
		gt.V(t, client).Equal(nil)
	})

	t.Run("create with zero app ID fails", func(t *testing.T) {
		appID := types.GitHubAppID(0)
		privateKey := types.GitHubAppPrivateKey("test-key")

		client, err := ghapp.New(appID, privateKey)
		gt.Error(t, err)
		gt.V(t, client).Equal(nil)
	})

	t.Run("HTTPClient returns error with invalid key", func(t *testing.T) {
		appID := types.GitHubAppID(12345)
		privateKey := types.GitHubAppPrivateKey("invalid-key")

		client, err := ghapp.New(appID, privateKey)
		gt.NoError(t, err)

		httpClient, err := client.HTTPClient(types.GitHubAppInstallID(67890))
		gt.Error(t, err)
		gt.V(t, httpClient).Equal(nil)
	})
}

func TestListInstallationRepos_Integration(t *testing.T) {
	appIDStr := os.Getenv("TEST_GITHUB_APP_ID")
	privateKey := os.Getenv("TEST_GITHUB_PRIVATE_KEY")
	owner := os.Getenv("TEST_GITHUB_OWNER")

	if appIDStr == "" || privateKey == "" || owner == "" {
		t.Skip("TEST_GITHUB_APP_ID, TEST_GITHUB_PRIVATE_KEY, and TEST_GITHUB_OWNER must be set")
	}

	appID, err := strconv.ParseInt(appIDStr, 10, 64)
	gt.NoError(t, err)

	client, err := ghapp.New(types.GitHubAppID(appID), types.GitHubAppPrivateKey(privateKey))
	gt.NoError(t, err)

	ctx := context.Background()

	// Get installation ID for owner
	installID, err := client.GetInstallationIDForOwner(ctx, owner)
	gt.NoError(t, err)
	gt.V(t, installID).NotEqual(types.GitHubAppInstallID(0))

	t.Logf("Found installation ID: %d for owner: %s", installID, owner)

	// List repos for installation
	repos, err := client.ListInstallationRepos(ctx, installID)
	gt.NoError(t, err)

	t.Logf("Found %d repositories for owner: %s", len(repos), owner)

	// Verify repos have expected fields
	for _, repo := range repos {
		gt.V(t, repo.Owner).NotEqual("")
		gt.V(t, repo.Name).NotEqual("")
		t.Logf("  - %s/%s (default_branch: %s, archived: %v, disabled: %v)",
			repo.Owner, repo.Name, repo.DefaultBranch, repo.Archived, repo.Disabled)
	}
}
