// Package sign проверяет подпись входящих запросов и подписывает ответы
// заголовком HashSHA256.
package sign

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"

	"github.com/d2cTool/rtmetrics/internal/hash"
)

// New возвращает middleware подписи. При пустом key запросы проходят без
// изменений. Middleware должен стоять после распаковки gzip, чтобы хеш
// считался от тех же байтов, что подписывал агент.
func New(log *slog.Logger, key string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if key == "" {
			return next
		}

		log = log.With(slog.String("component", "middleware/sign"))
		log.Info("sign middleware enabled")

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				log.Error("failed to read request body", slog.String("error", err.Error()))
				http.Error(w, "failed to read request body", http.StatusBadRequest)
				return
			}
			_ = r.Body.Close()
			r.Body = io.NopCloser(bytes.NewReader(body))

			// Запросы без подписи пропускаем: заголовок обязателен только для
			// клиентов, которым ключ известен.
			if signature := r.Header.Get(hash.Header); signature != "" {
				if !hash.Valid(body, key, signature) {
					log.Warn("request signature mismatch", slog.String("uri", r.RequestURI))
					http.Error(w, "invalid signature", http.StatusBadRequest)
					return
				}
			}

			sw := &signingResponseWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(sw, r)

			sw.flush(key)
		})
	}
}

// signingResponseWriter копит ответ целиком: подпись можно посчитать только
// когда известно всё тело, а заголовки уходят раньше первой записи.
type signingResponseWriter struct {
	http.ResponseWriter
	body        bytes.Buffer
	status      int
	wroteHeader bool
}

func (s *signingResponseWriter) WriteHeader(status int) {
	if s.wroteHeader {
		return
	}
	s.status = status
	s.wroteHeader = true
}

func (s *signingResponseWriter) Write(data []byte) (int, error) {
	if !s.wroteHeader {
		s.WriteHeader(http.StatusOK)
	}
	return s.body.Write(data)
}

func (s *signingResponseWriter) flush(key string) {
	s.Header().Set(hash.Header, hash.Sign(s.body.Bytes(), key))
	s.ResponseWriter.WriteHeader(s.status)
	if s.body.Len() > 0 {
		_, _ = s.ResponseWriter.Write(s.body.Bytes())
	}
}
