package option

import (
	"time"

	"github.com/goft-cloud/go-xxl-job-client/v2/constants"
)

const (
	defaultAdminAddr = "http://localhost:8080/xxl-job-admin/"
	defaultAppName   = "go-executor"
	defaultPort      = 8081
	defaultTimeout   = 5 * time.Second
	defaultBeatTime  = 20 * time.Second
)

type Option func(*ClientOptions)

type ClientOptions struct {
	// AdminAddr xxl admin 地址
	AdminAddr []string

	// AccessToken token
	AccessToken string

	// LogBasePath the job logs base dir path.
	LogBasePath string

	// AppName 执行器名
	AppName string

	// Port client 执行器端口
	Port int

	// EnableHttp 开启http协议
	EnableHttp bool

	// Timeout 请求admin超时时间
	Timeout time.Duration

	// BeatTime 执行器续约时间（超过30秒不续约admin会移除执行器，请设置到30秒以内）
	BeatTime time.Duration
}

// NewClientOptions instance
func NewClientOptions(opts ...Option) ClientOptions {
	options := ClientOptions{
		AdminAddr:   []string{defaultAdminAddr},
		AccessToken: "",
		AppName:     defaultAppName,
		Port:        defaultPort,
		Timeout:     defaultTimeout,
		BeatTime:    defaultBeatTime,
		LogBasePath: constants.BasePath,
	}

	for _, o := range opts {
		o(&options)
	}
	return options
}

// WithLogBasePath job logs base dir path
func WithLogBasePath(dirPath string) Option {
	return func(o *ClientOptions) {
		o.LogBasePath = dirPath
	}
}

// WithAdminAddress xxl admin address
func WithAdminAddress(addrs ...string) Option {
	return func(o *ClientOptions) {
		o.AdminAddr = addrs
	}
}

// WithAccessToken xxl admin accessToke
func WithAccessToken(token string) Option {
	return func(o *ClientOptions) {
		o.AccessToken = token
	}
}

// WithAppName app name
func WithAppName(appName string) Option {
	return func(o *ClientOptions) {
		o.AppName = appName
	}
}

// WithClientPort xxl client port
func WithClientPort(port int) Option {
	return func(o *ClientOptions) {
		o.Port = port
	}
}

// WithAdminTimeout xxl admin request timeout
func WithAdminTimeout(timeout time.Duration) Option {
	return func(o *ClientOptions) {
		o.Timeout = timeout
	}
}

// WithBeatTime xxl admin renew time
func WithBeatTime(beatTime time.Duration) Option {
	return func(o *ClientOptions) {
		o.BeatTime = beatTime
	}
}

// WithEnableHttp option
func WithEnableHttp(enable bool) Option {
	return func(o *ClientOptions) {
		o.EnableHttp = enable
	}
}
