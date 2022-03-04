package example

import (
	"context"
	"fmt"
	"time"

	xxl "github.com/goft-cloud/go-xxl-job-client/v2"
	"github.com/goft-cloud/go-xxl-job-client/v2/logger"
	"github.com/gookit/goutil/dump"
)

// NotStoppedJobHandler XXL-JOB job handler 回调. 模拟一直不退出的执行
func NotStoppedJobHandler(ctx context.Context) error {
	dump.P(xxl.GetParamObj(ctx))

	logger.LogJob(ctx, "test demo job handler!!!!!")

	cmd, ok := xxl.GetParam(ctx, "cmd")
	if !ok {
		return fmt.Errorf("param not exists: cmd")
	}

	for {
		select {
		case <-ctx.Done(): // 取出值即说明是结束信号
			fmt.Println("收到信号，父context的协程退出,time=", time.Now().Unix())
			return nil
		default:
			time.Sleep(3 * time.Second)
			fmt.Println("running ...")
		}
	}

	return fmt.Errorf("unknown cmd, cmd=%s", cmd)
}
