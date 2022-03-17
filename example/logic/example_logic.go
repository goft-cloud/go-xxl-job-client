package logic

import (
	"context"
	"strings"

	"github.com/goft-cloud/go-xxl-job-client/v2"
	"github.com/goft-cloud/go-xxl-job-client/v2/logger"
)

func Run(ctx context.Context) {
	cjp, err := xxl.GetParamObj(ctx)
	if err != nil {
		return
	}

	logger.LogJob(ctx, "example logic -", strings.Replace(cjp.InputParam, "\n", ", ", -1))
}
