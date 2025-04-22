package middleware

import (
	"net/http"
)

type Set struct {
	middlewares []func(next http.HandlerFunc) http.HandlerFunc
}

func NewSet(middlewares ...func(next http.HandlerFunc) http.HandlerFunc) *Set {
	return &Set{middlewares: middlewares}
}

func (s *Set) Append(middleware func(next http.HandlerFunc) http.HandlerFunc) Set {
	newMiddlewares := make([]func(next http.HandlerFunc) http.HandlerFunc, len(s.middlewares))
	copy(newMiddlewares, s.middlewares)

	newMiddlewares = append(newMiddlewares, middleware)
	return Set{middlewares: newMiddlewares}
}

func (s *Set) HandlerFunc(next http.HandlerFunc) http.HandlerFunc {
	for i := len(s.middlewares) - 1; i >= 0; i-- {
		next = s.middlewares[i](next)
	}
	return next
}
