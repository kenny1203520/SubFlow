package application

import "context"

// auditRequestMetaKey is an unexported context key, mirroring the txKey{}
// pattern already used by the PocketBase adapter for transaction handles
// (see backend/internal/adapters/pocketbase/repository.go).
type auditRequestMetaKey struct{}

// AuditRequestMeta carries the caller identity that audit() records
// alongside every event. The HTTP layer attaches it once per request via
// WithAuditRequestMeta; nothing else needs to thread it through call sites
// because context values propagate to every context derived from ctx,
// including the transaction context s.Stores.Transactions.Within hands to
// callbacks.
type AuditRequestMeta struct {
	IP        string
	UserAgent string
}

// WithAuditRequestMeta returns a context carrying meta for audit() to read.
func WithAuditRequestMeta(ctx context.Context, meta AuditRequestMeta) context.Context {
	return context.WithValue(ctx, auditRequestMetaKey{}, meta)
}

// auditRequestMetaFrom reads back what WithAuditRequestMeta attached, or the
// zero value for contexts that never went through the HTTP layer (background
// jobs like PostDueSubscriptions run from context.Background()).
func auditRequestMetaFrom(ctx context.Context) AuditRequestMeta {
	if meta, ok := ctx.Value(auditRequestMetaKey{}).(AuditRequestMeta); ok {
		return meta
	}
	return AuditRequestMeta{}
}
