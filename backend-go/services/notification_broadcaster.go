package services

import (
	"sync"
)

type NotificationPayload struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	RelatedID string `json:"related_id"`
	IsRead    bool   `json:"is_read"`
	CreatedAt string `json:"created_at"`
}

type Subscriber struct {
	Events chan NotificationPayload
	UserID string
}

type Broadcaster struct {
	mu          sync.RWMutex
	subscribers map[string]map[chan NotificationPayload]bool
}

var broadcasterInstance *Broadcaster
var broadcasterOnce sync.Once

func GetBroadcaster() *Broadcaster {
	broadcasterOnce.Do(func() {
		broadcasterInstance = &Broadcaster{
			subscribers: map[string]map[chan NotificationPayload]bool{},
		}
	})
	return broadcasterInstance
}

func (b *Broadcaster) Subscribe(userID string) chan NotificationPayload {
	ch := make(chan NotificationPayload, 128)
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.subscribers[userID] == nil {
		b.subscribers[userID] = map[chan NotificationPayload]bool{}
	}
	b.subscribers[userID][ch] = true
	return ch
}

func (b *Broadcaster) Unsubscribe(userID string, ch chan NotificationPayload) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if m, ok := b.subscribers[userID]; ok {
		delete(m, ch)
		if len(m) == 0 {
			delete(b.subscribers, userID)
		}
	}
	close(ch)
}

func (b *Broadcaster) Broadcast(userID string, payload NotificationPayload) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for ch := range b.subscribers[userID] {
		select {
		case ch <- payload:
		default:
			select {
			case <-ch:
			default:
			}
			select {
			case ch <- payload:
			default:
				go b.dropDeadSubscriber(userID, ch)
			}
		}
	}
}

func (b *Broadcaster) dropDeadSubscriber(userID string, ch chan NotificationPayload) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if m, ok := b.subscribers[userID]; ok {
		delete(m, ch)
		if len(m) == 0 {
			delete(b.subscribers, userID)
		}
	}
}
