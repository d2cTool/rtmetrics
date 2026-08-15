package audit

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

const httpTimeout = 3 * time.Second

type HTTPObserver struct {
	url    string
	client *http.Client
	log    *slog.Logger
}

func NewHTTPObserver(url string, log *slog.Logger) *HTTPObserver {
	return &HTTPObserver{
		url: url,
		client: &http.Client{
			Timeout: httpTimeout,
		},
		log: log,
	}
}

func (o *HTTPObserver) Observe(event Event) {
	body, err := json.Marshal(event)
	if err != nil {
		o.log.Error("failed to encode audit event", slog.String("error", err.Error()))
		return
	}

	resp, err := o.client.Post(o.url, "application/json", bytes.NewReader(body))
	if err != nil {
		o.log.Error("failed to post audit event", slog.String("url", o.url), slog.String("error", err.Error()))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		o.log.Error("audit receiver returned error",
			slog.String("url", o.url),
			slog.Int("status", resp.StatusCode),
		)
	}
}
