package httpapi_test

import (
	"context"
	"errors"
	"subflow/internal/application"
	"subflow/internal/domain"
	"subflow/internal/ports"
	"testing"
	"time"
)

type attachmentAuditFailure struct{ ports.AuditRepository }

func (r attachmentAuditFailure) Create(ctx context.Context, v *domain.AuditLog) error {
	if v.Outcome == "success" {
		return errors.New("audit offline")
	}
	return r.AuditRepository.Create(ctx, v)
}
func TestAttachmentAuditRollbackAndValidation(t *testing.T) {
	f := newAPITestApp(t)
	u := f.createUser(t, "attachment-rollback@example.com")
	ctx := context.Background()
	v, err := f.service.CreateIncome(ctx, u.Id, domain.Income{Title: "Income", AmountMinor: 100, Currency: domain.CurrencyTWD, ReceivedOn: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	target := application.AttachmentTarget{Type: "income", ID: v.ID, RequestID: "valid"}
	for _, tc := range []struct {
		name, requestID string
		data            []byte
		want            error
	}{
		{"request ID", "", []byte("%PDF-1.7"), domain.ErrInvalid},
		{"short HEIC", "short", []byte("0000ftypheic"), domain.ErrAttachmentFormat},
	} {
		t.Run(tc.name, func(t *testing.T) {
			target.RequestID = tc.requestID
			name := "receipt.pdf"
			if tc.name == "short HEIC" {
				name = "receipt.heic"
			}
			if _, err := f.service.UploadAttachment(ctx, u.Id, target, name, tc.data); !errors.Is(err, tc.want) {
				t.Fatalf("got %v want %v", err, tc.want)
			}
		})
	}
	target.RequestID = "upload"
	original := f.service.Stores.Audits
	f.service.Stores.Audits = attachmentAuditFailure{original}
	if _, err := f.service.UploadAttachment(ctx, u.Id, target, "receipt.pdf", []byte("%PDF-1.7")); !errors.Is(err, domain.ErrAuditUnavailable) {
		t.Fatalf("audit failure: %v", err)
	}
	items, err := f.stores.Attachments.List(ctx, "income", v.ID, time.Time{})
	if err != nil || len(items) != 0 {
		t.Fatalf("upload escaped rollback: %v %v", items, err)
	}
	f.service.Stores.Audits = original
	a, err := f.service.UploadAttachment(ctx, u.Id, target, "receipt.pdf", []byte("%PDF-1.7"))
	if err != nil {
		t.Fatal(err)
	}
	f.service.Stores.Audits = attachmentAuditFailure{original}
	if err = f.service.DeleteAttachment(ctx, u.Id, a.ID); err == nil {
		t.Fatal("expected deletion audit failure")
	}
	if _, err = f.stores.Attachments.Get(ctx, a.ID); err != nil {
		t.Fatalf("delete escaped rollback: %v", err)
	}
	if _, err = f.stores.Attachments.Read(ctx, a.ID); err != nil {
		t.Fatalf("blob lost on rollback: %v", err)
	}
}

func TestAttachmentGroupPermissionsAndHistory(t *testing.T) {
	f := newAPITestApp(t)
	owner := f.createUser(t, "attachment-admin@example.com")
	member := f.createUser(t, "attachment-member@example.com")
	ctx := context.Background()
	g := f.seedGroup(t, owner.Id, member.Id)
	assign := func(perms ...string) {
		t.Helper()
		r, err := f.service.CreateGroupRole(ctx, owner.Id, domain.Role{GroupID: g.ID, Name: "Attachments", Permissions: append([]string{"group.view", "ledger.expenses.read"}, perms...)})
		if err != nil {
			t.Fatal(err)
		}
		if err = f.service.AssignGroupRole(ctx, owner.Id, g.ID, member.Id, r.ID); err != nil {
			t.Fatal(err)
		}
	}
	v := domain.Expense{GroupID: g.ID, Title: "Receipt", PaidBy: owner.Id, AmountMinor: 100, Currency: domain.CurrencyTWD, IncurredOn: time.Now()}
	if err := f.stores.Expenses.Create(ctx, &v); err != nil {
		t.Fatal(err)
	}
	target := application.AttachmentTarget{Type: "expense", ID: v.ID, RequestID: "group-upload"}
	upload := func(actor string) (*domain.Attachment, error) {
		return f.service.UploadAttachment(ctx, actor, target, "receipt.pdf", []byte("%PDF-1.7"))
	}
	assign("ledger.attachments.upload")
	if _, err := upload(member.Id); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("missing read: %v", err)
	}
	assign("ledger.attachments.read")
	if _, err := upload(member.Id); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("missing upload: %v", err)
	}
	assign("ledger.attachments.read", "ledger.attachments.upload")
	a, err := upload(member.Id)
	if err != nil {
		t.Fatal(err)
	}
	if err = f.service.DeleteAttachment(ctx, member.Id, a.ID); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("missing delete: %v", err)
	}
	assign("ledger.attachments.read", "ledger.attachments.upload", "ledger.attachments.delete")
	other := f.seedGroup(t, owner.Id, "")
	strangerExpense := domain.Expense{GroupID: other.ID, Title: "Other group", PaidBy: owner.Id, AmountMinor: 100, Currency: domain.CurrencyTWD, IncurredOn: time.Now()}
	if err = f.stores.Expenses.Create(ctx, &strangerExpense); err != nil {
		t.Fatal(err)
	}
	if _, err = f.service.ListAttachments(ctx, member.Id, application.AttachmentTarget{Type: "expense", ID: strangerExpense.ID}); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("cross group: %v", err)
	}
	g.AttachmentMaxCount = 1
	if err = f.stores.Groups.Update(ctx, g); err != nil {
		t.Fatal(err)
	}
	target.RequestID = "count-limit"
	if _, err = upload(member.Id); !errors.Is(err, domain.ErrAttachmentCount) {
		t.Fatalf("count limit: %v", err)
	}
	g.AttachmentMaxCount = 10
	if err = f.stores.Groups.Update(ctx, g); err != nil {
		t.Fatal(err)
	}
	v.IncurredOn = time.Now().AddDate(0, -2, 0)
	if err = f.stores.Expenses.Update(ctx, &v); err != nil {
		t.Fatal(err)
	}
	target.RequestID = "historical"
	if _, err = upload(member.Id); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("historical upload: %v", err)
	}
	if err = f.service.DeleteAttachment(ctx, member.Id, a.ID); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("historical delete: %v", err)
	}
	assign("ledger.attachments.read", "ledger.attachments.upload", "ledger.attachments.delete", "ledger.records.historical_write")
	if _, err = upload(member.Id); err != nil {
		t.Fatal(err)
	}
	if err = f.service.DeleteAttachment(ctx, member.Id, a.ID); err != nil {
		t.Fatal(err)
	}
}

func TestSettlementApprovalSnapshotPreventsDirectAttachmentMutation(t *testing.T) {
	f := newAPITestApp(t)
	owner := f.createUser(t, "settlement-attachment-owner@example.com")
	member := f.createUser(t, "settlement-attachment-member@example.com")
	ctx := context.Background()
	group := f.seedGroup(t, owner.Id, member.Id)
	create := func() *domain.Settlement {
		t.Helper()
		value, err := f.service.CreateSettlement(ctx, owner.Id, domain.Settlement{
			GroupID: group.ID, FromUserID: owner.Id, ToUserID: member.Id, AmountMinor: 300,
			SettledOn: time.Now().Add(-time.Hour),
		})
		if err != nil {
			t.Fatal(err)
		}
		return value
	}

	legacy := create()
	if legacy.ApprovalRequired {
		t.Fatal("settlement created while approval was disabled must retain that rule snapshot")
	}
	group.SettlementApprovalEnabled = true
	if err := f.stores.Groups.Update(ctx, group); err != nil {
		t.Fatal(err)
	}
	legacyPage, err := f.service.ListAttachments(ctx, owner.Id, application.AttachmentTarget{Type: "settlement", ID: legacy.ID})
	if err != nil || !legacyPage.Access.CanUpload {
		t.Fatalf("changing the group setting must not alter legacy settlement attachment access: %#v (%v)", legacyPage.Access, err)
	}

	managed := create()
	if !managed.ApprovalRequired || managed.Status != "pending" {
		t.Fatalf("new settlement should be captured by the enabled review rule: %#v", managed)
	}
	reviews, err := f.service.ListSettlementReviews(ctx, member.Id, group.ID)
	if err != nil || len(reviews) != 1 {
		t.Fatalf("recipient should be able to find the pending settlement review: %#v (%v)", reviews, err)
	}
	if _, err = f.service.RespondSettlementReview(ctx, member.Id, reviews[0].ID, "accept", ""); err != nil {
		t.Fatal(err)
	}

	managedPage, err := f.service.ListAttachments(ctx, owner.Id, application.AttachmentTarget{Type: "settlement", ID: managed.ID})
	if err != nil {
		t.Fatal(err)
	}
	if !managedPage.Access.CanRead || managedPage.Access.CanUpload || managedPage.Access.CanDelete || managedPage.Access.CanRename || !managedPage.Access.ModificationRequiresReview {
		t.Fatalf("approved settlement attachments must remain readable but reject direct mutations: %#v", managedPage.Access)
	}
	if _, err = f.service.UploadAttachment(ctx, owner.Id, application.AttachmentTarget{Type: "settlement", ID: managed.ID, RequestID: "direct-settlement-attachment"}, "receipt.pdf", []byte("%PDF-1.7")); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("direct upload must not bypass settlement confirmation: %v", err)
	}
}
