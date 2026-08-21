package audit

import (
	"io"
	"net/http"
	"slices"
	"sync"
)

const eventBufferSize = 64

// Observer получает событие аудита. Ошибки доставки обрабатывает сам.
type Observer interface {
	Observe(Event)
}

// Subject хранит наблюдателей и рассылает им Event через фоновый воркер.
type Subject struct {
	mu        sync.RWMutex
	observers []Observer
	events    chan Event
	closed    bool
	wg        sync.WaitGroup
}

// NewSubject создаёт субъект и запускает воркер. Пока никто не подписан, Notify — no-op.
func NewSubject() *Subject {
	s := &Subject{
		events: make(chan Event, eventBufferSize),
	}
	s.wg.Add(1)
	go s.dispatch()
	return s
}

// Subscribe добавляет наблюдателя.
func (s *Subject) Subscribe(o Observer) {
	if s == nil || o == nil {
		return
	}
	s.mu.Lock()
	s.observers = append(s.observers, o)
	s.mu.Unlock()
}

// Notify ставит событие в очередь и сразу возвращается: Observe
// (в том числе HTTP с таймаутом 3s) не держит хендлер.
// Если очередь полна, событие отбрасывается — аудит best-effort.
func (s *Subject) Notify(event Event) {
	if s == nil {
		return
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closed {
		return
	}
	select {
	case s.events <- event:
	default:
	}
}

// NotifyRequest формирует событие из запроса и рассылает его.
func (s *Subject) NotifyRequest(r *http.Request, names []string) {
	if s == nil || len(names) == 0 {
		return
	}
	s.Notify(NewEvent(r, names))
}

// Close останавливает воркер и ждёт уже поставленные события. Повторный вызов безопасен.
func (s *Subject) Close() {
	if s == nil {
		return
	}
	s.mu.Lock()
	if !s.closed {
		s.closed = true
		close(s.events)
	}
	s.mu.Unlock()
	s.wg.Wait()

	s.mu.Lock()
	observers := slices.Clone(s.observers)
	s.mu.Unlock()
	for _, o := range observers {
		if c, ok := o.(io.Closer); ok {
			_ = c.Close()
		}
	}
}

func (s *Subject) dispatch() {
	defer s.wg.Done()
	for event := range s.events {
		s.mu.RLock()
		observers := slices.Clone(s.observers)
		s.mu.RUnlock()
		for _, o := range observers {
			o.Observe(event)
		}
	}
}
