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

func createMiddleware[C any](buf *bytes.Buffer, s string) func(next Handler[C]) Handler[C] {
	return func(next Handler[C]) Handler[C] {
		return HandlerFunc[C](func(ctx context.Context, hctx C, w http.ResponseWriter, req *http.Request) error {
			buf.WriteString(s)
			return next.Handle(ctx, hctx, w, req)
		})
	}
}

func ExampleRouter_Use() {
	var (
		router = NewRouter[struct{}]()
		buf    bytes.Buffer
		m1     = createMiddleware[struct{}](&buf, "m1")
		m2     = createMiddleware[struct{}](&buf, "m2")
		m3     = createMiddleware[struct{}](&buf, "m3")
	)

	router.Use(m1).Use(m2).Use(m3)

	handler := router.HandlerFunc(func(ctx context.Context, hctx struct{}, w http.ResponseWriter, _ *http.Request) error {
		fmt.Fprint(w, "foobar")
		return nil
	})

	ts := httptest.NewServer(adaptHandler(handler))
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

func ExampleRouter_Diverge() {
	var (
		errNotLoggedIn = errors.New("not logged in")
		errNotAdmin    = errors.New("not admin")
	)

	sessionMiddleware := Middleware[*string](func(next Handler[*string]) Handler[*string] {
		return HandlerFunc[*string](func(ctx context.Context, hctx *string, w http.ResponseWriter, req *http.Request) error {
			sessionID := req.URL.Query().Get("session")
			if sessionID == "" {
				return errNotLoggedIn
			}
			// fetch session from database
			return next.Handle(ctx, hctx, w, req)
		})
	})

	router := NewRouter[*string]().Use(sessionMiddleware)

	adminMiddleware := Middleware[*string](func(next Handler[*string]) Handler[*string] {
		return HandlerFunc[*string](func(ctx context.Context, hctx *string, w http.ResponseWriter, req *http.Request) error {
			role := req.URL.Query().Get("role") // fetch role from database
			if role != "admin" {
				return errNotAdmin
			}
			return next.Handle(ctx, hctx, w, req)
		})
	})

	adminRouter := router.Diverge().Use(adminMiddleware)

	mux := http.NewServeMux()
	mux.Handle("GET /dashboard", adaptHandler(router.HandlerFunc(func(ctx context.Context, hctx *string, w http.ResponseWriter, req *http.Request) error {
		io.WriteString(w, "this is dashboard")
		return nil
	})))
	mux.Handle("GET /admin", adaptHandler(adminRouter.HandlerFunc(func(ctx context.Context, hctx *string, w http.ResponseWriter, req *http.Request) error {
		io.WriteString(w, "this is admin")
		return nil
	})))
	mux.Handle("GET /other", adaptHandler(router.HandlerFunc(func(ctx context.Context, hctx *string, w http.ResponseWriter, req *http.Request) error {
		io.WriteString(w, "this is other")
		return nil
	})))

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

func adaptHandler[C any](handler Handler[C]) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		var hctx C

		err := handler.Handle(req.Context(), hctx, w, req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
}
