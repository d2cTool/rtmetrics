package audit

import (
	"encoding/json"
	"log/slog"
	"os"
	"sync"
)

type FileObserver struct {
	path string
	log  *slog.Logger
	mu   sync.Mutex
}

func NewFileObserver(path string, log *slog.Logger) *FileObserver {
	return &FileObserver{path: path, log: log}
}

func (o *FileObserver) Observe(event Event) {
	line, err := json.Marshal(event)
	if err != nil {
		o.log.Error("failed to encode audit event", slog.String("error", err.Error()))
		return
	}

	o.mu.Lock()
	defer o.mu.Unlock()

	f, err := os.OpenFile(o.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		o.log.Error("failed to open audit file", slog.String("path", o.path), slog.String("error", err.Error()))
		return
	}
	defer f.Close()

	if _, err := f.Write(append(line, '\n')); err != nil {
		o.log.Error("failed to write audit event", slog.String("path", o.path), slog.String("error", err.Error()))
	}
}
