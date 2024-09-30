package hutil

import (
	"context"
	"net/http"
)

type Middleware[C any] func(next Handler[C]) Handler[C]

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
	Handle(ctx context.Context, hctx HandlerContext, w http.ResponseWriter, req *http.Request) error
}

// HandlerFunc is the function version of a [Handler].
type HandlerFunc[C any] func(ctx context.Context, hctx C, w http.ResponseWriter, req *http.Request) error

// Handle implements [Handler].
func (f HandlerFunc[C]) Handle(ctx context.Context, hctx C, w http.ResponseWriter, req *http.Request) error {
	return f(ctx, hctx, w, req)
}

// Router is a stack of [Handler]. You stack a handler by calling [Router.Add].
//
// You can then call [Router.Handler] to get a handler that calls every handler sequentially.
//
// # Advantages
//
// First, since [Handler] returns an error the final handler returned [Router.Handler] stops as soon as it gets an error.
//
// Second, the fact that [Handler] takes an additional handler context makes it really easy to build up the context for a request in different handlers.
//
// Imagine the following flow implemented by some handlers:
//   - fetch a user if logged in from a database
//   - fetch a user profile data from a cache
//   - fetch a rate limiter for an ip
//
// We can store all this information in a handler context:
//
//	type User struct{}
//	type UserProfile struct{}
//
//	type myCtx struct {
//	    user        *User
//	    profile     *UserProfile
//	    rateLimiter <-chan struct{}
//	}
//
//	func main() {
//	    fetchUserHandler := hutil.HandlerFunc[*myCtx](func(ctx context.Context, hctx *myCtx, w http.ResponseWriter, req *http.Request) error {
//	        hctx.user = new(User) // fetch from database
//	        return nil
//	    })
//
//	    fetchUserProfileHandler := hutil.HandlerFunc[*myCtx](func(ctx context.Context, hctx *myCtx, w http.ResponseWriter, req *http.Request) error {
//	        hctx.profile = new(UserProfile) // fetch from database
//	        return nil
//	    })
//
//	    getRateLimiterHandler := hutil.HandlerFunc[*myCtx](func(ctx context.Context, hctx *myCtx, w http.ResponseWriter, req *http.Request) error {
//	        hctx.rateLimiter = make(chan struct{})
//	        return nil
//	    })
//
//	    stack := hutil.NewRouter(
//	    	fetchUserHandler,
//	    	fetchUserProfileHandler,
//	    	getRateLimiterHandler,
//	   	)
//
//	    stack.Handler(hutil.HandlerFunc[*myCtx](func(ctx context.Context, hctx *myCtx, w http.ResponseWriter, req *http.Request) error {
//	        <-hctx.rateLimiter
//
//	        // do stuff with user
//	        // do stuff with user profile
//	        // render page
//
//	        return nil
//	    }))
//	}
type Router[C any] struct {
	middlewares []Middleware[C]
}

// NewRouter creates a new [Router] with the provided handlers.
func NewRouter[C any](middleware ...Middleware[C]) *Router[C] {
	return &Router[C]{
		middlewares: middleware,
	}
}

// Add adds a [Handler] to the stack. Returns the stack as well.
//
// This can be used to chain calls:
//
//	loggedInHandlers := hutil.NewRouter[struct{}]().
//		Add(hutil.HandlerFunc[struct{}](func(ctx context.Context, hctx struct{}, w http.ResponseWriter, req *http.Request) error {
//			return nil
//		})).
//		Add(hutil.HandlerFunc[struct{}](func(ctx context.Context, hctx struct{}, w http.ResponseWriter, req *http.Request) error {
//			return nil
//		}))
func (s *Router[C]) Use(middleware Middleware[C]) *Router[C] {
	s.middlewares = append(s.middlewares, middleware)
	return s
}

// Diverge creates a new [Router] that is a clone of the current one.
//
// This simplifies defining trees of handlers. For example, imagine you have:
//   - a bunch of standard handlers
//   - handlers for logged in routes only
//   - handlers for admin in routes only
//
// Diverge can help with that:
//
//	stdHandlers := hutil.NewRouter(hutil.HandlerFunc[struct{}](func(ctx context.Context, hctx struct{}, w http.ResponseWriter, req *http.Request) error {
//		// std handler one
//		return nil
//	}))
//
//	// Logged in handlers diverge from the standard handlers
//	loggedInHandlers := stdHandlers.Diverge().AddFunc(func(ctx context.Context, hctx struct{}, w http.ResponseWriter, req *http.Request) error {
//		// fetch user
//		return nil
//	})
//
//	// Admin handlers diverge from the logged in handlersbecause to use an admin you must be logged in
//	adminHandlers := loggedInHandlers.Diverge().AddFunc(func(ctx context.Context, hctx struct{}, w http.ResponseWriter, req *http.Request) error {
//		// check if user is admin
//		return nil
//	})
//
//	dashboardHandler := loggedInHandlers.HandlerFunc(func(ctx context.Context, hctx struct{}, w http.ResponseWriter, req *http.Request) error {
//		return nil
//	})
//
//	adminDashboardHandler := adminHandlers.HandlerFunc(func(ctx context.Context, hctx struct{}, w http.ResponseWriter, req *http.Request) error {
//		return nil
//	})
func (s *Router[C]) Diverge() *Router[C] {
	tmp := make([]Middleware[C], len(s.middlewares))
	copy(tmp, s.middlewares)

	return &Router[C]{middlewares: tmp}
}

// Handler creates a [Handler] that calls every handler in the stack plus the `finalHandler` provided.
// Every handler is called sequentially; if a handler returns an error this error is returned and no further handler in the stack will be called.
func (s *Router[C]) Handler(finalHandler Handler[C]) Handler[C] {
	h := finalHandler

	count := 0
	for i := len(s.middlewares) - 1; i >= 0; i-- {
		middleware := s.middlewares[i]
		h = middleware(h)

		count++
	}

	return h
}

// HandlerFunc creates a [Handler] that calls every handler in the stack plus the final handler provided.
// This is just like [Router.Handler] except that it wraps the function `finalHandler` with [HandlerFunc].
//
// Every handler is called sequentially; if a handler returns an error this error is returned and no further handler in the stack will be called.
func (s *Router[C]) HandlerFunc(fn func(ctx context.Context, handlerContext C, w http.ResponseWriter, req *http.Request) error) Handler[C] {
	finalHandler := HandlerFunc[C](fn)

	return s.Handler(finalHandler)
}
