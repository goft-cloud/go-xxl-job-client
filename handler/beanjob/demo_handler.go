package beanjob

import (
	"context"

	"github.com/goft-cloud/go-xxl-job-client/v2/constants"
	"github.com/goft-cloud/go-xxl-job-client/v2/handler"
	"github.com/goft-cloud/go-xxl-job-client/v2/logger"
	"github.com/gookit/goutil/dump"
)

// DemoHandler struct
type DemoHandler struct {
}

// NewDemoHandler bean handler
func NewDemoHandler() handler.BeanJobRunFunc {
	ch := &DemoHandler{}
	return ch.Handle
}

// Handle task
func (ch DemoHandler) Handle(ctx context.Context) error {
	logger.Info("Start run demo handler++")

	cjp := ctx.Value(constants.CtxParamKey)
	dump.P(cjp)

	logger.LogJob(ctx, "bean job task run OK")

	logger.Info("End run demo handler--")
	return nil
}
