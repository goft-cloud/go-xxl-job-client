package utils

import (
	"reflect"

	"github.com/goft-cloud/go-xxl-job-client/v2/logger"
)

func ReflectStructToMap(ob interface{}) map[string]interface{} {
	res := make(map[string]interface{})
	object := reflect.ValueOf(ob)
	ref := object.Elem()
	typeOfType := ref.Type()

	if typeOfType.Kind() != reflect.Struct {
		logger.Error("Check type error not struct")
		return nil
	}

	for i := 0; i < ref.NumField(); i++ {
		field := ref.Field(i)
		res[typeOfType.Field(i).Name] = field.Interface()
	}
	return res
}
