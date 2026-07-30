package adapters

import (
	"fmt"
	"strings"

	"github.com/pocketbase/pocketbase/core"

	pbadapter "subflow/internal/adapters/pocketbase"
	"subflow/internal/ports"
)

type Stores struct {
	Groups        ports.GroupRepository
	Memberships   ports.MembershipRepository
	Invitations   ports.InvitationRepository
	Subscriptions ports.SubscriptionRepository
	Expenses      ports.ExpenseRepository
	Users         ports.UserDirectory
	Transactions  ports.TransactionManager
}

func New(driver string, app core.App) (Stores, error) {
	switch strings.ToLower(strings.TrimSpace(driver)) {
	case "", "pocketbase":
		s := pbadapter.NewStores(app)
		return Stores{s.Groups, s.Memberships, s.Invitations, s.Subscriptions, s.Expenses, s.Users, s.Transactions}, nil
	case "postgres", "mysql":
		return Stores{}, fmt.Errorf("SUBFLOW_DATA_DRIVER %q is reserved but not implemented", driver)
	default:
		return Stores{}, fmt.Errorf("unsupported SUBFLOW_DATA_DRIVER %q", driver)
	}
}
