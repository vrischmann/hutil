package hutil

import "net/http"

// Heavily inspired by https://github.com/rs/xhandler

// Chain is a helper for chaining middleware handlers together for easier management.
type Chain []func(next http.Handler) http.Handler

// Use appends a handler to the middleware chain.
func (c *Chain) Use(h func(next http.Handler) http.Handler) {
	*c = append(*c, h)
}

// Handler wraps the provided final handler with all the middleware appended to
// the chain and returns a http.Handler instance.
func (c Chain) Handler(h http.Handler) http.Handler {
	for i := len(c) - 1; i >= 0; i-- {
		h = c[i](h)
	}
	return h
}
