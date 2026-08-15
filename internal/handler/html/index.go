package html

import (
	"html/template"
	"log/slog"
	"net/http"

	"github.com/d2cTool/rtmetrics/internal/service"
	"github.com/go-chi/chi/v5/middleware"
)

type PageData struct {
	Counters map[string]int64
	Gauges   map[string]float64
}

func New(log *slog.Logger, svc service.MetricsService) http.HandlerFunc {
	tmpl := `<!DOCTYPE html>
<html>
<head>
    <title>Metrics Dashboard</title>
    <meta charset="utf-8">
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; background-color: #f5f5f5; }
        .container { max-width: 1200px; margin: 0 auto; background: white; padding: 20px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        h1 { color: #333; }
        h2 { color: #666; margin-top: 30px; }
        table { border-collapse: collapse; width: 100%; margin-top: 10px; }
        th, td { border: 1px solid #ddd; padding: 12px; text-align: left; }
        th { background-color: #4CAF50; color: white; }
        tr:nth-child(even) { background-color: #f9f9f9; }
        .empty { color: #999; font-style: italic; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Metrics Dashboard</h1>
        
        <h2>Counters</h2>
        {{if .Counters}}
        <table>
            <tr><th>Name</th><th>Value</th></tr>
            {{range $name, $value := .Counters}}
            <tr><td>{{$name}}</td><td>{{$value}}</td></tr>
            {{end}}
        </table>
        {{else}}
        <p class="empty">No counters available</p>
        {{end}}
        
        <h2>Gauges</h2>
        {{if .Gauges}}
        <table>
            <tr><th>Name</th><th>Value</th></tr>
            {{range $name, $value := .Gauges}}
            <tr><td>{{$name}}</td><td>{{$value}}</td></tr>
            {{end}}
        </table>
        {{else}}
        <p class="empty">No gauges available</p>
        {{end}}
    </div>
</body>
</html>`

	t := template.Must(template.New("metrics").Parse(tmpl))

	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handler.html.new"
		reqLog := log
		if log.Enabled(r.Context(), slog.LevelDebug) {
			reqLog = log.With(
				slog.String("op", op),
				slog.String("request_id", middleware.GetReqID(r.Context())),
			)
		}

		counters, err := svc.GetAllCounters(r.Context())
		if err != nil {
			reqLog.Error("failed to get counters", slog.String("error", err.Error()))
			counters = make(map[string]int64)
		}

		gauges, err := svc.GetAllGauges(r.Context())
		if err != nil {
			reqLog.Error("failed to get gauges", slog.String("error", err.Error()))
			gauges = make(map[string]float64)
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := t.Execute(w, PageData{Counters: counters, Gauges: gauges}); err != nil {
			reqLog.Error("failed to execute template", slog.String("error", err.Error()))
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		reqLog.Debug("html page rendered")
	}
}
