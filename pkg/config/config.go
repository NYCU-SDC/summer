package configutil

import (
	"errors"
	"reflect"
)

func Merge(base, override any) (any, error) {
	if base == nil {
		return nil, errors.New("base config cannot be nil")
	}
	if override == nil {
		return base, nil
	}

	final := base
	baseVal := reflect.ValueOf(&final).Elem()
	overrideVal := reflect.ValueOf(override).Elem()

	if baseVal.Type() != overrideVal.Type() {
		return nil, errors.New("config types do not match")
	}

	for i := 0; i < baseVal.NumField(); i++ {
		field := baseVal.Field(i)
		overrideField := overrideVal.Field(i)
		zero := reflect.Zero(field.Type()).Interface()

		if field.CanSet() && !reflect.DeepEqual(overrideField.Interface(), zero) {
			field.Set(overrideField)
		}
	}

	return &final, nil
}
