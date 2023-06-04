package husession

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrischmann/hutil/v3"
	"go.uber.org/zap/zaptest"
)

type testStore[ID SessionID, Value any] struct {
	data map[ID]*Value
}

type testSession struct {
	UserID   int
	UserName string
}

func newTestStore() Store[string, testSession] {
	return &testStore[string, testSession]{
		data: make(map[string]*testSession),
	}
}

func (s *testStore[ID, Value]) Fetch(ctx context.Context, id ID) (*Value, error) {
	return s.data[id], nil
}

func TestSessionMiddleware(t *testing.T) {
	logger := zaptest.NewLogger(t)

	//

	store := newTestStore()
	store.(*testStore[string, testSession]).data["foobar"] = &testSession{
		UserID:   20,
		UserName: "vincent",
	}

	extractor := func(value string) (string, error) { return value, nil }

	var s hutil.MiddlewareStack
	s.Use(NewMiddleware(logger, store, extractor))

	//

	fh := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		session := FromContext[testSession](req.Context())

		log.Printf("session: %+v", session)

		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "user id: %d ; user name: %s", session.UserID, session.UserName)
	})

	ts := httptest.NewServer(s.Handler(fh))
	defer ts.Close()

	resp, err := http.Get(ts.URL)
	require.NoError(t, err)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	resp.Body.Close()

	require.Equal(t, "user id: 20 ; user name: vincent", string(body))
}
