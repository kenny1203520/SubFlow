package application_test

import (
	"context"
	"errors"
	"subflow/internal/application"
	"subflow/internal/domain"
	"subflow/internal/ports"
	"testing"
	"time"
)

var errIncomeAuditUnavailable = errors.New("audit unavailable")

type unavailableIncomeAudit struct{ ports.AuditRepository }

func (r unavailableIncomeAudit) Create(context.Context, *domain.AuditLog) error {
	return errIncomeAuditUnavailable
}

func TestIncomePatchPreservesOmissionsAndAcceptsZeroAndEmpty(t *testing.T) {
	for _, group := range []bool{false, true} {
		t.Run(map[bool]string{false: "personal", true: "group"}[group], func(t *testing.T) {
			f := newMemberTransferFixture(t)
			ctx := context.Background()
			input := domain.Income{Title: "Salary", AmountMinor: 10000, Currency: domain.CurrencyTWD, ReceivedOn: time.Now().UTC().Truncate(time.Second), Notes: "remove me", SplitMode: domain.SplitEqual}
			var value *domain.Income
			var err error
			if group {
				input.GroupID = f.groupID
				value, err = f.service.CreateGroupIncome(ctx, f.ownerID, input)
			} else {
				value, err = f.service.CreateIncome(ctx, f.ownerID, input)
			}
			if err != nil {
				t.Fatal(err)
			}
			empty := ""
			updated, err := f.service.PatchIncome(ctx, f.ownerID, "", value.ID, application.IncomePatch{Notes: &empty})
			if err != nil {
				t.Fatal(err)
			}
			if updated.Notes != "" || updated.AmountMinor != 10000 || !updated.ReceivedOn.Equal(value.ReceivedOn) {
				t.Fatalf("omitted fields changed: %#v", updated)
			}
			zero := int64(0)
			splits := []*domain.IncomeSplit{{BaseSplit: domain.BaseSplit{UserID: f.ownerID}}}
			updated, err = f.service.PatchIncome(ctx, f.ownerID, "", value.ID, application.IncomePatch{AmountMinor: &zero, CategoryID: &empty, Splits: &splits})
			if err != nil {
				t.Fatal(err)
			}
			if updated.AmountMinor != 0 || updated.BaseAmountMinor != 0 || updated.CategoryID != "" {
				t.Fatalf("zero/clear ignored: %#v", updated)
			}
			if _, err = f.service.PatchIncome(ctx, f.outsiderID, "", value.ID, application.IncomePatch{Notes: &empty}); !errors.Is(err, domain.ErrForbidden) {
				t.Fatalf("outsider: %v", err)
			}
		})
	}
}

func TestIncomeAuditFailureRollsBackCreateUpdateDelete(t *testing.T) {
	for _, group := range []bool{false, true} {
		t.Run(map[bool]string{false: "personal", true: "group"}[group], func(t *testing.T) {
			f := newMemberTransferFixture(t)
			ctx := context.Background()
			input := domain.Income{Title: "Salary", AmountMinor: 10000, Currency: domain.CurrencyTWD, ReceivedOn: time.Now().UTC().Truncate(time.Second)}
			create := func() (*domain.Income, error) {
				if group {
					input.GroupID = f.groupID
					return f.service.CreateGroupIncome(ctx, f.ownerID, input)
				}
				return f.service.CreateIncome(ctx, f.ownerID, input)
			}
			value, err := create()
			if err != nil {
				t.Fatal(err)
			}
			f.service.Stores.Audits = unavailableIncomeAudit{f.stores.Audits}
			if _, err = create(); !errors.Is(err, errIncomeAuditUnavailable) {
				t.Fatalf("create: %v", err)
			}
			note := "changed"
			if _, err = f.service.PatchIncome(ctx, f.ownerID, "", value.ID, application.IncomePatch{Notes: &note}); !errors.Is(err, errIncomeAuditUnavailable) {
				t.Fatalf("update: %v", err)
			}
			if err = f.service.DeleteIncomeInScope(ctx, f.ownerID, "", value.ID); !errors.Is(err, errIncomeAuditUnavailable) {
				t.Fatalf("delete: %v", err)
			}
			stored, err := f.stores.Incomes.Get(ctx, value.ID)
			if err != nil || stored.Notes != "" {
				t.Fatalf("write escaped rollback: %#v %v", stored, err)
			}
			var records ports.Page[domain.Income]
			if group {
				records, err = f.stores.Incomes.List(ctx, f.groupID, ports.PageRequest{})
			} else {
				records, err = f.stores.Incomes.ListPersonal(ctx, f.ownerID, ports.PageRequest{})
			}
			if err != nil || records.TotalItems != 1 {
				t.Fatalf("create escaped rollback: %#v %v", records, err)
			}
		})
	}
}

func TestIncomeHistoryAndRouteScopeAreEnforced(t *testing.T) {
	f := newMemberTransferFixture(t)
	ctx := context.Background()
	f.service.Now = func() time.Time { return time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC) }
	input := domain.Income{GroupID: f.groupID, Title: "Historical", AmountMinor: 10000, Currency: domain.CurrencyTWD, ReceivedOn: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)}
	if _, err := f.service.CreateGroupIncome(ctx, f.fromID, input); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("historical create: %v", err)
	}
	value, err := f.service.CreateGroupIncome(ctx, f.ownerID, input)
	if err != nil {
		t.Fatal(err)
	}
	note := "changed"
	if _, err = f.service.PatchIncome(ctx, f.fromID, "", value.ID, application.IncomePatch{Notes: &note}); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("historical update: %v", err)
	}
	if _, err = f.service.PatchIncome(ctx, f.ownerID, "different-group", value.ID, application.IncomePatch{Notes: &note}); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("route mismatch: %v", err)
	}
	if err = f.service.DeleteIncomeInScope(ctx, f.ownerID, "different-group", value.ID); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("delete mismatch: %v", err)
	}
	logs, err := f.stores.Audits.List(ctx, f.groupID, ports.AuditQuery{PageRequest: ports.PageRequest{PerPage: 100}, Action: "income.created", Outcome: "failure"})
	if err != nil || len(logs.Items) == 0 {
		t.Fatalf("missing failure audit: %v", err)
	}
}
