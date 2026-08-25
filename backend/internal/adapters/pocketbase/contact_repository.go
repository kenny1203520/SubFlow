package pocketbase

import (
	"context"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"subflow/internal/domain"
	"subflow/internal/ports"
)

func (r *Repository) CreateContact(ctx context.Context, value *domain.Contact) error {
	record, err := newRecord(r.app(ctx), CollectionContacts)
	if err != nil {
		return err
	}
	writeContact(record, value)
	if err = r.app(ctx).Save(record); err != nil {
		return mapError(err)
	}
	value.ID = record.Id
	hydrateTimes(record, &value.CreatedAt, &value.UpdatedAt)
	return nil
}

func (r *Repository) GetContact(ctx context.Context, id string) (*domain.Contact, error) {
	record, err := r.app(ctx).FindRecordById(CollectionContacts, id)
	if err != nil {
		return nil, mapError(err)
	}
	return contactFrom(record), nil
}

func (r *Repository) ListContacts(ctx context.Context, ownerID string, request ports.PageRequest) (ports.Page[domain.Contact], error) {
	records, err := r.app(ctx).FindRecordsByFilter(CollectionContacts, "owner={:owner}", "name,email", 0, 0, dbx.Params{"owner": ownerID})
	if err != nil {
		return ports.Page[domain.Contact]{}, mapError(err)
	}
	items := make([]domain.Contact, 0, len(records))
	for _, record := range records {
		items = append(items, *contactFrom(record))
	}
	return slicePage(items, request), nil
}

func (r *Repository) UpdateContact(ctx context.Context, value *domain.Contact) error {
	record, err := r.app(ctx).FindRecordById(CollectionContacts, value.ID)
	if err != nil {
		return mapError(err)
	}
	writeContact(record, value)
	if err = r.app(ctx).Save(record); err != nil {
		return mapError(err)
	}
	hydrateTimes(record, &value.CreatedAt, &value.UpdatedAt)
	return nil
}

func (r *Repository) DeleteContact(ctx context.Context, id string) error {
	record, err := r.app(ctx).FindRecordById(CollectionContacts, id)
	if err != nil {
		return mapError(err)
	}
	return mapError(r.app(ctx).Delete(record))
}

func writeContact(record *core.Record, value *domain.Contact) {
	record.Set("owner", value.OwnerID)
	record.Set("name", strings.TrimSpace(value.Name))
	record.Set("email", strings.ToLower(strings.TrimSpace(value.Email)))
}

func contactFrom(record *core.Record) *domain.Contact {
	value := &domain.Contact{ID: record.Id, OwnerID: record.GetString("owner"), Name: record.GetString("name"), Email: strings.ToLower(record.GetString("email"))}
	hydrateTimes(record, &value.CreatedAt, &value.UpdatedAt)
	return value
}
