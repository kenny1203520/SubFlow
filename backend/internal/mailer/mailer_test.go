package mailer

import "testing"

func TestSMTPConfigurationRequiresHostAndSender(t *testing.T) {
	for _, m := range []*SMTP{{}, {Host: "smtp.example.com"}, {From: "noreply@example.com"}} {
		if m.Configured() {
			t.Fatalf("unexpected configured state: %#v", m)
		}
	}
	if !(&SMTP{Host: "smtp.example.com", From: "noreply@example.com"}).Configured() {
		t.Fatal("complete SMTP configuration should be accepted")
	}
}
