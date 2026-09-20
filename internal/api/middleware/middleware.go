package middleware

import "net/http"

// Middleware decora um handler HTTP.
type Middleware func(http.Handler) http.Handler

// Chain aplica os middlewares na ordem em que foram declarados: o primeiro
// da lista é o mais externo.
func Chain(h http.Handler, middlewares ...Middleware) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}
