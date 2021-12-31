package beanjob

import (
	"context"

	"github.com/goft-cloud/go-xxl-job-client/v2/handler"
)

// RunCmdHandler struct
type RunCmdHandler struct {
}

// NewCmdHandler bean handler
func NewCmdHandler() handler.BeanJobRunFunc {
	ch := &RunCmdHandler{}
	return ch.Handle
}

// Handle task
func (ch RunCmdHandler) Handle(ctx context.Context) error {
	// bin := "groovy"
	// cmd := exec.CommandContext(ctx, bin, args...)

	return nil
}
