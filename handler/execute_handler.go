package handler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"reflect"
	"runtime"
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

var scriptCmd = map[string]string{
	"GLUE_SHELL":      "bash",
	"GLUE_PYTHON":     "python",
	"GLUE_PHP":        "php",
	"GLUE_NODEJS":     "node",
	"GLUE_POWERSHELL": "powershell",
}

// ExecuteHandler interface
type ExecuteHandler interface {
	ParseJob(trigger *transport.TriggerParam) (runParam *JobRunParam, err error)
	Execute(jobId int32, glueType string, runParam *JobRunParam) error
}

// ScriptHandler struct
type ScriptHandler struct {
	sync.RWMutex
}

// ParseJob info
func (s *ScriptHandler) ParseJob(trigger *transport.TriggerParam) (jobParam *JobRunParam, err error) {
	suffix, ok := scriptMap[trigger.GlueType]
	if !ok {
		// logParam := make(map[string]interface{})
		// logParam["logId"] = trigger.LogId
		// logParam["jobId"] = trigger.JobId
		//
		// jobParamMap := make(map[string]map[string]interface{})
		// jobParamMap["logParam"] = logParam

		cjp := NewCtxJobParamByTpp(trigger)
		ctx := context.WithValue(context.Background(), "jobParam", cjp)

		msg := "暂不支持" + strings.ToLower(trigger.GlueType[constants.GluePrefixLen:]) + "脚本"
		logger.LogJobf(ctx, "job#%d parse error: %s", trigger.JobId, msg)
		return jobParam, errors.New(msg)
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
					return jobParam, err
				}
			}
		}

		if file != nil {
			defer file.Close()
			res, err := file.Write([]byte(trigger.GlueSource))
			if err != nil {
				return jobParam, err
			}
			if res <= 0 {
				return jobParam, errors.New("write script file failed")
			}
		}
	}

	inputParam := make(map[string]interface{})
	if trigger.ExecutorParams != "" {
		inputParam["param"] = trigger.ExecutorParams
	}

	jobParam = &JobRunParam{
		LogId:       trigger.LogId,
		LogDateTime: trigger.LogDateTime,
		JobName:     trigger.ExecutorHandler,
		JobTag:      path,
		InputParam:  inputParam,
	}
	if trigger.BroadcastTotal > 0 {
		jobParam.ShardIdx = trigger.BroadcastIndex
		jobParam.ShardTotal = trigger.BroadcastTotal
	}
	return jobParam, nil
}

// Execute script job
func (s *ScriptHandler) Execute(jobId int32, glueType string, runParam *JobRunParam) error {
	logId := runParam.LogId
	cjp := NewCtxJobParamByJrp(jobId, runParam)

	// logParam := make(map[string]interface{})
	// logParam["jobId"] = jobId
	// logParam["logId"] = logId
	// logParam["jobName"] = runParam.JobName
	// logParam["jobFunc"] = runParam.JobTag
	//
	// shardParam := make(map[string]interface{})
	// shardParam["shardingIdx"] = runParam.ShardIndex
	// shardParam["shardingTotal"] = runParam.ShardTotal
	//
	// jobParam := make(map[string]map[string]interface{})
	// jobParam["logParam"] = logParam
	// jobParam["inputParam"] = runParam.InputParam
	// jobParam["sharding"] = shardParam
	// dump.P(cjp)
	logger.Debugf("exec script job#%d task#%d, type: %s, params: %v", jobId, logId, glueType, cjp.String())
	ctx := context.WithValue(context.Background(), "jobParam", cjp)

	// ensure log dir created.
	logDir := logger.GetLogPath(time.Now())
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		s.Lock()
		os.MkdirAll(logDir, os.ModePerm)
		s.Unlock()
	}

	binName := scriptCmd[glueType]
	logfile := logDir + "/" + logger.LogfileName(runParam.LogId)

	cancelCtx, canFun := context.WithCancel(context.Background())
	defer canFun()
	runParam.CurrentCancelFunc = canFun

	var cmd *exec.Cmd

	// if binName == "bash" {
	// NOTICE: '-c' only for shell script
	// - use command pipe '>>logfile' sync log to file.
	// code := runParam.BuildCmdArgsString(logfile)
	// cmd = exec.CommandContext(cancelCtx, "bash", "-c", code)
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
			logger.Errorf("run task#%d script command error: %s", logId, errMsg)
		}

		logger.LogJobf(ctx, "run task#%d script failed, error: %s", logId, errMsg)
		return err
	}

	err = fh.Close() // close log file.
	logger.Debugf("run task#%d command script success", logId)
	return err
}

// BeanHandler struct
type BeanHandler struct {
	RunFunc JobHandlerFunc
}

// ParseJob info
func (b *BeanHandler) ParseJob(trigger *transport.TriggerParam) (jobParam *JobRunParam, err error) {
	if b.RunFunc == nil {
		logger.Errorf("the bean job#%d handler func not registered", trigger.JobId, trigger.LogId)
		return jobParam, errors.New("job run function not found")
	}

	inputParam := make(map[string]interface{})
	if trigger.ExecutorParams != "" {
		params := strings.Split(trigger.ExecutorParams, ",")
		if len(params) > 0 {
			for _, param := range params {
				if param != "" {
					jobP := strings.Split(param, "=")
					if len(jobP) > 1 {
						inputParam[jobP[0]] = jobP[1]
					}
				}
			}
		}
	}

	funName := getFunctionName(b.RunFunc)
	jobParam = &JobRunParam{
		LogId:       trigger.LogId,
		LogDateTime: trigger.LogDateTime,
		JobName:     trigger.ExecutorHandler,
		JobTag:      funName,
		InputParam:  inputParam,
	}
	return jobParam, err
}

// Execute bean handler func
func (b *BeanHandler) Execute(jobId int32, glueType string, runParam *JobRunParam) error {
	logId := runParam.LogId
	// see testdata/ctx-jobParam.json5
	// logParam := make(map[string]interface{})
	// logParam["jobId"] = jobId
	// logParam["logId"] = logId
	// logParam["jobName"] = runParam.JobName
	// logParam["jobFunc"] = runParam.JobTag
	//
	// shardParam := make(map[string]interface{})
	// shardParam["shardingIdx"] = runParam.ShardIdx
	// shardParam["shardingTotal"] = runParam.ShardTotal
	//
	// jobParam := make(map[string]map[string]interface{})
	// jobParam["logParam"] = logParam
	// jobParam["inputParam"] = runParam.InputParam
	// jobParam["sharding"] = shardParam

	cjp := NewCtxJobParamByJrp(jobId, runParam)
	baseCtx := context.Background()

	// add recover handle
	defer func() {
		if err := recover(); err != nil {
			var (
				errMsg string
				ok     bool
				e      error
			)

			if errMsg, ok = err.(string); !ok {
				if e, ok = err.(error); ok {
					errMsg = e.Error()
				}
			}

			if errMsg == "" {
				errMsg = "system error"
			}

			ctx := context.WithValue(baseCtx, "jobParam", cjp)
			logger.LogJobf(ctx, "bean job#%d task#%d run fatal! error: %s", jobId, logId, errMsg)
		}
	}()

	valueCtx, canFun := context.WithCancel(baseCtx)
	runParam.CurrentCancelFunc = canFun
	defer canFun()

	// with job params
	ctx := context.WithValue(valueCtx, "jobParam", cjp)
	err := b.RunFunc(ctx)
	if err != nil {
		logger.LogJobf(ctx, "job#%d task#%d run failed! error: %s", jobId, logId, err.Error())
		return err
	}

	return err
}

func getFunctionName(i interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}
