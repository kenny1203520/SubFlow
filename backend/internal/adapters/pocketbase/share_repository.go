package pocketbase

import (
	"context"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"subflow/internal/domain"
	"subflow/internal/ports"
)

func (r *Repository) CreateShare(ctx context.Context, value *domain.Share) error {
	record, err := newRecord(r.app(ctx), CollectionShares)
	if err != nil {
		return err
	}
	writeShare(record, value)
	if err = r.app(ctx).Save(record); err != nil {
		return err
	}
	value.ID = record.Id
	hydrateTimes(record, &value.CreatedAt, &value.UpdatedAt)
	return nil
}

func (r *Repository) GetShare(ctx context.Context, id string) (*domain.Share, error) {
	record, err := r.app(ctx).FindRecordById(CollectionShares, id)
	if err != nil {
		return nil, mapError(err)
	}
	return shareFrom(record), nil
}

func (r *Repository) GetShareByTokenHash(ctx context.Context, hash string) (*domain.Share, error) {
	record, err := r.app(ctx).FindFirstRecordByFilter(CollectionShares, "token_hash={:hash}", dbx.Params{"hash": hash})
	if err != nil {
		return nil, mapError(err)
	}
	return shareFrom(record), nil
}

func (r *Repository) ListShares(ctx context.Context, groupID, ownerID string, request ports.PageRequest) (ports.Page[domain.Share], error) {
	filter, params := "", dbx.Params{}
	if groupID != "" {
		filter, params = "group={:scope}", dbx.Params{"scope": groupID}
	}
	if ownerID != "" {
		filter, params = "owner={:scope}", dbx.Params{"scope": ownerID}
	}
	records, err := r.app(ctx).FindRecordsByFilter(CollectionShares, filter, "-created", 0, 0, params)
	if err != nil {
		return ports.Page[domain.Share]{}, err
	}
	items := make([]domain.Share, 0, len(records))
	for _, record := range records {
		items = append(items, *shareFrom(record))
	}
	return slicePage(items, request), nil
}

func (r *Repository) UpdateShare(ctx context.Context, value *domain.Share) error {
	record, err := r.app(ctx).FindRecordById(CollectionShares, value.ID)
	if err != nil {
		return mapError(err)
	}
	writeShare(record, value)
	if err = r.app(ctx).Save(record); err != nil {
		return err
	}
	hydrateTimes(record, &value.CreatedAt, &value.UpdatedAt)
	return nil
}

func (r *Repository) DeleteShare(ctx context.Context, id string) error {
	record, err := r.app(ctx).FindRecordById(CollectionShares, id)
	if err != nil {
		return mapError(err)
	}
	return r.app(ctx).Delete(record)
}

func (r *Repository) ReplaceShareViewers(ctx context.Context, shareID string, values []domain.ShareViewer) error {
	app := r.app(ctx)
	current, err := app.FindRecordsByFilter(CollectionShareViewers, "share={:share}", "", 0, 0, dbx.Params{"share": shareID})
	if err != nil {
		return err
	}
	for _, record := range current {
		if err = app.Delete(record); err != nil {
			return err
		}
	}
	for _, value := range values {
		record, createErr := newRecord(app, CollectionShareViewers)
		if createErr != nil {
			return createErr
		}
		record.Set("share", shareID)
		record.Set("user", value.UserID)
		if createErr = app.Save(record); createErr != nil {
			return createErr
		}
	}
	return nil
}

func (r *Repository) ListShareViewers(ctx context.Context, shareID string) ([]domain.ShareViewer, error) {
	records, err := r.app(ctx).FindRecordsByFilter(CollectionShareViewers, "share={:share}", "created", 0, 0, dbx.Params{"share": shareID})
	if err != nil {
		return nil, err
	}
	result := make([]domain.ShareViewer, 0, len(records))
	for _, record := range records {
		viewer := domain.ShareViewer{ID: record.Id, ShareID: shareID, UserID: record.GetString("user")}
		if user, userErr := r.GetUser(ctx, viewer.UserID); userErr == nil {
			viewer.Email, viewer.Name = user.Email, user.Name
		}
		result = append(result, viewer)
	}
	return result, nil
}

func writeShare(record *core.Record, value *domain.Share) {
	record.Set("group", value.GroupID)
	record.Set("owner", value.OwnerID)
	record.Set("name", strings.TrimSpace(value.Name))
	record.Set("token_hash", value.TokenHash)
	record.Set("access_mode", value.AccessMode)
	record.Set("password_hash", value.PasswordHash)
	record.Set("enabled", value.Enabled)
	record.Set("expires_at", value.ExpiresAt)
	record.Set("range_mode", value.RangeMode)
	record.Set("rolling_days", value.RollingDays)
	record.Set("starts_on", value.StartsOn)
	record.Set("ends_on", value.EndsOn)
	record.Set("show_summary", value.ShowSummary)
	record.Set("show_expenses", value.ShowExpenses)
	record.Set("show_subscriptions", value.ShowSubscriptions)
	record.Set("show_settlements", value.ShowSettlements)
	record.Set("show_identities", value.ShowIdentities)
	record.Set("show_notes", value.ShowNotes)
	record.Set("access_version", value.AccessVersion)
}

func shareFrom(record *core.Record) *domain.Share {
	value := &domain.Share{ID: record.Id, GroupID: record.GetString("group"), OwnerID: record.GetString("owner"), Name: record.GetString("name"), TokenHash: record.GetString("token_hash"), AccessMode: record.GetString("access_mode"), PasswordHash: record.GetString("password_hash"), Enabled: record.GetBool("enabled"), ExpiresAt: record.GetDateTime("expires_at").Time(), RangeMode: record.GetString("range_mode"), RollingDays: int(record.GetInt("rolling_days")), StartsOn: record.GetDateTime("starts_on").Time(), EndsOn: record.GetDateTime("ends_on").Time(), ShowSummary: record.GetBool("show_summary"), ShowExpenses: record.GetBool("show_expenses"), ShowSubscriptions: record.GetBool("show_subscriptions"), ShowSettlements: record.GetBool("show_settlements"), ShowIdentities: record.GetBool("show_identities"), ShowNotes: record.GetBool("show_notes"), AccessVersion: int(record.GetInt("access_version"))}
	hydrateTimes(record, &value.CreatedAt, &value.UpdatedAt)
	return value
}
