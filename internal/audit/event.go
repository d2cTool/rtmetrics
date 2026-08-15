package audit

import (
	"net"
	"net/http"
	"time"
)

type Event struct {
	TS        int64    `json:"ts"`
	Metrics   []string `json:"metrics"`
	IPAddress string   `json:"ip_address"`
}

func NewEvent(r *http.Request, names []string) Event {
	return Event{
		TS:        time.Now().Unix(),
		Metrics:   names,
		IPAddress: ClientIP(r),
	}
}

func ClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
