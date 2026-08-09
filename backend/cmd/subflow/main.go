package main

import (
	"context"
	"errors"
	"fmt"
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

func setupStartupNotice(link string) string {
	if link == "" {
		return ""
	}
	return fmt.Sprintf("\n\033[1;97;42m  FIRST-RUN SETUP REQUIRED  \033[0m\n\033[1;32m╔══════════════════════════════════════════════════════════════╗\n║ Open this one-time setup link in your browser:               ║\n╚══════════════════════════════════════════════════════════════╝\033[0m\n\033[1;97m%s\033[0m\n", link)
}

func main() {
	driver := os.Getenv("SUBFLOW_DATA_DRIVER")
	environment := os.Getenv("SUBFLOW_ENV")
	httpPort := os.Getenv("SUBFLOW_HTTP_PORT")
	args, err := config.WithServeAddress(os.Args, httpPort)
	if err != nil {
		log.Fatal(err)
	}
	os.Args = args
	appURL, err := config.PublicAppURL(os.Getenv("SUBFLOW_APP_URL"), httpPort, environment)
	if err != nil {
		log.Fatal(err)
	}
	if _, err = adapters.New(driver, nil); err != nil {
		log.Fatal(err)
	}
	app := pocketbase.New()
	events := realtime.NewBus()
	smtpMailer := mailer.FromEnv()
	var setupLink string
	realtime.BindRecordEvents(app, events)
	app.OnBootstrap().BindFunc(func(e *core.BootstrapEvent) error {
		if err := e.Next(); err != nil {
			return err
		}
		setupLink, err = pbadapter.EnsureSchemaWithSetupURL(e.App, appURL)
		if err != nil {
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
				app.Logger().Warn("subscription rate refresh failed", "operation", "refresh_automatic_subscriptions", "error", refreshErr)
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
		fmt.Print(setupStartupNotice(setupLink))
		return e.Next()
	})
	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
