package option

import (
	"time"

	"github.com/goft-cloud/go-xxl-job-client/v2/constants"
)

const (
	defaultAdminAddr = "http://localhost:8080/xxl-job-admin/"
	defaultAppName   = "xxl-go-executor"
	defaultPort      = 8081
	defaultTimeout   = 5 * time.Second
	defaultBeatTime  = 20 * time.Second
)

// OptionFunc func
type OptionFunc func(*ClientOptions)

// ClientOptions struct
type ClientOptions struct {
	// Enable client.
	Enable bool
	// AdminAddr xxl admin 地址
	AdminAddr []string
	// AccessToken token
	AccessToken string
	// LogBasePath the job logs base dir path.
	LogBasePath string
	// AppName 执行器名
	AppName string
	// ShellBin shell bin file. default is bash.
	ShellBin string
	// ClientPort client 执行器端口
	ClientPort int
	// EnableHttp 开启http协议
	EnableHttp bool
	// Timeout 请求admin超时时间
	Timeout time.Duration
	// BeatTime 执行器续约时间（超过30秒不续约admin会移除执行器，请设置到30秒以内）
	BeatTime time.Duration
}

// NewClientOptions instance
func NewClientOptions(opts ...OptionFunc) ClientOptions {
	options := ClientOptions{
		AdminAddr:   []string{defaultAdminAddr},
		AccessToken: "",
		AppName:     defaultAppName,
		ClientPort:  defaultPort,
		Timeout:     defaultTimeout,
		BeatTime:    defaultBeatTime,
		ShellBin:    constants.ShellBash,
		LogBasePath: constants.LogBasePath,
	}

	for _, o := range opts {
		o(&options)
	}
	return options
}

// WithOptionsFunc with custom option func
func WithOptionsFunc(fn func(opts *ClientOptions)) OptionFunc {
	return fn
}

// WithLogBasePath job logs base dir path
func WithLogBasePath(dirPath string) OptionFunc {
	return func(o *ClientOptions) {
		o.LogBasePath = dirPath
	}
}

// WithAdminAddress xxl admin address
func WithAdminAddress(addrs ...string) OptionFunc {
	return func(o *ClientOptions) {
		o.AdminAddr = addrs
	}
}

// WithAccessToken xxl admin accessToke
func WithAccessToken(token string) OptionFunc {
	return func(o *ClientOptions) {
		o.AccessToken = token
	}
}

// WithAppName app name
func WithAppName(appName string) OptionFunc {
	return func(o *ClientOptions) {
		o.AppName = appName
	}
}

// WithClientPort xxl client port
func WithClientPort(port int) OptionFunc {
	return func(o *ClientOptions) {
		o.ClientPort = port
	}
}

// WithAdminTimeout xxl admin request timeout
func WithAdminTimeout(timeout time.Duration) OptionFunc {
	return func(o *ClientOptions) {
		o.Timeout = timeout
	}
}

// WithBeatTime xxl admin renew time
func WithBeatTime(beatTime time.Duration) OptionFunc {
	return func(o *ClientOptions) {
		o.BeatTime = beatTime
	}
}

// WithEnableHttp option
func WithEnableHttp(enable bool) OptionFunc {
	return func(o *ClientOptions) {
		o.EnableHttp = enable
	}
}
