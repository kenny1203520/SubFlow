package realtime

import (
	"context"
	"sync"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"subflow/internal/domain"
)

type Bus struct {
	mu          sync.RWMutex
	next        int
	subscribers map[int]subscriber
}
type subscriber struct {
	groupID string
	ch      chan domain.Event
}

func NewBus() *Bus { return &Bus{subscribers: map[int]subscriber{}} }
func (b *Bus) Publish(_ context.Context, event domain.Event) error {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, s := range b.subscribers {
		if s.groupID == event.GroupID {
			select {
			case s.ch <- event:
			default:
			}
		}
	}
	return nil
}
func (b *Bus) Subscribe(ctx context.Context, groupID string) (<-chan domain.Event, func()) {
	b.mu.Lock()
	id := b.next
	b.next++
	ch := make(chan domain.Event, 32)
	b.subscribers[id] = subscriber{groupID, ch}
	b.mu.Unlock()
	var once sync.Once
	cancel := func() { once.Do(func() { b.mu.Lock(); delete(b.subscribers, id); close(ch); b.mu.Unlock() }) }
	go func() { <-ctx.Done(); cancel() }()
	return ch, cancel
}

func BindRecordEvents(app core.App, bus *Bus) {
	collections := []string{"groups", "group_members", "subscriptions", "expenses"}
	for _, collection := range collections {
		resource := collection
		publish := func(kind string, e *core.RecordEvent) error {
			groupID := e.Record.GetString("group")
			if resource == "groups" {
				groupID = e.Record.Id
			}
			_ = bus.Publish(e.Context, domain.Event{Type: kind, GroupID: groupID, Resource: resource, ResourceID: e.Record.Id, OccurredAt: time.Now().UTC()})
			return e.Next()
		}
		app.OnRecordAfterCreateSuccess(collection).BindFunc(func(e *core.RecordEvent) error { return publish("created", e) })
		app.OnRecordAfterUpdateSuccess(collection).BindFunc(func(e *core.RecordEvent) error { return publish("updated", e) })
		app.OnRecordAfterDeleteSuccess(collection).BindFunc(func(e *core.RecordEvent) error { return publish("deleted", e) })
	}
}
