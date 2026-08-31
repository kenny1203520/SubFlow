package pocketbase

import (
	"reflect"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"

	"subflow/internal/domain"
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
	for _, name := range []string{CollectionGroups, CollectionMembers, CollectionInvitations, CollectionSubscriptions, CollectionExpenses, CollectionIncomes} {
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

func TestMergePermissionsBackfillsMissingEntries(t *testing.T) {
	merged, changed := mergePermissions([]string{"group.view", "ledger.expenses.write"}, []string{"group.view", "ledger.expenses.write", "ledger.records.historical_write"})
	if !changed {
		t.Fatal("expected missing permissions to be backfilled")
	}
	if want := []string{"group.view", "ledger.expenses.write", "ledger.records.historical_write"}; !reflect.DeepEqual(merged, want) {
		t.Fatalf("merged permissions = %#v, want %#v", merged, want)
	}
}

func TestShareSchemaKeepsCiphertextHidden(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	if err = EnsureSchema(app); err != nil {
		t.Fatal(err)
	}
	shares, err := app.FindCollectionByNameOrId(CollectionShares)
	if err != nil {
		t.Fatal(err)
	}
	field := shares.Fields.GetByName("token_ciphertext")
	textField, ok := field.(*core.TextField)
	if !ok || !textField.Hidden {
		t.Fatalf("token_ciphertext must be a hidden text field, got %#v", field)
	}
}

func TestIncomeSplitSchemaAndBackfill(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	if err = EnsureSchema(app); err != nil {
		t.Fatal(err)
	}
	incomeSplits, err := app.FindCollectionByNameOrId(CollectionIncomeSplits)
	if err != nil {
		t.Fatal(err)
	}
	incomeField, ok := incomeSplits.Fields.GetByName("income").(*core.RelationField)
	if !ok || !incomeField.Required || !incomeField.CascadeDelete {
		t.Fatalf("income_splits.income must be a required cascading relation: %#v", incomeSplits.Fields.GetByName("income"))
	}
	userField, ok := incomeSplits.Fields.GetByName("user").(*core.RelationField)
	if !ok || !userField.Required {
		t.Fatalf("income_splits.user must be a required relation: %#v", incomeSplits.Fields.GetByName("user"))
	}
	if !strings.Contains(strings.Join(incomeSplits.Indexes, "\n"), "idx_splits_income_user") {
		t.Fatalf("income_splits must have a unique income/user index: %#v", incomeSplits.Indexes)
	}

	users, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatal(err)
	}
	user := core.NewRecord(users)
	user.Set("email", "income-backfill@example.com")
	user.Set("name", "Income Backfill")
	user.SetPassword("correct-horse-battery-staple")
	if err = app.Save(user); err != nil {
		t.Fatal(err)
	}
	stores := NewStores(app)
	group := &domain.Group{Name: "Income backfill", Currency: domain.CurrencyTWD, OwnerID: user.Id}
	if err = stores.Groups.Create(t.Context(), group); err != nil {
		t.Fatal(err)
	}
	incomes, err := app.FindCollectionByNameOrId(CollectionIncomes)
	if err != nil {
		t.Fatal(err)
	}
	income := core.NewRecord(incomes)
	income.Set("group", group.ID)
	income.Set("earned_by", user.Id)
	income.Set("title", "Legacy income")
	income.Set("category", "Other")
	income.Set("amount_minor", 12345)
	income.Set("currency", "TWD")
	income.Set("received_on", "2026-08-01 00:00:00.000Z")
	if err = app.Save(income); err != nil {
		t.Fatal(err)
	}
	if err = backfillFinance(app); err != nil {
		t.Fatal(err)
	}
	income, err = app.FindRecordById(CollectionIncomes, income.Id)
	if err != nil {
		t.Fatal(err)
	}
	if income.GetString("base_currency") != "TWD" || int64(income.GetFloat("base_amount_minor")) != 12345 || income.GetString("split_mode") != "amount" || income.GetString("category_ref") == "" {
		t.Fatalf("income defaults were not backfilled: %#v", income)
	}
	splits, err := app.FindRecordsByFilter(CollectionIncomeSplits, "income={:income}", "", 0, 0, map[string]any{"income": income.Id})
	if err != nil {
		t.Fatal(err)
	}
	if len(splits) != 1 || splits[0].GetString("user") != user.Id || int64(splits[0].GetFloat("amount_minor")) != 12345 || int64(splits[0].GetFloat("base_amount_minor")) != 12345 {
		t.Fatalf("income split was not backfilled: %#v", splits)
	}
	if err = backfillFinance(app); err != nil {
		t.Fatal(err)
	}
	splits, err = app.FindRecordsByFilter(CollectionIncomeSplits, "income={:income}", "", 0, 0, map[string]any{"income": income.Id})
	if err != nil {
		t.Fatal(err)
	}
	if len(splits) != 1 {
		t.Fatalf("income backfill must be idempotent, got %d splits", len(splits))
	}
	splits[0].Set("base_amount_minor", 0)
	if err = app.Save(splits[0]); err != nil {
		t.Fatal(err)
	}
	if err = backfillFinance(app); err != nil {
		t.Fatal(err)
	}
	splits, err = app.FindRecordsByFilter(CollectionIncomeSplits, "income={:income}", "", 0, 0, map[string]any{"income": income.Id})
	if err != nil {
		t.Fatal(err)
	}
	if len(splits) != 1 || int64(splits[0].GetFloat("base_amount_minor")) != 12345 {
		t.Fatalf("income split base amount was not backfilled: %#v", splits)
	}
	if err = app.Delete(income); err != nil {
		t.Fatal(err)
	}
	splits, err = app.FindRecordsByFilter(CollectionIncomeSplits, "income={:income}", "", 0, 0, map[string]any{"income": income.Id})
	if err != nil {
		t.Fatal(err)
	}
	if len(splits) != 0 {
		t.Fatalf("income split rows must cascade-delete with their income: %#v", splits)
	}
}
