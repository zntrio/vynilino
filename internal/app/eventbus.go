package app

import "sync"

// EventBus is a simple in-process pub/sub bus for subscription events,
// scoped per user ID so each subscriber only sees their own events.
type EventBus struct {
	mu          sync.RWMutex
	subscribers map[string][]chan RecordChangedEvent
}

// NewEventBus creates an EventBus.
func NewEventBus() *EventBus {
	return &EventBus{subscribers: make(map[string][]chan RecordChangedEvent)}
}

// Subscribe registers a channel for events belonging to userID.
// The caller must call the returned unsubscribe function when done.
func (b *EventBus) Subscribe(userID string) (<-chan RecordChangedEvent, func()) {
	ch := make(chan RecordChangedEvent, 16)
	b.mu.Lock()
	b.subscribers[userID] = append(b.subscribers[userID], ch)
	b.mu.Unlock()

	unsub := func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		subs := b.subscribers[userID]
		for i, s := range subs {
			if s == ch {
				b.subscribers[userID] = append(subs[:i], subs[i+1:]...)
				close(ch)
				break
			}
		}
	}
	return ch, unsub
}

// Publish sends an event to all subscribers for userID (non-blocking).
func (b *EventBus) Publish(userID string, event RecordChangedEvent) {
	b.mu.RLock()
	subs := b.subscribers[userID]
	b.mu.RUnlock()
	for _, ch := range subs {
		select {
		case ch <- event:
		default: // subscriber too slow; drop event
		}
	}
}
