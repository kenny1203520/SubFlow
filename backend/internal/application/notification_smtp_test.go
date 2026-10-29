package application_test

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/textproto"
	"strconv"
	"strings"
	"testing"
	"time"

	"subflow/internal/application"
	"subflow/internal/domain"
	"subflow/internal/mailer"
	"subflow/internal/ports"
)

// This is a real loopback SMTP transport; no external service or real address
// is used. Unit tests separately exercise the bounded retry schedule.
func notificationSMTP(t *testing.T, reject bool) (string, string, <-chan string) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	messages := make(chan string, 10)
	go func() {
		for {
			conn, acceptErr := listener.Accept()
			if acceptErr != nil {
				return
			}
			go func() {
				defer conn.Close()
				_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
				reader := textproto.NewReader(bufio.NewReader(conn))
				_, _ = fmt.Fprint(conn, "220 localhost ESMTP\r\n")
				for {
					line, readErr := reader.ReadLine()
					if readErr != nil {
						return
					}
					switch {
					case strings.HasPrefix(line, "EHLO"), strings.HasPrefix(line, "HELO"):
						_, _ = fmt.Fprint(conn, "250-localhost\r\n250 8BITMIME\r\n")
					case strings.HasPrefix(line, "RCPT") && reject:
						_, _ = fmt.Fprint(conn, "550 recipient rejected\r\n")
					case line == "DATA":
						_, _ = fmt.Fprint(conn, "354 send message\r\n")
						body, bodyErr := reader.ReadDotBytes()
						if bodyErr != nil {
							return
						}
						messages <- string(body)
						_, _ = fmt.Fprint(conn, "250 queued\r\n")
					case line == "QUIT":
						_, _ = fmt.Fprint(conn, "221 bye\r\n")
						return
					default:
						_, _ = fmt.Fprint(conn, "250 ok\r\n")
					}
				}
			}()
		}
	}()
	address := listener.Addr().(*net.TCPAddr)
	return "127.0.0.1", strconv.Itoa(address.Port), messages
}

func TestNotificationDeliveryUsesRealSMTPAndAuditsResult(t *testing.T) {
	for _, reject := range []bool{false, true} {
		t.Run(map[bool]string{false: "delivered", true: "SMTP rejection"}[reject], func(t *testing.T) {
			f := newHistoricalFixture(t)
			// PocketBase's TestApp normally replaces every SMTP client with a
			// fake. Remove that test-only interception to exercise TCP delivery.
			f.app.OnMailerSend().UnbindAll()
			host, port, messages := notificationSMTP(t, reject)
			settings := f.app.Settings()
			settings.SMTP.Enabled = true
			settings.SMTP.Host = host
			settings.SMTP.Port, _ = strconv.Atoi(port)
			settings.Meta.SenderAddress = "noreply@example.test"
			settings.Meta.AppURL = "https://localhost:8080"
			if err := f.app.Save(settings); err != nil {
				t.Fatal(err)
			}
			now := time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC)
			note := &domain.Notification{UserID: f.owner, Category: domain.NotificationCategorySecurity, Type: "password_changed", EventKey: "smtp-integration", Status: domain.NotificationStatusPending, CreatedAt: now, UpdatedAt: now}
			if err := f.stores.Notifications.Create(t.Context(), note); err != nil {
				t.Fatal(err)
			}
			job := &domain.NotificationEmailJob{NotificationID: note.ID, UserID: f.owner, Status: domain.NotificationEmailQueued}
			if err := f.stores.NotificationEmails.Enqueue(t.Context(), job); err != nil {
				t.Fatal(err)
			}
			delivery := application.NewNotificationDeliveryService(f.stores.Notifications, f.stores.NotificationEmails, f.stores.Transactions, f.stores.Users, f.service.RecordAudit, &mailer.Native{App: f.app}, func() time.Time { return now })
			err := delivery.ProcessDue(context.Background())
			if (err != nil) != reject {
				t.Fatalf("SMTP outcome: %v", err)
			}
			record, err := f.app.FindRecordById("notification_email_outbox", job.ID)
			if err != nil {
				t.Fatal(err)
			}
			status, outcome := domain.NotificationEmailSent, "success"
			if reject {
				status, outcome = domain.NotificationEmailQueued, "failure"
			}
			if record.GetString("status") != status || record.GetInt("attempts") != 1 {
				t.Fatalf("unexpected outbox state: %s / %d", record.GetString("status"), record.GetInt("attempts"))
			}
			logs, err := f.stores.Audits.List(t.Context(), "", ports.AuditQuery{PageRequest: ports.PageRequest{Page: 1, PerPage: 10}, Action: "notification.email.delivery"})
			if err != nil || len(logs.Items) != 1 || logs.Items[0].Outcome != outcome {
				t.Fatalf("delivery audit: %#v / %v", logs.Items, err)
			}
			if reject {
				if record.GetString("last_error") != "mail_delivery_failed" || strings.Contains(logs.Items[0].Summary, host) {
					t.Fatal("SMTP failure details must stay private")
				}
			} else {
				select {
				case message := <-messages:
					if !strings.Contains(message, "Password changed") || !strings.Contains(message, "owner@example.com") {
						t.Fatal("SMTP message did not reach the expected recipient")
					}
				case <-time.After(time.Second):
					t.Fatal("SMTP server received no email")
				}
				if err := delivery.ProcessDue(t.Context()); err != nil {
					t.Fatal(err)
				}
				select {
				case <-messages:
					t.Fatal("completed job was sent twice")
				default:
				}
			}
		})
	}
}
