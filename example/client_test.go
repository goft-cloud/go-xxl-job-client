package example

import (
	_ "net/http/pprof"
	"testing"

	xxl "github.com/goft-cloud/go-xxl-job-client/v2"
	"github.com/goft-cloud/go-xxl-job-client/v2/option"
	"github.com/sirupsen/logrus"
)

func TestXxlClient(t *testing.T) {
	client := xxl.NewXxlClient(
		option.WithAccessToken("edqedewfrqdrfrfr"),
		option.WithEnableHttp(true), // xxl_job v2.2之后的版本
		option.WithClientPort(8083),
		option.WithAdminAddress("http://localhost:8080/xxl-job-admin"),
	)

	client.SetGettyLogger(&logrus.Entry{
		Logger: logrus.New(),
		Level:  logrus.InfoLevel,
	})

	client.RegisterJob("testJob", JobTest)
	client.Run()
}
