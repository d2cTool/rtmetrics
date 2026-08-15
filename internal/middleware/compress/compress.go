package compress

import (
	"compress/gzip"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
)

var gzipWriterPool = sync.Pool{
	New: func() any {
		w, err := gzip.NewWriterLevel(io.Discard, gzip.DefaultCompression)
		if err != nil {
			return gzip.NewWriter(io.Discard)
		}
		return w
	},
}

var compressibleTypes = map[string]bool{
	"text/html":        true,
	"application/json": true,
}

func New(log *slog.Logger) func(next http.Handler) http.Handler {

	return func(next http.Handler) http.Handler {
		log = log.With(slog.String("component", "middleware/compress"))
		log.Info("compress middleware enabled")

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/debug/pprof") {
				next.ServeHTTP(w, r)
				return
			}

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

			gzw := &gzipResponseWriter{ResponseWriter: w}
			defer gzw.Close()

			next.ServeHTTP(gzw, r)
		})
	}
}

type gzipResponseWriter struct {
	http.ResponseWriter
	writer  *gzip.Writer
	written bool
}

func (g *gzipResponseWriter) WriteHeader(statusCode int) {
	contentType := g.Header().Get("Content-Type")
	if idx := strings.Index(contentType, ";"); idx != -1 {
		contentType = contentType[:idx]
	}
	contentType = strings.TrimSpace(contentType)
	shouldCompress := compressibleTypes[contentType]

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
			w := gzipWriterPool.Get().(*gzip.Writer)
			w.Reset(g.ResponseWriter)
			g.writer = w
		}
		return g.writer.Write(data)
	}

	return g.ResponseWriter.Write(data)
}

func (g *gzipResponseWriter) Close() {
	if g.writer != nil {
		_ = g.writer.Close()
		gzipWriterPool.Put(g.writer)
		g.writer = nil
	}
}
