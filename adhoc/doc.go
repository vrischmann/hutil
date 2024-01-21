// Package hutil contains helper functions and types to write HTTP handlers with Go.
//
// It is opiononated and minimalist:
// * `ShiftPath` is used to build routing
// * `Chain` is used to build a chain of middlewares
// * `NewLoggingMiddleware` is used to create a middleware which logs the requests. It is compatible with `Chain`
package hutil
