package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"

	hessian "github.com/apache/dubbo-go-hessian2"
	"github.com/goft-cloud/go-xxl-job-client/v2/admin"
	"github.com/goft-cloud/go-xxl-job-client/v2/logger"
	"github.com/goft-cloud/go-xxl-job-client/v2/transport"
)

const (
	MthRun  = "run"
	MthLog  = "log"
	MthKill = "kill"
	MthBeat = "beat"

	MthIdleBeat = "idleBeat"
)

// RequestProcess struct
type RequestProcess struct {
	sync.RWMutex

	JobManager  *JobManager
	ReqHandler  RequestHandler
	adminServer *admin.XxlAdminServer
}

// NewRequestProcess create object.
func NewRequestProcess(adminServer *admin.XxlAdminServer, handler RequestHandler) *RequestProcess {
	requestHandler := &RequestProcess{
		adminServer: adminServer,
		ReqHandler:  handler,
	}

	jobManager := &JobManager{
		QueueMap:     make(map[int32]*JobQueue),
		CallbackFunc: requestHandler.jobRunCallback,
	}

	requestHandler.JobManager = jobManager
	return requestHandler
}

// RegisterJob to job handler manager
func (rp *RequestProcess) RegisterJob(jobName string, beanJobFn BeanJobRunFunc) {
	rp.JobManager.RegisterJob(jobName, beanJobFn)
}

// push job to queue and run it
func (rp *RequestProcess) pushJob(trigger *transport.TriggerParam) {
	returns := transport.ReturnT{
		Code:    http.StatusOK,
		Content: "success",
	}

	// push job to queue and run it.
	err := rp.JobManager.PutJobToQueue(trigger)
	if err != nil {
		returns.Code = http.StatusInternalServerError
		returns.Content = err.Error()
		callback := &transport.HandleCallbackParam{
			LogId:         trigger.LogId,
			LogDateTim:    trigger.LogDateTime,
			ExecuteResult: returns,
		}

		rp.adminServer.CallbackAdmin([]*transport.HandleCallbackParam{callback})
	}
}

func (rp *RequestProcess) jobRunCallback(trigger *JobRunParam, runErr error) {
	returns := transport.ReturnT{
		Code:    http.StatusOK,
		Content: "success",
	}

	if runErr != nil {
		returns.Code = http.StatusInternalServerError
		returns.Content = runErr.Error()
	}

	callback := &transport.HandleCallbackParam{
		LogId:         trigger.LogId,
		LogDateTim:    trigger.LogDateTime,
		ExecuteResult: returns,
	}
	rp.adminServer.CallbackAdmin([]*transport.HandleCallbackParam{callback})
}

// RequestProcess handle
func (rp *RequestProcess) RequestProcess(ctx context.Context, r interface{}) (res []byte, err error) {
	response := transport.XxlRpcResponse{}
	returns := transport.ReturnT{
		Code:    http.StatusOK,
		Content: "success",
	}

	isOld := false
	reqId, accessToken, methodName, err := rp.ReqHandler.ParseParam(ctx, r)
	if err != nil {
		returns.Code = http.StatusInternalServerError
		returns.Msg = err.Error()
	} else {
		isOld = reqId != ""
		isContinue := reqId == ""
		// 老版本,处理非心跳请求
		if reqId != "" && "BEAT_PING_PONG" != reqId {
			response.RequestId = reqId
			isContinue = true
		}

		if isContinue {
			jt := rp.adminServer.GetToken()
			if accessToken != jt {
				returns.Code = http.StatusInternalServerError
				returns.Msg = "access token error"
			} else {
				if methodName != "beat" {
					mn := rp.ReqHandler.MethodName(ctx, r)
					logger.Debugf("received server method: %s, reqId: %s", mn, reqId)

					switch mn {
					case MthIdleBeat:
						jobId, err := rp.ReqHandler.IdleBeat(ctx, r)
						if err == nil {
							if rp.JobManager.HasRunning(jobId) {
								returns.Code = http.StatusInternalServerError
								returns.Content = "the server busy"
							}
						} else {
							returns.Code = http.StatusInternalServerError
							returns.Content = err.Error()
						}
					case MthLog:
						log, err := rp.ReqHandler.Log(ctx, r)
						if err == nil {
							returns.Content = log
						}
					case MthKill:
						jobId, err := rp.ReqHandler.Kill(ctx, r)
						if err == nil {
							rp.JobManager.cancelJob(jobId)
						}
					default: // MthRun
						// collect and build trigger params from r, then run job
						ta, err := rp.ReqHandler.Run(ctx, r)
						if err == nil {
							go rp.pushJob(ta)
						}
					}
				}
			}
		}
	}

	var bytes []byte
	if isOld {
		response.Result = returns
		e := hessian.NewEncoder()
		err = e.Encode(response)
		if err != nil {
			return nil, err
		}
		bytes = e.Buffer()
	} else {
		bytes, err = json.Marshal(&returns)
		if err != nil {
			return nil, err
		}
	}

	return bytes, nil
}

// UnregisterExecutor form xxl-job admin server
func (rp *RequestProcess) UnregisterExecutor() {
	rp.JobManager.clearJob()

	rp.adminServer.UnregisterExecutor()
}

// RegisterExecutor to xxl-job admin server
func (rp *RequestProcess) RegisterExecutor() {
	rp.adminServer.RegisterExecutor()

	go rp.adminServer.AutoRegisterJobGroup()
}
