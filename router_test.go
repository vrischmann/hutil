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

func TestHandlerStack(t *testing.T) {
	logger := zaptest.NewLogger(t)
	var mw Router[myHandlerContext]

	mw.Add(myHandlerFunc(func(ctx context.Context, handlerContext myHandlerContext, w http.ResponseWriter, req *http.Request) error {
		logger.Info("first one")
		return nil
	}))

	mw.Add(myHandlerFunc(func(ctx context.Context, handlerContext myHandlerContext, w http.ResponseWriter, req *http.Request) error {
		logger.Info("second one")
		return nil
	}))

	mw.Add(myHandlerFunc(func(ctx context.Context, handlerContext myHandlerContext, w http.ResponseWriter, req *http.Request) error {
		logger.Info("third one")
		return nil
	}))

	mw.Add(myHandlerFunc(func(ctx context.Context, handlerContext myHandlerContext, w http.ResponseWriter, req *http.Request) error {
		logger.Info("final one")

		w.WriteHeader(http.StatusFound)

		return nil
	}))

	require.Len(t, mw.handlers, 4)

	//

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		var hctx myHandlerContext

		handler := mw.Handler(myHandlerFunc(func(ctx context.Context, hctx myHandlerContext, w http.ResponseWriter, req *http.Request) error {
			return nil
		}))

		err := handler.Handle(req.Context(), hctx, w, req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}))
	defer ts.Close()

	resp, err := http.Get(ts.URL)
	require.NoError(t, err)
	require.Equal(t, http.StatusFound, resp.StatusCode)
}

func TestHandlerStackDiverge(t *testing.T) {
	logger := zaptest.NewLogger(t)

	calls := map[string]int{}

	first := myHandlerFunc(func(ctx context.Context, handlerContext myHandlerContext, w http.ResponseWriter, req *http.Request) error {
		calls["first"] += 1
		logger.Info("first one")
		return nil
	})
	second := myHandlerFunc(func(ctx context.Context, handlerContext myHandlerContext, w http.ResponseWriter, req *http.Request) error {
		calls["second"] += 1
		logger.Info("second one")
		return nil
	})
	third := myHandlerFunc(func(ctx context.Context, handlerContext myHandlerContext, w http.ResponseWriter, req *http.Request) error {
		calls["third"] += 1
		logger.Info("third one")
		return nil
	})
	final := myHandlerFunc(func(ctx context.Context, handlerContext myHandlerContext, w http.ResponseWriter, req *http.Request) error {
		calls["final"] += 1
		logger.Info("final one")
		w.WriteHeader(http.StatusConflict)
		return nil
	})

	stack := NewRouter(first, second)

	clonedMW := stack.Diverge()
	clonedMW.Add(third)
	require.Len(t, clonedMW.handlers, 3)

	stack.Add(final)
	require.Len(t, stack.handlers, 3)

	//

	serve := func(st *Router[myHandlerContext]) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			var hctx myHandlerContext

			handler := st.Handler(myHandlerFunc(func(ctx context.Context, hctx myHandlerContext, w http.ResponseWriter, req *http.Request) error {
				return nil
			}))

			err := handler.Handle(req.Context(), hctx, w, req)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		}))
	}

	{
		ts := serve(stack)
		defer ts.Close()

		resp, err := http.Get(ts.URL)
		require.NoError(t, err)
		require.Equal(t, http.StatusConflict, resp.StatusCode)

		require.Equal(t, map[string]int{"first": 1, "second": 1, "final": 1}, calls)
	}

	{
		ts := serve(clonedMW)
		defer ts.Close()

		resp, err := http.Get(ts.URL)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		require.Equal(t, map[string]int{"first": 2, "second": 2, "third": 1, "final": 1}, calls)
	}
}

func TestHandlerStackError(t *testing.T) {
	logger := zaptest.NewLogger(t)

	calls := map[string]int{}

	first := myHandlerFunc(func(ctx context.Context, handlerContext myHandlerContext, w http.ResponseWriter, req *http.Request) error {
		calls["first"] += 1
		logger.Info("first one")
		return nil
	})
	second := myHandlerFunc(func(ctx context.Context, handlerContext myHandlerContext, w http.ResponseWriter, req *http.Request) error {
		calls["second"] += 1
		logger.Error("second one")
		return errors.New("got error")
	})
	third := myHandlerFunc(func(ctx context.Context, handlerContext myHandlerContext, w http.ResponseWriter, req *http.Request) error {
		calls["third"] += 1
		logger.Info("third one")
		return nil
	})
	final := myHandlerFunc(func(ctx context.Context, handlerContext myHandlerContext, w http.ResponseWriter, req *http.Request) error {
		calls["final"] += 1
		logger.Info("final one")
		w.WriteHeader(http.StatusConflict)
		return nil
	})

	mw := NewRouter(
		first, second, third, final,
	)
	require.Len(t, mw.handlers, 4)

	//

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		var hctx myHandlerContext

		handler := mw.Handler(myHandlerFunc(func(ctx context.Context, hctx myHandlerContext, w http.ResponseWriter, req *http.Request) error {
			return nil
		}))

		err := handler.Handle(req.Context(), hctx, w, req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}))
	defer ts.Close()

	resp, err := http.Get(ts.URL)
	require.NoError(t, err)
	require.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	data, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, "got error\n", string(data))

	require.Equal(t, map[string]int{"first": 1, "second": 1}, calls)
}
