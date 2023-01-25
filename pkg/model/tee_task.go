package model

import (
	"strings"

	"github.com/ipld/go-ipld-prime/datamodel"
)

type TEEInputs struct {
	CliCommnadToExecute string
	// Entrypoint []string
	// Workdir    string
	// Mounts     IPLDMap[string, Resource]
	Outputs IPLDMap[string, datamodel.Node]
	// Env        IPLDMap[string, string]
}

var _ JobType = (*TEEInputs)(nil)

func (tee TEEInputs) UnmarshalInto(with string, spec *Spec) error {
	spec.Engine = EngineTEE
	spec.TEE = JobSpecTEE{
		ClICommandToExecute: tee.CliCommnadToExecute,
		// Entrypoint:       docker.Entrypoint,
		// WorkingDirectory: docker.Workdir,
	}

	// spec.Docker.EnvironmentVariables = []string{}
	// for key, val := range docker.Env.Values {
	// 	spec.Docker.EnvironmentVariables = append(spec.Docker.EnvironmentVariables, key, val)
	// }

	// inputData, err := parseInputs(docker.Mounts)
	// if err != nil {
	// 	return err
	// }
	// spec.Inputs = inputData

	spec.Outputs = []StorageSpec{}
	for path := range tee.Outputs.Values {
		spec.Outputs = append(spec.Outputs, StorageSpec{
			Path: path,
			Name: strings.Trim(path, "/"),
		})
	}
	return nil
}
