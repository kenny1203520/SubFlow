package application_test

import (
	"context"
	"testing"
	"time"

	"subflow/internal/domain"
	"subflow/internal/ports"
)

func TestSettlementApprovalRequestCorrectionAndVoidLifecycle(t *testing.T) {
	f := newSettlementFixture(t)
	ctx := context.Background()
	group, err := f.stores.Groups.Get(ctx, f.groupID)
	if err != nil {
		t.Fatal(err)
	}
	group.SettlementApprovalEnabled = true
	if err = f.stores.Groups.Update(ctx, group); err != nil {
		t.Fatal(err)
	}

	create := func(from, to string, amount int64) *domain.Settlement {
		t.Helper()
		value, createErr := f.service.CreateSettlement(ctx, from, domain.Settlement{GroupID: f.groupID, FromUserID: from, ToUserID: to, AmountMinor: amount, SettledOn: time.Date(2026, time.August, 10, 0, 0, 0, 0, time.UTC)})
		if createErr != nil {
			t.Fatal(createErr)
		}
		return value
	}

	pending := create(f.memberID, f.otherID, 500)
	if pending.Status != "pending" || !pending.ApprovalRequired || domain.SettlementIsEffective(*pending) {
		t.Fatalf("expected new settlement to await approval and remain outside balances, got %#v", pending)
	}
	submitterReviews, err := f.service.ListSettlementReviews(ctx, f.memberID, f.groupID)
	if err != nil || len(submitterReviews) != 1 || submitterReviews[0].CanRespond {
		t.Fatalf("submitter should see the pending request without response controls, got %#v (%v)", submitterReviews, err)
	}
	if err = f.service.WithdrawSettlementReview(ctx, f.memberID, submitterReviews[0].ID); err != nil {
		t.Fatalf("creator should be able to withdraw their pending settlement: %v", err)
	}
	withdrawn, err := f.stores.Settlements.Get(ctx, pending.ID)
	if err != nil || withdrawn.Status != "withdrawn" || domain.SettlementIsEffective(*withdrawn) {
		t.Fatalf("withdrawn settlement must remain outside balances: %#v (%v)", withdrawn, err)
	}

	approved := create(f.memberID, f.otherID, 500)
	recipientReviews, err := f.service.ListSettlementReviews(ctx, f.otherID, f.groupID)
	if err != nil || len(recipientReviews) != 1 || !recipientReviews[0].CanRespond {
		t.Fatalf("recipient should receive an actionable confirmation, got %#v (%v)", recipientReviews, err)
	}
	if _, err = f.service.RespondSettlementReview(ctx, f.otherID, recipientReviews[0].ID, "accept", ""); err != nil {
		t.Fatalf("recipient should be able to confirm without an override reason: %v", err)
	}
	approved, err = f.stores.Settlements.Get(ctx, approved.ID)
	if err != nil || approved.Status != "confirmed" || !domain.SettlementIsEffective(*approved) {
		t.Fatalf("confirmed settlement should enter balances: %#v (%v)", approved, err)
	}

	change, err := f.service.UpdateSettlement(ctx, f.memberID, approved.ID, domain.Settlement{FromUserID: f.memberID, ToUserID: f.otherID, AmountMinor: 800, SettledOn: approved.SettledOn, Notes: "revised"})
	if err != nil || change.AmountMinor != 500 {
		t.Fatalf("an approved change should leave the original row effective while pending, got %#v (%v)", change, err)
	}
	recipientReviews, err = f.service.ListSettlementReviews(ctx, f.otherID, f.groupID)
	if err != nil || len(recipientReviews) != 1 || recipientReviews[0].Action != "change" {
		t.Fatalf("expected a separate pending change proposal, got %#v (%v)", recipientReviews, err)
	}
	if _, err = f.service.RespondSettlementReview(ctx, f.otherID, recipientReviews[0].ID, "accept", ""); err != nil {
		t.Fatal(err)
	}
	approved, err = f.stores.Settlements.Get(ctx, approved.ID)
	if err != nil || approved.AmountMinor != 800 {
		t.Fatalf("accepted change should update the effective settlement atomically: %#v (%v)", approved, err)
	}

	voidReview, err := f.service.RequestSettlementVoid(ctx, f.memberID, approved.ID, "duplicate entry")
	if err != nil {
		t.Fatal(err)
	}
	unchanged, err := f.stores.Settlements.Get(ctx, approved.ID)
	if err != nil || unchanged.Status != "confirmed" || unchanged.AmountMinor != 800 {
		t.Fatalf("void request must preserve the original accounting effect before decision: %#v (%v)", unchanged, err)
	}
	if _, err = f.service.RespondSettlementReview(ctx, f.otherID, voidReview.ID, "reject", "keep the original record"); err != nil {
		t.Fatal(err)
	}
	unchanged, _ = f.stores.Settlements.Get(ctx, approved.ID)
	if unchanged.Status != "confirmed" {
		t.Fatalf("rejecting a void request must preserve the original record, got %#v", unchanged)
	}
	voidReview, err = f.service.RequestSettlementVoid(ctx, f.memberID, approved.ID, "confirmed duplicate")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.service.RespondSettlementReview(ctx, f.otherID, voidReview.ID, "accept", ""); err != nil {
		t.Fatal(err)
	}
	voided, err := f.stores.Settlements.Get(ctx, approved.ID)
	if err != nil || voided.Status != "voided" || domain.SettlementIsEffective(*voided) {
		t.Fatalf("accepted void should remove the settlement from balances: %#v (%v)", voided, err)
	}

	managerSettlement := create(f.ownerID, f.memberID, 250)
	managerReviews, err := f.service.ListSettlementReviews(ctx, f.ownerID, f.groupID)
	if err != nil || len(managerReviews) != 1 || !managerReviews[0].CanManage || managerReviews[0].SettlementID != managerSettlement.ID {
		t.Fatalf("settlement manager should see the pending request and their override role: %#v (%v)", managerReviews, err)
	}
	if _, err = f.service.RespondSettlementReview(ctx, f.ownerID, managerReviews[0].ID, "accept", ""); err != domain.ErrInvalid {
		t.Fatalf("manager override should require a reason, got %v", err)
	}
	if _, err = f.service.RespondSettlementReview(ctx, f.ownerID, managerReviews[0].ID, "accept", "I recorded this payment myself"); err != nil {
		t.Fatalf("manager should be able to confirm their outgoing settlement with attribution: %v", err)
	}
	successes, err := f.stores.Audits.List(ctx, f.groupID, ports.AuditQuery{PageRequest: ports.PageRequest{Page: 1, PerPage: 10}, Action: "settlement.review.accept", Outcome: "success"})
	if err != nil || len(successes.Items) == 0 {
		t.Fatalf("manager self-confirmation should write a success audit, got %#v (%v)", successes, err)
	}
	failures, err := f.stores.Audits.List(ctx, f.groupID, ports.AuditQuery{PageRequest: ports.PageRequest{Page: 1, PerPage: 10}, Action: "settlement.review.accept", Outcome: "failure"})
	if err != nil || len(failures.Items) == 0 {
		t.Fatalf("missing manager override reason should write a failure audit, got %#v (%v)", failures, err)
	}
}
