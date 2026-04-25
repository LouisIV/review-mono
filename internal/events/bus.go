package events

import (
	"sync"
	"time"

	"review/internal/models"
)

type Bus struct {
	mu   sync.Mutex
	next int
	subs map[int]chan models.Event
}

func NewBus() *Bus {
	return &Bus{subs: map[int]chan models.Event{}}
}

func (b *Bus) Subscribe() (int, <-chan models.Event) {
	b.mu.Lock()
	defer b.mu.Unlock()

	id := b.next
	b.next++
	ch := make(chan models.Event, 32)
	b.subs[id] = ch

	return id, ch
}

func (b *Bus) Unsubscribe(id int) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if ch, ok := b.subs[id]; ok {
		delete(b.subs, id)
		close(ch)
	}
}

func (b *Bus) Publish(event models.Event) {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	for _, ch := range b.subs {
		select {
		case ch <- event:
		default:
		}
	}
}
