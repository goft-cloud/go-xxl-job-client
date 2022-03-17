package example

import (
	_ "net/http/pprof"
	"testing"

	getty "github.com/apache/dubbo-getty"
	xxl "github.com/goft-cloud/go-xxl-job-client/v2"
	"github.com/goft-cloud/go-xxl-job-client/v2/handler/beanjob"
	"github.com/goft-cloud/go-xxl-job-client/v2/option"
	"github.com/gookit/goutil/strutil"
)

func TestXxlClient(t *testing.T) {
	// TIP: 可以在开发时打开调试模式，可以看到更多信息
	option.SetRunMode(option.ModeDebug)

	var clientOpts = []option.OptionFunc{
		// option.WithAccessToken("edqedewfrqdrfrfr"),
		option.WithEnableHttp(true), // xxl_job v2.2之后的版本
		option.WithClientPort(8083), // 客户端启动端口
		// option.WithAdminAddress("http://localhost:8080/xxl-job-admin"),
	}

	// 更多选项配置
	admAddr := "http://localhost:8686/xxl-job-admin"
	// var admAddr = config.String("xxl-job-addr", "")
	clientOpts = append(clientOpts, option.WithAdminAddress(admAddr))

	logPath := "./xxl-logs"
	// var logPath = config.String("xxl-job-log-path", "")
	if !strutil.IsBlank(logPath) {
		clientOpts = append(clientOpts, option.WithLogBasePath(logPath))
	}

	client := xxl.NewXxlClient(clientOpts...)

	// set getty logger level
	client.SetGettyLogLevel(getty.LoggerLevelInfo)

	// 注册JobHandler(Bean模式任务的handler)
	client.RegisterJob("test_job", JobTest)
	client.RegisterJob("cmd_handler", beanjob.NewCmdHandler([]string{}))
	client.RegisterJob("not_stopped_job", NotStoppedJobHandler)

	// 启动客户端
	// client.Run()
	client.MustRun()
}
