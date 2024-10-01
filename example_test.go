package hutil

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
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
