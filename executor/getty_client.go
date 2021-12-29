package executor

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/dubbogo/getty"
	"github.com/dubbogo/gost/sync"
	"github.com/goft-cloud/go-xxl-job-client/v2/logger"
)

const (
	cronPeriod      = 20e9 / 1e6
	queueLen        = 128
	maxMsgLen       = 102400
	wqLen           = 512
	keepAliveTime   = 3 * time.Minute
	writeTimeout    = 5 * time.Second
	ReadBufferSize  = 262144
	writeBufferSize = 65536
)

var (
	onceTaskPoll sync.Once
	taskPool     *gxsync.TaskPool
)

// GettyClient client server struct
type GettyClient struct {
	// ServeCloserFn on client server close.
	ServeCloserFn func()
	// PkgHandler for request
	PkgHandler getty.ReadWriter

	EventListener getty.EventListener
}

// NewGettyClient create.
func NewGettyClient(pkgHandler getty.ReadWriter, eventListener getty.EventListener) *GettyClient {
	return &GettyClient{
		PkgHandler:    pkgHandler,
		EventListener: eventListener,
	}
}

func (c *GettyClient) Run(port, taskSize int) {
	onceTaskPoll.Do(func() {
		taskPool = gxsync.NewTaskPool(
			gxsync.WithTaskPoolTaskQueueLength(queueLen),
			gxsync.WithTaskPoolTaskQueueNumber(taskSize),
			gxsync.WithTaskPoolTaskPoolSize(taskSize/2+1),
		)
	})

	portStr := ":" + strconv.Itoa(port)
	server := getty.NewTCPServer(
		getty.WithLocalAddress(portStr),
	)

	server.RunEventLoop(func(session getty.Session) error {
		err := c.initialSession(session)
		if err != nil {
			return err
		}
		if taskPool != nil {
			session.SetTaskPool(taskPool)
		}
		return err
	})

	// util.WaitCloseSignals(server)
	waitCloseSignals(func() {
		logger.Info("client server closing ......")

		server.Close()
		if c.ServeCloserFn != nil {
			c.ServeCloserFn()
		}
	})
}

func (c *GettyClient) initialSession(session getty.Session) (err error) {
	tcpConn, ok := session.Conn().(*net.TCPConn)
	if !ok {
		panic(fmt.Sprintf("newSession: %s, session.conn{%#v} is not tcp connection", session.Stat(), session.Conn()))
	}

	if err = tcpConn.SetNoDelay(true); err != nil {
		return err
	}
	if err = tcpConn.SetKeepAlive(true); err != nil {
		return err
	}
	if err = tcpConn.SetKeepAlivePeriod(keepAliveTime); err != nil {
		return err
	}
	if err = tcpConn.SetReadBuffer(ReadBufferSize); err != nil {
		return err
	}
	if err = tcpConn.SetWriteBuffer(writeBufferSize); err != nil { // 考虑查看日志时候返回数据可能会多，会不会太小？
		return err
	}

	session.SetName("tcp")
	session.SetMaxMsgLen(maxMsgLen)
	session.SetWQLen(wqLen)
	session.SetReadTimeout(time.Second)
	session.SetWriteTimeout(writeTimeout)
	session.SetCronPeriod(int(cronPeriod))
	session.SetWaitTime(time.Second)
	session.SetPkgHandler(c.PkgHandler)
	session.SetEventListener(c.EventListener)
	return err
}

func waitCloseSignals(closer func()) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, os.Kill, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
	<-signals
	// closer.Close()
	closer()
}
