package constants

const (
	DateTimeFormat = "2006-01-02 15:04:05"
	DateFormat     = "2006-01-02"
	BasePath       = "/data/applogs/xxl-job/jobhandler"
	GlueSourceName = "gluesource"
	GluePrefix     = "GLUE_"
	GluePrefixLen  = len(GluePrefix)

	HttpProtocol = "http"

	// CtxParamKey name
	CtxParamKey = "jobParam"

	// EnvXxlShardIdx env name
	EnvXxlShardIdx = "XXL_SHARD_IDX"
	// EnvXxlShardTotal env name
	EnvXxlShardTotal = "XXL_SHARD_TOTAL"
)
