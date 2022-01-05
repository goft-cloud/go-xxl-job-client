package example

import (
	_ "net/http/pprof"
	"testing"

	getty "github.com/apache/dubbo-getty"
	xxl "github.com/goft-cloud/go-xxl-job-client/v2"
	"github.com/goft-cloud/go-xxl-job-client/v2/option"
)

func TestXxlClient(t *testing.T) {
	// TIP: 可以在开发时打开调试模式，可以看到更多信息
	option.SetRunMode(option.ModeDebug)

	var clientOpts = []option.OptionFunc{
		// option.WithAccessToken("edqedewfrqdrfrfr"),
		option.WithEnableHttp(true), // xxl_job v2.2之后的版本
		option.WithClientPort(8083), // 客户端启动端口
		option.WithAdminAddress("http://localhost:8080/xxl-job-admin"),
	}

	client := xxl.NewXxlClient(clientOpts...)

	// 更多选项配置
	// var admAddr = config.String("xxl-job-addr", "")
	// if !strutil.IsBlank(admAddr) {
	// 	clientOpts = append(clientOpts, option.WithAdminAddress(admAddr))
	// }
	//
	// var logPath = config.String("xxl-job-log-path", "")
	// if !strutil.IsBlank(logPath) {
	// 	clientOpts = append(clientOpts, option.WithLogBasePath(logPath))
	// }

	// set getty logger level
	client.SetGettyLogLevel(getty.LoggerLevelInfo)

	// 注册JobHandler(Bean模式任务的handler)
	client.RegisterJob("my_job_handler", JobTest)

	// 启动客户端
	// client.Run()
	client.MustRun()
}
