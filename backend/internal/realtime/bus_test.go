package realtime

import (
	"context"
	"testing"
	"time"

	"subflow/internal/domain"
)

func TestBusIsolatesGroups(t *testing.T) {
	bus := NewBus()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	a, stopA := bus.Subscribe(ctx, "a")
	defer stopA()
	b, stopB := bus.Subscribe(ctx, "b")
	defer stopB()
	want := domain.Event{Type: "created", GroupID: "a", Resource: "expenses", ResourceID: "1", OccurredAt: time.Now()}
	if err := bus.Publish(ctx, want); err != nil {
		t.Fatal(err)
	}
	select {
	case got := <-a:
		if got.ResourceID != want.ResourceID {
			t.Fatalf("unexpected event: %#v", got)
		}
	case <-time.After(time.Second):
		t.Fatal("group a did not receive event")
	}
	select {
	case got := <-b:
		t.Fatalf("group b received leaked event: %#v", got)
	case <-time.After(20 * time.Millisecond):
	}
}
