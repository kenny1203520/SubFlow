package application_test

import (
	"context"
	"errors"
	"testing"

	"subflow/internal/adapters"
	"subflow/internal/application"
	"subflow/internal/domain"
	"subflow/internal/ports"
)

var errProviderUnlink = errors.New("provider unlink failed")
var errProviderAudit = errors.New("provider audit failed")

type unlinkTxKey struct{}

type unlinkUsers struct {
	ports.UserDirectory
	unlinked bool
	err      error
}

func (u *unlinkUsers) UnlinkExternalAuth(context.Context, string, string) error {
	if u.err != nil {
		return u.err
	}
	u.unlinked = true
	return nil
}

type unlinkAuditRepository struct {
	ports.AuditRepository
	logs      []domain.AuditLog
	failCount int
	failEvery bool
}

func (r *unlinkAuditRepository) Create(ctx context.Context, log *domain.AuditLog) error {
	if r.failEvery || r.failCount > 0 {
		if r.failCount > 0 {
			r.failCount--
		}
		return errProviderAudit
	}
	copy := *log
	copy.Hash = ""
	if inTransaction, _ := ctx.Value(unlinkTxKey{}).(bool); inTransaction {
		copy.Scope = "transaction"
	}
	r.logs = append(r.logs, copy)
	return nil
}

type unlinkTransactionManager struct {
	users  *unlinkUsers
	audits *unlinkAuditRepository
}

func (m *unlinkTransactionManager) Within(ctx context.Context, fn func(context.Context) error) error {
	wasUnlinked := m.users.unlinked
	logCount := len(m.audits.logs)
	err := fn(context.WithValue(ctx, unlinkTxKey{}, true))
	if err != nil {
		m.users.unlinked = wasUnlinked
		m.audits.logs = m.audits.logs[:logCount]
	}
	return err
}

func newUnlinkTestService(users *unlinkUsers, audits *unlinkAuditRepository) *application.Service {
	return application.New(adapters.Stores{
		Users: users, Audits: audits,
		Transactions: &unlinkTransactionManager{users: users, audits: audits},
	}, "provider-unlink-test-key")
}

func TestUnlinkProviderWritesSuccessAuditInUnlinkTransaction(t *testing.T) {
	users, audits := &unlinkUsers{}, &unlinkAuditRepository{}
	service := newUnlinkTestService(users, audits)

	if err := service.UnlinkProvider(context.Background(), "user-1", "google"); err != nil {
		t.Fatal(err)
	}
	if !users.unlinked {
		t.Fatal("expected provider unlink to commit")
	}
	if len(audits.logs) != 1 || audits.logs[0].Outcome != "success" || audits.logs[0].Scope != "transaction" {
		t.Fatalf("expected success audit in transaction, got %#v", audits.logs)
	}
}

func TestUnlinkProviderAuditsOperationFailureOutsideTransaction(t *testing.T) {
	users, audits := &unlinkUsers{err: errProviderUnlink}, &unlinkAuditRepository{}
	service := newUnlinkTestService(users, audits)

	err := service.UnlinkProvider(context.Background(), "user-1", "google")
	if !errors.Is(err, errProviderUnlink) {
		t.Fatalf("expected original unlink error, got %v", err)
	}
	if users.unlinked {
		t.Fatal("failed provider unlink must not commit")
	}
	if len(audits.logs) != 1 || audits.logs[0].Outcome != "failure" || audits.logs[0].Scope == "transaction" {
		t.Fatalf("expected failure audit outside transaction, got %#v", audits.logs)
	}
	if audits.logs[0].Summary == "" {
		t.Fatal("expected a fixed failure summary")
	}
}

func TestUnlinkProviderRollsBackWhenSuccessAuditFailsThenRecordsFailure(t *testing.T) {
	users, audits := &unlinkUsers{}, &unlinkAuditRepository{failCount: 1}
	service := newUnlinkTestService(users, audits)

	err := service.UnlinkProvider(context.Background(), "user-1", "google")
	if !errors.Is(err, errProviderAudit) {
		t.Fatalf("expected success audit failure to remain visible, got %v", err)
	}
	if users.unlinked {
		t.Fatal("provider unlink should roll back when its success audit cannot be committed")
	}
	if len(audits.logs) != 1 || audits.logs[0].Outcome != "failure" || audits.logs[0].Scope == "transaction" {
		t.Fatalf("expected the outside failure audit to be retained, got %#v", audits.logs)
	}
}

func TestUnlinkProviderPreservesOperationAndAuditErrors(t *testing.T) {
	users, audits := &unlinkUsers{err: errProviderUnlink}, &unlinkAuditRepository{failEvery: true}
	service := newUnlinkTestService(users, audits)

	err := service.UnlinkProvider(context.Background(), "user-1", "google")
	if !errors.Is(err, errProviderUnlink) || !errors.Is(err, domain.ErrAuditUnavailable) {
		t.Fatalf("expected operation and audit errors, got %v", err)
	}
}
