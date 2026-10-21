package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"subflow/internal/domain"
	"subflow/internal/ports"
)

const maxNotificationEmailAttempts = 5

type NotificationDeliveryService struct {
	notifications ports.NotificationRepository
	emails        ports.NotificationEmailRepository
	transactions  ports.TransactionManager
	users         ports.UserDirectory
	audit         func(context.Context, string, string, string, string, string, string, ...string) error
	mailer        ports.NotificationMailer
	now           func() time.Time
}

func NewNotificationDeliveryService(notifications ports.NotificationRepository, emails ports.NotificationEmailRepository, transactions ports.TransactionManager, users ports.UserDirectory, audit func(context.Context, string, string, string, string, string, string, ...string) error, mailer ports.NotificationMailer, now func() time.Time) *NotificationDeliveryService {
	if now == nil {
		now = time.Now
	}
	return &NotificationDeliveryService{notifications: notifications, emails: emails, transactions: transactions, users: users, audit: audit, mailer: mailer, now: now}
}

func (s *NotificationDeliveryService) ProcessDue(ctx context.Context) error {
	if s.mailer == nil || !s.mailer.Configured() {
		return nil
	}
	now := s.now().UTC()
	var jobs []domain.NotificationEmailJob
	if err := s.transactions.Within(ctx, func(tx context.Context) error {
		var err error
		jobs, err = s.emails.ClaimDue(tx, now, 25)
		return err
	}); err != nil {
		return err
	}
	var failures []error
	for i := range jobs {
		if err := s.deliver(ctx, &jobs[i]); err != nil {
			failures = append(failures, err)
		}
	}
	return errors.Join(failures...)
}

func (s *NotificationDeliveryService) deliver(ctx context.Context, job *domain.NotificationEmailJob) error {
	notification, err := s.notifications.Get(ctx, job.NotificationID)
	if err != nil || notification.UserID != job.UserID {
		if err == nil {
			err = domain.ErrForbidden
		}
		return s.finish(ctx, job, domain.NotificationEmailCancelled, "notification_unavailable", "cancelled", notificationGroup(notification), err)
	}
	if notification.Category != domain.NotificationCategorySecurity && notification.Status != domain.NotificationStatusPending {
		return s.finish(ctx, job, domain.NotificationEmailCancelled, "event_ended", "cancelled", notification.GroupID, nil)
	}
	preferences, err := s.notifications.GetPreferences(ctx, job.UserID)
	if err != nil {
		return s.retry(ctx, job, notification.GroupID, err)
	}
	if !emailEnabled(preferences, notification.Category) {
		return s.finish(ctx, job, domain.NotificationEmailCancelled, "preference_disabled", "cancelled", notification.GroupID, nil)
	}
	user, err := s.users.Get(ctx, job.UserID)
	if err != nil || user.Email == "" {
		return s.finish(ctx, job, domain.NotificationEmailCancelled, "recipient_unavailable", "cancelled", notification.GroupID, err)
	}
	subject, body := notificationEmailContent(notification)
	if err := s.mailer.SendNotification(ctx, user.Email, subject, body); err != nil {
		return s.retry(ctx, job, notification.GroupID, err)
	}
	return s.finish(ctx, job, domain.NotificationEmailSent, "", "success", notification.GroupID, nil)
}

func (s *NotificationDeliveryService) retry(ctx context.Context, job *domain.NotificationEmailJob, groupID string, cause error) error {
	status, next := domain.NotificationEmailQueued, s.now().UTC().Add(emailRetryDelay(job.Attempts))
	if job.Attempts >= maxNotificationEmailAttempts {
		status, next = domain.NotificationEmailFailed, time.Time{}
	}
	job.Status = status
	job.LastError = "mail_delivery_failed"
	job.NextAttemptAt = nil
	if !next.IsZero() {
		job.NextAttemptAt = &next
	}
	if err := s.saveAndAudit(ctx, job, groupID, "failure", cause); err != nil {
		return err
	}
	return cause
}

func (s *NotificationDeliveryService) finish(ctx context.Context, job *domain.NotificationEmailJob, status, lastError, outcome, groupID string, cause error) error {
	job.Status = status
	job.LastError = lastError
	job.NextAttemptAt = nil
	return s.saveAndAudit(ctx, job, groupID, outcome, cause)
}

func (s *NotificationDeliveryService) saveAndAudit(ctx context.Context, job *domain.NotificationEmailJob, groupID, outcome string, cause error) error {
	action := "notification.email.delivery"
	err := s.transactions.Within(ctx, func(tx context.Context) error {
		if err := s.emails.Update(tx, job); err != nil {
			return err
		}
		return s.audit(tx, "", groupID, action, "notification_email", job.ID, outcome, encodeAuditSummary(map[string]any{"attempt": job.Attempts, "result": job.Status, "reason": job.LastError}, nil))
	})
	if err != nil {
		return errors.Join(err, cause)
	}
	if job.Status == domain.NotificationEmailFailed {
		return fmt.Errorf("notification email %s exhausted retries: %w", job.ID, cause)
	}
	return nil
}

func notificationGroup(note *domain.Notification) string {
	if note == nil {
		return ""
	}
	return note.GroupID
}

func emailEnabled(value domain.NotificationPreferences, category string) bool {
	switch category {
	case domain.NotificationCategoryTask:
		return value.TasksEmail
	case domain.NotificationCategoryException:
		return value.ExceptionsEmail
	case domain.NotificationCategorySecurity:
		return true
	default:
		return false
	}
}

func emailRetryDelay(attempt int) time.Duration {
	delays := []time.Duration{time.Minute, 5 * time.Minute, 15 * time.Minute, time.Hour}
	index := attempt - 1
	if index < 0 {
		index = 0
	}
	if index >= len(delays) {
		index = len(delays) - 1
	}
	return delays[index]
}

func notificationEmailContent(note *domain.Notification) (string, string) {
	titles := map[string]string{
		"group_invitation":             "Group invitation",
		"ownership_transfer":           "Ownership transfer request",
		"member_transfer":              "Member transfer request",
		"system_role_changed":          "System role changed",
		"subscription_post_failed":     "Subscription posting needs attention",
		"recurring_income_post_failed": "Recurring income posting needs attention",
	}
	title := titles[note.Type]
	if title == "" {
		title = "Important SubFlow notice"
	}
	return "SubFlow: " + title, "You have a new SubFlow notice: " + title + ". Sign in to review the details and any available actions."
}

func (s *Service) ListFailedNotificationEmails(ctx context.Context, userID string, req ports.PageRequest) (ports.Page[domain.NotificationEmailJob], error) {
	if err := s.systemPermission(ctx, userID, "system.settings.manage"); err != nil {
		return ports.Page[domain.NotificationEmailJob]{}, err
	}
	return s.Stores.NotificationEmails.ListFailed(ctx, req)
}

func (s *Service) RetryNotificationEmail(ctx context.Context, userID, id string) error {
	if err := s.systemPermission(ctx, userID, "system.settings.manage"); err != nil {
		return err
	}
	now := s.Now().UTC()
	err := s.Stores.Transactions.Within(ctx, func(tx context.Context) error {
		if err := s.Stores.NotificationEmails.Retry(tx, id, now); err != nil {
			return err
		}
		return s.audit(tx, userID, "", "notification.email.retried", "notification_email", id, "success")
	})
	if err != nil {
		if auditErr := s.audit(ctx, userID, "", "notification.email.retried", "notification_email", id, "failure"); auditErr != nil {
			return domain.ErrAuditUnavailable
		}
	}
	return err
}
