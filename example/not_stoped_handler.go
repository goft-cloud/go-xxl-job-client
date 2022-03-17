package example

import (
	"context"
	"strings"
	"time"

	xxl "github.com/goft-cloud/go-xxl-job-client/v2"
	"github.com/goft-cloud/go-xxl-job-client/v2/example/logic"
	"github.com/goft-cloud/go-xxl-job-client/v2/logger"
	"github.com/gookit/goutil/dump"
)

// NotStoppedJobHandler XXL-JOB job handler 回调. 模拟一直不退出的执行
//
// TIP: only use for tests
func NotStoppedJobHandler(ctx context.Context) error {
	cjp, err := xxl.GetParamObj(ctx)
	if err != nil {
		return err
	}

	dump.P(cjp)
	logger.LogJob(ctx, "test not stopped job handler!")

	for {
		select {
		case <-ctx.Done(): // 取出值即说明是结束信号
			logger.LogJob(ctx, "收到信号，父context的协程退出")
			return nil
		default:
			time.Sleep(1 * time.Second)
			logger.LogJob(ctx, "job running -", strings.Replace(cjp.InputParam, "\n", ", ", -1))
			logic.Run(ctx)
		}
	}
}
