package hutil

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

type myHandlerContext struct {
	userID string
}

type myHandler = Handler[myHandlerContext]
type myHandlerFunc = HandlerFunc[myHandlerContext]

var errNotOK = errors.New("not ok")

func statusHandler(ctx context.Context, handlerCtx myHandlerContext, w http.ResponseWriter, req *http.Request) error {
	data, err := io.ReadAll(req.Body)
	if err != nil {
		return fmt.Errorf("unable to read body, err: %w", err)
	}

	if string(data) == "KO" {
		return errNotOK
	} else if string(data) == "FOO" {
		return errors.New("foo")
	}

	w.WriteHeader(http.StatusOK)
	io.WriteString(w, "OK")

	return nil
}

func wrapHandler(logger *zap.Logger, handler myHandler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ctx, cancel := context.WithTimeout(req.Context(), 1*time.Second)
		defer cancel()

		hctx := myHandlerContext{
			userID: req.PathValue("user_id"),
		}
		// do authentication etc.

		err := handler.Handle(ctx, hctx, w, req)
		switch {
		case errors.Is(err, errNotOK):
			http.Error(w, "not ok !!", http.StatusInternalServerError)
		case err != nil:
			logger.Error("handler failed", zap.Error(err))
			http.Error(w, "unknown error", http.StatusInternalServerError)
		}
	})
}

func TestHandler(t *testing.T) {
	logger := zaptest.NewLogger(t)
	handler := wrapHandler(logger, myHandlerFunc(statusHandler))

	ts := httptest.NewServer(handler)
	defer ts.Close()

	testCases := []struct {
		body          string
		expStatusCode int
		expResp       string
	}{
		{`{}`, 200, "OK"},
		{`KO`, 500, "not ok !!"},
		{`FOO`, 500, "unknown error"},
	}

	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			resp, err := http.Post(ts.URL, "application/json", strings.NewReader(tc.body))
			require.NoError(t, err)
			require.Equal(t, tc.expStatusCode, resp.StatusCode)

			data, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			tmp := strings.TrimSpace(string(data))
			require.Equal(t, tc.expResp, tmp)
		})
	}
}
