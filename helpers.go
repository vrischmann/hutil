package hutil

import (
	"fmt"
	"io"
	"net/http"
)

// WriteError writes a 500 Internal Server Error response with the text of the err.Error().
func WriteError(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	io.WriteString(w, err.Error())
}

// WriteOK writes a 200 OK response with the text provided.
func WriteOK(w http.ResponseWriter, text string, args ...interface{}) {
	WriteText(w, http.StatusOK, text, args...)
}

// WriteBadRequest writes a 400 Bad Request response with the text provided.
func WriteBadRequest(w http.ResponseWriter, text string, args ...interface{}) {
	WriteText(w, http.StatusBadRequest, text, args...)
}

// WriteUnauthorized writes a 401 Unauthorized response with the text provided.
func WriteUnauthorized(w http.ResponseWriter, text string, args ...interface{}) {
	WriteText(w, http.StatusUnauthorized, text, args...)
}

// WriteServiceUnavailable writes a 503 Service Unavailable response with the text provided.
func WriteServiceUnavailable(w http.ResponseWriter, text string, args ...interface{}) {
	WriteText(w, http.StatusServiceUnavailable, text, args...)
}

// WriteText writes the provided status code and the provided text to a ResponseWriter.
func WriteText(w http.ResponseWriter, code int, text string, args ...interface{}) {
	w.WriteHeader(code)
	fmt.Fprintf(w, text, args...)
}
