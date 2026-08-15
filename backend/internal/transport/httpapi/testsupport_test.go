package httpapi_test

import (
	"context"
	"net/url"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"

	"subflow/internal/adapters"
	"subflow/internal/adapters/pocketbase"
	"subflow/internal/application"
	"subflow/internal/domain"
	"subflow/internal/transport/httpapi"
)

// apiFixture wires a PocketBase test app + Service exactly like the
// application package's own fixtures (e.g. newHistoricalFixture), and adds
// what tests.ApiScenario needs to drive requests through the real HTTP
// router instead of calling Service methods directly.
type apiFixture struct {
	app     *tests.TestApp
	stores  adapters.Stores
	service *application.Service
}

// newAPITestApp seeds schema only (no users/groups) so each test file can
// seed exactly the fixture data its scenarios need.
func newAPITestApp(t *testing.T) *apiFixture {
	t.Helper()
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)
	if err = pocketbase.EnsureSchema(app); err != nil {
		t.Fatal(err)
	}
	stores, err := adapters.New("pocketbase", app)
	if err != nil {
		t.Fatal(err)
	}
	return &apiFixture{app: app, stores: stores, service: application.New(stores)}
}

// newUninitializedAPITestApp is like newAPITestApp but also captures the
// one-time setup link's token, for exercising the setup/initialize flow
// which newAPITestApp's plain EnsureSchema call discards.
func newUninitializedAPITestApp(t *testing.T) (*apiFixture, string) {
	t.Helper()
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)
	setupLink, err := pocketbase.EnsureSchemaWithSetupURL(app, "http://localhost:8080")
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(setupLink)
	if err != nil {
		t.Fatal(err)
	}
	stores, err := adapters.New("pocketbase", app)
	if err != nil {
		t.Fatal(err)
	}
	return &apiFixture{app: app, stores: stores, service: application.New(stores)}, parsed.Query().Get("token")
}

// factory adapts this fixture's already-seeded app into a
// tests.ApiScenario.TestAppFactory, so scenarios reuse it instead of each
// getting a fresh empty one.
func (f *apiFixture) factory() func(t testing.TB) *tests.TestApp {
	return func(t testing.TB) *tests.TestApp { return f.app }
}

// beforeTest registers the real API routes exactly like
// cmd/subflow/main.go's OnServe hook does, so scenarios exercise routing,
// auth middleware, and handler wiring, not just Service methods directly.
func (f *apiFixture) beforeTest() func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
	return func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
		(&httpapi.API{Service: f.service}).RegisterRoutes(e)
	}
}

func (f *apiFixture) createUser(t *testing.T, email string) *core.Record {
	t.Helper()
	users, err := f.app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatal(err)
	}
	record := core.NewRecord(users)
	record.Set("email", email)
	record.Set("name", email)
	record.Set("timezone", "UTC")
	record.SetPassword("correct-horse-battery-staple")
	if err = f.app.Save(record); err != nil {
		t.Fatal(err)
	}
	return record
}

func (f *apiFixture) token(t *testing.T, record *core.Record) string {
	t.Helper()
	token, err := record.NewAuthToken()
	if err != nil {
		t.Fatal(err)
	}
	return token
}

// seedGroup creates a group owned by owner, with member (if non-empty)
// added as a regular member.
func (f *apiFixture) seedGroup(t *testing.T, owner, member string) *domain.Group {
	t.Helper()
	ctx := context.Background()
	group := &domain.Group{Name: "API Test Group", Currency: domain.CurrencyTWD, Color: "#7057e8", OwnerID: owner, Timezone: "UTC"}
	if err := f.stores.Groups.Create(ctx, group); err != nil {
		t.Fatal(err)
	}
	if err := f.stores.Memberships.Create(ctx, &domain.Membership{GroupID: group.ID, UserID: owner, Role: domain.RoleOwner}); err != nil {
		t.Fatal(err)
	}
	if member != "" {
		if err := f.stores.Memberships.Create(ctx, &domain.Membership{GroupID: group.ID, UserID: member, Role: domain.RoleMember}); err != nil {
			t.Fatal(err)
		}
	}
	return group
}
