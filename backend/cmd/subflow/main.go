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
	"subflow/internal/domain"
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
		if environment != "" && environment != "development" && !e.App.Settings().SMTP.Enabled {
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
		base.Rates = exchange.NewOpenERAPIProvider()
		app.OnRecordRequestPasswordResetRequest("users").BindFunc(func(event *core.RecordRequestPasswordResetRequestEvent) error {
			if err := base.VerifyCaptcha(event.Request.Context(), domain.CaptchaFlowPasswordReset, event.Request.Header.Get("X-SubFlow-Captcha"), event.RealIP()); err != nil {
				return event.BadRequestError("captcha_verification_failed", nil)
			}
			return event.Next()
		})
		app.OnRecordRequestOTPRequest("users").BindFunc(func(event *core.RecordCreateOTPRequestEvent) error {
			if err := base.VerifyCaptcha(event.Request.Context(), domain.CaptchaFlowOTPRequest, event.Request.Header.Get("X-SubFlow-Captcha"), event.RealIP()); err != nil {
				return event.BadRequestError("captcha_verification_failed", nil)
			}
			return event.Next()
		})
		// Password login has no built-in hook of its own -- fires after
		// identity lookup but before the password is even checked, so a
		// rejected captcha short-circuits before that check runs. Disabled by
		// default (see captchaFlowsFrom's migration defaults), so existing
		// installs see no behavior change until an admin opts in.
		app.OnRecordAuthWithPasswordRequest("users").BindFunc(func(event *core.RecordAuthWithPasswordRequestEvent) error {
			if err := base.VerifyCaptcha(event.Request.Context(), domain.CaptchaFlowLogin, event.Request.Header.Get("X-SubFlow-Captcha"), event.RealIP()); err != nil {
				return event.BadRequestError("captcha_verification_failed", nil)
			}
			return event.Next()
		})
		app.OnRecordAuthWithOAuth2Request("users").BindFunc(func(event *core.RecordAuthWithOAuth2RequestEvent) error {
			if event.IsNewRecord {
				settings, settingsErr := base.SetupStatus(event.Request.Context())
				if settingsErr != nil || !settings.Initialized || !settings.AllowOIDCRegistration {
					return event.ForbiddenError("New user registration is disabled", nil)
				}
				if event.OAuth2User == nil || event.OAuth2User.Email == "" {
					return event.BadRequestError("oidc_verified_email_required", nil)
				}
			}
			return event.Next()
		})
		app.Cron().MustAdd("subflow_exchange_rates", "15 */6 * * *", func() {
			if refreshErr := base.RefreshReferenceRates(context.Background()); refreshErr != nil {
				app.Logger().Warn("exchange rate refresh failed", "error", refreshErr)
			}
			if refreshErr := base.RefreshAutomaticSubscriptions(context.Background()); refreshErr != nil {
				app.Logger().Warn("subscription rate refresh failed", "operation", "refresh_automatic_subscriptions", "error", refreshErr)
			}
		})
		app.Cron().MustAdd("subflow_subscription_posting", "* * * * *", func() {
			if postErr := base.PostDueSubscriptions(context.Background()); postErr != nil {
				app.Logger().Warn("subscription posting failed", "operation", "post_due_subscriptions", "error", postErr)
			}
		})
		go func() {
			if refreshErr := base.RefreshReferenceRates(context.Background()); refreshErr != nil {
				app.Logger().Warn("initial exchange rate refresh failed", "error", refreshErr)
			}
		}()
		(&httpapi.API{Service: base}).RegisterRoutes(e)
		(&httpapi.CollaborationAPI{Service: &application.CollaborationService{Base: base, Events: events, Mailer: &mailer.Native{App: e.App}, Environment: environment, AppURL: appURL}}).RegisterRoutes(e)
		web.Register(e)
		fmt.Print(setupStartupNotice(setupLink))
		return e.Next()
	})
	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
