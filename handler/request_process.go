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

// RequestProcess struct
type RequestProcess struct {
	sync.RWMutex

	JobHandler  *JobManager
	ReqHandler  RequestHandler
	adminServer *admin.XxlAdminServer
}

// NewRequestProcess create object.
func NewRequestProcess(adminServer *admin.XxlAdminServer, handler RequestHandler) *RequestProcess {
	requestHandler := &RequestProcess{
		adminServer: adminServer,
		ReqHandler:  handler,
	}

	jobHandler := &JobManager{
		QueueMap:     make(map[int32]*JobQueue),
		CallbackFunc: requestHandler.jobRunCallback,
	}

	requestHandler.JobHandler = jobHandler
	return requestHandler
}

// RegisterJob to job handler manager
func (j *RequestProcess) RegisterJob(jobName string, beanJobFn BeanJobRunFunc) {
	j.JobHandler.RegisterJob(jobName, beanJobFn)
}

// push job to queue and run it
func (j *RequestProcess) pushJob(trigger *transport.TriggerParam) {
	returns := transport.ReturnT{
		Code:    http.StatusOK,
		Content: "success",
	}

	// push job to queue and run it.
	err := j.JobHandler.PutJobToQueue(trigger)
	if err != nil {
		returns.Code = http.StatusInternalServerError
		returns.Content = err.Error()
		callback := &transport.HandleCallbackParam{
			LogId:         trigger.LogId,
			LogDateTim:    trigger.LogDateTime,
			ExecuteResult: returns,
		}

		j.adminServer.CallbackAdmin([]*transport.HandleCallbackParam{callback})
	}
}

func (r *RequestProcess) jobRunCallback(trigger *JobRunParam, runErr error) {
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
	r.adminServer.CallbackAdmin([]*transport.HandleCallbackParam{callback})
}

// RequestProcess handle
func (j *RequestProcess) RequestProcess(ctx context.Context, r interface{}) (res []byte, err error) {
	response := transport.XxlRpcResponse{}
	returns := transport.ReturnT{
		Code:    http.StatusOK,
		Content: "success",
	}

	isOld := false
	reqId, accessToken, methodName, err := j.ReqHandler.ParseParam(ctx, r)
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
			jt := j.adminServer.GetToken()
			if accessToken != jt {
				returns.Code = http.StatusInternalServerError
				returns.Msg = "access token error"
			} else {
				if methodName != "beat" {
					mn := j.ReqHandler.MethodName(ctx, r)
					logger.Debugf("received server method: %s, reqId: %s", mn, reqId)

					switch mn {
					case "idleBeat":
						jobId, err := j.ReqHandler.IdleBeat(ctx, r)
						if err == nil {
							if j.JobHandler.HasRunning(jobId) {
								returns.Content = http.StatusInternalServerError
								returns.Content = "the server busy"
							}
						} else {
							returns.Content = http.StatusInternalServerError
							returns.Content = err.Error()
						}
					case "log":
						log, err := j.ReqHandler.Log(ctx, r)
						if err == nil {
							returns.Content = log
						}
					case "kill":
						jobId, err := j.ReqHandler.Kill(ctx, r)
						if err == nil {
							j.JobHandler.cancelJob(jobId)
						}
					default:
						// collect and build trigger params from r
						ta, err := j.ReqHandler.Run(ctx, r)
						if err == nil {
							go j.pushJob(ta)
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
func (j *RequestProcess) UnregisterExecutor() {
	j.JobHandler.clearJob()

	j.adminServer.UnregisterExecutor()
}

// RegisterExecutor to xxl-job admin server
func (j *RequestProcess) RegisterExecutor() {
	j.adminServer.RegisterExecutor()

	go j.adminServer.AutoRegisterJobGroup()
}
