package mailer

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"os"
	"strings"

	"subflow/internal/domain"
)

type SMTP struct{ Host, Port, User, Password, From string }

func FromEnv() *SMTP {
	return &SMTP{Host: os.Getenv("SUBFLOW_SMTP_HOST"), Port: value("SUBFLOW_SMTP_PORT", "587"), User: os.Getenv("SUBFLOW_SMTP_USER"), Password: os.Getenv("SUBFLOW_SMTP_PASSWORD"), From: os.Getenv("SUBFLOW_SMTP_FROM")}
}
func value(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
func (m *SMTP) Configured() bool { return m.Host != "" && m.From != "" }
func (m *SMTP) SendInvitation(ctx context.Context, inv domain.Invitation, group domain.Group, url string) error {
	if !m.Configured() {
		return fmt.Errorf("smtp is not configured")
	}
	subject := "SubFlow 群組邀請"
	body := fmt.Sprintf("你已受邀加入 %s。\r\n\r\n%s\r\n", group.Name, url)
	message := []byte("From: " + m.From + "\r\nTo: " + inv.Email + "\r\nSubject: " + subject + "\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n" + body)
	address := net.JoinHostPort(m.Host, m.Port)
	auth := smtp.Auth(nil)
	if m.User != "" {
		auth = smtp.PlainAuth("", m.User, m.Password, m.Host)
	}
	if m.Port == "465" {
		dialer := &net.Dialer{}
		conn, err := tls.DialWithDialer(dialer, "tcp", address, &tls.Config{ServerName: m.Host, MinVersion: tls.VersionTLS12})
		if err != nil {
			return err
		}
		client, err := smtp.NewClient(conn, m.Host)
		if err != nil {
			return err
		}
		defer client.Close()
		if auth != nil {
			if err = client.Auth(auth); err != nil {
				return err
			}
		}
		if err = client.Mail(m.From); err != nil {
			return err
		}
		if err = client.Rcpt(inv.Email); err != nil {
			return err
		}
		w, err := client.Data()
		if err != nil {
			return err
		}
		if _, err = w.Write(message); err != nil {
			return err
		}
		return w.Close()
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	return smtp.SendMail(address, auth, m.From, []string{inv.Email}, message)
}

type DevelopmentSink struct{}

func (*DevelopmentSink) Configured() bool { return true }
func (*DevelopmentSink) SendInvitation(context.Context, domain.Invitation, domain.Group, string) error {
	return nil
}
func ModeMailer(environment string) *SMTP { return FromEnv() }
func IsDevelopment(environment string) bool {
	return strings.EqualFold(environment, "development") || environment == ""
}
