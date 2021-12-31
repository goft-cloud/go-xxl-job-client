package xxl

import (
	"context"
	"errors"

	"github.com/goft-cloud/go-xxl-job-client/v2/constants"
	"github.com/goft-cloud/go-xxl-job-client/v2/param"
)

// CtxParamKey name
const CtxParamKey = constants.CtxParamKey

// GetParam get input param by name.
func GetParam(ctx context.Context, name string) (string, bool) {
	if obj, err := GetParamObj(ctx); err == nil {
		return obj.TryParam(name)
	}
	return "", false
}

// GetFullParam string
func GetFullParam(ctx context.Context) string {
	if obj, err := GetParamObj(ctx); err == nil {
		return obj.InputParam
	}
	return ""
}

// GetParamObj object
func GetParamObj(ctx context.Context) (*param.CtxJobParam, error) {
	val := ctx.Value(CtxParamKey)
	if val == nil {
		return nil, errors.New("job param not exists")
	}

	if cjp, ok := val.(*param.CtxJobParam); ok {
		return cjp, nil
	}

	return nil, errors.New("job param not exists")
}

// GetSharding info
func GetSharding(ctx context.Context) (shardIdx, shardTotal int32) {
	if obj, err := GetParamObj(ctx); err == nil {
		shardIdx = obj.ShardIndex
		shardTotal = obj.ShardTotal
	}
	return
}
