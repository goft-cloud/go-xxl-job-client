package example

import (
	"context"
	"log"

	xxl "github.com/goft-cloud/go-xxl-job-client/v2"
	"github.com/goft-cloud/go-xxl-job-client/v2/logger"
)

func JobTest(ctx context.Context) error {
	val, _ := xxl.GetParam(ctx, "test")

	log.Print("job param:", val)
	logger.LogJob(ctx, "test job!!!!!")
	param, _ := xxl.GetParam(ctx, "name") // 获取输入参数
	logger.LogJob(ctx, "the input param:", param)
	shardingIdx, shardingTotal := xxl.GetSharding(ctx) // 获取分片参数
	logger.LogJob(ctx, "the sharding param: idx:", shardingIdx, ", total:", shardingTotal)

	return nil
}
