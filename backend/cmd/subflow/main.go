package main

import (
	"context"
	"errors"
	"log"
	"os"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	_ "github.com/pocketbase/pocketbase/migrations"

	"subflow/internal/adapters"
	pbadapter "subflow/internal/adapters/pocketbase"
	"subflow/internal/application"
	"subflow/internal/config"
	"subflow/internal/exchange"
	"subflow/internal/mailer"
	"subflow/internal/realtime"
	"subflow/internal/transport/httpapi"
	"subflow/internal/web"
)

func main() {
	driver := os.Getenv("SUBFLOW_DATA_DRIVER")
	environment := os.Getenv("SUBFLOW_ENV")
	httpPort := os.Getenv("SUBFLOW_HTTP_PORT")
	args, err := config.WithServeAddress(os.Args, httpPort)
	if err != nil {
		log.Fatal(err)
	}
	os.Args = args
	appURL := config.AppURL(os.Getenv("SUBFLOW_APP_URL"), httpPort)
	if _, err = adapters.New(driver, nil); err != nil {
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
		if err := pbadapter.EnsureSchema(e.App); err != nil {
			return err
		}
		return mailer.ConfigurePocketBase(e.App, smtpMailer)
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
		base.Rates = exchange.NewCBCProvider()
		app.Cron().MustAdd("subflow_exchange_rates", "15 */6 * * *", func() {
			if refreshErr := base.RefreshReferenceRates(context.Background()); refreshErr != nil {
				app.Logger().Warn("exchange rate refresh failed", "error", refreshErr)
			}
			if refreshErr := base.RefreshAutomaticSubscriptions(context.Background()); refreshErr != nil {
				app.Logger().Warn("subscription rate refresh failed", "error", refreshErr)
			}
		})
		go func() {
			if refreshErr := base.RefreshReferenceRates(context.Background()); refreshErr != nil {
				app.Logger().Warn("initial exchange rate refresh failed", "error", refreshErr)
			}
		}()
		(&httpapi.API{Service: base}).RegisterRoutes(e)
		(&httpapi.CollaborationAPI{Service: &application.CollaborationService{Base: base, Events: events, Mailer: smtpMailer, Environment: environment, AppURL: appURL}}).RegisterRoutes(e)
		web.Register(e)
		return e.Next()
	})
	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
