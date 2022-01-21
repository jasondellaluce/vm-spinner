package vmjobs

import (
	_ "embed"
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

var kmodDefaultImages = cli.StringSlice{
	"generic/fedora33",
	"generic/fedora35",
	"ubuntu/focal64",
	"ubuntu/bionic64",
	"generic/debian10",
	"generic/centos8",
	"bento/amazonlinux-2",
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

func (j *kmodJob) String() string {
	return "kmod"
}

func (j *kmodJob) Desc() string {
	return "Run kmod build job."
}

func (j *kmodJob) Flags() []cli.Flag {
	return flagsForBpfKmodTest(&kmodDefaultImages)
}

func (j *kmodJob) ParseCfg(c *cli.Context) error {
	btJob, err := newBuildTestJob(c, false, []string{"VM", "GCC", "Linux", "Kmod_built"})
	if err != nil {
		return err
	}
	j.buildTestJob = btJob
	j.kmodInfos = initKmodInfoMap(c.StringSlice("image"))
	return nil
}

func (j *kmodJob) Cmd() string {
	return j.cmd
}

func (j *kmodJob) Process(output VMOutput) {
	outputs := strings.Split(output.Line, ": ")
	info := j.kmodInfos[output.VM]
	switch outputs[0] {
	case "GCC_VERSION":
		info.gcc = outputs[1]
	case "LINUX_VERSION":
		info.linux = outputs[1]
	case "DRIVER_BUILT", "ERROR":
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
