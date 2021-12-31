package example

import (
	"context"

	xxl "github.com/goft-cloud/go-xxl-job-client/v2"
	"github.com/goft-cloud/go-xxl-job-client/v2/logger"
	"github.com/gookit/goutil/dump"
)

func JobTest(ctx context.Context) error {
	cjp, err := xxl.GetParamObj(ctx)
	if err != nil {
		return err
	}

	dump.P(cjp)
	logger.LogJob(ctx, "test job!!!!!")

	param := cjp.InputParam // 获取输入参数
	logger.LogJob(ctx, "the input param:", param)

	shardingIdx, shardingTotal := xxl.GetSharding(ctx) // 获取分片参数
	logger.LogJob(ctx, "the sharding param - idx:", shardingIdx, ", total:", shardingTotal)

	return nil
}
