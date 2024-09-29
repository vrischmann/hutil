package hutil

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
)

func ExampleMiddlewareStack() {
	createMiddleware := func(buf *bytes.Buffer, s string) func(next http.Handler) http.Handler {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				buf.WriteString(s)
				next.ServeHTTP(w, req)
			})
		}
	}

	var (
		s   MiddlewareStack
		buf bytes.Buffer
	)

	var (
		m1 = createMiddleware(&buf, "m1")
		m2 = createMiddleware(&buf, "m2")
		m3 = createMiddleware(&buf, "m3")
	)

	s.Use(m1)
	s.Use(m2)
	s.Use(m3)

	ts := httptest.NewServer(s.Handler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "foobar")
	})))
	defer ts.Close()

	res, _ := http.Get(ts.URL)

	data, _ := io.ReadAll(res.Body)
	defer res.Body.Close()

	fmt.Println(string(data))
	fmt.Println(buf.String())
	// Output:
	// foobar
	// m1m2m3
}

func ExampleHandlerStack() {
	var (
		errNotLoggedIn = errors.New("not logged in")
		errNotAdmin    = errors.New("not admin")
	)

	// This handler stack simulates a handler that fetches a user from a database using a session id.
	loggedInHandlers := NewHandlerStack(HandlerFunc[*string](func(ctx context.Context, hctx *string, w http.ResponseWriter, req *http.Request) error {
		sessionID := req.URL.Query().Get("session")
		if sessionID == "" {
			return errNotLoggedIn
		}
		// fetch session from database
		return nil
	}))

	// This handler stack simulats a handler that verifies that a user is an administrator.
	adminHandlers := loggedInHandlers.Diverge().Add(HandlerFunc[*string](func(ctx context.Context, hctx *string, w http.ResponseWriter, req *http.Request) error {
		role := req.URL.Query().Get("role") // fetch role from database
		if role != "admin" {
			return errNotAdmin
		}
		return nil
	}))

	makeHandlerAdapter := func(handler Handler[*string]) func(http.ResponseWriter, *http.Request) {
		return func(w http.ResponseWriter, req *http.Request) {
			var hctx string

			if err := handler.Handle(req.Context(), &hctx, w, req); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /dashboard", makeHandlerAdapter(loggedInHandlers.Handler(HandlerFunc[*string](func(ctx context.Context, hctx *string, w http.ResponseWriter, req *http.Request) error {
		io.WriteString(w, "this is dashboard")
		return nil
	}))))
	mux.HandleFunc("GET /admin", makeHandlerAdapter(adminHandlers.Handler(HandlerFunc[*string](func(ctx context.Context, hctx *string, w http.ResponseWriter, req *http.Request) error {
		io.WriteString(w, "this is admin")
		return nil
	}))))
	mux.HandleFunc("GET /other", makeHandlerAdapter(loggedInHandlers.Handler(HandlerFunc[*string](func(ctx context.Context, hctx *string, w http.ResponseWriter, req *http.Request) error {
		io.WriteString(w, "this is other")
		return nil
	}))))

	ts := httptest.NewServer(mux)
	defer ts.Close()

	//

	fetch := func(path string) string {
		res, _ := http.Get(ts.URL + path)

		data, _ := io.ReadAll(res.Body)
		defer res.Body.Close()

		return strings.TrimSpace(string(data))
	}

	fmt.Println(fetch("/dashboard"))
	fmt.Println(fetch("/dashboard?session=foobar"))
	fmt.Println(fetch("/admin?session=foobar"))
	fmt.Println(fetch("/admin?session=foobar&role=admin"))
	fmt.Println(fetch("/other?session=hello"))

	// Output:
	// not logged in
	// this is dashboard
	// not admin
	// this is admin
	// this is other
}
