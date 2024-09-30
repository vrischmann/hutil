package hutil

import (
	"net/http"
	"time"

	"github.com/vrischmann/hutil/v5/internal"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type loggingMiddlewareOptions struct {
	level      zapcore.Level
	logCheckFn func(*http.Request) bool
}

func newDefaultLoggingMiddlewareOptions() *loggingMiddlewareOptions {
	return &loggingMiddlewareOptions{
		level: zapcore.InfoLevel,
	}
}

type LoggingMiddlewareOption func(*loggingMiddlewareOptions)

// LogLevel sets the [zapcore.Level] at which non-errors requests are logged.
func LogLevel(level zapcore.Level) LoggingMiddlewareOption {
	return func(opts *loggingMiddlewareOptions) {
		opts.level = level
	}
}

// LogCheck sets a callback that will be called to verify if a particular request should be logged when it finished with a non-error status code.
// In other words:
// * if the status code is < 400 `fn` is called to verify if we should log the request or not
// * if the status code is >= 400 `fn` is not called and the request is logged anyway
//
// This is useful if you have a route for which you only care about errors, for example a status route:
//
//	LogCheck(func(r *http.Request) bool {
//	    // Log every request except /status
//	    return r.URL.Path != "/status"
//	})
func LogCheck(fn func(*http.Request) bool) LoggingMiddlewareOption {
	return func(opts *loggingMiddlewareOptions) {
		opts.logCheckFn = fn
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
			lw := internal.NewLoggingWriter(w)

			ru := *r.URL
			ru.Host = ""

			start := time.Now()
			next.ServeHTTP(lw, r)
			elapsed := time.Since(start)

			switch {
			case lw.StatusCode < 400:
				// Check if we should log non-errors. By default everything is logged.
				if opts.logCheckFn != nil && !opts.logCheckFn(r) {
					return
				}

				// Log using the provided level for non-errors

				logger.Log(opts.level, "request handled",
					zap.Stringer("url", &ru),
					zap.String("method", r.Method),
					zap.Int("status_code", lw.StatusCode),
					zap.Int("response_size", lw.Size),
					zap.Duration("elapsed", elapsed),
				)

			case lw.StatusCode >= 400 && lw.StatusCode < 500:
				// Log using the WARN level for 4xx errors

				logger.Warn("request handled",
					zap.Stringer("url", &ru),
					zap.String("method", r.Method),
					zap.Int("status_code", lw.StatusCode),
					zap.Int("response_size", lw.Size),
					zap.Duration("elapsed", elapsed),
				)

			case lw.StatusCode > 500:
				// Log using the ERROR level for 5xx errors and up

				logger.Error("request handled",
					zap.Stringer("url", &ru),
					zap.String("method", r.Method),
					zap.Int("status_code", lw.StatusCode),
					zap.Int("response_size", lw.Size),
					zap.Duration("elapsed", elapsed),
				)
			}
		})
	}
}
