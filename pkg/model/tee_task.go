package model

import (
	"strings"

	"github.com/ipld/go-ipld-prime/datamodel"
)

type TEEInputs struct {
	DiskImageAddress string
	Outputs          IPLDMap[string, datamodel.Node]
}

var _ JobType = (*TEEInputs)(nil)

func (tee TEEInputs) UnmarshalInto(with string, spec *Spec) error {
	spec.Engine = EngineTEE
	spec.TEE = JobSpecTEE{
		DiskImageAddress: tee.DiskImageAddress,
	}

	spec.Outputs = []StorageSpec{}
	for path := range tee.Outputs.Values {
		spec.Outputs = append(spec.Outputs, StorageSpec{
			Path: path,
			Name: strings.Trim(path, "/"),
		})
	}
	return nil
}
