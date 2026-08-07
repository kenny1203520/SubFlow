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
	userID  string
	groups  map[string]bool
	ch      chan domain.Event
}

func NewBus() *Bus { return &Bus{subscribers: map[int]subscriber{}} }
func (b *Bus) Publish(_ context.Context, event domain.Event) error {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, s := range b.subscribers {
		if s.groupID == event.GroupID || (s.userID != "" && (event.UserID == s.userID || (event.GroupID != "" && s.groups[event.GroupID]))) {
			select {
			case s.ch <- event:
			default:
			}
		}
	}
	return nil
}

func (b *Bus) SubscribeWorkspace(ctx context.Context, userID string, groupIDs []string) (<-chan domain.Event, func()) {
	b.mu.Lock()
	id := b.next
	b.next++
	ch := make(chan domain.Event, 32)
	groups := map[string]bool{}
	for _, groupID := range groupIDs {
		groups[groupID] = true
	}
	b.subscribers[id] = subscriber{userID: userID, groups: groups, ch: ch}
	b.mu.Unlock()
	var once sync.Once
	cancel := func() { once.Do(func() { b.mu.Lock(); delete(b.subscribers, id); close(ch); b.mu.Unlock() }) }
	go func() { <-ctx.Done(); cancel() }()
	return ch, cancel
}
func (b *Bus) Subscribe(ctx context.Context, groupID string) (<-chan domain.Event, func()) {
	b.mu.Lock()
	id := b.next
	b.next++
	ch := make(chan domain.Event, 32)
	b.subscribers[id] = subscriber{groupID: groupID, ch: ch}
	b.mu.Unlock()
	var once sync.Once
	cancel := func() { once.Do(func() { b.mu.Lock(); delete(b.subscribers, id); close(ch); b.mu.Unlock() }) }
	go func() { <-ctx.Done(); cancel() }()
	return ch, cancel
}

func BindRecordEvents(app core.App, bus *Bus) {
	collections := []string{"groups", "group_members", "subscriptions", "expenses", "settlements"}
	for _, collection := range collections {
		resource := collection
		publish := func(kind string, e *core.RecordEvent) error {
			groupID := e.Record.GetString("group")
			userID := e.Record.GetString("owner")
			if resource == "group_members" {
				userID = e.Record.GetString("user")
			}
			if resource == "groups" {
				groupID = e.Record.Id
			}
			_ = bus.Publish(e.Context, domain.Event{Type: kind, GroupID: groupID, UserID: userID, Resource: resource, ResourceID: e.Record.Id, OccurredAt: time.Now().UTC()})
			return e.Next()
		}
		app.OnRecordAfterCreateSuccess(collection).BindFunc(func(e *core.RecordEvent) error { return publish("created", e) })
		app.OnRecordAfterUpdateSuccess(collection).BindFunc(func(e *core.RecordEvent) error { return publish("updated", e) })
		app.OnRecordAfterDeleteSuccess(collection).BindFunc(func(e *core.RecordEvent) error { return publish("deleted", e) })
	}
}
