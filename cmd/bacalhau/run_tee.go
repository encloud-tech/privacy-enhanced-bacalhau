package bacalhau

import (
	"context"
	"fmt"
	"strings"

	"github.com/filecoin-project/bacalhau/pkg/bacerrors"
	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	jobutils "github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/util/templates"
	"github.com/filecoin-project/bacalhau/pkg/version"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/kubectl/pkg/util/i18n"
	"sigs.k8s.io/yaml"
)

var (
	teeRunLong = templates.LongDesc(i18n.T(`
		Runs a Job using a TEE Based Executor on the Node
		`))

	//nolint:lll // Documentation
	teeRunExample = templates.Examples(i18n.T(`
		Documentation to be added here
	`))
)

type TEERunOptions struct {
	Engine           string   // Executor - executor.Executor
	DeploymentType   string   //Type-Constellation/Anjuna
	Verifier         string   // Verifier - verifier.Verifier
	Publisher        string   // Publisher - publisher.Publisher
	Inputs           []string // Array of input CIDs
	InputUrls        []string // Array of input URLs (will be copied to IPFS)
	InputVolumes     []string // Array of input volumes in 'CID:mount point' form
	OutputVolumes    []string // Array of output volumes in 'name:mount point' form
	Env              []string // Array of environment variables
	IDOnly           bool     // Only print the job ID
	Concurrency      int      // Number of concurrent jobs to run
	Confidence       int      // Minimum number of nodes that must agree on a verification result
	MinBids          int      // Minimum number of bids before they will be accepted (at random)
	Timeout          float64  // Job execution timeout in seconds
	CPU              string
	Memory           string
	GPU              string
	WorkingDirectory string   // Working directory for docker
	Labels           []string // Labels for the job on the Bacalhau network (for searching)

	Image      string   // Image to execute
	Entrypoint []string // Entrypoint to the docker image

	SkipSyntaxChecking bool // Verify the syntax using shellcheck

	DryRun bool // Don't submit the jobspec, print it to STDOUT

	RunTimeSettings RunTimeSettings // Settings for running the job

	DownloadFlags ipfs.IPFSDownloadSettings // Settings for running Download

	ShardingGlobPattern string
	ShardingBasePath    string
	ShardingBatchSize   int

	FilPlus bool // add a "filplus" label to the job to grab the attention of fil+ moderators
}

func NewTEERunOptions() *TEERunOptions {
	return &TEERunOptions{
		Engine:             "TEE",
		Verifier:           "noop",
		Publisher:          "estuary",
		Inputs:             []string{},
		InputUrls:          []string{},
		InputVolumes:       []string{},
		OutputVolumes:      []string{},
		Env:                []string{},
		Concurrency:        1,
		Confidence:         0,
		MinBids:            0, // 0 means no minimum before bidding
		Timeout:            DefaultTimeout.Seconds(),
		CPU:                "",
		Memory:             "",
		GPU:                "",
		SkipSyntaxChecking: false,
		WorkingDirectory:   "",
		Labels:             []string{},
		DownloadFlags:      *ipfs.NewIPFSDownloadSettings(),
		RunTimeSettings:    *NewRunTimeSettings(),

		ShardingGlobPattern: "",
		ShardingBasePath:    "/inputs",
		ShardingBatchSize:   1,

		FilPlus: false,
	}
}

func newTEECmd() *cobra.Command {
	TEECmd := &cobra.Command{
		Use:   "tee",
		Short: "Run a Job using the TEE based Executor present on the Network",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			// Check that the server version is compatible with the client version
			serverVersion, _ := GetAPIClient().Version(cmd.Context()) // Ok if this fails, version validation will skip
			if err := ensureValidVersion(cmd.Context(), version.Get(), serverVersion); err != nil {
				cmd.Println(err.Error())
				return err
			}
			return nil
		},
	}
	TEECmd.AddCommand(newTEERunCmd())
	return TEECmd
}

func newTEERunCmd() *cobra.Command {
	v1 := NewTEERunOptions()

	TEERunCmd := &cobra.Command{
		Use:     "run",
		Short:   "Run a TEE Based Job on the Network",
		Long:    teeRunLong,
		Example: teeRunExample,
		Args:    cobra.MinimumNArgs(1),
		PreRun:  applyPorcelainLogLevel,
		RunE: func(cmd *cobra.Command, cmdArgs []string) error {
			return teeRun(cmd, cmdArgs, v1)
		},
	}
	TEERunCmd.PersistentFlags().StringVar(
		&v1.DeploymentType, "DeploymentType", v1.DeploymentType,
		`What TEE Framework to use(Anjuna/Constellation)`,
	)

	TEERunCmd.PersistentFlags().StringSliceVarP(
		&v1.Inputs, "inputs", "i", v1.Inputs,
		`CIDs to use on the job. Mounts them at '/inputs' in the execution.`,
	)

	return TEERunCmd
}

func teeRun(cmd *cobra.Command, cmdArgs []string, v1 *TEERunOptions) error {
	cm := system.NewCleanupManager()
	defer cm.Cleanup()
	ctx := cmd.Context()

	ctx, rootSpan := system.NewRootSpan(ctx, system.GetTracer(), "cmd/bacalhau/dockerRun")
	defer rootSpan.End()
	cm.RegisterCallback(system.CleanupTraceProvider)

	j, err := createTEEJob(ctx, cmdArgs, v1)
	if err != nil {
		Fatal(cmd, fmt.Sprintf("Error creating job: %s", err), 1)
		return nil
	}

	err = jobutils.VerifyJob(ctx, j)
	if err != nil {
		if _, ok := err.(*bacerrors.ImageNotFound); ok {
			Fatal(cmd, fmt.Sprintf("Error to be logged out"), 1)
			return nil
		} else {
			Fatal(cmd, fmt.Sprintf("Error verifying job: %s", err), 1)
			return nil
		}
	}

	if v1.DryRun {
		// Converting job to yaml
		var yamlBytes []byte
		yamlBytes, err = yaml.Marshal(j)
		if err != nil {
			Fatal(cmd, fmt.Sprintf("Error converting job to yaml: %s", err), 1)
			return nil
		}
		cmd.Print(string(yamlBytes))
		return nil
	}

	return ExecuteJob(ctx,
		cm,
		cmd,
		j,
		v1.RunTimeSettings,
		v1.DownloadFlags,
		nil,
	)
}

func createTEEJob(ctx context.Context,
	cmdArgs []string,
	v1 *TEERunOptions) (*model.Job, error) {

	//nolint:ineffassign,staticcheck
	_, span := system.GetTracer().Start(ctx, "cmd/bacalhau/dockerRun.ProcessAndExecuteJob")
	defer span.End()

	//Specifying the Entrypoint(TODO):

	swarmAddresses := v1.DownloadFlags.IPFSSwarmAddrs

	if swarmAddresses == "" {
		swarmAddresses = strings.Join(system.Envs[system.Production].IPFSSwarmAddresses, ",")
	}

	v1.DownloadFlags = ipfs.IPFSDownloadSettings{
		TimeoutSecs:    v1.DownloadFlags.TimeoutSecs,
		OutputDir:      v1.DownloadFlags.OutputDir,
		IPFSSwarmAddrs: swarmAddresses,
	}

	engineType, err := model.ParseEngine(v1.Engine)
	if err != nil {
		return &model.Job{}, err
	}

	verifierType, err := model.ParseVerifier(v1.Verifier)
	if err != nil {
		return &model.Job{}, err
	}

	publisherType, err := model.ParsePublisher(v1.Publisher)
	if err != nil {
		return &model.Job{}, err
	}

	// for _, i := range v1.Inputs {
	//  v1.InputVolumes = append(v1.InputVolumes, v1.Sprintf("%s:/inputs", i))
	// }

	if len(v1.WorkingDirectory) > 0 {
		err = system.ValidateWorkingDir(v1.WorkingDirectory)

		if err != nil {
			return &model.Job{}, errors.Wrap(err, "CreateJobSpecAndDeal:")
		}
	}

	labels := v1.Labels

	if v1.FilPlus {
		labels = append(labels, "filplus")
	}

	j, err := jobutils.ConstructTEEJob(
		model.APIVersionLatest(),
		engineType,
		verifierType,
		publisherType,
		v1.CPU, v1.CPU, v1.GPU,
		v1.InputUrls,
		v1.InputVolumes,
		v1.OutputVolumes,
		v1.Env,
		v1.Entrypoint,
		v1.Image,
		v1.Concurrency,
		v1.Confidence,
		v1.MinBids,
		v1.Timeout,
		labels,
		v1.WorkingDirectory,
		v1.ShardingGlobPattern,
		v1.ShardingBasePath,
		v1.ShardingBatchSize,
		doNotTrack,
	)
	if err != nil {
		return &model.Job{}, errors.Wrap(err, "CreateJobSpecAndDeal")
	}

	return j, nil

}
