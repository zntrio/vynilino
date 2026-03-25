package app

import (
	"context"
	"testing"
	"time"
)

// TestEventBus_BufferFits256 verifies that 256 events fit in the subscriber
// buffer without any drops, confirming the buffer was increased from 16.
func TestEventBus_BufferFits256(t *testing.T) {
	bus := NewEventBus()
	_, unsub := bus.Subscribe("user1")
	defer unsub()

	ctx := context.Background()
	const n = 256

	before := bus.dropped.Load()
	for i := range n {
		bus.Publish(ctx, "user1", RecordChangedEvent{Type: "CREATED", RecordID: string(rune('a' + i%26))})
	}
	after := bus.dropped.Load()

	if after != before {
		t.Fatalf("expected zero drops for %d events (buffer=256), got %d drop(s)", n, after-before)
	}
}

// TestEventBus_BufferSaturation verifies that events published with an active
// draining subscriber are all received within a reasonable time.
func TestEventBus_BufferSaturation(t *testing.T) {
	bus := NewEventBus()
	ch, unsub := bus.Subscribe("user1")
	defer unsub()

	ctx := context.Background()
	const n = 256

	received := make(chan struct{}, n)
	go func() {
		for range ch {
			received <- struct{}{}
		}
	}()

	for i := range n {
		bus.Publish(ctx, "user1", RecordChangedEvent{Type: "CREATED", RecordID: string(rune('a' + i%26))})
	}

	deadline := time.After(2 * time.Second)
	got := 0
	for got < n {
		select {
		case <-received:
			got++
		case <-deadline:
			t.Fatalf("only received %d/%d events before deadline", got, n)
		}
	}
}

func TestEventBus_DropCounter(t *testing.T) {
	bus := NewEventBus()
	_, unsub := bus.Subscribe("user1")
	defer unsub()

	ctx := context.Background()

	// Publish more events than the buffer can hold without draining.
	for range 300 {
		bus.Publish(ctx, "user1", RecordChangedEvent{Type: "CREATED", RecordID: "x"})
	}

	if bus.dropped.Load() == 0 {
		t.Fatal("expected drop counter > 0 when buffer overflows")
	}
}

func TestEventBus_HasSubscribers(t *testing.T) {
	bus := NewEventBus()

	if bus.HasSubscribers("user1") {
		t.Fatal("expected no subscribers initially")
	}

	_, unsub := bus.Subscribe("user1")
	if !bus.HasSubscribers("user1") {
		t.Fatal("expected subscriber after Subscribe")
	}

	unsub()
	if bus.HasSubscribers("user1") {
		t.Fatal("expected no subscribers after unsubscribe")
	}
}
