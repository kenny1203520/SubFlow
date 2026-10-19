package httpapi_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"subflow/internal/application"
	"subflow/internal/domain"
	"subflow/internal/ports"
)

type attachmentEveryAuditFailure struct{ ports.AuditRepository }

func (r attachmentEveryAuditFailure) Create(context.Context, *domain.AuditLog) error {
	return errors.New("audit offline")
}

func TestAttachmentRenamePermissionsHistoryAndRollback(t *testing.T) {
	f := newAPITestApp(t)
	ctx := context.Background()
	owner := f.createUser(t, "rename-group-owner@example.com")
	member := f.createUser(t, "rename-group-member@example.com")
	outsider := f.createUser(t, "rename-group-outsider@example.com")
	group := f.seedGroup(t, owner.Id, member.Id)
	expense := domain.Expense{GroupID: group.ID, PaidBy: owner.Id, Title: "Receipt", AmountMinor: 100, Currency: domain.CurrencyTWD, IncurredOn: time.Now().UTC()}
	if err := f.stores.Expenses.Create(ctx, &expense); err != nil {
		t.Fatal(err)
	}
	target := application.AttachmentTarget{Type: "expense", ID: expense.ID, RequestID: "rename-group"}
	a, err := f.service.UploadAttachment(ctx, owner.Id, target, "original.pdf", []byte("%PDF-1.7"))
	if err != nil {
		t.Fatal(err)
	}
	assign := func(perms ...string) {
		t.Helper()
		role, err := f.service.CreateGroupRole(ctx, owner.Id, domain.Role{GroupID: group.ID, Name: "Attachment editor", Permissions: append([]string{"group.view", "ledger.expenses.read"}, perms...)})
		if err != nil {
			t.Fatal(err)
		}
		if err = f.service.AssignGroupRole(ctx, owner.Id, group.ID, member.Id, role.ID); err != nil {
			t.Fatal(err)
		}
	}
	assign("ledger.attachments.upload")
	if _, err = f.service.RenameAttachment(ctx, member.Id, a.ID, "denied.pdf"); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("missing attachment read: %v", err)
	}
	assign("ledger.attachments.read")
	page, err := f.service.ListAttachments(ctx, member.Id, target)
	if err != nil || page.Access.CanRename {
		t.Fatalf("rename access without upload: %v %v", page, err)
	}
	if _, err = f.service.RenameAttachment(ctx, member.Id, a.ID, "denied.pdf"); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("missing upload: %v", err)
	}
	assign("ledger.attachments.read", "ledger.attachments.upload")
	if _, err = f.service.RenameAttachment(ctx, outsider.Id, a.ID, "denied.pdf"); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("other group user: %v", err)
	}
	if _, err = f.service.RenameAttachment(ctx, member.Id, a.ID, "current.pdf"); err != nil {
		t.Fatal(err)
	}
	expense.IncurredOn = time.Now().AddDate(0, -2, 0)
	if err = f.stores.Expenses.Update(ctx, &expense); err != nil {
		t.Fatal(err)
	}
	page, err = f.service.ListAttachments(ctx, member.Id, target)
	if err != nil || page.Access.CanRename {
		t.Fatalf("historical access: %v %v", page, err)
	}
	if _, err = f.service.RenameAttachment(ctx, member.Id, a.ID, "denied.pdf"); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("missing historical write: %v", err)
	}
	assign("ledger.attachments.read", "ledger.attachments.upload", "ledger.records.historical_write")
	// Legacy over-limit records remain renameable; only new uploads are limited.
	group.AttachmentMaxCount = 1
	if err = f.stores.Groups.Update(ctx, group); err != nil {
		t.Fatal(err)
	}
	copy := *a
	copy.ID = ""
	copy.RequestID = "legacy-over-limit"
	if err = f.stores.Attachments.Create(ctx, &copy, []byte("%PDF-1.7")); err != nil {
		t.Fatal(err)
	}
	unicodeName := strings.Repeat("收", 251) + ".pdf"
	if _, err = f.service.RenameAttachment(ctx, member.Id, a.ID, unicodeName); err != nil {
		t.Fatal(err)
	}
	originalAudits := f.service.Stores.Audits
	f.service.Stores.Audits = attachmentAuditFailure{originalAudits}
	if _, err = f.service.RenameAttachment(ctx, member.Id, a.ID, "rollback.pdf"); !errors.Is(err, domain.ErrAuditUnavailable) {
		t.Fatalf("audit failure: %v", err)
	}
	saved, err := f.stores.Attachments.Get(ctx, a.ID)
	if err != nil || saved.FileName != unicodeName {
		t.Fatalf("rename escaped rollback: %v %v", saved, err)
	}
	f.service.Stores.Audits = attachmentEveryAuditFailure{originalAudits}
	if _, err = f.service.RenameAttachment(ctx, member.Id, a.ID, "invalid.exe"); !errors.Is(err, domain.ErrAuditUnavailable) {
		t.Fatalf("failure audit unavailable: %v", err)
	}
	logs, err := f.stores.Audits.List(ctx, group.ID, ports.AuditQuery{Action: "attachment.renamed", PageRequest: ports.PageRequest{Page: 1, PerPage: 100}})
	if err != nil {
		t.Fatal(err)
	}
	success, failure := 0, 0
	for _, log := range logs.Items {
		if log.Outcome == "success" {
			success++
		} else {
			failure++
		}
	}
	if success != 2 || failure < 4 {
		t.Fatalf("missing atomic rename audits: successes=%d failures=%d", success, failure)
	}
}

func TestAttachmentDerivedExpenseUsesCanonicalReadPermission(t *testing.T) {
	f := newAPITestApp(t)
	ctx := context.Background()
	owner := f.createUser(t, "canonical-owner@example.com")
	member := f.createUser(t, "canonical-member@example.com")
	group := f.seedGroup(t, owner.Id, member.Id)
	at := time.Now().UTC().Truncate(time.Millisecond)
	sub := domain.Subscription{GroupID: group.ID, PaidBy: owner.Id, Name: "Monthly", Currency: domain.CurrencyTWD, AmountMinor: 100, BillingCycle: domain.BillingMonthly, BillingInterval: 1, StartsOn: at, NextBilling: at, Status: domain.SubscriptionActive, SplitMode: domain.SplitAmount}
	if err := f.stores.Subscriptions.Create(ctx, &sub); err != nil {
		t.Fatal(err)
	}
	expense := domain.Expense{GroupID: group.ID, PaidBy: owner.Id, Title: "Posted", SubscriptionID: sub.ID, Currency: domain.CurrencyTWD, AmountMinor: 100, IncurredOn: at}
	if err := f.stores.Expenses.Create(ctx, &expense); err != nil {
		t.Fatal(err)
	}
	target := application.AttachmentTarget{Type: "expense", ID: expense.ID, RequestID: "canonical"}
	a, err := f.service.UploadAttachment(ctx, owner.Id, target, "receipt.pdf", []byte("%PDF-1.7"))
	if err != nil {
		t.Fatal(err)
	}
	perms := []string{"group.view", "ledger.expenses.read", "ledger.attachments.read", "ledger.attachments.upload"}
	for _, withSubscriptionRead := range []bool{false, true} {
		if withSubscriptionRead {
			perms = append(perms, "ledger.subscriptions.read")
		}
		role, err := f.service.CreateGroupRole(ctx, owner.Id, domain.Role{GroupID: group.ID, Name: "Reader", Permissions: perms})
		if err != nil {
			t.Fatal(err)
		}
		if err = f.service.AssignGroupRole(ctx, owner.Id, group.ID, member.Id, role.ID); err != nil {
			t.Fatal(err)
		}
		page, listErr := f.service.ListAttachments(ctx, member.Id, target)
		_, renameErr := f.service.RenameAttachment(ctx, member.Id, a.ID, "renamed.pdf")
		if !withSubscriptionRead {
			if !errors.Is(listErr, domain.ErrForbidden) || !errors.Is(renameErr, domain.ErrForbidden) {
				t.Fatalf("canonical scope bypass: list=%v rename=%v", listErr, renameErr)
			}
		} else if listErr != nil || renameErr != nil || !page.Access.CanRename || len(page.Items) != 1 {
			t.Fatalf("canonical scope access: list=%v rename=%v page=%v", listErr, renameErr, page)
		}
	}
}
