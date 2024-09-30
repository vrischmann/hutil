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

// Router is a stack of [Handler]. You stack a handler by calling [Router.Add].
//
// You can then call [Router.Handler] to get a handler that calls every handler sequentially.
//
// This type is similar to [MiddlewareStack] but better because it works nicely with [Handler].
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
	handlers []Handler[C]
}

// NewRouter creates a new [Router] with the provided handlers.
func NewRouter[C any](handler ...Handler[C]) *Router[C] {
	return &Router[C]{
		handlers: handler,
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
func (s *Router[C]) Add(handler Handler[C]) *Router[C] {
	s.handlers = append(s.handlers, handler)
	return s
}

// AddFunc adds a [Handler] to the stack by wrapping `handler` into a [HandlerFunc]. Returns the stack as well.
//
// This can be used to chain calls:
//
//	stdHandlers := hutil.NewRouter[struct{}]()
//
//	loggedInHandlers := stdHandlers.Diverge().
//		AddFunc(func(ctx context.Context, hctx struct{}, w http.ResponseWriter, req *http.Request) error {
//			return nil
//		}).
//		AddFunc(func(ctx context.Context, hctx struct{}, w http.ResponseWriter, req *http.Request) error {
//			return nil
//		})
func (s *Router[C]) AddFunc(handler func(ctx context.Context, hctx C, w http.ResponseWriter, req *http.Request) error) *Router[C] {
	s.handlers = append(s.handlers, HandlerFunc[C](handler))
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
	tmp := make([]Handler[C], len(s.handlers))
	copy(tmp, s.handlers)

	return &Router[C]{handlers: tmp}
}

// Handler creates a [Handler] that calls every handler in the stack plus the `finalHandler` provided.
// Every handler is called sequentially; if a handler returns an error this error is returned and no further handler in the stack will be called.
func (s *Router[C]) Handler(finalHandler Handler[C]) Handler[C] {
	return HandlerFunc[C](func(ctx context.Context, handlerContext C, w http.ResponseWriter, req *http.Request) error {
		for _, handler := range s.handlers {
			err := handler.Handle(ctx, handlerContext, w, req)
			if err != nil {
				return err
			}
		}

		return finalHandler.Handle(ctx, handlerContext, w, req)
	})
}

// HandlerFunc creates a [Handler] that calls every handler in the stack plus the final handler provided.
// This is just like [Router.Handler] except that it wraps the function `finalHandler` with [HandlerFunc].
//
// Every handler is called sequentially; if a handler returns an error this error is returned and no further handler in the stack will be called.
func (s *Router[C]) HandlerFunc(fn func(ctx context.Context, handlerContext C, w http.ResponseWriter, req *http.Request) error) Handler[C] {
	finalHandler := HandlerFunc[C](fn)

	return HandlerFunc[C](func(ctx context.Context, handlerContext C, w http.ResponseWriter, req *http.Request) error {
		for _, handler := range s.handlers {
			err := handler.Handle(ctx, handlerContext, w, req)
			if err != nil {
				return err
			}
		}

		return finalHandler.Handle(ctx, handlerContext, w, req)
	})
}
