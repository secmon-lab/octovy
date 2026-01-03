package trivy

import "time"

type ConfigFile struct {
	Architecture string `json:"architecture,omitempty"`
	Author       string `json:"author,omitempty"`
	Container    string `json:"container,omitempty"`
	// Created       Time    `json:"created,omitempty"`
	Created       string    `json:"created,omitempty"`
	DockerVersion string    `json:"docker_version,omitempty"`
	History       []History `json:"history,omitempty"`
	OS            string    `json:"os,omitempty"`
	RootFS        *RootFS   `json:"rootfs,omitempty"`
	Config        *Config   `json:"config,omitempty"`

	// BigQuery does not support a field name with a dot, so we need to skip this field.
	// OSVersion  string `json:"os.version,omitempty"`
	// OSFeatures []string `json:"os.features,omitempty"`

	Variant string `json:"variant,omitempty"`
}

type History struct {
	Author     string `json:"author,omitempty"`
	Created    string `json:"created,omitempty"`
	CreatedBy  string `json:"created_by,omitempty"`
	Comment    string `json:"comment,omitempty"`
	EmptyLayer bool   `json:"empty_layer,omitempty"`
}

type OS struct {
	Family string
	Name   string
	Eosl   bool `json:"EOSL,omitempty"`

	// This field is used for enhanced security maintenance programs such as Ubuntu ESM, Debian Extended LTS.
	Extended bool `json:"extended,omitempty"`
}

type RootFS struct {
	Type    string `json:"type,omitempty"`
	DiffIDs []Hash `json:"diff_ids,omitempty"`
}

type Hash struct {
	// Algorithm holds the algorithm used to compute the hash.
	Algorithm string

	// Hex holds the hex portion of the content hash.
	Hex string
}

type Config struct {
	AttachStderr    bool                `json:"AttachStderr,omitempty"`
	AttachStdin     bool                `json:"AttachStdin,omitempty"`
	AttachStdout    bool                `json:"AttachStdout,omitempty"`
	Cmd             []string            `json:"Cmd,omitempty"`
	Healthcheck     *HealthConfig       `json:"Healthcheck,omitempty"`
	Domainname      string              `json:"Domainname,omitempty"`
	Entrypoint      []string            `json:"Entrypoint,omitempty"`
	Env             []string            `json:"Env,omitempty"`
	Hostname        string              `json:"Hostname,omitempty"`
	Image           string              `json:"Image,omitempty"`
	Labels          map[string]string   `json:"Labels,omitempty"`
	OnBuild         []string            `json:"OnBuild,omitempty"`
	OpenStdin       bool                `json:"OpenStdin,omitempty"`
	StdinOnce       bool                `json:"StdinOnce,omitempty"`
	Tty             bool                `json:"Tty,omitempty"`
	User            string              `json:"User,omitempty"`
	Volumes         map[string]struct{} `json:"Volumes,omitempty"`
	WorkingDir      string              `json:"WorkingDir,omitempty"`
	ExposedPorts    map[string]struct{} `json:"ExposedPorts,omitempty"`
	ArgsEscaped     bool                `json:"ArgsEscaped,omitempty"`
	NetworkDisabled bool                `json:"NetworkDisabled,omitempty"`
	MacAddress      string              `json:"MacAddress,omitempty"`
	StopSignal      string              `json:"StopSignal,omitempty"`
	Shell           []string            `json:"Shell,omitempty"`
}

type HealthConfig struct {
	// Test is the test to perform to check that the container is healthy.
	// An empty slice means to inherit the default.
	// The options are:
	// {} : inherit healthcheck
	// {"NONE"} : disable healthcheck
	// {"CMD", args...} : exec arguments directly
	// {"CMD-SHELL", command} : run command with system's default shell
	Test []string `json:",omitempty"`

	// Zero means to inherit. Durations are expressed as integer nanoseconds.
	Interval    time.Duration `json:",omitempty"` // Interval is the time to wait between checks.
	Timeout     time.Duration `json:",omitempty"` // Timeout is the time to wait before considering the check to have hung.
	StartPeriod time.Duration `json:",omitempty"` // The start period for the container to initialize before the retries starts to count down.

	// Retries is the number of consecutive failures needed to consider a container as unhealthy.
	// Zero means inherit.
	Retries int `json:",omitempty"`
}
