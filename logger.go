package hutil

import (
	"bufio"
	"errors"
	"net"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// NewLoggingMiddleware returns a middleware that logs the requests and the results of the request as processed by the next middleware
// (or handler) in the chain.
//
// The middleware doesn't log by itself, instead the logFn you pass should do that. THe middleware merely
// gives you the data to log.
//
// Note about the execution time. Depending on where in the chain you place the logging handler, you will get
// different execution times.
// Take the following chain example:
// Middleware1 -> Logging -> FinalHandler (your business-logic handler, or muxer for example)
//
// The logging middleware, when called, will start counting AFTER the two handlers before in the chain, meaning it will only
// measure the execution time of the final handler.
// This isn't always what you want, because if you have a middleware in the chain that can take some measurable time, you probably
// want to count it too in the execution time.
// Thus, make sure you place the logging handler at the correct place in the chain.
func NewLoggingMiddleware(logger *zap.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			lw := &loggingWriter{underlying: w}

			ru := *r.URL
			ru.Host = ""

			// Preserve the original path.
			// When using ShiftPath the request is altered and thus the logging call reports a wrong path.
			// originalPath := ru.Path

			start := time.Now()
			next.ServeHTTP(lw, r)
			elapsed := time.Since(start)

			// r.URL.Path = originalPath
			logger.Info("request handled",
				zap.Stringer("url", &ru),
				zap.Int("status_code", lw.statusCode),
				zap.Int("response_size", lw.size),
				zap.Duration("elapsed", elapsed),
			)
		})
	}
}

type loggingWriter struct {
	underlying http.ResponseWriter
	statusCode int
	size       int
}

func (w *loggingWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	v, ok := w.underlying.(http.Hijacker)
	if !ok {
		panic(errors.New("response writer does not implement Hijacker"))
	}

	return v.Hijack()
}

func (w *loggingWriter) Header() http.Header {
	return w.underlying.Header()
}

func (w *loggingWriter) Write(b []byte) (n int, err error) {
	w.size += len(b)
	return w.underlying.Write(b)
}

func (w *loggingWriter) WriteHeader(code int) {
	w.statusCode = code
	w.underlying.WriteHeader(code)
}

var _ http.ResponseWriter = (*loggingWriter)(nil)
