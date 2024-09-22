package hutil

import (
	"bufio"
	"errors"
	"net"
	"net/http"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type loggingMiddlewareOptions struct {
	level      zapcore.Level
	onlyErrors bool
}

func newDefaultLoggingMiddlewareOptions() *loggingMiddlewareOptions {
	return &loggingMiddlewareOptions{
		level:      zapcore.InfoLevel,
		onlyErrors: false,
	}
}

type LoggingMiddlewareOption func(*loggingMiddlewareOptions)

func LogLevel(level zapcore.Level) LoggingMiddlewareOption {
	return func(opts *loggingMiddlewareOptions) {
		opts.level = level
	}
}

func LogOnlyErrors(enabled bool) LoggingMiddlewareOption {
	return func(opts *loggingMiddlewareOptions) {
		opts.onlyErrors = enabled
	}
}

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
func NewLoggingMiddleware(logger *zap.Logger, opt ...LoggingMiddlewareOption) Middleware {
	opts := newDefaultLoggingMiddlewareOptions()
	for _, o := range opt {
		o(opts)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			lw := &loggingWriter{underlying: w}

			ru := *r.URL
			ru.Host = ""

			start := time.Now()
			next.ServeHTTP(lw, r)
			elapsed := time.Since(start)

			switch {
			case opts.onlyErrors && lw.statusCode < 400:
				// No logging for non-errors

			case !opts.onlyErrors && lw.statusCode < 400:
				// Log using the provided level for non-errors

				logger.Log(opts.level, "request handled",
					zap.Stringer("url", &ru),
					zap.Int("status_code", lw.statusCode),
					zap.Int("response_size", lw.size),
					zap.Duration("elapsed", elapsed),
				)

			case lw.statusCode >= 400 && lw.statusCode < 500:
				// Log using the WARN level for 4xx errors

				logger.Warn("request handled",
					zap.Stringer("url", &ru),
					zap.Int("status_code", lw.statusCode),
					zap.Int("response_size", lw.size),
					zap.Duration("elapsed", elapsed),
				)

			case lw.statusCode > 500:
				// Log using the ERROR level for 5xx errors and up

				logger.Error("request handled",
					zap.Stringer("url", &ru),
					zap.Int("status_code", lw.statusCode),
					zap.Int("response_size", lw.size),
					zap.Duration("elapsed", elapsed),
				)
			}
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
