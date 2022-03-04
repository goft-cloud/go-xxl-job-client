package handler

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/goft-cloud/go-xxl-job-client/v2/constants"
	"github.com/goft-cloud/go-xxl-job-client/v2/logger"
	"github.com/goft-cloud/go-xxl-job-client/v2/param"
	"github.com/goft-cloud/go-xxl-job-client/v2/queue"
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
	return &JobRunParam{}
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

// ExecuteHandler interface
type ExecuteHandler interface {
	ParseJob(trigger *transport.TriggerParam) (runParam *JobRunParam, err error)
	Execute(jobId int32, glueType string, runParam *JobRunParam) error
}

// JobQueue struct
type JobQueue struct {
	ExecuteHandler
	// JobId value
	JobId int32
	Run   int32 // 0 stop, 1 run
	// GlueType name
	GlueType   string
	CurrentJob *JobRunParam
	Queue      *queue.Queue
	Callback   func(trigger *JobRunParam, runErr error)
}

// StopJob check
func (jq *JobQueue) StopJob() bool {
	return atomic.CompareAndSwapInt32(&jq.Run, 1, 0)
}

// StartJob run
func (jq *JobQueue) StartJob() {
	if atomic.CompareAndSwapInt32(&jq.Run, 0, 1) {
		jq.asyRunJob()
	}
}

func (jq *JobQueue) asyRunJob() {
	go func() {
		for {
			has, node := jq.Queue.Poll()
			if has {
				jq.CurrentJob = node.(*JobRunParam)
				jq.Callback(jq.CurrentJob, jq.Execute(jq.JobId, jq.GlueType, jq.CurrentJob))
			} else {
				jq.StopJob()
				break
			}
		}
	}()
}

// JobHandler struct
type JobHandler struct {
	sync.RWMutex

	jobMap   map[string]BeanJobRunFunc
	QueueMap map[int32]*JobQueue

	CallbackFunc func(trigger *JobRunParam, runErr error)
}

// BeanJobLength size
func (j *JobHandler) BeanJobLength() int {
	if j.jobMap == nil {
		return 0
	}
	return len(j.jobMap)
}

// RegisterJob handler
func (j *JobHandler) RegisterJob(jobName string, beanJobFn BeanJobRunFunc) {
	j.Lock()
	defer j.Unlock()
	if j.jobMap == nil {
		j.jobMap = make(map[string]BeanJobRunFunc)
	} else {
		_, ok := j.jobMap[jobName]
		if ok {
			panic("the job had already register, job name can't be repeated:" + jobName)
		}
	}
	j.jobMap[jobName] = beanJobFn
}

func (j *JobHandler) HasRunning(jobId int32) bool {
	qu, has := j.QueueMap[jobId]
	if has {
		if qu.Run > 0 || qu.Queue.HasNext() {
			return true
		}
	}
	return false
}

// PutJobToQueue push job to queue and run it.
func (j *JobHandler) PutJobToQueue(trigger *transport.TriggerParam) (err error) {
	logger.Debugf("put and start job#%d, info: %#v", trigger.JobId, trigger)

	qu, has := j.QueueMap[trigger.JobId] // map value是地址，读不加锁
	if has {
		runParam, err := qu.ParseJob(trigger)
		if err != nil {
			return err
		}

		err = qu.Queue.Put(runParam)
		if err == nil {
			qu.StartJob()
			return err
		}

		return err
	}

	// 任务map初始化锁
	j.Lock()
	defer j.Unlock()

	jobQueue := &JobQueue{
		GlueType: trigger.GlueType,
		JobId:    trigger.JobId,
		Callback: j.CallbackFunc,
	}

	// switch job exec handler.
	if trigger.ExecutorHandler != "" {
		if j.jobMap == nil && len(j.jobMap) <= 0 {
			return errors.New("bean job handler not found")
		}

		fun, ok := j.jobMap[trigger.ExecutorHandler]
		if !ok {
			return errors.New("bean job handler not found")
		}

		jobQueue.ExecuteHandler = &BeanHandler{RunFunc: fun}
	} else {
		// use script handler
		jobQueue.ExecuteHandler = &ScriptHandler{}
	}

	runParam, err := jobQueue.ParseJob(trigger)
	if err != nil {
		return err
	}

	q := queue.NewQueue()
	err = q.Put(runParam)
	if err != nil {
		return err
	}

	jobQueue.Queue = q
	j.QueueMap[trigger.JobId] = jobQueue
	jobQueue.StartJob()
	return err
}

func (j *JobHandler) cancelJob(jobId int32) {
	jobQueue, has := j.QueueMap[jobId]
	if has {
		logger.Infof("the job#%d be xxl-job admin canceled", jobId)
		res := jobQueue.StopJob()

		if res {
			if jobQueue.CurrentJob != nil && jobQueue.CurrentJob.CurrentCancelFunc != nil {
				jobQueue.CurrentJob.CurrentCancelFunc()

				go func() {
					logId := jobQueue.CurrentJob.LogId

					// logParam := make(map[string]interface{})
					// logParam["logId"] = logId
					// logParam["jobId"] = jobId
					// logParam["jobName"] = jobQueue.CurrentJob.JobName
					// logParam["jobFunc"] = jobQueue.CurrentJob.JobTag

					cjp := NewCtxJobParamByJrp(jobId, jobQueue.CurrentJob)
					ctx := context.WithValue(context.Background(), constants.CtxParamKey, cjp)

					logger.LogJobf(ctx, "job#%d task#%d canceled by admin!", jobId, logId)
				}()
			} else {
				logger.Errorf("cancel job#%d error, current running task not found", jobId)
			}

			if jobQueue.Queue != nil {
				jobQueue.Queue.Clear()
			}
		}
	} else {
		logger.Errorf("cancel job#%d error, job not found", jobId)
	}
}

func (j *JobHandler) clearJob() {
	j.jobMap = map[string]BeanJobRunFunc{}
	j.QueueMap = make(map[int32]*JobQueue)
}
