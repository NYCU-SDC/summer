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

// Append return a new Set with the new middleware added to the end of the chain, won't change the original one
func (s *Set) Append(middleware func(next http.HandlerFunc) http.HandlerFunc) Set {
	newMiddlewares := make([]func(next http.HandlerFunc) http.HandlerFunc, len(s.middlewares))
	copy(newMiddlewares, s.middlewares)

	newMiddlewares = append(newMiddlewares, middleware)
	return Set{middlewares: newMiddlewares}
}

// HandlerFunc returns a http.HandlerFunc that applies all middlewares in the set in the appended order to the given handler
func (s *Set) HandlerFunc(next http.HandlerFunc) http.HandlerFunc {
	for i := len(s.middlewares) - 1; i >= 0; i-- {
		next = s.middlewares[i](next)
	}
	return next
}
