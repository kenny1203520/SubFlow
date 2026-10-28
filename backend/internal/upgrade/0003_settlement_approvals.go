package upgrade

import (
	"fmt"

	"ganigeorgiev.com/pocketbase/pocketbase/core"

	pbadapter "subflow/internal/adapters/pocketbase"
)

const (
	SettlementApprovalsID       = "0003_settlement_approvals"
	SettlementApprovalsChecksum = "sha256:2e1712085079711a05a48fd3bfa8aeea0546702b1870244f4b5bf28eed0c622b"
)

func settlementApprovals() Migration {
	return Migration{
		ID:       SettlementApprovalsID,
		Checksum: SettlementApprovalsChecksum,
		Requires: []string{AuditIDLimitID},
		Precondition: func(app core.App) error {
			for _, name := range []string{pbadapter.CollectionGroups, pbadapter.CollectionSettlements} {
				if _, err := app.FindCollectionByNameOrId(name); err != nil {
					return fmt.Errorf("required collection %s is missing", name)
				}
			}
			return nil
		},
		Up: func(app core.App) error {
			if err := pbadapter.EnsureSettlementApprovalSchema(app); err != nil {
				return err
			}
			settlements, err := app.FindRecordsByFilter(pbadapter.CollectionSettlements, "", "", 0, 0, nil)
			if err != nil {
				return err
			}
			for _, settlement := range settlements {
				changed := false
				if settlement.GetString("status") == "" {
					settlement.Set("status", "confirmed")
					changed = true
				}
				if settlement.GetInt("version") < 1 {
					settlement.Set("version", 1)
					changed = true
				}
				if changed {
					if err := app.Save(settlement); err != nil {
						return err
					}
				}
			}
			return nil
		},
		Verify: func(app core.App) error {
			groups, err := app.FindCollectionByNameOrId(pbadapter.CollectionGroups)
			if err != nil {
				return err
			}
			if _, ok := groups.Fields.GetByName("settlement_approval_enabled").(*core.BoolField); !ok {
				return fmt.Errorf("group settlement approval setting is missing")
			}
			settlements, err := app.FindCollectionByNameOrId(pbadapter.CollectionSettlements)
			if err != nil {
				return err
			}
			if _, ok := settlements.Fields.GetByName("status").(*core.SelectField); !ok {
				return fmt.Errorf("settlement status is missing")
			}
			if _, ok := settlements.Fields.GetByName("approval_required").(*core.BoolField); !ok {
				return fmt.Errorf("settlement approval rule is missing")
			}
			if _, ok := settlements.Fields.GetByName("version").(*core.NumberField); !ok {
				return fmt.Errorf("settlement version is missing")
			}
			if _, err := app.FindCollectionByNameOrId(pbadapter.CollectionSettlementReviews); err != nil {
				return fmt.Errorf("settlement review collection is missing")
			}
			records, err := app.FindRecordsByFilter(pbadapter.CollectionSettlements, "", "", 0, 0, nil)
			if err != nil {
				return err
			}
			for _, settlement := range records {
				if settlement.GetString("status") == "" || settlement.GetInt("version") < 1 {
					return fmt.Errorf("legacy settlement %s was not initialized", settlement.Id)
				}
			}
			return nil
		},
	}
}
