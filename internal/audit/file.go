package audit

import (
	"encoding/json"
	"log/slog"
	"os"
	"sync"
)

// FileObserver дописывает событие в конец файла одной JSON-строкой.
type FileObserver struct {
	file *os.File
	log  *slog.Logger
	mu   sync.Mutex
}

func NewFileObserver(path string, log *slog.Logger) (*FileObserver, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	return &FileObserver{file: f, log: log}, nil
}

func (o *FileObserver) Observe(event Event) {
	line, err := json.Marshal(event)
	if err != nil {
		o.log.Error("failed to encode audit event", slog.String("error", err.Error()))
		return
	}

	o.mu.Lock()
	defer o.mu.Unlock()
	if o.file == nil {
		return
	}

	if _, err := o.file.Write(append(line, '\n')); err != nil {
		o.log.Error("failed to write audit event",
			slog.String("path", o.file.Name()),
			slog.String("error", err.Error()),
		)
	}
}

// Close закрывает файл. Повторный вызов безопасен.
func (o *FileObserver) Close() error {
	if o == nil {
		return nil
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.file == nil {
		return nil
	}
	err := o.file.Close()
	o.file = nil
	return err
}
