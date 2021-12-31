package beanjob

import (
	"context"

	"github.com/goft-cloud/go-xxl-job-client/v2/handler"
)

// GroovyHandler struct
type GroovyHandler struct {
}

// NewGroovyHandler bean handler
func NewGroovyHandler() handler.JobHandlerFunc {
	gh := &GroovyHandler{}
	return gh.Handle
}

// Handle task
func (gh GroovyHandler) Handle(ctx context.Context) error {
	// bin := "groovy"
	// cmd := exec.CommandContext(ctx, bin, args...)

	return nil
}
