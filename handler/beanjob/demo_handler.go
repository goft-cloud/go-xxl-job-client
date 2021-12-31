package beanjob

import (
	"context"

	"github.com/goft-cloud/go-xxl-job-client/v2/handler"
	"github.com/goft-cloud/go-xxl-job-client/v2/logger"
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
	// bin := "groovy"
	// cmd := exec.CommandContext(ctx, bin, args...)
	logger.LogJob(ctx, "bean job task run OK")
	return nil
}
