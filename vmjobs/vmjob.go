package vmjobs

import (
	"errors"
	"fmt"
	"github.com/urfave/cli"
	"os"
	"plugin"
)

var ImageParamDesc = "VM image to run the command on. Specify it multiple times for multiple vms."

type VMJob interface {
	// Stringer -> name for the job
	fmt.Stringer

	// Desc -> job description used as cmd line sub cmd description
	Desc() string
	// Flags -> list of cli.Flag supported specifically by the job.
	// if missing, an "image/i" required flag is automatically enforced
	Flags() []cli.Flag
	// ParseCfg -> called when program starts on a job, to parse job specific config
	ParseCfg(c *cli.Context) error
	// Cmd -> returns command to be used
	Cmd() string
	// Process -> processes each output line
	Process(VM, outputLine string)
	// Done -> called at the end of program, to let job flush its data if needed
	Done()
}

// pluginJob type is used to wrap the VMJob provided
// by plugins. Right now it is only used in main to
// distinguish between plugins and internal jobs
type pluginJob struct {
	VMJob
}

var (
	jobs               = make(map[string]VMJob)
	alreadyExistentErr = errors.New("job already registered")
	symbolNotFoundErr  = errors.New("failed to find a pluginJob exported symbol that implements VMJob interface")
	notVMJobErr        = errors.New("symbol does not implement VMJob interface")
)

// RegisterJob is used by internal plugins to register themselves in their init()
func RegisterJob(name string, job VMJob) error {
	if _, ok := jobs[name]; !ok {
		jobs[name] = job
		return nil
	}
	return alreadyExistentErr
}

func ListJobs() []VMJob {
	jSlice := make([]VMJob, 0, len(jobs))
	for _, j := range jobs {
		jSlice = append(jSlice, j)
	}
	return jSlice
}

func LoadPlugins(folder string) error {
	files, err := os.ReadDir(folder)
	if err != nil {
		return err
	}

	for _, f := range files {
		handle, err := plugin.Open(folder + "/" + f.Name())
		if err != nil {
			// Skip all non .so files
			continue
		}
		sym, err := handle.Lookup("PluginJob")
		if err != nil {
			return symbolNotFoundErr
		}
		pl, ok := sym.(VMJob)
		if !ok {
			return notVMJobErr
		}
		err = RegisterJob(pl.String(), pluginJob{pl})
		if err != nil {
			return err
		}
	}
	return nil
}

func IsPluginJob(job VMJob) bool {
	_, isPlugin := job.(pluginJob)
	return isPlugin
}
