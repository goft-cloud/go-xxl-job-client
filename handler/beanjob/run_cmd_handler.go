package beanjob

import (
	"context"
	"errors"
	"io"
	"os/exec"

	"github.com/goft-cloud/go-xxl-job-client/v2/handler"
	"github.com/goft-cloud/go-xxl-job-client/v2/logger"
	"github.com/gookit/goutil/arrutil"
	"github.com/gookit/goutil/cliutil/cmdline"
	"github.com/gookit/goutil/strutil"
)

// RunCmdHandler struct
type RunCmdHandler struct {
	allowCmds []string
}

// NewCmdHandler bean handler
func NewCmdHandler(allowCmds []string) handler.BeanJobRunFunc {
	ch := &RunCmdHandler{
		allowCmds: allowCmds,
	}
	return ch.Handle
}

// Handle task
func (ch RunCmdHandler) Handle(ctx context.Context) error {
	obj, err := handler.GetCtxJobParam(ctx)
	if err != nil {
		return err
	}

	cmdName := obj.Param("cmd")
	if strutil.IsBlank(cmdName) {
		return errors.New("job params: cmd is required")
	}

	if cmdName == "help" {
		logger.LogJob(ctx, ch.BuildHelp())
		return nil
	}

	// allow limit cmd
	if len(ch.allowCmds) > 0 && arrutil.StringsHas(ch.allowCmds, cmdName) {
		logger.LogJob(ctx, ch.BuildHelp())
		return nil
	}

	// with cmd args.
	var args []string
	if argsLine := obj.Param("args"); len(argsLine) > 0 {
		args = cmdline.ParseLine(argsLine)
	}

	jobId := obj.JobID
	logId := obj.LogID

	// build command
	cmd := exec.CommandContext(ctx, cmdName, args...)
	stdout, _ := cmd.StdoutPipe()

	// see: https://stackoverflow.com/questions/48926982/write-stdout-stream-to-file
	logfile := logger.LogfilePath(logId)
	fh, err := logger.OpenLogFile(logfile)
	if err != nil {
		return err
	}

	go io.Copy(fh, stdout)

	logger.LogJobf(ctx, "will run the cmdline: %s", cmd.String())
	logger.Debugf("cmd job#%d will run task#%d cmdline: '%s', logfile: %s", jobId, logId, cmd.String(), logfile)
	if err := cmd.Run(); err != nil {
		_ = fh.Close() // close log file.

		errMsg := err.Error()
		if ee, ok := err.(*exec.ExitError); ok {
			errMsg = string(ee.Stderr) + "; " + ee.String()
			logger.Errorf("cmd job#%d - run task#%d cmdline error: %s", jobId, logId, errMsg)
		}

		return err
	}

	// close log file.
	return fh.Close()
}

// BuildHelp task
func (ch RunCmdHandler) BuildHelp() string {
	return `run input cmd with args.

Params:
cmd 	the command bin name.
args	the arguments for cmd run.

Examples:
cmd = php
args = /path/to/my.php arg0 arg1
`
}
