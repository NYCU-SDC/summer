package logutil

import (
	"context"
	"fmt"
	"reflect"
	"sort"

	"go.uber.org/zap"
)

const (
	// the maximum number of elements to display in a slice before truncating
	maxSliceDisplayLength = 20
)

type MethodTracker struct {
	logger     *zap.Logger
	methodName string
}

// StartMethod begins tracking a method execution and logs the method entry with normalized parameters.
// It returns a MethodTracker that should be used to log the method completion.
//
// Parameters:
//   - ctx: The context for the method execution (currently unused but reserved for future use)
//   - logger: The zap logger instance to use for logging
//   - name: The name of the method being tracked
//   - params: A map of parameter names to values that will be normalized before logging
//
// Example:
//
//	tracker := logutil.StartMethod(ctx, logger, "CreateUser", map[string]interface{}{
//		"username": "john_doe",
//		"email": "john@example.com",
//	})
func StartMethod(ctx context.Context, logger *zap.Logger, name string, params map[string]interface{}) *MethodTracker {
	normalized := normalizeParams(params)
	paramKeys := extractSortedKeys(normalized)

	logger = logger.WithOptions(zap.AddCallerSkip(1))

	logger.Info(fmt.Sprintf("Method %s started with params: %v", name, paramKeys),
		zap.String("method.name", name),
		zap.Any("method.params", normalized),
	)

	return &MethodTracker{
		logger:     logger,
		methodName: name,
	}
}

// Complete logs the completion of the tracked method with normalized results.
// This should typically be called using defer to ensure it's always executed.
//
// Parameters:
//   - result: A map of result names to values that will be normalized before logging
//
// Example:
//
//	defer tracker.Complete(map[string]interface{}{
//		"user_id": userID,
//		"created_at": timestamp,
//	})
func (t *MethodTracker) Complete(result map[string]interface{}) {
	normalized := normalizeParams(result)
	logMsg := fmt.Sprintf("Method %s completed", t.methodName)

	t.logger.Info(logMsg,
		zap.String("method.name", t.methodName),
		zap.Any("method.result", normalized),
	)
}

func extractSortedKeys(params interface{}) []string {
	if asMap, ok := params.(map[string]interface{}); ok {
		keys := make([]string, 0, len(asMap))
		for k := range asMap {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		return keys
	}
	return []string{}
}

func normalizeParams(v interface{}) interface{} {
	if v == nil {
		return nil
	}

	val := reflect.ValueOf(v)

	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil
		}
		return normalizeParams(val.Elem().Interface())
	}

	if _, ok := v.(fmt.Stringer); ok {
		return v
	}

	switch val.Kind() {
	case reflect.Slice, reflect.Array:
		return normalizeSlice(val)

	case reflect.Struct:
		return normalizeStruct(val)

	case reflect.Map:
		return normalizeMap(val)

	default:
		return v
	}
}

func normalizeSlice(val reflect.Value) interface{} {
	length := val.Len()

	if length > maxSliceDisplayLength {
		truncated := make([]interface{}, maxSliceDisplayLength)
		for i := 0; i < maxSliceDisplayLength; i++ {
			truncated[i] = normalizeParams(val.Index(i).Interface())
		}
		return fmt.Sprintf("%v... (total: %d)", truncated, length)
	}

	result := make([]interface{}, length)
	for i := 0; i < length; i++ {
		result[i] = normalizeParams(val.Index(i).Interface())
	}
	return fmt.Sprintf("%v (total: %d)", result, length)
}

func normalizeStruct(val reflect.Value) interface{} {
	typ := val.Type()
	result := make(map[string]interface{}, val.NumField())

	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		result[field.Name] = normalizeParams(val.Field(i).Interface())
	}

	return result
}

func normalizeMap(val reflect.Value) interface{} {
	result := make(map[string]interface{}, val.Len())
	iter := val.MapRange()

	for iter.Next() {
		key := iter.Key().String()
		result[key] = normalizeParams(iter.Value().Interface())
	}

	return result
}
