package compress

import (
	"compress/gzip"
	"log/slog"
	"net/http"
	"strings"
)

func New(log *slog.Logger) func(next http.Handler) http.Handler {

	return func(next http.Handler) http.Handler {
		log = log.With(slog.String("component", "middleware/compress"))
		log.Info("compress middleware enabled")

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
				gz, err := gzip.NewReader(r.Body)
				if err != nil {
					log.Error("invalid gzip body")
					http.Error(w, "invalid gzip body", http.StatusBadRequest)
					return
				}
				defer gz.Close()
				r.Body = gz
			}

			acceptsGzip := false
			acceptEncoding := r.Header.Get("Accept-Encoding")
			for _, enc := range strings.Split(acceptEncoding, ",") {
				enc = strings.TrimSpace(enc)
				if strings.HasPrefix(enc, "gzip") {
					acceptsGzip = true
					break
				}
			}

			if !acceptsGzip {
				next.ServeHTTP(w, r)
				return
			}

			gzw := &gzipResponseWriter{
				ResponseWriter: w,
				whitelist: map[string]bool{
					"text/html":        true,
					"application/json": true,
				},
			}
			defer gzw.Close()

			next.ServeHTTP(gzw, r)
		})
	}
}

type gzipResponseWriter struct {
	http.ResponseWriter
	writer    *gzip.Writer
	whitelist map[string]bool
	written   bool
}

func (g *gzipResponseWriter) WriteHeader(statusCode int) {
	contentType := g.Header().Get("Content-Type")
	if idx := strings.Index(contentType, ";"); idx != -1 {
		contentType = contentType[:idx]
	}
	contentType = strings.TrimSpace(contentType)
	shouldCompress := g.whitelist[contentType]

	if shouldCompress {
		g.Header().Set("Content-Encoding", "gzip")
		g.Header().Del("Content-Length")
	}

	g.ResponseWriter.WriteHeader(statusCode)
}

func (g *gzipResponseWriter) Write(data []byte) (int, error) {
	if g.written {
		if g.writer != nil {
			return g.writer.Write(data)
		}
		return g.ResponseWriter.Write(data)
	}

	g.written = true

	if g.Header().Get("Content-Encoding") == "gzip" {
		if g.writer == nil {
			var err error
			g.writer, err = gzip.NewWriterLevel(g.ResponseWriter, gzip.DefaultCompression)
			if err != nil {
				return g.ResponseWriter.Write(data)
			}
		}
		return g.writer.Write(data)
	}

	return g.ResponseWriter.Write(data)
}

func (g *gzipResponseWriter) Close() {
	if g.writer != nil {
		g.writer.Close()
	}
}
