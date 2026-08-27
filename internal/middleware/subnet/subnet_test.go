package subnet

import (
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/d2cTool/rtmetrics/internal/realip"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAllowsTrustedIP(t *testing.T) {
	_, network, err := net.ParseCIDR("192.168.0.0/16")
	require.NoError(t, err)

	handler := New(slog.Default(), network)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/update", nil)
	req.Header.Set(realip.Header, "192.168.10.5")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestNewForbidsForeignIP(t *testing.T) {
	_, network, err := net.ParseCIDR("192.168.0.0/16")
	require.NoError(t, err)

	handler := New(slog.Default(), network)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name string
		ip   string
	}{
		{name: "чужая подсеть", ip: "10.0.0.1"},
		{name: "пустой заголовок", ip: ""},
		{name: "не IP", ip: "localhost"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/updates/", nil)
			if tt.ip != "" {
				req.Header.Set(realip.Header, tt.ip)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			assert.Equal(t, http.StatusForbidden, rec.Code)
		})
	}
}
