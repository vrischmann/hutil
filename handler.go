package hutil

import (
	"context"
	"net/http"
)

type Handler[HandlerContext any] interface {
	Handle(ctx context.Context, handlerCtx HandlerContext, w http.ResponseWriter, req *http.Request) error
}

type HandlerFunc[HandlerContext any] func(ctx context.Context, handlerCtx HandlerContext, w http.ResponseWriter, req *http.Request) error

func (f HandlerFunc[HandlerContext]) Handle(ctx context.Context, handlerCtx HandlerContext, w http.ResponseWriter, req *http.Request) error {
	return f(ctx, handlerCtx, w, req)
}
