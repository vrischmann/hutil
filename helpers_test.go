package hutil

import (
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWriteError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		WriteError(w, errors.New("foobar"))
	}))
	defer ts.Close()

	resp, err := http.Get(ts.URL)
	require.Nil(t, err)

	data, err := ioutil.ReadAll(resp.Body)
	require.Nil(t, err)
	resp.Body.Close()

	require.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	require.Equal(t, "foobar", string(data))
}

func TestWriteOK(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		WriteOK(w, "foobar")
	}))
	defer ts.Close()

	resp, err := http.Get(ts.URL)
	require.Nil(t, err)

	data, err := ioutil.ReadAll(resp.Body)
	require.Nil(t, err)
	resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "foobar", string(data))
}

func TestWriteBadRequest(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		WriteBadRequest(w, "foobar")
	}))
	defer ts.Close()

	resp, err := http.Get(ts.URL)
	require.Nil(t, err)

	data, err := ioutil.ReadAll(resp.Body)
	require.Nil(t, err)
	resp.Body.Close()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	require.Equal(t, "foobar", string(data))
}

func TestWriteServiceUnavailable(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		WriteServiceUnavailable(w, "foobar")
	}))
	defer ts.Close()

	resp, err := http.Get(ts.URL)
	require.Nil(t, err)

	data, err := ioutil.ReadAll(resp.Body)
	require.Nil(t, err)
	resp.Body.Close()

	require.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
	require.Equal(t, "foobar", string(data))
}
