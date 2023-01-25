package tee

import (
	"context"
	"fmt"

	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/storage"
)

type Executor struct {
	Jobs map[string]*model.Job

	executors executor.ExecutorProvider
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
	shard.Job.Spec.Engine = model.EngineTEE

	return &model.RunCommandResult{}, nil
}

func NewExecutor(
	ctx context.Context,
	storageProvider storage.StorageProvider,
) (*Executor, error) {
	// TODO: add host-specific config about TEE runtimes need to provided
	// engine := wazero.NewRuntime(ctx)

	// executor := &Executor{
	// 	Engine:          engine,
	// 	StorageProvider: storageProvider,
	// }

	return (*Executor)(nil), nil
}

// IsInstalled checks if tee cli tool itself is installed.
func (e *Executor) IsInstalled(ctx context.Context) (bool, error) {
	teeExecutor, err := e.executors.GetExecutor(ctx, model.EngineTEE)
	if err != nil {
		return false, err
	}
	return teeExecutor.IsInstalled(ctx)
}

func (e *Executor) HasStorageLocally(context.Context, model.StorageSpec) (bool, error) {
	return true, nil
}

func (e *Executor) GetVolumeSize(context.Context, model.StorageSpec) (uint64, error) {
	return 0, nil
}

func (e *Executor) CancelShard(ctx context.Context, shard model.JobShard) error {
	teeExecutor, err := e.executors.GetExecutor(ctx, model.EngineTEE)
	if err != nil {
		return err
	}
	return teeExecutor.CancelShard(ctx, shard)
}

// Compile-time check that Executor implements the Executor interface.
var _ executor.Executor = (*Executor)(nil)
