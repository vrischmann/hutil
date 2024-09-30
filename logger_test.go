package hutil

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestLoggingHandler(t *testing.T) {
	zapCore, zapObservedLogs := observer.New(zapcore.DebugLevel)
	logger := zap.New(zapCore)

	//

	router := NewRouter(NewLoggingMiddleware[struct{}](logger))

	fh := func(ctx context.Context, hctx struct{}, w http.ResponseWriter, req *http.Request) error {
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, "foobar")
		time.Sleep(500 * time.Millisecond)
		return nil
	}

	ts := httptest.NewServer(adaptHandler(router.HandlerFunc(fh)))
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

	logs := zapObservedLogs.All()
	if n := len(logs); n != 1 {
		t.Fatalf("expected one log to be produced, got %d", n)
	}

	logContext := logs[0].Context

	if n := len(logContext); n != 5 {
		t.Fatalf("expected 5 fields on the log line, got %d", n)
	}

	require.Equal(t, "url", logContext[0].Key)
	require.Equal(t, "/foo/bar/baz", logContext[0].Interface.(*url.URL).String())
	require.Equal(t, zapcore.StringerType, logContext[0].Type)

	require.Equal(t, "method", logContext[1].Key)
	require.Equal(t, "GET", logContext[1].String)
	require.Equal(t, zapcore.StringType, logContext[1].Type)

	require.Equal(t, "status_code", logContext[2].Key)
	require.Equal(t, int64(200), logContext[2].Integer)
	require.Equal(t, zapcore.Int64Type, logContext[2].Type)

	require.Equal(t, "response_size", logContext[3].Key)
	require.Equal(t, int64(6), logContext[3].Integer)
	require.Equal(t, zapcore.Int64Type, logContext[3].Type)

	require.Equal(t, "elapsed", logContext[4].Key)
	require.InDelta(t, int64(500*time.Millisecond), logContext[4].Integer, float64(10e6)) // delta of 10ms (10 000 000 ns)
	require.Equal(t, zapcore.DurationType, logContext[4].Type)
}
