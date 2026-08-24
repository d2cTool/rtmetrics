package decrypt

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/d2cTool/rtmetrics/internal/rsaenc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecryptsEncryptedBody(t *testing.T) {
	t.Parallel()

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	plain := []byte(`{"id":"Alloc","type":"gauge","value":1}`)
	enc, err := rsaenc.Encrypt(&priv.PublicKey, plain)
	require.NoError(t, err)

	var got []byte
	h := New(slog.Default(), priv)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		got, err = io.ReadAll(r.Body)
		require.NoError(t, err)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewReader(enc))
	req.Header.Set(rsaenc.Header, rsaenc.HeaderValue)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, plain, got)
}

func TestPassesPlaintextWithoutHeader(t *testing.T) {
	t.Parallel()

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	plain := []byte(`{"ok":true}`)
	var got []byte
	h := New(slog.Default(), priv)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		got, err = io.ReadAll(r.Body)
		require.NoError(t, err)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(plain))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, plain, got)
}

func TestRejectsInvalidCiphertext(t *testing.T) {
	t.Parallel()

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	h := New(slog.Default(), priv)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("handler must not be called")
	}))

	req := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewReader([]byte("garbage-ciphertext-not-rsa")))
	req.Header.Set(rsaenc.Header, rsaenc.HeaderValue)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
