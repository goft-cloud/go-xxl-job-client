package handler

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"

	"github.com/goft-cloud/go-xxl-job-client/v2/constants"
	"github.com/goft-cloud/go-xxl-job-client/v2/logger"
	"github.com/goft-cloud/go-xxl-job-client/v2/queue"
	"github.com/goft-cloud/go-xxl-job-client/v2/transport"
)

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
	// Callback on job exec completed.
	Callback func(trigger *JobRunParam, runErr error)
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

// JobManager struct
type JobManager struct {
	sync.RWMutex

	// key is jobName by registered.
	//
	// TIP: one jobName corresponds to one jobId
	jobMap map[string]BeanJobRunFunc
	// key is jobId.
	//
	// TIP: one jobName corresponds to one jobId
	QueueMap map[int32]*JobQueue
	// CallbackFunc call on job task completed, notify xxl-job admin
	CallbackFunc func(trigger *JobRunParam, runErr error)
}

// RegisterJob handler
func (jm *JobManager) RegisterJob(jobName string, beanJobFn BeanJobRunFunc) {
	jm.Lock()
	defer jm.Unlock()

	if jm.jobMap == nil {
		jm.jobMap = make(map[string]BeanJobRunFunc)
	} else if _, ok := jm.jobMap[jobName]; ok {
		panic("the job had already register, job name can't be repeated:" + jobName)
	}

	jm.jobMap[jobName] = beanJobFn
}

// HasRunning of the jobId
func (jm *JobManager) HasRunning(jobId int32) bool {
	qu, has := jm.QueueMap[jobId]
	if has {
		if qu.Run > 0 || qu.Queue.HasNext() {
			return true
		}
	}
	return false
}

// PutJobToQueue push job to queue and run it.
func (jm *JobManager) PutJobToQueue(ttp *transport.TriggerParam) (err error) {
	logger.Debugf("put and start job#%d, trigger info: %#v", ttp.JobId, ttp)

	jq, has := jm.QueueMap[ttp.JobId] // map value是地址，读不加锁
	if has {
		runParam, err := jq.ParseJob(ttp)
		if err != nil {
			return err
		}

		err = jq.Queue.Put(runParam)
		if err == nil {
			jq.StartJob()
		}
		return err
	}

	// 任务map初始化锁
	jm.Lock()
	defer jm.Unlock()

	jq = &JobQueue{
		GlueType: ttp.GlueType,
		JobId:    ttp.JobId,
		Callback: jm.CallbackFunc,
	}

	// switch bean job exec handler.
	if ttp.ExecutorHandler != "" {
		if jm.jobMap == nil && len(jm.jobMap) <= 0 {
			return errors.New("bean job handler not found")
		}

		fun, ok := jm.jobMap[ttp.ExecutorHandler]
		if !ok {
			return errors.New("bean job handler not found")
		}

		jq.ExecuteHandler = &BeanHandler{RunFunc: fun}
	} else {
		// use script handler
		jq.ExecuteHandler = &ScriptHandler{}
	}

	runParam, err := jq.ParseJob(ttp)
	if err != nil {
		return err
	}

	jq.Queue = queue.NewQueue()

	err = jq.Queue.Put(runParam)
	if err != nil {
		return err
	}

	jm.QueueMap[ttp.JobId] = jq
	jq.StartJob()
	return err
}

// cancel job run by admin kill notify
func (jm *JobManager) cancelJob(jobId int32) {
	jq, has := jm.QueueMap[jobId]
	if !has {
		logger.Errorf("cancel job#%d error, job not found", jobId)
		return
	}

	logger.Infof("the job#%d will be cancel by xxl-job admin notify", jobId)

	ok := jq.StopJob()
	if ok {
		if jq.CurrentJob != nil && jq.CurrentJob.CurrentCancelFunc != nil {
			jq.CurrentJob.CurrentCancelFunc()

			go func() {
				logId := jq.CurrentJob.LogId

				cjp := NewCtxJobParamByJrp(jobId, jq.CurrentJob)
				ctx := context.WithValue(context.Background(), constants.CtxParamKey, cjp)

				logger.LogJobf(ctx, "job#%d task#%d canceled by admin!", jobId, logId)
			}()
		} else {
			logger.Errorf("cancel job#%d error, current running task not found", jobId)
		}

		if jq.Queue != nil {
			jq.Queue.Clear()
		}
	}
}

// BeanJobLength size
func (jm *JobManager) BeanJobLength() int {
	if jm.jobMap == nil {
		return 0
	}
	return len(jm.jobMap)
}

func (jm *JobManager) clearJob() {
	jm.jobMap = map[string]BeanJobRunFunc{}
	jm.QueueMap = make(map[int32]*JobQueue)
}
