package hutil

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func createMiddleware(buf *bytes.Buffer, s string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			buf.WriteString(s)
			next.ServeHTTP(w, req)
		})
	}
}

func TestChain(t *testing.T) {
	var (
		c   Chain
		buf bytes.Buffer
	)

	var (
		m1 = createMiddleware(&buf, "m1")
		m2 = createMiddleware(&buf, "m2")
		m3 = createMiddleware(&buf, "m3")
	)

	c.Use(m1)
	c.Use(m2)
	c.Use(m3)

	ts := httptest.NewServer(c.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "foobar")
	})))
	defer ts.Close()

	res, err := http.Get(ts.URL)
	require.Nil(t, err)

	data, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()
	require.Nil(t, err)
	require.Equal(t, "foobar", string(data))
	require.Equal(t, "m1m2m3", buf.String())
}
