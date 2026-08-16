package adapters

import (
	"fmt"
	"strings"

	"github.com/pocketbase/pocketbase/core"

	pbadapter "subflow/internal/adapters/pocketbase"
	"subflow/internal/ports"
)

type Stores struct {
	Groups             ports.GroupRepository
	Memberships        ports.MembershipRepository
	Invitations        ports.InvitationRepository
	OwnershipTransfers ports.OwnershipTransferRepository
	MemberTransfers    ports.MemberTransferRepository
	Notifications      ports.NotificationRepository
	Subscriptions      ports.SubscriptionRepository
	Expenses           ports.ExpenseRepository
	Settlements        ports.SettlementRepository
	Categories         ports.CategoryRepository
	ExchangeRates      ports.ExchangeRateRepository
	Roles              ports.RoleRepository
	Audits             ports.AuditRepository
	Users              ports.UserDirectory
	Settings           ports.SystemSettingsRepository
	Transactions       ports.TransactionManager
}

func New(driver string, app core.App) (Stores, error) {
	switch strings.ToLower(strings.TrimSpace(driver)) {
	case "", "pocketbase":
		s := pbadapter.NewStores(app)
		return Stores{s.Groups, s.Memberships, s.Invitations, s.OwnershipTransfers, s.MemberTransfers, s.Notifications, s.Subscriptions, s.Expenses, s.Settlements, s.Categories, s.ExchangeRates, s.Roles, s.Audits, s.Users, s.Settings, s.Transactions}, nil
	case "postgres", "mysql":
		return Stores{}, fmt.Errorf("SUBFLOW_DATA_DRIVER %q is reserved but not implemented", driver)
	default:
		return Stores{}, fmt.Errorf("unsupported SUBFLOW_DATA_DRIVER %q", driver)
	}
}
