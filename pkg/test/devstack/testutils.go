package devstack

import (
	"fmt"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/devstack"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/jsonrpc"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/test/scenario"
	"github.com/filecoin-project/bacalhau/pkg/types"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	"github.com/stretchr/testify/assert"

	"github.com/rs/zerolog/log"
)

var STORAGE_DRIVER_NAMES = []string{
	storage.IPFS_FUSE_DOCKER,
	storage.IPFS_API_COPY,
}

func SetupTest(
	t *testing.T,
	nodes int,
	badActors int,
) (*devstack.DevStack, *system.CancelContext) {
	cancelContext := system.GetCancelContextWithSignals()
	getExecutors := func(ipfsMultiAddress string, nodeIndex int) (map[string]executor.Executor, error) {
		return executor.NewDockerIPFSExecutors(cancelContext, ipfsMultiAddress, fmt.Sprintf("devstacknode%d", nodeIndex))
	}
	stack, err := devstack.NewDevStack(
		cancelContext,
		nodes,
		badActors,
		getExecutors,
	)
	assert.NoError(t, err)
	if err != nil {
		log.Fatal().Msg(fmt.Sprintf("Unable to create devstack: %s", err))
	}
	return stack, cancelContext
}

// this might be called multiple times if KEEP_STACK is active
// the first time - once the test has completed, this function will be called
// it will reset the KEEP_STACK variable so the user can ctrl+c the running stack
func TeardownTest(stack *devstack.DevStack, cancelContext *system.CancelContext) {
	if !system.ShouldKeepStack() {
		cancelContext.Stop()
	} else {
		stack.PrintNodeInfo()
		select {}
	}
}

// re-use the docker executor tests but full end to end with libp2p transport
// and 3 nodes
func devStackDockerStorageTest(
	t *testing.T,
	testCase scenario.TestCase,
	nodeCount int,
) {

	stack, cancelContext := SetupTest(
		t,
		nodeCount,
		0,
	)

	defer TeardownTest(stack, cancelContext)

	rpcHost := "127.0.0.1"
	rpcPort := stack.Nodes[0].JSONRpcNode.Port

	inputStorageList, err := testCase.SetupStorage(stack, storage.IPFS_API_COPY, nodeCount)
	assert.NoError(t, err)

	// this is stdout mode
	outputs := []types.StorageSpec{}

	jobSpec := &types.JobSpec{
		Engine:   string(executor.EXECUTOR_DOCKER),
		Verifier: string(verifier.VERIFIER_NOOP),
		Vm:       testCase.GetJobSpec(),
		Inputs:   inputStorageList,
		Outputs:  outputs,
	}

	jobDeal := &types.JobDeal{
		Concurrency: nodeCount,
	}

	job, err := jsonrpc.SubmitJob(jobSpec, jobDeal, rpcHost, rpcPort)
	assert.NoError(t, err)

	if err != nil {
		t.FailNow()
	}

	err = stack.WaitForJob(job.Id, map[string]int{
		system.JOB_STATE_COMPLETE: nodeCount,
	}, []string{
		system.JOB_STATE_BID_REJECTED,
		system.JOB_STATE_ERROR,
	})
	assert.NoError(t, err)

	// jobs, err := jobutils.ListJobs(rpcHost, rpcPort)
	// assert.NoError(t, err)
}
