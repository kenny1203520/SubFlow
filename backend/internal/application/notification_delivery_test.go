package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"subflow/internal/application"
	"subflow/internal/domain"
	"subflow/internal/ports"
)

type failingNotificationMailer struct{ calls int }

func (m *failingNotificationMailer) Configured() bool { return true }
func (m *failingNotificationMailer) SendNotification(context.Context, string, string, string) error {
	m.calls++
	return errors.New("smtp password and host must never be persisted")
}

func TestNotificationEmailDeliveryRetriesAndStoresOnlySafeFailureCode(t *testing.T) {
	f := newHistoricalFixture(t)
	ctx := context.Background()
	now := time.Date(2026, time.August, 11, 0, 0, 0, 0, time.UTC)
	note := &domain.Notification{UserID: f.owner, GroupID: f.group.ID, Category: domain.NotificationCategoryException, Type: "subscription_post_failed", EventKey: "delivery-test-event", Status: domain.NotificationStatusPending, Title: "Posting failed", Message: "Review the subscription", CreatedAt: now, UpdatedAt: now}
	if err := f.stores.Notifications.Create(ctx, note); err != nil {
		t.Fatal(err)
	}
	job := &domain.NotificationEmailJob{NotificationID: note.ID, UserID: f.owner, Status: domain.NotificationEmailQueued, CreatedAt: now, UpdatedAt: now}
	if err := f.stores.NotificationEmails.Enqueue(ctx, job); err != nil {
		t.Fatal(err)
	}
	mailer := &failingNotificationMailer{}
	delivery := application.NewNotificationDeliveryService(f.stores.Notifications, f.stores.NotificationEmails, f.stores.Transactions, f.stores.Users, f.service.RecordAudit, mailer, func() time.Time { return now })

	delays := []time.Duration{0, time.Minute, 5 * time.Minute, 15 * time.Minute, time.Hour}
	for attempt, delay := range delays {
		if attempt > 0 {
			now = now.Add(delay)
		}
		if err := delivery.ProcessDue(ctx); err == nil {
			t.Fatalf("attempt %d should surface the SMTP failure to the scheduler", attempt+1)
		}
	}
	if mailer.calls != 5 {
		t.Fatalf("expected five bounded attempts, got %d", mailer.calls)
	}
	failed, err := f.stores.NotificationEmails.ListFailed(ctx, ports.PageRequest{Page: 1, PerPage: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(failed.Items) != 1 || failed.Items[0].Attempts != 5 || failed.Items[0].LastError != "mail_delivery_failed" {
		t.Fatalf("expected one safely recorded exhausted job, got %#v", failed.Items)
	}
}
