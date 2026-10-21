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

func TestPrivateNotificationEventOnlyReachesRecipientWorkspace(t *testing.T) {
	bus := NewBus()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	recipient, stopRecipient := bus.SubscribeWorkspace(ctx, "user-a", []string{"group-1"})
	defer stopRecipient()
	otherMember, stopOther := bus.SubscribeWorkspace(ctx, "user-b", []string{"group-1"})
	defer stopOther()
	groupOnly, stopGroup := bus.Subscribe(ctx, "group-1")
	defer stopGroup()

	if err := bus.Publish(ctx, domain.Event{Type: "created", UserID: "user-a", Resource: "notifications", ResourceID: "note-1"}); err != nil {
		t.Fatal(err)
	}
	select {
	case event := <-recipient:
		if event.ResourceID != "note-1" {
			t.Fatalf("unexpected notification event: %#v", event)
		}
	default:
		t.Fatal("recipient did not receive notification hint")
	}
	for name, channel := range map[string]<-chan domain.Event{"other member": otherMember, "group subscriber": groupOnly} {
		select {
		case event := <-channel:
			t.Fatalf("private notification leaked to %s: %#v", name, event)
		default:
		}
	}
}
