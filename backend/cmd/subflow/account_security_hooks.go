package main

import (
	"context"
	"fmt"

	"ganigeorgiev.com/pocketbase/pocketbase/core"

	"subflow/internal/adapters"
	"subflow/internal/application"
	"subflow/internal/domain"
	"subflow/internal/realtime"
)

func registerAccountSecurityConfirmationHooks(app core.App, service *application.Service, events *realtime.Bus, encryptionKey string) {
	app.OnRecordConfirmPasswordResetRequest("users").BindFunc(func(event *core.RecordConfirmPasswordResetRequestEvent) error {
		userID := event.Record.Id
		requestApp := event.App
		var notice *domain.Notification
		if err := requestApp.RunInTransaction(func(txApp core.App) error {
			event.App = txApp
			defer func() { event.App = requestApp }()
			if err := event.Next(); err != nil {
				return err
			}
			txStores, err := adapters.New("pocketbase", txApp)
			if err != nil {
				return err
			}
			now := service.Now().UTC()
			note := &domain.Notification{UserID: userID, Category: domain.NotificationCategorySecurity, Type: "password_changed", EventKey: fmt.Sprintf("password-changed:%s:%d", userID, now.UnixNano()), Title: "password_changed", Status: domain.NotificationStatusPending, CreatedAt: now, UpdatedAt: now}
			if err := txStores.Notifications.Create(event.Request.Context(), note); err != nil {
				return err
			}
			notice = note
			if err := txStores.NotificationEmails.Enqueue(event.Request.Context(), &domain.NotificationEmailJob{NotificationID: note.ID, UserID: userID, Status: domain.NotificationEmailQueued, NextAttemptAt: &now}); err != nil {
				return err
			}
			return application.New(txStores, encryptionKey).RecordAudit(event.Request.Context(), userID, "", "account.password.changed", "user", userID, "success")
		}); err != nil {
			return err
		}
		publishAccountSecurityNotice(event.Request.Context(), service, events, notice, userID)
		return nil
	})

	app.OnRecordConfirmEmailChangeRequest("users").BindFunc(func(event *core.RecordConfirmEmailChangeRequestEvent) error {
		userID := event.Record.Id
		oldEmail := event.Record.Email()
		requestApp := event.App
		var notice *domain.Notification
		if err := requestApp.RunInTransaction(func(txApp core.App) error {
			event.App = txApp
			defer func() { event.App = requestApp }()
			if err := event.Next(); err != nil {
				return err
			}
			txStores, err := adapters.New("pocketbase", txApp)
			if err != nil {
				return err
			}
			now := service.Now().UTC()
			note := &domain.Notification{UserID: userID, Category: domain.NotificationCategorySecurity, Type: "account_email_changed", EventKey: fmt.Sprintf("account-email-changed:%s:%d", userID, now.UnixNano()), Title: "account_email_changed", Status: domain.NotificationStatusPending, CreatedAt: now, UpdatedAt: now}
			if err := txStores.Notifications.Create(event.Request.Context(), note); err != nil {
				return err
			}
			notice = note
			if err := txStores.NotificationEmails.Enqueue(event.Request.Context(), &domain.NotificationEmailJob{NotificationID: note.ID, UserID: userID, RecipientEmail: oldEmail, Status: domain.NotificationEmailQueued, NextAttemptAt: &now}); err != nil {
				return err
			}
			return application.New(txStores, encryptionKey).RecordAudit(event.Request.Context(), userID, "", "account.email.changed", "user", userID, "success", `{"reason":"email_change_confirmed"}`)
		}); err != nil {
			return err
		}
		publishAccountSecurityNotice(event.Request.Context(), service, events, notice, userID)
		return nil
	})
}

func publishAccountSecurityNotice(ctx context.Context, service *application.Service, events *realtime.Bus, notice *domain.Notification, userID string) {
	if notice == nil {
		return
	}
	_ = events.Publish(ctx, domain.Event{Type: "created", UserID: userID, Resource: "notifications", ResourceID: notice.ID, OccurredAt: service.Now().UTC()})
}
