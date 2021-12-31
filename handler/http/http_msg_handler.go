package http

import (
	"context"
	"net/http"
	"time"

	getty "github.com/apache/dubbo-getty"
	"github.com/goft-cloud/go-xxl-job-client/v2/logger"
	"github.com/goft-cloud/go-xxl-job-client/v2/transport"
)

const (
	cronTime = 20e9
	// timeout for write pkg
	writePkgTimeout = 5 * time.Second
)

// MessageHandler struct
type MessageHandler struct {
	GettyClient *transport.GettyRPCClient
	MsgHandle   func(ctx context.Context, pkg interface{}) (res []byte, err error)
}

func NewHttpMessageHandler(transport *transport.GettyRPCClient, msgHandler func(ctx context.Context, pkg interface{}) (res []byte, err error)) getty.EventListener {
	return &MessageHandler{
		GettyClient: transport,
		MsgHandle:   msgHandler,
	}
}

// OnOpen session
func (h *MessageHandler) OnOpen(session getty.Session) error {
	logger.Infof("OnOpen session: %s", session.Stat())

	h.GettyClient.AddSession(session)
	return nil
}

func (h *MessageHandler) OnError(session getty.Session, err error) {
	logger.Infof("OnError session{%s} got error{%v}, will be closed.", session.Stat(), err)
}

func (h *MessageHandler) OnClose(session getty.Session) {
	logger.Infof("OnClose session{%s} is closing......", session.Stat())
	h.GettyClient.RemoveSession(session)
}

func (h *MessageHandler) OnMessage(session getty.Session, pkg interface{}) {
	s, ok := pkg.([]*transport.HttpRequestPkg)
	if !ok {
		logger.Errorf("illegal package: {%#v}", pkg)
		return
	}

	for _, v := range s {
		if v != nil {
			res, err := h.MsgHandle(context.Background(), v)
			reply(session, res, err)
		}
	}
}

func (h *MessageHandler) OnCron(session getty.Session) {
	active := session.GetActive()

	if cronTime < time.Since(active).Nanoseconds() {
		logger.Infof("OnCorn session{%s} timeout{%s}", session.Stat(), time.Since(active).String())
		session.Close()
		h.GettyClient.RemoveSession(session)
	}
}

func reply(session getty.Session, resBy []byte, err error) {
	pkg := transport.NewHttpResponsePkg(http.StatusOK, resBy)
	if err != nil || resBy == nil {
		pkg = transport.NewHttpResponsePkg(http.StatusInternalServerError, resBy)
	}

	_, _, err = session.WritePkg(pkg, writePkgTimeout)
	if err != nil {
		logger.Errorf("WritePkg error: %#v, pkg: %#v", err, pkg)
	}
}
