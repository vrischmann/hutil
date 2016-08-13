package hutil

import (
	"log"
	"net/http"
	"time"
)

// LoggingOptions is used to control the behavior of the logging middleware.
type LoggingOptions struct {
	// WithExecutionTime controls whether the execution time is logged.
	WithExecutionTime bool
	// WithHeaders controls whether the headers are logged.
	WithHeaders bool
	// Log is a function used to produce the log. If nil, a default function which uses the `log` package will be used.
	Log func(format string, args ...interface{})
}

// NewLoggingMiddleware returns a middleware that logs the requests and the results of the request as processed by the next middleware
// (or handler) in the chain.
//
// By default, only the status code, URL path and response length is logged.
// You can pass a LoggingOptions with the `WithHeaders` or `WithExecutionTime` fields set to true
// to add more data to the log.
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
func NewLoggingMiddleware(options *LoggingOptions) func(http.Handler) http.Handler {
	logFunc := log.Printf
	if options != nil && options.Log != nil {
		logFunc = options.Log
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			lw := &loggingWriter{underlying: w}

			start := time.Now()

			next.ServeHTTP(lw, r)

			elapsed := time.Since(start)

			switch {
			case options == nil:
				logFunc("%d %s l:%d", lw.statusCode, r.URL.Path, lw.size)
			case options.WithExecutionTime && options.WithHeaders:
				logFunc("%d %s headers:%s l:%d e:%d", lw.statusCode, r.URL.Path, r.Header, lw.size, elapsed/time.Millisecond)
			case options.WithExecutionTime:
				logFunc("%d %s l:%d e:%d", lw.statusCode, r.URL.Path, lw.size, elapsed/time.Millisecond)
			case options.WithHeaders:
				logFunc("%d %s headers:%v l:%d", lw.statusCode, r.URL.Path, r.Header, lw.size)
			}
		})
	}
}

type loggingWriter struct {
	underlying http.ResponseWriter
	statusCode int
	size       int
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
