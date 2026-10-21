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
	NotificationEmails ports.NotificationEmailRepository
	Subscriptions      ports.SubscriptionRepository
	Expenses           ports.ExpenseRepository
	Incomes            ports.IncomeRepository
	RecurringIncomes   ports.RecurringIncomeRepository
	Settlements        ports.SettlementRepository
	Attachments        ports.AttachmentRepository
	Categories         ports.CategoryRepository
	ExchangeRates      ports.ExchangeRateRepository
	Roles              ports.RoleRepository
	Audits             ports.AuditRepository
	Shares             ports.ShareRepository
	Contacts           ports.ContactRepository
	Users              ports.UserDirectory
	Settings           ports.SystemSettingsRepository
	Transactions       ports.TransactionManager
}

func New(driver string, app core.App) (Stores, error) {
	switch strings.ToLower(strings.TrimSpace(driver)) {
	case "", "pocketbase":
		s := pbadapter.NewStores(app)
		return Stores{Groups: s.Groups, Memberships: s.Memberships, Invitations: s.Invitations, OwnershipTransfers: s.OwnershipTransfers, MemberTransfers: s.MemberTransfers, Notifications: s.Notifications, NotificationEmails: s.NotificationEmails, Subscriptions: s.Subscriptions, Expenses: s.Expenses, Incomes: s.Incomes, RecurringIncomes: s.RecurringIncomes, Settlements: s.Settlements, Attachments: s.Attachments, Categories: s.Categories, ExchangeRates: s.ExchangeRates, Roles: s.Roles, Audits: s.Audits, Shares: s.Shares, Contacts: s.Contacts, Users: s.Users, Settings: s.Settings, Transactions: s.Transactions}, nil
	case "postgres", "mysql":
		return Stores{}, fmt.Errorf("SUBFLOW_DATA_DRIVER %q is reserved but not implemented", driver)
	default:
		return Stores{}, fmt.Errorf("unsupported SUBFLOW_DATA_DRIVER %q", driver)
	}
}
