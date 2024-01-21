package hutil

import "net/http"

// Middleware is a function taking a HTTP handler and returning a new handler that wraps it.
// For example, you could have a middleware that injects a session in the request's context:
//
//	// Create a middleware that injects a session in the request context
//	func sessionMiddleware(next http.Handler) Middleware {
//		return func(w http.ResponseWriter, req *http.Request) {
//			session := fetchSession(req)
//			ctx := contet.WithValue(req.Context(), "session", session)
//			req = req.WithContext(ctx)
//			next.ServeHTTP(w, req)
//		}
//	}
//
// Note that the you _have_ to call `next.ServeHTTP` otherwise requests stop in this handler.
//
// Next you wrap your own handler with this middleware:
//
//	myHandler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
//		...
//	)
//	handler := sessionMiddleware(myHandler)
//
// Now whenever your handler is called it can easily get the session
type Middleware func(next http.Handler) http.Handler

// MiddlewareStack as its name imply, stacks middlewares.
// You stack a middleware by calling `Use`.
// Keep in mind that the first middleware you stack will be the middleware that wraps everything else,
// this means if you have something like a logging middleware it should be stacked first.
//
// Here is an example:
//
//	var stack MiddlewareStack
//	stack.Use(newLoggingMiddleware())
//	stack.Use(newRateLimitMiddleware())
//	stack.Use(newSessionMiddleware())
//	stack.Use(newUserMiddleware())
//
// This will create the following chain of middleware calls:
// logging -> ratelimit -> session -> user
type MiddlewareStack []Middleware

// Use stacks a middleware.
func (s *MiddlewareStack) Use(h Middleware) {
	*s = append(*s, h)
}

// Handler wraps the provided final handler with all the middleware stacked and returns a http.Handler instance.
//
// Here is an example:
//
//	myHandler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
//		...
//	)
//
//	var stack MiddlewareStack
//	stack.Use(newLoggingMiddleware())
//	stack.Use(newRateLimitMiddleware())
//	handler := stack.Handler(myHandler)
//
// This will create the following chain of calls:
// logging -> ratelimit -> myHandler
func (s MiddlewareStack) Handler(h http.Handler) http.Handler {
	for i := len(s) - 1; i >= 0; i-- {
		h = s[i](h)
	}
	return h
}
