package sign

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/d2cTool/rtmetrics/internal/hash"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testKey = "secret"

// echoHandler отдаёт тело запроса обратно, чтобы проверить, что middleware
// не мешает хендлеру читать его.
func echoHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_, _ = w.Write(body)
	})
}

func doRequest(t *testing.T, key, body, signature string) *http.Response {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/updates/", strings.NewReader(body))
	if signature != "" {
		req.Header.Set(hash.Header, signature)
	}

	rec := httptest.NewRecorder()
	New(slog.Default(), key)(echoHandler()).ServeHTTP(rec, req)
	return rec.Result()
}

func TestValidSignaturePassesAndResponseIsSigned(t *testing.T) {
	t.Parallel()

	const body = `[{"id":"Alloc","type":"gauge","value":1.5}]`

	resp := doRequest(t, testKey, body, hash.Sign([]byte(body), testKey))
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	got, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, body, string(got), "хендлер должен видеть тело запроса")
	assert.Equal(t, hash.Sign(got, testKey), resp.Header.Get(hash.Header), "ответ должен быть подписан")
}

func TestInvalidSignatureIsRejected(t *testing.T) {
	t.Parallel()

	const body = `[{"id":"Alloc","type":"gauge","value":1.5}]`

	resp := doRequest(t, testKey, body, hash.Sign([]byte("другое тело"), testKey))
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestRequestWithoutSignatureIsAccepted(t *testing.T) {
	t.Parallel()

	resp := doRequest(t, testKey, `{"id":"Alloc","type":"gauge","value":1.5}`, "")
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.NotEmpty(t, resp.Header.Get(hash.Header), "ответ подписывается и без подписи в запросе")
}

func TestHandlerStatusIsPreserved(t *testing.T) {
	t.Parallel()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	})

	req := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader("{}"))
	rec := httptest.NewRecorder()
	New(slog.Default(), testKey)(handler).ServeHTTP(rec, req)

	resp := rec.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	assert.NotEmpty(t, resp.Header.Get(hash.Header))
}
