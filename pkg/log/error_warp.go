package logutil

import "errors"

type InfoCarrier interface {
	LogInfo() map[string]any
}

type InfoError[K ~string, V any] struct {
	Base error
	Info map[K]V
}

func (e *InfoError[K, V]) Error() string {
	if e.Base != nil {
		return e.Base.Error()
	}
	return ""
}

func (e *InfoError[K, V]) Unwrap() error {
	return e.Base
}

func (e *InfoError[K, V]) LogInfo() map[string]any {
	if len(e.Info) == 0 {
		return nil
	}

	out := make(map[string]any, len(e.Info))
	for k, v := range e.Info {
		out[string(k)] = v
	}

	return out
}

func NewInfoError[K ~string, V any](base error, info map[K]V) error {
	return &InfoError[K, V]{
		Base: base,
		Info: info,
	}
}

func WrapInfoError[K ~string, V any](base error, info map[K]V) error {
	return &InfoError[K, V]{
		Base: base,
		Info: info,
	}
}

func InfoFromError(err error) map[string]any {
	var carrier InfoCarrier
	if errors.As(err, &carrier) {
		return carrier.LogInfo()
	}

	return nil
}