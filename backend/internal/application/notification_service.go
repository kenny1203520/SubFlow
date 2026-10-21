package application

import (
	"context"
	"time"

	"subflow/internal/domain"
	"subflow/internal/ports"
)

// NotificationService owns notification inbox operations. It receives only
// the ports it uses, keeping inbox behavior independent from invitation and
// finance workflows.
type NotificationService struct {
	repository   ports.NotificationRepository
	transactions ports.TransactionManager
	now          func() time.Time
	audit        func(context.Context, string, string, string, string, string, string) error
}

func NewNotificationService(repository ports.NotificationRepository, transactions ports.TransactionManager, now func() time.Time, audit func(context.Context, string, string, string, string, string, string) error) *NotificationService {
	if now == nil {
		now = time.Now
	}
	return &NotificationService{repository: repository, transactions: transactions, now: now, audit: audit}
}

func (s *NotificationService) ListForUser(ctx context.Context, userID string, page ports.PageRequest) (ports.Page[domain.Notification], error) {
	return s.repository.ListForUser(ctx, userID, page)
}

func (s *NotificationService) ListFiltered(ctx context.Context, userID string, query ports.NotificationQuery) (ports.Page[domain.Notification], error) {
	return s.repository.ListFiltered(ctx, userID, query)
}

func (s *NotificationService) Summary(ctx context.Context, userID string) (ports.NotificationSummary, error) {
	return s.repository.Summary(ctx, userID, s.now().UTC())
}

func (s *NotificationService) Preferences(ctx context.Context, userID string) (domain.NotificationPreferences, error) {
	return s.repository.GetPreferences(ctx, userID)
}

func (s *NotificationService) UpdatePreferences(ctx context.Context, userID string, value domain.NotificationPreferences) error {
	value.UserID = userID
	// Required account security notices cannot be suppressed by client input.
	value.SecurityToast = true
	value.SecurityEmail = true
	err := s.transactions.Within(ctx, func(tx context.Context) error {
		if err := s.repository.SavePreferences(tx, value); err != nil {
			return err
		}
		if s.audit != nil {
			return s.audit(tx, userID, "", "notification.preferences.updated", "notification_preferences", userID, "success")
		}
		return nil
	})
	return s.auditFailure(ctx, userID, nil, userID, "notification.preferences.updated", err)
}

func (s *NotificationService) MarkRead(ctx context.Context, userID, id string) error {
	var note *domain.Notification
	err := s.transactions.Within(ctx, func(tx context.Context) error {
		var err error
		note, err = s.repository.Get(tx, id)
		if err != nil {
			return err
		}
		if note.UserID != userID {
			return domain.ErrForbidden
		}
		if note.ReadAt == nil {
			if err = s.repository.MarkRead(tx, id, s.now().UTC()); err != nil {
				return err
			}
			if s.audit != nil {
				return s.audit(tx, userID, note.GroupID, "notification.read", "notification", id, "success")
			}
		}
		return nil
	})
	if err == nil {
		return nil
	}
	// The failure audit must use the outer context; writing it inside the
	// transaction that just failed would roll the audit row back as well.
	if s.audit != nil {
		groupID := ""
		if note != nil {
			groupID = note.GroupID
		}
		if auditErr := s.audit(ctx, userID, groupID, "notification.read", "notification", id, "failure"); auditErr != nil {
			return domain.ErrAuditUnavailable
		}
	}
	return err
}

func (s *NotificationService) MarkUnread(ctx context.Context, userID, id string) error {
	var note *domain.Notification
	err := s.transactions.Within(ctx, func(tx context.Context) error {
		var err error
		note, err = s.repository.Get(tx, id)
		if err != nil {
			return err
		}
		if note.UserID != userID {
			return domain.ErrForbidden
		}
		if err = s.repository.MarkUnread(tx, userID, id); err != nil {
			return err
		}
		if s.audit != nil {
			return s.audit(tx, userID, note.GroupID, "notification.unread", "notification", id, "success")
		}
		return nil
	})
	return s.auditFailure(ctx, userID, note, id, "notification.unread", err)
}

func (s *NotificationService) MarkReadAll(ctx context.Context, userID string) error {
	when := s.now().UTC()
	err := s.transactions.Within(ctx, func(tx context.Context) error {
		if err := s.repository.MarkReadAll(tx, userID, when, when); err != nil {
			return err
		}
		if s.audit != nil {
			return s.audit(tx, userID, "", "notification.read_all", "notification", userID, "success")
		}
		return nil
	})
	return s.auditFailure(ctx, userID, nil, userID, "notification.read_all", err)
}

func (s *NotificationService) MarkViewed(ctx context.Context, userID string, ids []string) error {
	when := s.now().UTC()
	err := s.transactions.Within(ctx, func(tx context.Context) error {
		if err := s.repository.MarkViewed(tx, userID, ids, when); err != nil {
			return err
		}
		if s.audit != nil {
			return s.audit(tx, userID, "", "notification.viewed", "notification", userID, "success")
		}
		return nil
	})
	return s.auditFailure(ctx, userID, nil, userID, "notification.viewed", err)
}

func (s *NotificationService) auditFailure(ctx context.Context, userID string, note *domain.Notification, resourceID, action string, err error) error {
	if err == nil {
		return nil
	}
	if s.audit != nil {
		groupID := ""
		if note != nil {
			groupID = note.GroupID
		}
		if auditErr := s.audit(ctx, userID, groupID, action, "notification", resourceID, "failure"); auditErr != nil {
			return domain.ErrAuditUnavailable
		}
	}
	return err
}
