package pocketbase

import (
	"context"
	"io"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"github.com/pocketbase/pocketbase/tools/types"
	"subflow/internal/domain"
)

func attachmentFrom(r *core.Record) domain.Attachment {
	return domain.Attachment{ID: r.Id, GroupID: r.GetString("group"), OwnerID: r.GetString("owner"), TargetType: r.GetString("target_type"), TargetID: r.GetString("target_id"), BillingAt: r.GetDateTime("billing_at").Time(), UploadedBy: r.GetString("uploaded_by"), FileName: r.GetString("file_name"), MediaType: r.GetString("media_type"), Size: int64(r.GetFloat("file_size")), CreatedAt: r.GetDateTime("created").Time()}
}

func (r *AttachmentRepo) Create(ctx context.Context, value *domain.Attachment, content []byte) error {
	app := r.app(ctx)
	file, err := filesystem.NewFileFromBytes(content, value.FileName)
	if err != nil {
		return err
	}
	rec, err := newRecord(app, CollectionAttachments)
	if err != nil {
		return err
	}
	rec.Set("group", value.GroupID)
	rec.Set("owner", value.OwnerID)
	rec.Set("target_type", value.TargetType)
	rec.Set("target_id", value.TargetID)
	rec.Set("billing_at", value.BillingAt)
	rec.Set("uploaded_by", value.UploadedBy)
	rec.Set("media_type", value.MediaType)
	rec.Set("file_name", value.FileName)
	rec.Set("file_size", len(content))
	rec.Set("request_id", value.RequestID)
	rec.Set("file", file)
	if err = app.Save(rec); err != nil {
		return err
	}
	value.ID = rec.Id
	value.Size = int64(len(content))
	value.CreatedAt = rec.GetDateTime("created").Time()
	return nil
}
func (r *AttachmentRepo) Get(ctx context.Context, id string) (*domain.Attachment, error) {
	rec, err := r.app(ctx).FindRecordById(CollectionAttachments, id)
	if err != nil {
		return nil, mapError(err)
	}
	v := attachmentFrom(rec)
	return &v, nil
}
func (r *AttachmentRepo) Rename(ctx context.Context, id, name string) error {
	record, err := r.app(ctx).FindRecordById(CollectionAttachments, id)
	if err != nil {
		return mapError(err)
	}
	record.Set("file_name", name)
	return r.app(ctx).Save(record)
}
func (r *AttachmentRepo) FindByRequest(ctx context.Context, userID, requestID string) (*domain.Attachment, error) {
	rec, err := r.app(ctx).FindFirstRecordByFilter(CollectionAttachments, "uploaded_by={:user} && request_id={:request}", dbx.Params{"user": userID, "request": requestID})
	if err != nil {
		return nil, mapError(err)
	}
	v := attachmentFrom(rec)
	return &v, nil
}
func (r *AttachmentRepo) List(ctx context.Context, typ, id string, billing time.Time) ([]domain.Attachment, error) {
	params := dbx.Params{"type": typ, "target": id}
	filter := "target_type={:type} && target_id={:target}"
	if !billing.IsZero() {
		date, err := types.ParseDateTime(billing)
		if err != nil {
			return nil, err
		}
		filter += " && billing_at={:billing}"
		params["billing"] = date
	}
	records, err := r.app(ctx).FindRecordsByFilter(CollectionAttachments, filter, "created", 0, 0, params)
	if err != nil {
		return nil, err
	}
	items := make([]domain.Attachment, 0, len(records))
	for _, rec := range records {
		items = append(items, attachmentFrom(rec))
	}
	return items, nil
}
func (r *AttachmentRepo) ListGroup(ctx context.Context, groupID string) ([]domain.Attachment, error) {
	records, err := r.app(ctx).FindRecordsByFilter(CollectionAttachments, "group={:group}", "", 0, 0, dbx.Params{"group": groupID})
	if err != nil {
		return nil, err
	}
	items := make([]domain.Attachment, 0, len(records))
	for _, rec := range records {
		items = append(items, attachmentFrom(rec))
	}
	return items, nil
}
func (r *AttachmentRepo) Read(ctx context.Context, id string) ([]byte, error) {
	rec, err := r.app(ctx).FindRecordById(CollectionAttachments, id)
	if err != nil {
		return nil, mapError(err)
	}
	fs, err := r.app(ctx).NewFilesystem()
	if err != nil {
		return nil, err
	}
	defer fs.Close()
	reader, err := fs.GetReader(rec.BaseFilesPath() + "/" + rec.GetString("file"))
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}
func (r *AttachmentRepo) Delete(ctx context.Context, id string) error {
	rec, err := r.app(ctx).FindRecordById(CollectionAttachments, id)
	if err != nil {
		return mapError(err)
	}
	return r.app(ctx).Delete(rec)
}
func (r *AttachmentRepo) DeleteTarget(ctx context.Context, typ, id string) error {
	records, err := r.app(ctx).FindRecordsByFilter(CollectionAttachments, "target_type={:type} && target_id={:target}", "", 0, 0, dbx.Params{"type": typ, "target": id})
	if err != nil {
		return err
	}
	for _, rec := range records {
		if err = r.app(ctx).Delete(rec); err != nil {
			return err
		}
	}
	return nil
}
func (r *AttachmentRepo) DeletePeriod(ctx context.Context, typ, id string, billing time.Time) error {
	date, err := types.ParseDateTime(billing)
	if err != nil {
		return err
	}
	records, err := r.app(ctx).FindRecordsByFilter(CollectionAttachments, "target_type={:type} && target_id={:target} && billing_at={:billing}", "", 0, 0, dbx.Params{"type": typ, "target": id, "billing": date})
	if err != nil {
		return err
	}
	for _, rec := range records {
		if err = r.app(ctx).Delete(rec); err != nil {
			return err
		}
	}
	return nil
}
func (r *AttachmentRepo) ReassignTarget(ctx context.Context, fromType, fromID, toType, toID string) error {
	records, err := r.app(ctx).FindRecordsByFilter(CollectionAttachments, "target_type={:type} && target_id={:target}", "", 0, 0, dbx.Params{"type": fromType, "target": fromID})
	if err != nil {
		return err
	}
	for _, rec := range records {
		rec.Set("target_type", toType)
		rec.Set("target_id", toID)
		if err = r.app(ctx).Save(rec); err != nil {
			return err
		}
	}
	return nil
}
func (r *AttachmentRepo) ReassignPeriod(ctx context.Context, fromType, fromID string, billing time.Time, toType, toID string) error {
	date, err := types.ParseDateTime(billing)
	if err != nil {
		return err
	}
	records, err := r.app(ctx).FindRecordsByFilter(CollectionAttachments, "target_type={:type} && target_id={:target} && billing_at={:billing}", "", 0, 0, dbx.Params{"type": fromType, "target": fromID, "billing": date})
	if err != nil {
		return err
	}
	for _, rec := range records {
		rec.Set("target_type", toType)
		rec.Set("target_id", toID)
		rec.Set("billing_at", time.Time{})
		if err = r.app(ctx).Save(rec); err != nil {
			return err
		}
	}
	return nil
}
