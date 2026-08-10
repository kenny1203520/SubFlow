package pocketbase

import (
	"strings"
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

func TestEnsureSchemaWithSetupURLReturnsOneTimeLink(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	link, err := EnsureSchemaWithSetupURL(app, "http://localhost:5173")
	if err != nil || !strings.HasPrefix(link, "http://localhost:5173/setup?token=") {
		t.Fatalf("first setup link = %q, %v", link, err)
	}
	settings, err := app.FindFirstRecordByFilter(CollectionSystemSettings, "key='primary'", nil)
	if err != nil || settings.GetString("setup_secret_hash") == "" || strings.Contains(settings.GetString("setup_secret_hash"), link) {
		t.Fatalf("setup token must be stored only as a hash: %v", err)
	}
	if second, secondErr := EnsureSchemaWithSetupURL(app, "http://localhost:5173"); secondErr != nil || second != "" {
		t.Fatalf("second setup link = %q, %v", second, secondErr)
	}
}
