package tee

import (
	"context"
	"fmt"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/storage"
)

type Executor struct {
	// used to allow multiple temperory executors to run against the compute node
	ID string

	// the storage providers we can implement for a job
	StorageProvider storage.StorageProvider

	// Client
}

func (e *Executor) RunShard(
	ctx context.Context,
	shard model.JobShard,
	jobResultsDir string,
) (*model.RunCommandResult, error) {

	if shard.Job.Spec.TEE.ClICommandToExecute != "" {

		fmt.Println(shard.Job.Spec.TEE.ClICommandToExecute)
	}

	//nolint:ineffassign,staticcheck
	shard.Job.Spec.Engine = model.EngineCLI

	return &model.RunCommandResult{}, nil
}
