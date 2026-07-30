package main

import (
	"errors"
	"log"
	"os"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	_ "github.com/pocketbase/pocketbase/migrations"

	"subflow/internal/adapters"
	pbadapter "subflow/internal/adapters/pocketbase"
	"subflow/internal/application"
	"subflow/internal/mailer"
	"subflow/internal/realtime"
	"subflow/internal/transport/httpapi"
)

func main() {
	driver := os.Getenv("SUBFLOW_DATA_DRIVER")
	environment := os.Getenv("SUBFLOW_ENV")
	appURL := os.Getenv("SUBFLOW_APP_URL")
	if appURL == "" {
		appURL = "http://localhost:8080"
	}
	if _, err := adapters.New(driver, nil); err != nil {
		log.Fatal(err)
	}

	app := pocketbase.New()
	events := realtime.NewBus()
	smtpMailer := mailer.FromEnv()
	realtime.BindRecordEvents(app, events)
	app.OnBootstrap().BindFunc(func(e *core.BootstrapEvent) error {
		if err := e.Next(); err != nil {
			return err
		}
		return pbadapter.EnsureSchema(e.App)
	})
	app.OnRecordRequestPasswordResetRequest("users").BindFunc(func(e *core.RecordRequestPasswordResetRequestEvent) error {
		if environment != "" && environment != "development" && !smtpMailer.Configured() {
			return errors.New("SMTP is required for password reset outside development")
		}
		return e.Next()
	})
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		stores, err := adapters.New(driver, e.App)
		if err != nil {
			return err
		}
		base := application.New(stores)
		(&httpapi.API{Service: base}).RegisterRoutes(e)
		(&httpapi.CollaborationAPI{Service: &application.CollaborationService{
			Base: base, Events: events, Mailer: smtpMailer,
			Environment: environment, AppURL: appURL,
		}}).RegisterRoutes(e)
		return e.Next()
	})
	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
