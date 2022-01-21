package vmjobs

import (
	_ "embed"
	"fmt"
	"github.com/olekukonko/tablewriter"
	log "github.com/sirupsen/logrus"
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
	table *tablewriter.Table
	cmd   string
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

//go:embed scripts/bpf_kmod_job.sh
var bpfKmodCmdFmt string

func newBuildTestJob(c *cli.Context, isBpf bool, headers []string) buildTestJob {
	commitHash := c.String("commithash")
	forkName := c.String("forkname")

	if len(commitHash) == 0 {
		log.Fatalf("empty 'commithash' value")
	}
	if len(forkName) == 0 {
		log.Fatalf("empty 'forkname' value")
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(headers)
	// Markdown tables!
	table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	table.SetCenterSeparator("|")
	return buildTestJob{
		table: table,
		cmd:   fmt.Sprintf(bpfKmodCmdFmt, forkName, commitHash, isBpf),
	}
}

func newBpfJob(c *cli.Context) (*bpfJob, error) {
	return &bpfJob{
		buildTestJob: newBuildTestJob(c, true, []string{"VM", "Clang", "Linux", "Scap_built", "Probe_built", "Res"}),
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
	return j.cmd
}

func (j *bpfJob) Process(output VMOutput) {
	outputs := strings.Split(output.Line, ": ")
	info := j.bpfInfos[output.VM]
	switch outputs[0] {
	case "CLANG_VERSION":
		info.clang = outputs[1]
	case "LINUX_VERSION":
		info.linux = outputs[1]
	case "SCAP_BUILT":
		info.scapBuilt, _ = strconv.ParseBool(outputs[1])
	case "PROBE_BUILT":
		info.probeBuilt, _ = strconv.ParseBool(outputs[1])
	case "VERIFIER_TEST", "ERROR":
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
