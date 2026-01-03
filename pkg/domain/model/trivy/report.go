package trivy

import (
	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/octovy/pkg/domain/types"
)

type Report struct {
	SchemaVersion int          `json:",omitempty"`
	ReportID      string       `json:",omitempty"`
	CreatedAt     string       `json:",omitempty"`
	ArtifactID    string       `json:",omitempty"`
	ArtifactName  string       `json:",omitempty"`
	ArtifactType  ArtifactType `json:",omitempty"`
	Metadata      Metadata     `json:",omitempty"`
	Results       Results      `json:",omitempty"`
}

// Validate checks the required fields are filled. Currently, it checks only schema version.
func (x *Report) Validate() error {
	if x.SchemaVersion == 0 {
		return goerr.Wrap(types.ErrValidationFailed, "schema version is empty")
	}
	return nil
}

// Metadata represents a metadata of artifact
type Metadata struct {
	Size int64 `json:",omitempty"`
	OS   *OS   `json:",omitempty"`

	// Container image
	ImageID     string      `json:",omitempty"`
	DiffIDs     []string    `json:",omitempty"`
	RepoTags    []string    `json:",omitempty"`
	RepoDigests []string    `json:",omitempty"`
	ImageConfig *ConfigFile `json:",omitempty"`

	// Repository metadata
	RepoURL   string `json:",omitempty"`
	Branch    string `json:",omitempty"`
	Commit    string `json:",omitempty"`
	CommitMsg string `json:",omitempty"`
	Author    string `json:",omitempty"`
	Committer string `json:",omitempty"`
}

type Results []Result

type Result struct {
	Target            string                     `json:"Target"`
	Class             ResultClass                `json:"Class,omitempty"`
	Type              string                     `json:"Type,omitempty"`
	Packages          []Package                  `json:"Packages,omitempty"`
	Vulnerabilities   []DetectedVulnerability    `json:"Vulnerabilities,omitempty"`
	MisconfSummary    *MisconfSummary            `json:"MisconfSummary,omitempty"`
	Misconfigurations []DetectedMisconfiguration `json:"Misconfigurations,omitempty"`
	Secrets           []SecretFinding            `json:"Secrets,omitempty"`
	Licenses          []DetectedLicense          `json:"Licenses,omitempty"`
	// CustomResources   []ftypes.CustomResource    `json:"CustomResources,omitempty"`
}

type ResultClass string
type Compliance = string
type Format string
type ArtifactType string
type Digest string

type Status int

type Repository struct {
	Family  string `json:",omitempty"`
	Release string `json:",omitempty"`
}

type Layer struct {
	Digest    string `json:",omitempty"`
	DiffID    string `json:",omitempty"`
	CreatedBy string `json:",omitempty"`
}

type Package struct {
	ID         string   `json:",omitempty"`
	Name       string   `json:",omitempty"`
	Identifier *PackageIdentifier `json:",omitempty"`
	Version    string   `json:",omitempty"`
	Release    string   `json:",omitempty"`
	Epoch      int      `json:",omitempty"`
	Arch       string   `json:",omitempty"`
	Dev        bool     `json:",omitempty"`
	SrcName    string   `json:",omitempty"`
	SrcVersion string   `json:",omitempty"`
	SrcRelease string   `json:",omitempty"`
	SrcEpoch   int      `json:",omitempty"`
	Licenses   []string `json:",omitempty"`
	Maintainer string   `json:",omitempty"`

	Modularitylabel string     `json:",omitempty"` // only for Red Hat based distributions
	BuildInfo       *BuildInfo `json:",omitempty"` // only for Red Hat

	Ref      string `json:",omitempty"` // identifier which can be used to reference the component elsewhere
	Indirect bool   `json:",omitempty"` // this package is direct dependency of the project or not

	// Dependencies of this package
	// Note:　it may have interdependencies, which may lead to infinite loops.
	DependsOn []string `json:",omitempty"`

	Layer *Layer `json:",omitempty"`

	// Each package metadata have the file path, while the package from lock files does not have.
	FilePath string `json:",omitempty"`

	// This is required when using SPDX formats. Otherwise, it will be empty.
	Digest Digest `json:",omitempty"`

	// lines from the lock file where the dependency is written
	Locations []Location `json:",omitempty"`

	// Relationship with the project (root, direct, indirect)
	Relationship string `json:",omitempty"`
}

type PackageIdentifier struct {
	PURL string `json:",omitempty"`
	UID  string `json:",omitempty"`
}

type Location struct {
	StartLine int `json:",omitempty"`
	EndLine   int `json:",omitempty"`
}

// BuildInfo represents information under /root/buildinfo in RHEL
type BuildInfo struct {
	ContentSets []string `json:",omitempty"`
	Nvr         string   `json:",omitempty"`
	Arch        string   `json:",omitempty"`
}
