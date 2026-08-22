// Package decrypt расшифровывает тела запросов, зашифрованных агентом (RSA+AES).
package decrypt

import (
	"bytes"
	"crypto/rsa"
	"io"
	"log/slog"
	"net/http"

	"github.com/d2cTool/rtmetrics/internal/rsaenc"
)

// New возвращает middleware расшифровки. Подключать только при ненулевом priv.
// Запросы без заголовка X-Encrypted проходят без изменений.
func New(log *slog.Logger, priv *rsa.PrivateKey) func(next http.Handler) http.Handler {
	log = log.With(slog.String("component", "middleware/decrypt"))
	log.Info("decrypt middleware enabled")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get(rsaenc.Header) == "" {
				next.ServeHTTP(w, r)
				return
			}

			body, err := io.ReadAll(r.Body)
			_ = r.Body.Close()
			if err != nil {
				log.Debug("failed to read encrypted body", slog.String("error", err.Error()))
				http.Error(w, "failed to read request body", http.StatusBadRequest)
				return
			}

			plain, err := rsaenc.Decrypt(priv, body)
			if err != nil {
				log.Debug("failed to decrypt body", slog.String("error", err.Error()))
				http.Error(w, "failed to decrypt request body", http.StatusBadRequest)
				return
			}

			r.Body = io.NopCloser(bytes.NewReader(plain))
			r.ContentLength = int64(len(plain))
			r.Header.Del("Content-Length")
			next.ServeHTTP(w, r)
		})
	}
}
