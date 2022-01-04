package rpc

import (
	"context"
	"net/http"
	"time"

	getty "github.com/apache/dubbo-getty"
	"github.com/goft-cloud/go-xxl-job-client/v2/constants"
	"github.com/goft-cloud/go-xxl-job-client/v2/logger"
	"github.com/goft-cloud/go-xxl-job-client/v2/transport"
)

const (
	// timeout for write pkg
	writePkgTimeout = 5 * time.Second
)

// MessageHandler tcp event message listener
type MessageHandler struct {
	GettyClient *transport.GettyRPCClient
	MsgHandle   func(ctx context.Context, pkg interface{}) (res []byte, err error)
}

func NewRpcMessageHandler(transport *transport.GettyRPCClient, msgHandler func(ctx context.Context, pkg interface{}) (res []byte, err error)) getty.EventListener {
	return &MessageHandler{
		GettyClient: transport,
		MsgHandle:   msgHandler,
	}
}

func (h *MessageHandler) OnOpen(session getty.Session) error {
	logger.Infof("Tcp.OnOpen - open session: %s", session.Stat())
	h.GettyClient.AddSession(session)
	return nil
}

func (h *MessageHandler) OnError(session getty.Session, err error) {
	logger.Infof("Tcp.OnError - session{%s} got error{%v}, will be closed.", session.Stat(), err)
}

func (h *MessageHandler) OnClose(session getty.Session) {
	logger.Infof("Tcp.OnClose - close session{%s} is closing ......", session.Stat())

	h.GettyClient.RemoveSession(session)
}

func (h *MessageHandler) OnMessage(session getty.Session, pkg interface{}) {
	s, ok := pkg.([]interface{})
	if !ok {
		logger.Errorf("Tcp.OnMessage - illegal package{%#v}", pkg)
		return
	}

	for _, v := range s {
		if v != nil {
			logger.Debugf("Tcp.OnMessage - message package item{%#v}", v)
			res, err := h.MsgHandle(context.Background(), v)
			reply(session, res, err)
		}
	}
}

func (h *MessageHandler) OnCron(sess getty.Session) {
	active := sess.GetActive()
	actDtime := active.Format(constants.DateTimeFormat2)
	logger.Debugf("Tcp.OnCron - session heartbeat check, last active: %s", actDtime)

	if constants.CronPeriod < time.Since(active).Nanoseconds() {
		logger.Infof(
			"Tcp.OnCorn - session{%s} timeout{%s}(last active:%s)",
			sess.Stat(),
			time.Since(active).String(),
			actDtime,
		)

		sess.Close()
		h.GettyClient.RemoveSession(sess)
	}

	// session.UpdateActive()
}

func reply(sess getty.Session, resp []byte, err error) {
	if sess.IsClosed() {
		logger.Errorf("Tcp.OnMessage - reply error: session closed, err: %#v, resp: %s", err, string(resp))
		return
	}

	pkg := transport.NewHttpResponsePkg(http.StatusOK, resp)
	if err != nil || resp == nil {
		pkg = transport.NewHttpResponsePkg(http.StatusInternalServerError, resp)
	}

	_, _, err = sess.WritePkg(pkg, writePkgTimeout)
	if err != nil {
		logger.Errorf("Tcp.WritePkg error: %#v, pkg: %#v", err, pkg)
	}
}
