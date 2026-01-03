package types

import "log/slog"

type (
	GitHubAppID         int64
	GitHubAppInstallID  int64
	GitHubAppSecret     string
	GitHubAppPrivateKey string
	GitHubRepoID        string
	BranchName          string
	TargetID            string
	CommitSHA           string
	ScanStatus          string
)

const (
	ScanStatusSuccess ScanStatus = "success"
	ScanStatusFailure ScanStatus = "failure"
	ScanStatusPending ScanStatus = "pending"
)

func (x GitHubAppSecret) LogValue() slog.Value {
	return slog.StringValue("***********")
}

func (x GitHubAppSecret) String() string {
	return "***********"
}

func (x GitHubAppPrivateKey) LogValue() slog.Value {
	return slog.StringValue("***********")
}

func (x GitHubAppPrivateKey) String() string {
	return "***********"
}
