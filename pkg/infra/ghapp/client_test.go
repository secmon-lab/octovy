package ghapp_test

import (
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
