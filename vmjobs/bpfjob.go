package vmjobs

import (
	_ "embed"
	"fmt"
	"github.com/olekukonko/tablewriter"
	"github.com/urfave/cli"
	"os"
	"strconv"
	"strings"
)

type bpfInfo struct {
	clang      string
	linux      string
	scapBuilt  bool
	probeBuilt bool
	res        string
}

type buildTestJob struct {
	table       *tablewriter.Table
	checkoutCmd string
}

type bpfJob struct {
	buildTestJob
	bpfInfos map[string]*bpfInfo
}

var BpfDefaultImages = cli.StringSlice{
	"generic/fedora33",
	"generic/fedora35",
	"ubuntu/focal64",
	"ubuntu/bionic64",
	"generic/debian10",
	"generic/centos8",
	"bento/amazonlinux-2",
}

//go:embed scripts/bpf_job.sh
var bpfCmdFmt string

func newBuildTestJob(c *cli.Context, headers []string) buildTestJob {
	var checkoutCmd string
	if c.IsSet("commithash") {
		checkoutCmd = fmt.Sprintf("git checkout %s", c.String("commithash"))
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(headers)
	// Markdown tables!
	table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	table.SetCenterSeparator("|")
	return buildTestJob{
		table:       table,
		checkoutCmd: checkoutCmd,
	}
}

func newBpfJob(c *cli.Context) (*bpfJob, error) {
	return &bpfJob{
		buildTestJob: newBuildTestJob(c, []string{"VM", "Clang", "Linux", "Scap_built", "Probe_built", "Res"}),
		bpfInfos:     initBpfInfoMap(c.StringSlice("image")),
	}, nil
}

// Preinitialize map with meaningful values so that we will access it readonly,
// and there will be no need for concurrent access strategies
func initBpfInfoMap(images []string) map[string]*bpfInfo {
	bpfInfos := make(map[string]*bpfInfo)
	for _, image := range images {
		bpfInfos[image] = &bpfInfo{
			clang:      "N/A",
			linux:      "N/A",
			scapBuilt:  false,
			probeBuilt: false,
			res:        "N/A",
		}
	}
	return bpfInfos
}

func (j *bpfJob) Cmd() string {
	return fmt.Sprintf(bpfCmdFmt, j.checkoutCmd)
}

func (j *bpfJob) Process(output VMOutput) {
	outputs := strings.Split(output.Line, ": ")
	info := j.bpfInfos[output.VM]
	switch outputs[0] {
	case "BPF_VERIFIER_CLANG":
		info.clang = outputs[1]
	case "BPF_VERIFIER_LINUX":
		info.linux = outputs[1]
	case "BPF_VERIFIER_SCAP":
		info.scapBuilt, _ = strconv.ParseBool(outputs[1])
	case "BPF_VERIFIER_PROBE":
		info.probeBuilt, _ = strconv.ParseBool(outputs[1])
	case "BPF_VERIFIER_TEST", "ERROR":
		info.res = outputs[1]
	}
}

func (j *bpfJob) Done() {
	for vm, info := range j.bpfInfos {
		j.table.Append([]string{vm, info.clang, info.linux,
			strconv.FormatBool(info.scapBuilt),
			strconv.FormatBool(info.probeBuilt),
			info.res})
	}
	j.table.Render()
}
