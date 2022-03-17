package handler

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/goft-cloud/go-xxl-job-client/v2/constants"
	"github.com/goft-cloud/go-xxl-job-client/v2/param"
	"github.com/goft-cloud/go-xxl-job-client/v2/transport"
	"github.com/gookit/goutil/strutil"
)

// GetCtxJobParam object
func GetCtxJobParam(ctx context.Context) (*param.CtxJobParam, error) {
	val := ctx.Value(constants.CtxParamKey)
	if val == nil {
		return nil, errors.New("ctx job param not exists")
	}

	if cjp, ok := val.(*param.CtxJobParam); ok {
		return cjp, nil
	}

	return nil, errors.New("ctx job param is invalid")
}

// NewCtxJobParamByTpp create
func NewCtxJobParamByTpp(ttp *transport.TriggerParam) *param.CtxJobParam {
	return &param.CtxJobParam{
		JobID: ttp.JobId,
		LogID: ttp.LogId,
		// JobName:    "",
		JobFunc: ttp.ExecutorHandler,
		// user input param string.
		InputParam: ttp.ExecutorParams,
		ShardIndex: ttp.BroadcastIndex,
		ShardTotal: ttp.BroadcastTotal,
	}
}

// NewCtxJobParamByJrp create.
func NewCtxJobParamByJrp(jobId int32, jrp *JobRunParam) *param.CtxJobParam {
	return &param.CtxJobParam{
		JobID:   jobId,
		LogID:   jrp.LogId,
		JobName: jrp.JobName,
		JobFunc: jrp.JobTag,
		// user input param string.
		InputParam:  jrp.InputParam["fullParam"],
		InputParams: jrp.InputParam,
		// sharding params
		ShardIndex: jrp.ShardIdx,
		ShardTotal: jrp.ShardTotal,
	}
}

// JobRunParam struct
type JobRunParam struct {
	LogId       int64
	LogDateTime int64
	JobName     string
	// JobTag on script, this is script file path.
	JobTag string
	// InputParam user input params, key 'fullParam' always exists.
	InputParam map[string]string
	ShardIdx   int32
	ShardTotal int32
	// CurrentCancelFunc use for kill running job
	CurrentCancelFunc context.CancelFunc
}

// NewJobRunParam create
func NewJobRunParam(ttp *transport.TriggerParam) *JobRunParam {
	return &JobRunParam{
		LogId:       ttp.LogId,
		LogDateTime: ttp.LogDateTime,
		JobName:     ttp.ExecutorHandler,
	}
}

// WithOptionFn for config param
func (jrp *JobRunParam) WithOptionFn(fn func(jrp *JobRunParam)) *JobRunParam {
	fn(jrp)
	return jrp
}

// BuildCmdArgs list
func (jrp *JobRunParam) BuildCmdArgs(logfile ...string) []string {
	var cmdArgs = []string{jrp.JobTag}

	if len(jrp.InputParam) > 0 {
		ps, ok := jrp.InputParam["fullParam"]
		if ok {
			// 参数可用换行隔开
			params := strings.Split(ps, "\n")
			for _, v := range params {
				cmdArgs = append(cmdArgs, strings.TrimSpace(v))
			}
		}
	}

	if jrp.ShardTotal > 0 {
		cmdArgs = append(cmdArgs, strutil.MustString(jrp.ShardIdx), strutil.MustString(jrp.ShardTotal))
	}

	// TIP: use command pipe write log to file.
	// can also: https://stackoverflow.com/questions/48926982/write-stdout-stream-to-file
	if len(logfile) > 0 {
		cmdArgs = append(cmdArgs, ">>"+logfile[0])
	}

	return cmdArgs
}

// BuildCmdArgsString string
func (jrp *JobRunParam) BuildCmdArgsString(logfile string) string {
	var buffer bytes.Buffer

	// set job script file
	buffer.WriteString(jrp.JobTag)

	if len(jrp.InputParam) > 0 {
		ps, ok := jrp.InputParam["fullParam"]
		if ok {
			// 参数可用空格或者换行隔开
			params := strings.Split(ps, "\n")
			for _, v := range params {
				buffer.WriteString(" ")
				buffer.WriteString(strings.TrimSpace(v))
			}
		}
	}

	if jrp.ShardTotal > 0 {
		buffer.WriteString(fmt.Sprintf(" %d %d", jrp.ShardIdx, jrp.ShardTotal))
	}

	// TIP: use command pipe write log to file.
	// can also: https://stackoverflow.com/questions/48926982/write-stdout-stream-to-file
	buffer.WriteString(" >>")
	buffer.WriteString(logfile)

	return buffer.String()
}
