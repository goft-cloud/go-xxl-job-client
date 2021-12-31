package constants

const (
	DateTimeFormat  = "2006-01-02 15:04:05"
	DateTimeFormat2 = "2006/01/02 15:04:05"

	DateFormat  = "2006-01-02"
	LogBasePath = "/var/tmp/xxl-job-go-logs"

	GlueSourceName = "gluesource"
	GluePrefix     = "GLUE_"
	GluePrefixLen  = len(GluePrefix)

	// CronPeriod is connection session heartbeat check interval.
	// nanoseconds, time.Duration(CronPeriod).Seconds()
	// - 设置为30s, 2.1.2 xxl-job 的ping beat 是30s一个
	CronPeriod = 30e9

	HttpProtocol = "http"

	// CtxParamKey name
	CtxParamKey = "jobParam"

	// EnvXxlShardIdx env name
	EnvXxlShardIdx = "XXL_SHARD_IDX"
	// EnvXxlShardTotal env name
	EnvXxlShardTotal = "XXL_SHARD_TOTAL"
)
