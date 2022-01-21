package vmjobs

import (
	_ "embed"
	"errors"
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
	table *tablewriter.Table
	cmd   string
}

type bpfJob struct {
	buildTestJob
	bpfInfos map[string]*bpfInfo
}

var bpfDefaultImages = cli.StringSlice{
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

func (j *bpfJob) String() string {
	return "bpf"
}

func (j *bpfJob) Desc() string {
	return "Run bpf build + verifier job."
}

func (j *bpfJob) Flags() []cli.Flag {
	return flagsForBpfKmodTest(&bpfDefaultImages)
}

func (j *bpfJob) ParseCfg(c *cli.Context) error {
	btJob, err := newBuildTestJob(c, true, []string{"VM", "Clang", "Linux", "Scap_built", "Probe_built", "Res"})
	if err != nil {
		return err
	}
	j.buildTestJob = btJob
	j.bpfInfos = initBpfInfoMap(c.StringSlice("image"))
	return nil
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

func newBuildTestJob(c *cli.Context, isBpf bool, headers []string) (buildTestJob, error) {
	commitHash := c.String("commithash")
	forkName := c.String("forkname")

	if len(commitHash) == 0 {
		return buildTestJob{}, errors.New("empty 'commithash' value")
	}
	if len(forkName) == 0 {
		return buildTestJob{}, errors.New("empty 'forkname' value")
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(headers)
	// Markdown tables!
	table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	table.SetCenterSeparator("|")
	return buildTestJob{
		table: table,
		cmd:   fmt.Sprintf(bpfKmodCmdFmt, forkName, commitHash, isBpf),
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

func flagsForBpfKmodTest(defImages *cli.StringSlice) []cli.Flag {
	return []cli.Flag{
		cli.StringSliceFlag{
			Name:  "image,i",
			Usage: imageParamDesc,
			Value: defImages,
		},
		cli.StringFlag{
			Name:  "forkname",
			Usage: "libs fork to clone from.",
			Value: "falcosecurity",
		},
		cli.StringFlag{
			Name:  "commithash",
			Usage: "libs commit hash to run the test against.",
			Value: "master",
		},
	}
}
