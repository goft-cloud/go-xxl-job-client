package param

import "fmt"

// CtxJobParam struct
type CtxJobParam struct {
	JobID int32
	LogID int64
	// JobName job handler name.
	JobName string
	// JobFunc on script mode, is script file path.
	JobFunc string
	// ShardIndex sharding info params
	ShardIndex int32
	ShardTotal int32
	// InputParam is full user input param string. equals to InputParams["fullParam"]
	InputParam  string
	InputParams map[string]string
}

// Param get input param by name.
func (cjp *CtxJobParam) Param(name string) string {
	return cjp.InputParams[name]
}

// TryParam try to get input param by name.
func (cjp *CtxJobParam) TryParam(name string) (string, bool) {
	val, ok := cjp.InputParams[name]
	return val, ok
}

// String to string.
func (cjp *CtxJobParam) String() string {
	return fmt.Sprintf("%#v", *cjp)
}
