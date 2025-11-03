package config

import (
	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/opac"
	"github.com/urfave/cli/v2"
)

type Policy struct {
	files cli.StringSlice
}

func (x *Policy) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringSliceFlag{
			Name:        "policy-file",
			Usage:       "Policy files to evaluate",
			EnvVars:     []string{"OCTOVY_POLICY_FILE"},
			Destination: &x.files,
		},
	}
}

func (x *Policy) Configure() (*opac.Client, error) {
	if len(x.files.Value()) == 0 {
		return nil, nil
	}

	client, err := opac.New(opac.Files(x.files.Value()...))
	if err != nil {
		return nil, goerr.Wrap(err, "Failed to initialize policy engine", goerr.V("files", x.files.Value()))
	}

	return client, nil
}
