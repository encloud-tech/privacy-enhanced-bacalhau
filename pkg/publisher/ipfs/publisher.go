package ipfs

import (
	"context"
	"fmt"

	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/publisher"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/trace"
)

type IPFSPublisher struct {
	IPFSClient  *ipfs.Client
	JobLoader   job.JobLoader
	StateLoader job.StateLoader
}

func NewIPFSPublisher(
	cm *system.CleanupManager,
	ipfsAPIAddr string,
	jobLoader job.JobLoader,
	stateLoader job.StateLoader,
) (*IPFSPublisher, error) {
	cl, err := ipfs.NewClient(ipfsAPIAddr)
	if err != nil {
		return nil, err
	}

	log.Debug().Msgf("IPFS publisher initialized for node: %s", ipfsAPIAddr)
	return &IPFSPublisher{
		IPFSClient:  cl,
		JobLoader:   jobLoader,
		StateLoader: stateLoader,
	}, nil
}

func (publisher *IPFSPublisher) IsInstalled(ctx context.Context) (bool, error) {
	ctx, span := newSpan(ctx, "IsInstalled")
	defer span.End()

	_, err := publisher.IPFSClient.ID(ctx)
	return err == nil, err
}

func (publisher *IPFSPublisher) PublishShardResult(
	ctx context.Context,
	hostID string,
	jobID string,
	shardIndex int,
	shardResultPath string,
) (*storage.StorageSpec, error) {
	ctx, span := newSpan(ctx, "PublishShardResult")
	defer span.End()
	log.Debug().Msgf(
		"Uploading results folder to ipfs: %s %s %d %s",
		hostID,
		jobID,
		shardIndex,
		shardResultPath,
	)
	cid, err := publisher.IPFSClient.Put(ctx, shardResultPath)
	if err != nil {
		return nil, err
	}
	return &storage.StorageSpec{
		Name:   fmt.Sprintf("job-%s-shard-%d-host-%s", jobID, shardIndex, hostID),
		Engine: storage.StorageSourceIPFS,
		Cid:    cid,
	}, nil
}

func (publisher *IPFSPublisher) ComposeResultReferences(
	ctx context.Context,
	jobID string,
) ([]storage.StorageSpec, error) {
	results := []storage.StorageSpec{}
	ctx, span := newSpan(ctx, "ComposeResultSet")
	defer span.End()
	resolver := publisher.getStateResolver()
	shardResults, err := resolver.GetResults(ctx, jobID)
	if err != nil {
		return results, nil
	}
	for _, shardResult := range shardResults {
		results = append(results, storage.StorageSpec{
			Name:   fmt.Sprintf("shard%d", shardResult.ShardIndex),
			Path:   fmt.Sprintf("shard%d", shardResult.ShardIndex),
			Engine: storage.StorageSourceIPFS,
			Cid:    string(shardResult.ResultsProposal),
		})
	}
	return results, nil
}

func (publisher *IPFSPublisher) getStateResolver() *job.StateResolver {
	return job.NewStateResolver(
		publisher.JobLoader,
		publisher.StateLoader,
	)
}

func newSpan(ctx context.Context, apiName string) (context.Context, trace.Span) {
	return system.Span(ctx, "publisher/ipfs", apiName)
}

// Compile-time check that Verifier implements the correct interface:
var _ publisher.Publisher = (*IPFSPublisher)(nil)
