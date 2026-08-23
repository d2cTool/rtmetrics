package agent

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

func TestClientSetsRealIPHeader(t *testing.T) {
	var got string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get(realip.Header)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL, "", slog.Default())
	require.NoError(t, client.SendGauge(t.Context(), "Alloc", 1))

	ip := net.ParseIP(got)
	require.NotNil(t, ip, "X-Real-IP должен содержать IP хоста, получено %q", got)
	assert.NotNil(t, ip.To4())
}
