package handler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/goft-cloud/go-xxl-job-client/v2/constants"
	"github.com/goft-cloud/go-xxl-job-client/v2/logger"
	"github.com/goft-cloud/go-xxl-job-client/v2/transport"
	"github.com/gookit/goutil/strutil"
)

var scriptMap = map[string]string{
	"GLUE_SHELL":      ".sh",
	"GLUE_PYTHON":     ".py",
	"GLUE_PHP":        ".php",
	"GLUE_NODEJS":     ".js",
	"GLUE_POWERSHELL": ".ps1",
}

var scriptBin = map[string]string{
	"GLUE_SHELL":      constants.ShellBash,
	"GLUE_PYTHON":     "python",
	"GLUE_PHP":        "php",
	"GLUE_NODEJS":     "node",
	"GLUE_POWERSHELL": "powershell",
}

// SetShellBin custom set shell bin
func SetShellBin(binName string) {
	scriptBin["GLUE_SHELL"] = binName
}

// ScriptHandler struct
type ScriptHandler struct {
	sync.RWMutex
}

// ParseJob info
func (s *ScriptHandler) ParseJob(trigger *transport.TriggerParam) (jrp *JobRunParam, err error) {
	suffix, ok := scriptMap[trigger.GlueType]
	if !ok {
		cjp := NewCtxJobParamByTpp(trigger)
		ctx := context.WithValue(context.Background(), constants.CtxParamKey, cjp)

		msg := "暂不支持" + strings.ToLower(trigger.GlueType[constants.GluePrefixLen:]) + "脚本"
		logger.LogJobf(ctx, "job#%d parse error: %s", trigger.JobId, msg)
		return jrp, errors.New(msg)
	}

	// path := fmt.Sprintf("%s_%d_%d%s", constants.GlueSourcePath, trigger.JobId, trigger.GlueUpdatetime, suffix)
	path := fmt.Sprintf("%s/job%d_%d%s", logger.GlueSourcePath(), trigger.JobId, trigger.GlueUpdatetime, suffix)
	_, err = os.Stat(path)
	if err != nil && os.IsNotExist(err) {
		s.Lock()
		defer s.Unlock()
		file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0750)
		if err != nil && os.IsNotExist(err) {
			// err = os.MkdirAll(constants.GlueSourcePath, os.ModePerm)
			err = os.MkdirAll(logger.GlueSourcePath(), os.ModePerm)
			if err == nil {
				file, err = os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0750)
				if err != nil {
					return jrp, err
				}
			}
		}

		if file != nil {
			defer file.Close()
			res, err := file.Write([]byte(trigger.GlueSource))
			if err != nil {
				return jrp, err
			}
			if res <= 0 {
				return jrp, errors.New("write script file failed")
			}
		}
	}

	inputParam := make(map[string]string)
	// ensure 'fullParam' key always exists.
	inputParam["fullParam"] = trigger.ExecutorParams

	jrp = &JobRunParam{
		LogId:       trigger.LogId,
		LogDateTime: trigger.LogDateTime,
		JobName:     trigger.ExecutorHandler,
		JobTag:      path,
		InputParam:  inputParam,
	}

	if trigger.BroadcastTotal > 0 {
		jrp.ShardIdx = trigger.BroadcastIndex
		jrp.ShardTotal = trigger.BroadcastTotal
	}
	return jrp, nil
}

// Execute script job
func (s *ScriptHandler) Execute(jobId int32, glueType string, runParam *JobRunParam) error {
	logId := runParam.LogId
	cjp := NewCtxJobParamByJrp(jobId, runParam)

	// dump.P(cjp)
	logger.Debugf("exec script job#%d task#%d, type: %s, params: %v", jobId, logId, glueType, cjp.String())
	ctx := context.WithValue(context.Background(), constants.CtxParamKey, cjp)

	// ensure log dir created.
	// up: 不创建子目录
	logDir := logger.GetLogPath(time.Now())
	// if _, err := os.Stat(logDir); os.IsNotExist(err) {
	// 	s.Lock()
	// 	os.MkdirAll(logDir, os.ModePerm)
	// 	s.Unlock()
	// }

	binName := scriptBin[glueType]
	// logfile := logDir + "/" + logger.LogfileName(runParam.LogId)
	logfile := logDir + "_" + logger.LogfileName(runParam.LogId)

	cancelCtx, canFun := context.WithCancel(context.Background())
	defer canFun()
	runParam.CurrentCancelFunc = canFun

	var cmd *exec.Cmd

	// if binName == ShellBash {
	// NOTICE: '-c' only for shell script
	// - use command pipe '>>logfile' sync log to file.
	// code := runParam.BuildCmdArgsString(logfile)
	// cmd = exec.CommandContext(cancelCtx, binName, "-c", code)
	// } else {
	// TIP: use args the pipe mark >> no effect.
	// cmdArgs := runParam.BuildCmdArgs(logfile)
	args := runParam.BuildCmdArgs()
	cmd = exec.CommandContext(cancelCtx, binName, args...)
	stdout, _ := cmd.StdoutPipe()

	// see: https://stackoverflow.com/questions/48926982/write-stdout-stream-to-file
	fh, err := logger.OpenLogFile(logfile)
	if err != nil {
		return err
	}

	// defer fh.Close()
	// go io.Copy(io.MultiWriter(f, os.Stdout), stdout)
	go io.Copy(fh, stdout)
	// }

	if runParam.ShardTotal > 0 {
		cmd.Env = []string{
			constants.EnvXxlShardIdx + "=" + strutil.MustString(runParam.ShardIdx),
			constants.EnvXxlShardTotal + "=" + strutil.MustString(runParam.ShardTotal),
		}
	}

	logger.Debugf("job#%d will run task#%d script command: '%s', logfile: %s", jobId, logId, cmd.String(), logfile)
	if err := cmd.Run(); err != nil {
		_ = fh.Close() // close log file.

		errMsg := err.Error()
		if ee, ok := err.(*exec.ExitError); ok {
			errMsg = string(ee.Stderr)
			logger.Errorf("job#%d - run task#%d script command error: %s", jobId, logId, errMsg)
		}

		logger.LogJobf(ctx, "job#%d - run task#%d script failed, error: %s", jobId, logId, errMsg)
		return err
	}

	err = fh.Close() // close log file.
	logger.Debugf("job#%d - run task#%d command script success", jobId, logId)
	return err
}
