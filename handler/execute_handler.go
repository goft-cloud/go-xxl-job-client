package handler

import (
	"bytes"
	"context"
	"fmt"
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
	"github.com/pkg/errors"
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

type ExecuteHandler interface {
	ParseJob(trigger *transport.TriggerParam) (runParam *JobRunParam, err error)
	Execute(jobId int32, glueType string, runParam *JobRunParam) error
}

type ScriptHandler struct {
	sync.RWMutex
}

func (s *ScriptHandler) ParseJob(trigger *transport.TriggerParam) (jobParam *JobRunParam, err error) {
	suffix, ok := scriptMap[trigger.GlueType]
	if !ok {
		logParam := make(map[string]interface{})
		logParam["logId"] = trigger.LogId
		logParam["jobId"] = trigger.JobId

		jobParamMap := make(map[string]map[string]interface{})
		jobParamMap["logParam"] = logParam
		ctx := context.WithValue(context.Background(), "jobParam", jobParamMap)

		msg := "暂不支持" + strings.ToLower(trigger.GlueType[constants.GluePrefixLen:]) + "脚本"
		logger.LogJob(ctx, "job parse error:", msg)
		return jobParam, errors.New(msg)
	}

	// path := fmt.Sprintf("%s_%d_%d%s", constants.GlueSourcePath, trigger.JobId, trigger.GlueUpdatetime, suffix)
	path := fmt.Sprintf("%s/gs_%d_%d%s", logger.GlueSourcePath(), trigger.JobId, trigger.GlueUpdatetime, suffix)
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
	logParam := make(map[string]interface{})
	logParam["logId"] = runParam.LogId
	logParam["jobId"] = jobId
	logParam["jobName"] = runParam.JobName
	logParam["jobFunc"] = runParam.JobTag

	shardParam := make(map[string]interface{})
	shardParam["shardingIdx"] = runParam.ShardIdx
	shardParam["shardingTotal"] = runParam.ShardTotal

	jobParam := make(map[string]map[string]interface{})
	jobParam["logParam"] = logParam
	jobParam["inputParam"] = runParam.InputParam
	jobParam["sharding"] = shardParam

	logger.Debugf("exec script job, type: %s, ID: %d, params: %v", glueType, jobId, jobParam)
	ctx := context.WithValue(context.Background(), "jobParam", jobParam)

	// ensure log dir created.
	logDir := logger.GetLogPath(time.Now())
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		s.Lock()
		os.MkdirAll(logDir, os.ModePerm)
		s.Unlock()
	}

	logfile := logDir + "/" + logger.LogfileName(runParam.LogId)

	var buffer bytes.Buffer
	buffer.WriteString(runParam.JobTag)
	if len(runParam.InputParam) > 0 {
		ps, ok := runParam.InputParam["param"]
		if ok {
			// 参数要用逗号隔开，用空格或者换行可能有问题
			params := strings.Split(ps.(string), ",")
			for _, v := range params {
				buffer.WriteString(" ")
				buffer.WriteString(strings.TrimSpace(v))
			}
		}
	}

	if runParam.ShardTotal > 0 {
		buffer.WriteString(fmt.Sprintf(" %d %d", runParam.ShardIdx, runParam.ShardTotal))
	}

	// use command pipe write log to file.
	buffer.WriteString(" >> ")
	buffer.WriteString(logfile)

	cancelCtx, canFun := context.WithCancel(context.Background())
	defer canFun()
	c := buffer.String()
	logger.Debugf("script job contents: %s", c)

	runParam.CurrentCancelFunc = canFun
	// NOTICE: '-c' only for shell script
	cmd := exec.CommandContext(cancelCtx, scriptCmd[glueType], "-c", c)

	// cmd.Output() must be waite command complete.
	output, err := cmd.Output()
	if err != nil {
		logger.LogJob(ctx, "run script job result:", string(output), ", error: ", err.Error())
		return err
	}

	return err
}

type BeanHandler struct {
	RunFunc JobHandlerFunc
}

func (b *BeanHandler) ParseJob(trigger *transport.TriggerParam) (jobParam *JobRunParam, err error) {
	if b.RunFunc == nil {
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

func (b *BeanHandler) Execute(jobId int32, glueType string, runParam *JobRunParam) error {
	logParam := make(map[string]interface{})
	logParam["logId"] = runParam.LogId
	logParam["jobId"] = jobId
	logParam["jobName"] = runParam.JobName
	logParam["jobFunc"] = runParam.JobTag

	shardParam := make(map[string]interface{})
	shardParam["shardingIdx"] = runParam.ShardIdx
	shardParam["shardingTotal"] = runParam.ShardTotal

	jobParam := make(map[string]map[string]interface{})
	jobParam["logParam"] = logParam
	jobParam["inputParam"] = runParam.InputParam
	jobParam["sharding"] = shardParam

	valueCtx, canFun := context.WithCancel(context.Background())
	defer canFun()

	runParam.CurrentCancelFunc = canFun
	ctx := context.WithValue(valueCtx, "jobParam", jobParam)
	err := b.RunFunc(ctx)
	if err != nil {
		logger.LogJob(ctx, "job#%d run failed! msg:", jobId, err.Error())
		return err
	}

	return err
}

func getFunctionName(i interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}
