package handler_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/d2cTool/rtmetrics/internal/handler/get"
	"github.com/d2cTool/rtmetrics/internal/handler/html"
	"github.com/d2cTool/rtmetrics/internal/handler/ping"
	"github.com/d2cTool/rtmetrics/internal/handler/post"
	"github.com/d2cTool/rtmetrics/internal/handler/update"
	"github.com/d2cTool/rtmetrics/internal/handler/updates"
	"github.com/d2cTool/rtmetrics/internal/handler/value"
	"github.com/d2cTool/rtmetrics/internal/service"
	"github.com/d2cTool/rtmetrics/internal/storage"
	"github.com/go-chi/chi/v5"
)

func exampleServer() http.Handler {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := service.New(storage.New())
	r := chi.NewRouter()
	r.Post("/update", update.New(log, svc))
	r.Post("/update/", update.New(log, svc))
	r.Post("/updates/", updates.New(log, svc))
	r.Post("/update/{mtype}/{name}/{value}", post.New(log, svc))
	r.Post("/value", value.New(log, svc))
	r.Get("/value/{mtype}/{name}", get.New(log, svc))
	r.Get("/", html.New(log, svc))
	r.Get("/ping", ping.New(log, okPinger{}))
	return r
}

type okPinger struct{}

func (okPinger) PingContext(context.Context) error { return nil }

func do(h http.Handler, method, path, contentType, body string) *httptest.ResponseRecorder {
	var rdr io.Reader
	if body != "" {
		rdr = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, rdr)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func printResp(rec *httptest.ResponseRecorder) {
	fmt.Println(rec.Code)
	fmt.Print(strings.TrimRight(rec.Body.String(), "\n"))
	if rec.Body.Len() > 0 {
		fmt.Println()
	}
}

// Example_updateJSON — POST /update: одна gauge в JSON.
func Example_updateJSON() {
	srv := exampleServer()
	printResp(do(srv, http.MethodPost, "/update", "application/json",
		`{"id":"Alloc","type":"gauge","value":123.4}`))
	// Output:
	// 200
	// {"id":"Alloc","type":"gauge","value":123.4}
}

// Example_updatePath — POST /update/{mtype}/{name}/{value}.
func Example_updatePath() {
	srv := exampleServer()
	printResp(do(srv, http.MethodPost, "/update/counter/PollCount/5", "", ""))
	printResp(do(srv, http.MethodPost, "/update/counter/PollCount/3", "", ""))
	// Output:
	// 200
	// 5
	// 200
	// 8
}

// Example_updatesBatch — POST /updates/: пачка метрик.
func Example_updatesBatch() {
	srv := exampleServer()
	printResp(do(srv, http.MethodPost, "/updates/", "application/json",
		`[{"id":"Alloc","type":"gauge","value":10},{"id":"PollCount","type":"counter","delta":2}]`))
	printResp(do(srv, http.MethodGet, "/value/gauge/Alloc", "", ""))
	printResp(do(srv, http.MethodGet, "/value/counter/PollCount", "", ""))
	// Output:
	// 200
	// 200
	// 10
	// 200
	// 2
}

// Example_valueJSON — POST /value: чтение метрики JSON-ом.
func Example_valueJSON() {
	srv := exampleServer()
	do(srv, http.MethodPost, "/update", "application/json",
		`{"id":"Alloc","type":"gauge","value":1.5}`)
	printResp(do(srv, http.MethodPost, "/value", "application/json",
		`{"id":"Alloc","type":"gauge"}`))
	// Output:
	// 200
	// {"id":"Alloc","type":"gauge","value":1.5}
}

// Example_valuePath — GET /value/{mtype}/{name}.
func Example_valuePath() {
	srv := exampleServer()
	do(srv, http.MethodPost, "/update/gauge/RandomValue/0.25", "", "")
	printResp(do(srv, http.MethodGet, "/value/gauge/RandomValue", "", ""))
	// Output:
	// 200
	// 0.25
}

// Example_index — GET /: HTML-дашборд после записи метрики.
func Example_index() {
	srv := exampleServer()
	do(srv, http.MethodPost, "/update/gauge/Alloc/42", "", "")
	rec := do(srv, http.MethodGet, "/", "", "")
	fmt.Println(rec.Code)
	fmt.Println(rec.Header().Get("Content-Type"))
	fmt.Println(strings.Contains(rec.Body.String(), "Alloc"))
	// Output:
	// 200
	// text/html; charset=utf-8
	// true
}

// Example_ping — GET /ping при доступной БД.
func Example_ping() {
	srv := exampleServer()
	printResp(do(srv, http.MethodGet, "/ping", "", ""))
	// Output:
	// 200
}
