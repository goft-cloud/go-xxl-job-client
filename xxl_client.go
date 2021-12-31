package xxl

import (
	getty "github.com/apache/dubbo-getty"
	"github.com/apache/dubbo-go-hessian2"
	"github.com/goft-cloud/go-xxl-job-client/v2/admin"
	"github.com/goft-cloud/go-xxl-job-client/v2/constants"
	executor2 "github.com/goft-cloud/go-xxl-job-client/v2/executor"
	"github.com/goft-cloud/go-xxl-job-client/v2/handler"
	"github.com/goft-cloud/go-xxl-job-client/v2/handler/http"
	"github.com/goft-cloud/go-xxl-job-client/v2/handler/rpc"
	"github.com/goft-cloud/go-xxl-job-client/v2/logger"
	"github.com/goft-cloud/go-xxl-job-client/v2/option"
	"github.com/goft-cloud/go-xxl-job-client/v2/transport"
)

// XxlClient struct
type XxlClient struct {
	options  option.ClientOptions
	executor *executor2.Executor
	// request handler
	requestHandler *handler.RequestProcess
}

// NewXxlClient create
func NewXxlClient(opts ...option.Option) *XxlClient {
	clientOps := option.NewClientOptions(opts...)

	executor := executor2.NewExecutor(
		"",
		clientOps.AppName,
		clientOps.Port,
	)

	adminServer := admin.NewAdminServer(
		clientOps.AdminAddr,
		clientOps.Timeout,
		clientOps.BeatTime,
		executor,
	)

	// set log path.
	logger.SetLogBasePath(clientOps.LogBasePath)

	adminServer.AccessToken = map[string]string{
		"XXL-JOB-ACCESS-TOKEN": clientOps.AccessToken,
	}

	var requestHandler *handler.RequestProcess
	var gettyClient *executor2.GettyClient
	if clientOps.EnableHttp {
		requestHandler = handler.NewRequestProcess(adminServer, &http.HttpRequestHandler{})
		executor.Protocol = constants.HttpProtocol

		gettyClient = executor2.NewGettyClient(
			http.NewHttpPackageHandler(),
			http.NewHttpMessageHandler(&transport.GettyRPCClient{}, requestHandler.RequestProcess),
		)
	} else {
		// register java POJO
		hessian.RegisterPOJO(&transport.XxlRpcRequest{})
		hessian.RegisterPOJO(&transport.TriggerParam{})
		hessian.RegisterPOJO(&transport.Beat{})
		hessian.RegisterPOJO(&transport.XxlRpcResponse{})
		hessian.RegisterPOJO(&transport.ReturnT{})
		hessian.RegisterPOJO(&transport.HandleCallbackParam{})
		hessian.RegisterPOJO(&logger.LogResult{})
		hessian.RegisterPOJO(&transport.RegistryParam{})

		requestHandler = handler.NewRequestProcess(adminServer, &rpc.RpcRequestHandler{})
		gettyClient = &executor2.GettyClient{
			PkgHandler:    rpc.NewPackageHandler(),
			EventListener: rpc.NewRpcMessageHandler(&transport.GettyRPCClient{}, requestHandler.RequestProcess),
		}
	}

	// remove executor on client server close
	gettyClient.ServeCloserFn = requestHandler.UnregisterExecutor

	executor.SetClient(gettyClient)

	return &XxlClient{
		requestHandler: requestHandler,
		// other
		executor: executor,
		options:  clientOps,
	}
}

// MustRun start and run, will panic on error
func (c *XxlClient) MustRun() {
	if err := c.Run(); err != nil {
		panic(err)
	}
}

// Run start and run client.
func (c *XxlClient) Run() error {
	logger.Infof("go executor client run on mode: %s", option.RunMode())
	logger.Infof("xxl-job admin address list is: %v", c.options.AdminAddr)
	// logger.Infof("registered executor name is: %s", c.executor.AppName)

	// register to xxl-job admin
	c.requestHandler.RegisterExecutor()

	err := logger.InitLogPath()
	if err != nil {
		return err
	}

	logger.Infof("go executor client started on port: %d", c.options.Port)

	c.executor.Run(c.requestHandler.JobHandler.BeanJobLength())
	return nil
}

// RegisterJob add job handler.
func (c *XxlClient) RegisterJob(jobName string, function handler.BeanJobRunFunc) {
	c.requestHandler.RegisterJob(jobName, function)
}

// SetGettyLogger set logger to getty.
func (c *XxlClient) SetGettyLogger(logger getty.Logger) {
	getty.SetLogger(logger)
}

// Unregister from xxl-job admin.
func (c *XxlClient) Unregister() {
	c.requestHandler.UnregisterExecutor()
}

// Options gets
func (c *XxlClient) Options() option.ClientOptions {
	return c.options
}
