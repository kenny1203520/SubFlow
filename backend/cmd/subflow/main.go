package main

import (
	"log"
	"os"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	_ "github.com/pocketbase/pocketbase/migrations"

	"subflow/internal/adapters"
	pbadapter "subflow/internal/adapters/pocketbase"
	"subflow/internal/application"
	"subflow/internal/transport/httpapi"
)

func main() {
	driver := os.Getenv("SUBFLOW_DATA_DRIVER")
	if _, err := adapters.New(driver, nil); err != nil {
		log.Fatal(err)
	}
	app := pocketbase.New()
	app.OnBootstrap().BindFunc(func(e *core.BootstrapEvent) error {
		if err := e.Next(); err != nil {
			return err
		}
		return pbadapter.EnsureSchema(e.App)
	})
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		stores, err := adapters.New(driver, e.App)
		if err != nil {
			return err
		}
		(&httpapi.API{Service: application.New(stores)}).RegisterRoutes(e)
		return e.Next()
	})
	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
