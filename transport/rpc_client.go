package transport

import (
	"sync"

	getty "github.com/apache/dubbo-getty"
	"github.com/goft-cloud/go-xxl-job-client/v2/logger"
)

// GettyRPCClient struct
type GettyRPCClient struct {
	sync.RWMutex
	sessions []getty.Session
}

func (c *GettyRPCClient) AddSession(session getty.Session) {
	if session == nil {
		return
	}

	c.Lock()
	defer c.Unlock()
	if c.sessions == nil {
		c.sessions = make([]getty.Session, 0, 16)
	}

	c.sessions = append(c.sessions, session)
}

// RemoveSession from sessions
func (c *GettyRPCClient) RemoveSession(session getty.Session) {
	if session == nil {
		return
	}

	c.Lock()
	defer c.Unlock()
	if c.sessions == nil || len(c.sessions) == 0 {
		return
	}

	for i, s := range c.sessions {
		if s == session {
			c.sessions = append(c.sessions[:i], c.sessions[i+1:]...)
			logger.Infof("remove session{%s}, its index{%d}", session.Stat(), i)
			break
		}
	}

	logger.Infof("after remove session{%s}, left number:%d", session.Stat(), len(c.sessions))
}
