package sse

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/buithean2010/mail-tracker/backend/internal/domain"
)

// Broker routes SSE events per user using plain Go channels.
type Broker struct {
	subs   map[uuid.UUID][]chan domain.SSEEvent
	mu     sync.RWMutex
}

func NewBroker() *Broker {
	return &Broker{
		subs: make(map[uuid.UUID][]chan domain.SSEEvent),
	}
}

func (b *Broker) Publish(userID uuid.UUID, event string, data any) {
	payload, _ := json.Marshal(data)
	ev := domain.SSEEvent{Event: event, Data: string(payload)}

	b.mu.RLock()
	channels := b.subs[userID]
	b.mu.RUnlock()

	for _, ch := range channels {
		select {
		case ch <- ev:
		default: // drop if receiver is slow
		}
	}
}

func (b *Broker) Subscribe(userID uuid.UUID) (<-chan domain.SSEEvent, func()) {
	ch := make(chan domain.SSEEvent, 16)

	b.mu.Lock()
	b.subs[userID] = append(b.subs[userID], ch)
	b.mu.Unlock()

	cancel := func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		chans := b.subs[userID]
		for i, c := range chans {
			if c == ch {
				b.subs[userID] = append(chans[:i], chans[i+1:]...)
				close(ch)
				break
			}
		}
		if len(b.subs[userID]) == 0 {
			delete(b.subs, userID)
		}
	}

	return ch, cancel
}

// Heartbeat sends a heartbeat event to all subscribers every 30 seconds to keep proxy connections alive.
func (b *Broker) Heartbeat(stop <-chan struct{}) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			b.mu.RLock()
			for _, channels := range b.subs {
				ev := domain.SSEEvent{Event: "heartbeat", Data: "{}"}
				for _, ch := range channels {
					select {
					case ch <- ev:
					default:
					}
				}
			}
			b.mu.RUnlock()
		case <-stop:
			return
		}
	}
}
