package bun

import (
	"net/http"
	"time"

	"github.com/uptrace/bunrouter"
	"github.com/vrischmann/hutil/v4/internal"
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
func NewLoggingMiddleware(logger *zap.Logger, opt ...LoggingMiddlewareOption) bunrouter.MiddlewareFunc {
	opts := newDefaultLoggingMiddlewareOptions()
	for _, o := range opt {
		o(opts)
	}

	return func(next bunrouter.HandlerFunc) bunrouter.HandlerFunc {
		return func(w http.ResponseWriter, r bunrouter.Request) error {
			lw := internal.NewLoggingWriter(w)

			ru := *r.URL
			ru.Host = ""

			start := time.Now()
			next.ServeHTTP(lw, r.Request)
			elapsed := time.Since(start)

			switch {
			case opts.onlyErrors && lw.StatusCode < 400:
				// No logging for non-errors
				return nil

			case !opts.onlyErrors && lw.StatusCode < 400:
				// Log using the provided level for non-errors

				logger.Log(opts.level, "request handled",
					zap.Stringer("url", &ru),
					zap.Int("status_code", lw.StatusCode),
					zap.Int("response_size", lw.Size),
					zap.Duration("elapsed", elapsed),
				)

			case lw.StatusCode >= 400 && lw.StatusCode < 500:
				// Log using the WARN level for 4xx errors

				logger.Warn("request handled",
					zap.Stringer("url", &ru),
					zap.Int("status_code", lw.StatusCode),
					zap.Int("response_size", lw.Size),
					zap.Duration("elapsed", elapsed),
				)

			case lw.StatusCode > 500:
				// Log using the ERROR level for 5xx errors and up

				logger.Error("request handled",
					zap.Stringer("url", &ru),
					zap.Int("status_code", lw.StatusCode),
					zap.Int("response_size", lw.Size),
					zap.Duration("elapsed", elapsed),
				)
			}

			return nil
		}
	}
}
