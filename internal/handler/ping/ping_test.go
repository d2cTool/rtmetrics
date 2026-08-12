package ping

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/stretchr/testify/require"
)

type mockPinger struct {
	err error
}

func (m *mockPinger) PingContext(ctx context.Context) error {
	return m.err
}

func doRequest(t *testing.T, db Pinger) *httptest.ResponseRecorder {
	t.Helper()

	handler := New(slog.Default(), db)
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Get("/ping", handler)
	r.ServeHTTP(w, req)

	return w
}

func TestNew_Success(t *testing.T) {
	w := doRequest(t, &mockPinger{})
	require.Equal(t, http.StatusOK, w.Code)
}

func TestNew_PingError_InternalServerError(t *testing.T) {
	w := doRequest(t, &mockPinger{err: errors.New("connection refused")})
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestNew_NilDatabase_InternalServerError(t *testing.T) {
	w := doRequest(t, nil)
	require.Equal(t, http.StatusInternalServerError, w.Code)
}
