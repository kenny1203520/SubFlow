package application

import (
	"context"
	"time"

	"subflow/internal/domain"
)

func (s *Service) createNotification(ctx context.Context, userID, category, eventKey, title, groupID, resourceID string) (*domain.Notification, error) {
	note := &domain.Notification{UserID: userID, Category: category, EventKey: eventKey, Type: title, Title: title, GroupID: groupID, ResourceID: resourceID, Status: domain.NotificationStatusPending}
	if err := s.Stores.Notifications.Create(ctx, note); err != nil {
		return nil, err
	}
	return note, nil
}

func (s *Service) publishNotification(ctx context.Context, note *domain.Notification, kind string) {
	if note == nil || s.Events == nil {
		return
	}
	// Notification events are private inbox hints. Omitting GroupID prevents a
	// group-scoped SSE subscriber from learning that another member received one.
	_ = s.Events.Publish(ctx, domain.Event{Type: kind, UserID: note.UserID, Resource: "notifications", ResourceID: note.ID, OccurredAt: s.now().UTC()})
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}
