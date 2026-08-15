package audit

import "net/http"

// Observer получает событие аудита. Ошибки доставки обрабатывает сам.
type Observer interface {
	Observe(Event)
}

// Subject хранит наблюдателей и рассылает им Event.
type Subject struct {
	observers []Observer
}

// NewSubject создаёт пустой субъект. Пока никто не подписан, Notify — no-op.
func NewSubject() *Subject {
	return &Subject{}
}

// Subscribe добавляет наблюдателя.
func (s *Subject) Subscribe(o Observer) {
	if s == nil || o == nil {
		return
	}
	s.observers = append(s.observers, o)
}

// Notify отправляет событие всем наблюдателям. Нил-субъект безопасен.
func (s *Subject) Notify(event Event) {
	if s == nil {
		return
	}
	for _, o := range s.observers {
		o.Observe(event)
	}
}

// NotifyRequest формирует событие из запроса и рассылает его.
func (s *Subject) NotifyRequest(r *http.Request, names []string) {
	if s == nil || len(names) == 0 {
		return
	}
	s.Notify(NewEvent(r, names))
}
