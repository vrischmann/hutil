package hutil

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func createMiddleware(buf *bytes.Buffer, s string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			buf.WriteString(s)
			next.ServeHTTP(w, req)
		})
	}
}

func TestMiddlewareStack(t *testing.T) {
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

	ts := httptest.NewServer(s.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "foobar")
	})))
	defer ts.Close()

	res, err := http.Get(ts.URL)
	if err != nil {
		t.Fatal(err)
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if exp, got := "foobar", string(data); exp != got {
		t.Fatalf("expected body %q, got %q", exp, got)
	}
	if exp, got := "m1m2m3", buf.String(); exp != got {
		t.Fatalf("expected %q, got %q", exp, got)
	}
}
