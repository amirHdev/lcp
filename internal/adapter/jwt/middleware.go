// Package jwt provides JWT middleware utilities.
package jwt

import "net/http"

// Middleware is a lightweight placeholder that forwards requests without
// performing JWT verification. It keeps the package compilable while a proper
// authentication layer is designed.
type Middleware func(http.Handler) http.Handler

// New returns a middleware that simply calls the next handler.
func New(_ string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}
}
