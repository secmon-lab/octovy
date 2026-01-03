package types

import "log/slog"

type (
	GitHubAppID         int64
	GitHubAppInstallID  int64
	GitHubAppSecret     string
	GitHubAppPrivateKey string
)

func (x GitHubAppSecret) LogValue() slog.Value {
	return slog.StringValue("***********")
}

func (x GitHubAppPrivateKey) LogValue() slog.Value {
	return slog.StringValue("***********")
}
