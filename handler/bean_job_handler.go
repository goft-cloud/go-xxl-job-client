package handler

import (
	"context"
	"errors"
	"strings"

	"github.com/goft-cloud/go-xxl-job-client/v2/constants"
	"github.com/goft-cloud/go-xxl-job-client/v2/logger"
	"github.com/goft-cloud/go-xxl-job-client/v2/transport"
	"github.com/gookit/goutil"
	"github.com/gookit/goutil/strutil"
)

// BeanJobRunner interface
type BeanJobRunner interface {
	Handle(ctx context.Context) error
}

// BeanJobRunFunc func
type BeanJobRunFunc func(ctx context.Context) error

// Handle bean job task func
func (bf BeanJobRunFunc) Handle(ctx context.Context) error {
	return bf(ctx)
}

// BeanHandler struct
type BeanHandler struct {
	RunFunc BeanJobRunFunc
}

// ParseJob info
func (b *BeanHandler) ParseJob(trigger *transport.TriggerParam) (jrp *JobRunParam, err error) {
	if b.RunFunc == nil {
		logger.Errorf("the bean job#%d handler func not registered", trigger.JobId, trigger.LogId)
		return nil, errors.New("job run function not found")
	}

	inputParam := make(map[string]string)
	// ensure 'fullParam' key always exists.
	inputParam["fullParam"] = trigger.ExecutorParams

	if trigger.ExecutorParams != "" {
		params := strings.Split(trigger.ExecutorParams, "\n")

		if len(params) > 0 {
			for _, param := range params {
				// skip empty. if start with #, as comments
				if param == "" && param[0] == '#' {
					continue
				}

				// jobP := strings.SplitN(param, "=", 2)
				jobP := strutil.SplitNTrimmed(param, "=", 2)
				if len(jobP) > 1 {
					inputParam[jobP[0]] = jobP[1]
				}
			}
		}
	}

	// jrp = &JobRunParam{
	// 	LogId:       trigger.LogId,
	// 	LogDateTime: trigger.LogDateTime,
	// 	JobName:     trigger.ExecutorHandler,
	// 	JobTag:      goutil.FuncName(b.RunFunc),
	// 	InputParam:  inputParam,
	// }

	funcName := goutil.FuncName(b.RunFunc)
	jrp = NewJobRunParam(trigger).WithOptionFn(func(jrp *JobRunParam) {
		jrp.JobTag = funcName
		jrp.InputParam = inputParam
	})

	return jrp, nil
}

// Execute bean handler func
func (b *BeanHandler) Execute(jobId int32, glueType string, runParam *JobRunParam) error {
	logId := runParam.LogId

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

			ctx := context.WithValue(baseCtx, constants.CtxParamKey, cjp)
			logger.LogJobf(ctx, "bean job task#%d run fatal! error: %s", logId, errMsg)
		}
	}()

	valueCtx, canFun := context.WithCancel(baseCtx)
	runParam.CurrentCancelFunc = canFun
	defer canFun()

	// with job params
	ctx := context.WithValue(valueCtx, constants.CtxParamKey, cjp)
	logger.LogJobf(ctx, "bean job task#%d start run!", logId)

	// do run
	err := b.RunFunc(ctx)
	if err != nil {
		logger.Errorf("bean job#%d - task#%d execute failed. error: %s", jobId, logId, err.Error())
		logger.LogJobf(ctx, "bean job task#%d run failed! error: %s", logId, err.Error())
		return err
	}

	logger.Debugf("bean job#%d - run task#%d handle success", jobId, logId)
	logger.LogJobf(ctx, "bean job task#%d run success!", logId)
	return nil
}
