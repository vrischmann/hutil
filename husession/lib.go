package husession

import (
	"context"
	"net/http"

	"github.com/vrischmann/hutil/v3"
	"go.uber.org/zap"
)

type SessionID interface {
	comparable
}

type IDExtractor[ID SessionID] func(string) (ID, error)

type Store[ID SessionID, Value any] interface {
	Fetch(ctx context.Context, id ID) (*Value, error)
}

type middleware[ID SessionID, Value any] struct {
	logger    *zap.Logger
	store     Store[ID, Value]
	extractor IDExtractor[ID]
}

func NewMiddleware[ID SessionID, Value any](logger *zap.Logger, store Store[ID, Value], extractor IDExtractor[ID]) hutil.Middleware {
	mw := &middleware[ID, Value]{
		logger:    logger,
		store:     store,
		extractor: extractor,
	}

	return func(next http.Handler) http.Handler {
		return mw.makeHandler(next)
	}
}

var sessionContextKey struct{}

// FromContext extracts a session from the context. If there's no session it returns nil.
func FromContext[Value any](ctx context.Context) *Value {
	value := ctx.Value(sessionContextKey)
	if value == nil {
		return nil
	}

	return value.(*Value)
}

func (m *middleware[ID, Session]) makeHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// 1. Try to extract the session ID

		// TODO(vincent): extract the real ID
		id, err := m.extractor("foobar")
		if err != nil {
			m.logger.Error("unable to extract session id", zap.Error(err))
			next.ServeHTTP(w, req)
			return
		}

		// 2. We have the ID, fetch the session

		session, err := m.store.Fetch(req.Context(), id)
		if err != nil {
			m.logger.Error("unable to fetch session", zap.Error(err))
			next.ServeHTTP(w, req)
			return
		}

		// 3. If we have a session set it in the request context
		if session != nil {
			m.logger.Debug("got session", zap.Any("session", session))

			ctx := context.WithValue(req.Context(), sessionContextKey, session)
			req = req.WithContext(ctx)
		}

		// 4. Finally, whatever happens call the next handler
		next.ServeHTTP(w, req)
	})
}
