package pocketbase

import (
	"testing"

	"github.com/pocketbase/pocketbase/tests"

	"subflow/internal/ports"
)

func TestFreshSchemaAndPortContracts(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	if err := EnsureSchema(app); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{CollectionGroups, CollectionMembers, CollectionInvitations, CollectionSubscriptions, CollectionExpenses} {
		if _, err := app.FindCollectionByNameOrId(name); err != nil {
			t.Fatalf("missing %s: %v", name, err)
		}
	}

	stores := NewStores(app)
	var _ ports.GroupRepository = stores.Groups
	var _ ports.MembershipRepository = stores.Memberships
	var _ ports.InvitationRepository = stores.Invitations
	var _ ports.SubscriptionRepository = stores.Subscriptions
	var _ ports.ExpenseRepository = stores.Expenses
	var _ ports.UserDirectory = stores.Users
	var _ ports.TransactionManager = stores.Transactions
}
