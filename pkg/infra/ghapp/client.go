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
	"github.com/m-mizutani/octovy/pkg/domain/model"
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

func (x *Client) ListInstallationRepos(ctx context.Context, installID types.GitHubAppInstallID) ([]*model.GitHubAPIRepository, error) {
	client, err := x.buildGithubClient(installID)
	if err != nil {
		return nil, err
	}

	var allRepos []*model.GitHubAPIRepository
	opts := &github.ListOptions{PerPage: 100}

	for {
		result, resp, err := client.Apps.ListRepos(ctx, opts)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to list installation repos")
		}

		for _, repo := range result.Repositories {
			allRepos = append(allRepos, &model.GitHubAPIRepository{
				Owner:         repo.GetOwner().GetLogin(),
				Name:          repo.GetName(),
				DefaultBranch: repo.GetDefaultBranch(),
				Archived:      repo.GetArchived(),
				Disabled:      repo.GetDisabled(),
			})
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	logging.From(ctx).Info("Listed installation repos",
		slog.Int("count", len(allRepos)),
		slog.Any("installID", installID),
	)

	return allRepos, nil
}

func (x *Client) buildAppClient() (*github.Client, error) {
	tr := http.DefaultTransport
	itr, err := ghinstallation.NewAppsTransport(tr, int64(x.appID), []byte(x.pem))
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create app transport")
	}
	return github.NewClient(&http.Client{Transport: itr}), nil
}

func (x *Client) GetInstallationIDForOwner(ctx context.Context, owner string) (types.GitHubAppInstallID, error) {
	client, err := x.buildAppClient()
	if err != nil {
		return 0, err
	}

	// Try organization installation first
	installation, resp, orgErr := client.Apps.FindOrganizationInstallation(ctx, owner)
	if orgErr == nil && installation != nil {
		logging.From(ctx).Info("Found organization installation",
			slog.String("owner", owner),
			slog.Int64("installID", installation.GetID()),
		)
		return types.GitHubAppInstallID(installation.GetID()), nil
	}

	// If not found as org (404), try user installation
	if resp != nil && resp.StatusCode == http.StatusNotFound {
		installation, _, userErr := client.Apps.FindUserInstallation(ctx, owner)
		if userErr != nil {
			return 0, goerr.Wrap(userErr, "failed to find user installation for owner",
				goerr.V("owner", owner),
			)
		}

		if installation != nil {
			logging.From(ctx).Info("Found user installation",
				slog.String("owner", owner),
				slog.Int64("installID", installation.GetID()),
			)
			return types.GitHubAppInstallID(installation.GetID()), nil
		}
	}

	// If org lookup failed with non-404 error, propagate it
	if orgErr != nil {
		return 0, goerr.Wrap(orgErr, "failed to find organization installation for owner",
			goerr.V("owner", owner),
		)
	}

	return 0, goerr.Wrap(types.ErrInvalidGitHubData, "installation not found for owner",
		goerr.V("owner", owner),
	)
}
