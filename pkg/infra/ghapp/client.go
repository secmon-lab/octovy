package ghapp

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v53/github"
	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/octovy/pkg/domain/interfaces"
	"github.com/m-mizutani/octovy/pkg/domain/types"
	"github.com/m-mizutani/octovy/pkg/utils/logging"
)

type Client struct {
	appID types.GitHubAppID
	pem   types.GitHubAppPrivateKey
}

var _ interfaces.GitHubApp = (*Client)(nil)

func New(appID types.GitHubAppID, pem types.GitHubAppPrivateKey) (*Client, error) {
	if appID == 0 {
		return nil, goerr.Wrap(types.ErrInvalidOption, "appID is empty")
	}
	if pem == "" {
		return nil, goerr.Wrap(types.ErrInvalidOption, "pem is empty")
	}

	client := &Client{
		appID: appID,
		pem:   pem,
	}

	return client, nil
}

func (x *Client) buildGithubClient(installID types.GitHubAppInstallID) (*github.Client, error) {
	httpClient, err := x.buildGithubHTTPClient(installID)
	if err != nil {
		return nil, err
	}
	return github.NewClient(httpClient), nil
}

func (x *Client) buildGithubHTTPClient(installID types.GitHubAppInstallID) (*http.Client, error) {
	tr := http.DefaultTransport
	itr, err := ghinstallation.New(tr, int64(x.appID), int64(installID), []byte(x.pem))

	if err != nil {
		return nil, goerr.Wrap(err, "Failed to create github client")
	}

	client := &http.Client{Transport: itr}
	return client, nil
}

func (x *Client) GetArchiveURL(ctx context.Context, input *interfaces.GetArchiveURLInput) (*url.URL, error) {
	logging.From(ctx).Info("Sending GetArchiveLink request",
		slog.Any("appID", x.appID),
		slog.Any("privateKey", x.pem),
		slog.Any("input", input),
	)

	client, err := x.buildGithubClient(input.InstallID)
	if err != nil {
		return nil, err
	}

	opt := &github.RepositoryContentGetOptions{
		Ref: input.CommitID,
	}

	// https://docs.github.com/en/rest/reference/repos#downloads
	// https://docs.github.com/en/rest/repos/contents?apiVersion=2022-11-28#get-archive-link
	url, r, err := client.Repositories.GetArchiveLink(ctx, input.Owner, input.Repo, github.Zipball, opt, false)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get archive link")
	}
	if r.StatusCode != http.StatusFound {
		body, _ := io.ReadAll(r.Body)
		return nil, goerr.New("Failed to get archive link", goerr.V("status", r.StatusCode), goerr.V("body", string(body)))
	}

	logging.From(ctx).Debug("GetArchiveLink response", slog.Any("url", url), slog.Any("r", r))

	return url, nil
}

func (x *Client) HTTPClient(installID types.GitHubAppInstallID) (*http.Client, error) {
	return x.buildGithubHTTPClient(installID)
}
