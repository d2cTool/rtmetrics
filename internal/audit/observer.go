package audit

import "net/http"

type Observer interface {
	Observe(Event)
}

type Subject struct {
	observers []Observer
}

func NewSubject() *Subject {
	return &Subject{}
}

func (s *Subject) Subscribe(o Observer) {
	if s == nil || o == nil {
		return
	}
	s.observers = append(s.observers, o)
}

func (s *Subject) Notify(event Event) {
	if s == nil {
		return
	}
	for _, o := range s.observers {
		o.Observe(event)
	}
}

func (s *Subject) NotifyRequest(r *http.Request, names []string) {
	if s == nil || len(names) == 0 {
		return
	}
	s.Notify(NewEvent(r, names))
}
