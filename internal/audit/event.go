package audit

import (
	"net"
	"net/http"
	"time"
)

// Event — запись аудита успешно принятых метрик.
type Event struct {
	TS        int64    `json:"ts"`
	Metrics   []string `json:"metrics"`
	IPAddress string   `json:"ip_address"`
}

// NewEvent собирает событие из имён метрик и входящего запроса.
func NewEvent(r *http.Request, names []string) Event {
	return Event{
		TS:        time.Now().Unix(),
		Metrics:   names,
		IPAddress: clientIP(r),
	}
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
