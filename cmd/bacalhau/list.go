package bacalhau

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/types"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

func init() {
	listCmd.PersistentFlags().StringVar(
		&listOutputFormat, "output", "text",
		`The output format for the list of jobs (json or text)`,
	)
	listCmd.PersistentFlags().BoolVar(
		&tableOutputWide, "wide", false,
		`Print full values in the table results`,
	)
	listCmd.PersistentFlags().IntVarP(
		&tableMaxJobs, "number", "n", 10,
		`print the first NUM jobs instead of the first 10.`,
	)
	listCmd.PersistentFlags().BoolVar(&tableHideHeader, "hide-header", false,
		`do not print the column headers.`)

	listCmd.PersistentFlags().Var(&tableSortBy, "sort-by",
		`sort by field, defaults to creation time, with newest first [Allowed "id", "created_at"].`)
	listCmd.PersistentFlags().BoolVar(&tableSortReverse, "reverse", false,
		`reverse order of table - for time sorting, this will be newest first.`)
	listCmd.PersistentFlags().StringVar(&tableIdFilter, "id-filter", "", `filter by Job List to IDs matching substring.`)
	listCmd.PersistentFlags().BoolVar(&tableNoStyle, "no-style", false, `remove all styling from table output.`)
}

// From: https://stackoverflow.com/questions/50824554/permitted-flag-values-for-cobra
type ColumnEnum string

const (
	ColumnID        ColumnEnum = "id"
	ColumnCreatedAt ColumnEnum = "created_at"
)

func (c *ColumnEnum) String() string {
	return string(*c)
}

// Type is only used in help text
func (c *ColumnEnum) Type() string {
	return "Column"
}

// Set must have pointer receiver so it doesn't change the value of a copy
func (c *ColumnEnum) Set(v string) error {
	switch v {
	case string(ColumnID), string(ColumnCreatedAt):
		*c = ColumnEnum(v)
		return nil
	default:
		return errors.New(`must be one of "id", or "created_at"`)
	}
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List jobs on the network",
	RunE: func(cmd *cobra.Command, cmdArgs []string) error {
		jobs, err := getAPIClient().List()
		if err != nil {
			return err
		}

		if listOutputFormat == "json" {
			msgBytes, err := json.MarshalIndent(jobs, "", "    ")
			if err != nil {
				return err
			}

			fmt.Printf("%s\n", msgBytes)
			return nil
		}

		t := table.NewWriter()
		t.SetOutputMirror(cmd.OutOrStderr())
		if !tableHideHeader {
			t.AppendHeader(table.Row{"id", "job", "creation_time", "inputs", "outputs", "concurrency", "node", "state", "result"})
		}
		t.SetColumnConfigs([]table.ColumnConfig{
			{Number: 1, AutoMerge: true},
			{Number: 2, AutoMerge: true},
			{Number: 3, AutoMerge: true},
			{Number: 4, AutoMerge: true},
			{Number: 5, AutoMerge: true},
		})

		// Create an external structure to order the print out of the jobs map
		keysToSort := make([]string, 0, len(jobs))
		mappedJobs := make(map[string]*types.Job, len(jobs))
		for jobSpec, job := range jobs {
			var k string
			switch tableSortBy {
			case ColumnID:
				k = job.Id
			case ColumnCreatedAt:
				k = job.CreatedAt.Format(time.RFC3339)
			default:
				k = jobSpec // The existing sort
			}
			keysToSort = append(keysToSort, k)
			mappedJobs[k] = job
		}
		sort.Strings(keysToSort)
		if tableSortReverse {
			keysToSort = ReverseList(keysToSort)
		}

		numberInTable := Min(tableMaxJobs, len(keysToSort))

		for _, key := range keysToSort[0:numberInTable] {
			job := mappedJobs[key]
			jobDesc := []string{
				job.Spec.Engine,
			}

			if job.Spec.Engine == string(executor.EXECUTOR_DOCKER) {
				jobDesc = append(jobDesc, job.Spec.Vm.Image)
				jobDesc = append(jobDesc, strings.Join(job.Spec.Vm.Entrypoint, " "))
			}

			if len(job.State) == 0 {
				t.AppendRows([]table.Row{
					{
						shortId(job.Id),
						shortenString(strings.Join(jobDesc, " ")),
						job.CreatedAt.Format("06-01-02-15:04:05"),
						len(job.Spec.Inputs),
						len(job.Spec.Outputs),
						job.Deal.Concurrency,
						"",
						"waiting",
						"",
					},
				})
			} else {
				for node, jobState := range job.State {
					t.AppendRows([]table.Row{
						{
							shortId(job.Id),
							shortenString(strings.Join(jobDesc, " ")),
							job.CreatedAt.Format("06-01-02-15:04:05"),
							len(job.Spec.Inputs),
							len(job.Spec.Outputs),
							job.Deal.Concurrency,
							shortId(node),
							shortenString(jobState.State),
							shortenString(getJobResult(job, jobState)),
						},
					})
				}
			}

		}
		if tableNoStyle {
			t.SetStyle(table.Style{
				Name:   "StyleDefault",
				Box:    table.StyleBoxDefault,
				Color:  table.ColorOptionsDefault,
				Format: table.FormatOptionsDefault,
				HTML:   table.DefaultHTMLOptions,
				Options: table.Options{
					DrawBorder:      false,
					SeparateColumns: false,
					SeparateFooter:  false,
					SeparateHeader:  false,
					SeparateRows:    false,
				},
				Title: table.TitleOptionsDefault,
			})
		} else {
			t.SetStyle(table.StyleColoredGreenWhiteOnBlack)
		}
		t.Render()

		return nil
	},
}
