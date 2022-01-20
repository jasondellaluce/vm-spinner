package vmjobs

import (
	_ "embed"
	"fmt"
	"github.com/urfave/cli"
	"strconv"
	"strings"
)

type kmodInfo struct {
	gcc       string
	linux     string
	kmodBuilt bool
}

type kmodJob struct {
	buildTestJob
	kmodInfos map[string]*kmodInfo
}

var KmodDefaultImages = cli.StringSlice{
	"generic/fedora33",
	"generic/fedora35",
	"ubuntu/focal64",
	"ubuntu/bionic64",
	"generic/debian10",
	"generic/centos8",
	"bento/amazonlinux-2",
}

//go:embed scripts/kmod_job.sh
var kmodCmdFmt string

func newKmodJob(c *cli.Context) (*kmodJob, error) {
	return &kmodJob{
		buildTestJob: newBuildTestJob(c, []string{"VM", "GCC", "Linux", "Kmod_built"}),
		kmodInfos:    initKmodInfoMap(c.StringSlice("image")),
	}, nil
}

// Preinitialize map with meaningful values so that we will access it readonly,
// and there will be no need for concurrent access strategies
func initKmodInfoMap(images []string) map[string]*kmodInfo {
	kmodInfos := make(map[string]*kmodInfo)
	for _, image := range images {
		kmodInfos[image] = &kmodInfo{
			gcc:       "N/A",
			linux:     "N/A",
			kmodBuilt: false,
		}
	}
	return kmodInfos
}

func (j *kmodJob) Cmd() string {
	return fmt.Sprintf(kmodCmdFmt, j.forkName, j.commitHash)
}

func (j *kmodJob) Process(output VMOutput) {
	outputs := strings.Split(output.Line, ": ")
	info := j.kmodInfos[output.VM]
	switch outputs[0] {
	case "KMOD_DRIVER_GCC":
		info.gcc = outputs[1]
	case "KMOD_DRIVER_LINUX":
		info.linux = outputs[1]
	case "KMOD_DRIVER_BUILT":
		info.kmodBuilt, _ = strconv.ParseBool(outputs[1])
	}
}

func (j *kmodJob) Done() {
	for vm, info := range j.kmodInfos {
		j.table.Append([]string{vm, info.gcc, info.linux,
			strconv.FormatBool(info.kmodBuilt)})
	}
	j.table.Render()
}
