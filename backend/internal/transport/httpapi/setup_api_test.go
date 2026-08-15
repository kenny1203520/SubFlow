package httpapi_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/tests"
)

func TestSetupStatusBeforeInitialization(t *testing.T) {
	f, _ := newUninitializedAPITestApp(t)
	scenario := tests.ApiScenario{
		Name:                  "setup status reports uninitialized before any admin exists",
		Method:                http.MethodGet,
		URL:                   "/api/subflow/v1/setup/status",
		TestAppFactory:        f.factory(),
		BeforeTestFunc:        f.beforeTest(),
		DisableTestAppCleanup: true,
		ExpectedStatus:        http.StatusOK,
		ExpectedContent:       []string{`"initialized":false`},
	}
	scenario.Test(t)
}

func TestInitializeSetupRejectsInvalidToken(t *testing.T) {
	f, _ := newUninitializedAPITestApp(t)
	body := `{"adminName":"Admin","email":"admin@example.com","password":"correct-horse-battery-staple","siteName":"SubFlow","defaultTimezone":"UTC","defaultCurrency":"TWD","token":"wrong-token"}`
	scenario := tests.ApiScenario{
		Name:                  "initialize setup rejects a token that doesn't match the issued one",
		Method:                http.MethodPost,
		URL:                   "/api/subflow/v1/setup/initialize",
		Body:                  strings.NewReader(body),
		TestAppFactory:        f.factory(),
		BeforeTestFunc:        f.beforeTest(),
		DisableTestAppCleanup: true,
		ExpectedStatus:        http.StatusForbidden,
		ExpectedContent:       []string{`"code":"setup_token_invalid"`},
	}
	scenario.Test(t)
}

func TestInitializeSetupSucceedsWithValidToken(t *testing.T) {
	f, token := newUninitializedAPITestApp(t)
	body := `{"adminName":"Admin","email":"admin@example.com","password":"correct-horse-battery-staple","siteName":"SubFlow","defaultTimezone":"UTC","defaultCurrency":"TWD","token":"` + token + `"}`
	scenario := tests.ApiScenario{
		Name:                  "initialize setup succeeds with the issued token and creates the first admin",
		Method:                http.MethodPost,
		URL:                   "/api/subflow/v1/setup/initialize",
		Body:                  strings.NewReader(body),
		TestAppFactory:        f.factory(),
		BeforeTestFunc:        f.beforeTest(),
		DisableTestAppCleanup: true,
		ExpectedStatus:        http.StatusCreated,
		ExpectedContent:       []string{`"id":`},
	}
	scenario.Test(t)
}

func TestRegisterRejectsBeforeSetupHasRun(t *testing.T) {
	f := newAPITestApp(t)
	body := `{"adminName":"New User","email":"newuser@example.com","password":"correct-horse-battery-staple"}`
	scenario := tests.ApiScenario{
		Name:                  "register is rejected while the instance has no initialized settings yet",
		Method:                http.MethodPost,
		URL:                   "/api/subflow/v1/auth/register",
		Body:                  strings.NewReader(body),
		TestAppFactory:        f.factory(),
		BeforeTestFunc:        f.beforeTest(),
		DisableTestAppCleanup: true,
		ExpectedStatus:        http.StatusForbidden,
		ExpectedContent:       []string{`"code":"forbidden"`},
	}
	scenario.Test(t)
}
