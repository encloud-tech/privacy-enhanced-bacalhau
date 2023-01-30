package bacalhau

import (
	"github.com/filecoin-project/bacalhau/pkg/downloader/util"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/util/templates"
	"github.com/filecoin-project/bacalhau/pkg/version"
	"github.com/spf13/cobra"
	"k8s.io/kubectl/pkg/util/i18n"
)

var (
	teeRunLong = templates.LongDesc(i18n.T(`
		Runs a job using the Docker executor on the node.
		`))

	//nolint:lll // Documentation
	teeRunExample = templates.Examples(i18n.T(`
		Documentation to be added here
	`))
)

type TEERunOptions struct {
	Engine           string   // Executor - executor.Executor
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
	Networking       model.Network
	NetworkDomains   []string
	WorkingDirectory string   // Working directory for docker
	Labels           []string // Labels for the job on the Bacalhau network (for searching)
	NodeSelector     string   // Selector (label query) to filter nodes on which this job can be executed

	Image      string   // Image to execute
	Entrypoint []string // Entrypoint to the docker image

	SkipSyntaxChecking bool // Verify the syntax using shellcheck

	DryRun bool // Don't submit the jobspec, print it to STDOUT

	RunTimeSettings RunTimeSettings // Settings for running the job

	DownloadFlags model.DownloaderSettings // Settings for running Download

	ShardingGlobPattern string
	ShardingBasePath    string
	ShardingBatchSize   int

	FilPlus bool // add a "filplus" label to the job to grab the attention of fil+ moderators
}

func NewTEERunOptions() *TEERunOptions {
	return &TEERunOptions{
		Engine:             "EngineTEE",
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
		Networking:         model.NetworkNone,
		NetworkDomains:     []string{},
		SkipSyntaxChecking: false,
		WorkingDirectory:   "",
		Labels:             []string{},
		NodeSelector:       "",
		DownloadFlags:      *util.NewDownloadSettings(),
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
	return TEERunCmd
}
