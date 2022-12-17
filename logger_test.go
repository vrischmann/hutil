package hutil

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLoggingHandler(t *testing.T) {
	var data struct {
		StatusCode   int
		Req          *http.Request
		ResponseSize int
		Elapsed      time.Duration
	}

	logFn := func(req *http.Request, statusCode int, responseSize int, elapsed time.Duration) {
		data.StatusCode = statusCode
		data.Req = req
		data.ResponseSize = responseSize
		data.Elapsed = elapsed
	}

	//

	var c Chain
	c.Use(NewLoggingMiddleware(logFn))

	fh := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, "foobar")
		time.Sleep(500 * time.Millisecond)
	})

	ts := httptest.NewServer(c.Handler(fh))
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/foo/bar/baz")
	if err != nil {
		t.Fatal(err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if exp, got := http.StatusOK, resp.StatusCode; exp != got {
		t.Fatalf("expected status code %d, got %d", exp, got)
	}
	if exp, got := "foobar", string(body); exp != got {
		t.Fatalf("expected body %q, got %q", exp, got)
	}
	if exp, got := "/foo/bar/baz", data.Req.URL.Path; exp != got {
		t.Fatalf("expected path %s, got %s", exp, got)
	}
	if exp, got := len(body), data.ResponseSize; exp != got {
		t.Fatalf("expected response size %d, got %d", exp, got)
	}
	if exp, got := 500*time.Millisecond, data.Elapsed; got <= exp {
		t.Fatalf("expected elapsed time to be > %s, got %s", exp, got)
	}
}
