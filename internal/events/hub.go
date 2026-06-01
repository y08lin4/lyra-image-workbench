package events

import "sync"

type Event struct {
	Event   string `json:"event"`
	Code    string `json:"code"`
	English string `json:"english"`
	Chinese string `json:"chinese"`
	Data    any    `json:"data,omitempty"`
}

type Hub struct {
	mu   sync.Mutex
	subs map[string]map[chan Event]struct{}
}

func NewHub() *Hub {
	return &Hub{subs: make(map[string]map[chan Event]struct{})}
}

func (h *Hub) Subscribe(jobID string) (<-chan Event, func()) {
	ch := make(chan Event, 16)
	h.mu.Lock()
	if h.subs[jobID] == nil {
		h.subs[jobID] = make(map[chan Event]struct{})
	}
	h.subs[jobID][ch] = struct{}{}
	h.mu.Unlock()
	cancel := func() {
		h.mu.Lock()
		if h.subs[jobID] != nil {
			delete(h.subs[jobID], ch)
			if len(h.subs[jobID]) == 0 {
				delete(h.subs, jobID)
			}
		}
		h.mu.Unlock()
		close(ch)
	}
	return ch, cancel
}

func (h *Hub) Publish(jobID string, event Event) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subs[jobID] {
		select {
		case ch <- event:
		default:
		}
	}
}
