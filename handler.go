package hutil

import (
	"context"
	"net/http"
)

// Handler is my own version of a HTTP handler.
//
// The standard [http.Handler] is almost perfect except for the fact that it can't return an error, this is the main change I added.
//
// I also added a `HandlerContext`, this is an arbitrary type that can be used to pass around things needed by every handlers, for
// example a database handle, a logger, etc.
// Having it as an argument to the handler makes it easier to write handlers as functions instead of structs of methods on a struct.
// This is a personal preference thing.
//
// Finally, I explicitly add a [context.Context] instead of relying on the request' context: this is because I want every handler to
// have a context with a timeout. I _could_ create a new request with this new context but that is an expensive operation.
type Handler[HandlerContext any] interface {
	Handle(ctx context.Context, handlerCtx HandlerContext, w http.ResponseWriter, req *http.Request) error
}

// HandlerFunc is the function version of a [Handler].
type HandlerFunc[HandlerContext any] func(ctx context.Context, handlerCtx HandlerContext, w http.ResponseWriter, req *http.Request) error

// Handle implements [Handler].
func (f HandlerFunc[HandlerContext]) Handle(ctx context.Context, handlerCtx HandlerContext, w http.ResponseWriter, req *http.Request) error {
	return f(ctx, handlerCtx, w, req)
}
