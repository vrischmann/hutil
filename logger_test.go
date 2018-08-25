package hutil

import (
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type loggingHandlerTestCase struct {
	path      string
	expected  string
	exact     bool
	sleepTime time.Duration
}

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
	require.Nil(t, err)

	body, err := ioutil.ReadAll(resp.Body)
	require.Nil(t, err)
	resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "foobar", string(body))

	require.Equal(t, http.StatusOK, data.StatusCode)
	require.Equal(t, "/foo/bar/baz", data.Req.URL.Path)
	require.Equal(t, len(body), data.ResponseSize)
	require.True(t, data.Elapsed > 500*time.Millisecond)
}
