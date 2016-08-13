package hutil

import (
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type loggingHandlerTestCase struct {
	path      string
	expected  string
	exact     bool
	sleepTime time.Duration
	options   *LoggingOptions
}

// TODO(vincent): fix the test cases with the headers; it's a map so the tests fail randomly.
var loggingHandlerTestCases = []loggingHandlerTestCase{
	{
		"/",
		"200 / l:6\n",
		true,
		0,
		nil,
	},
	// {
	// 	"/",
	// 	"200 / headers:map[User-Agent:[Go-http-client/1.1] Accept-Encoding:[gzip]] l:6\n",
	// 	true,
	// 	0,
	// 	&LoggingOptions{
	// 		WithHeaders: true,
	// 	},
	// },
	{
		"/",
		"200 / l:6 e:15",
		false,
		time.Millisecond * 150,
		&LoggingOptions{WithExecutionTime: true},
	},
	// {
	// 	"/",
	// 	"200 / headers:map[User-Agent:[Go-http-client/1.1] Accept-Encoding:[gzip]] l:6 e:15",
	// 	false,
	// 	time.Millisecond * 150,
	// 	&LoggingOptions{
	// 		WithHeaders:       true,
	// 		WithExecutionTime: true,
	// 	},
	// },
}

func TestLoggingHandler(t *testing.T) {
	for _, tc := range loggingHandlerTestCases {
		var buf bytes.Buffer
		log.SetFlags(0)
		log.SetOutput(&buf)

		var c Chain
		c.Use(NewLoggingMiddleware(tc.options))

		fh := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			io.WriteString(w, "foobar")
			time.Sleep(tc.sleepTime)
		})

		ts := httptest.NewServer(c.Handler(fh))
		defer ts.Close()

		resp, err := http.Get(ts.URL + tc.path)
		require.Nil(t, err)

		data, err := ioutil.ReadAll(resp.Body)
		require.Nil(t, err)
		resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.Equal(t, "foobar", string(data))

		t.Log(buf.String())
		if tc.exact {
			require.Equal(t, tc.expected, buf.String())
		} else {
			require.True(t, strings.HasPrefix(buf.String(), tc.expected))
		}
	}
}
