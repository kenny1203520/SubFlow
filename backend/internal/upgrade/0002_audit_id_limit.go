package upgrade

import (
	"fmt"

	"ganigeorgiev.com/pocketbase/pocketbase/core"
)

const (
	AuditIDLimitID       = "0002_audit_id_limit"
	AuditIDLimitChecksum = "sha256:1f903a9762cf4f19c870c500942015ee7f63bb25a53a15de6e6c6633a0d9ca2e"
	AuditResourceIDLimit = 64
)

func auditIdLimit() Migration {
	return Migration{
		ID:       AuditIDLimitID,
		Checksum: AuditIDLimitChecksum,
		Precondition: func(app core.App) error {
			collection, err := app.FindCollectionByNameOrId("audit_logs")
			if err != nil {
				return fmt.Errorf("audit log collection is missing")
			}
			if _, ok := collection.Fields.GetByName("resource_id").(*core.TextField); !ok {
				return fmt.Errorf("audit resource ID field is not a text field")
			}
			return nil
		},
		Up: func(app core.App) error {
			collection, err := app.FindCollectionByNameOrId("audit_logs")
			if err != nil {
				return fmt.Errorf("audit log collection is missing")
			}
			field, ok := collection.Fields.GetByName("resource_id").(*core.TextField)
			if !ok {
				return fmt.Errorf("audit resource ID field is not a text field")
			}
			if field.Max == 0 || field.Max >= AuditResourceIDLimit {
				return nil
			}
			field.Max = AuditResourceIDLimit
			return app.Save(collection)
		},
		Verify: func(app core.App) error {
			collection, err := app.FindCollectionByNameOrId("audit_logs")
			if err != nil {
				return fmt.Errorf("audit log collection is missing")
			}
			field, ok := collection.Fields.GetByName("resource_id").(*core.TextField)
			if !ok || (field.Max > 0 && field.Max < AuditResourceIDLimit) {
				return fmt.Errorf("audit resource ID limit was not expanded")
			}
			return nil
		},
	}
}
