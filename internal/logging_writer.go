package internal

import (
	"bufio"
	"errors"
	"net"
	"net/http"
)

type LoggingWriter struct {
	underlying http.ResponseWriter
	StatusCode int
	Size       int
}

func NewLoggingWriter(underlying http.ResponseWriter) *LoggingWriter {
	return &LoggingWriter{underlying: underlying}
}

func (w *LoggingWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	v, ok := w.underlying.(http.Hijacker)
	if !ok {
		panic(errors.New("response writer does not implement Hijacker"))
	}

	return v.Hijack()
}

func (w *LoggingWriter) Header() http.Header {
	return w.underlying.Header()
}

func (w *LoggingWriter) Write(b []byte) (n int, err error) {
	w.Size += len(b)
	return w.underlying.Write(b)
}

func (w *LoggingWriter) WriteHeader(code int) {
	w.StatusCode = code
	w.underlying.WriteHeader(code)
}

var _ http.ResponseWriter = (*LoggingWriter)(nil)
