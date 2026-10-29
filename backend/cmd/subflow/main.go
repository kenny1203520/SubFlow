package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"ganigeorgiev.com/pocketbase/pocketbase"
	"ganigeorgiev.com/pocketbase/pocketbase/core"
	_ "ganigeorgiev.com/pocketbase/pocketbase/migrations"

	"subflow/internal/adapters"
	"subflow/internal/application"
	"subflow/internal/config"
	"subflow/internal/domain"
	"subflow/internal/exchange"
	"subflow/internal/mailer"
	"subflow/internal/realtime"
	"subflow/internal/security"
	"subflow/internal/transport/httpapi"
	"subflow/internal/upgrade"
	"subflow/internal/web"
)

func setupStartupNotice(link string) string {
	if link == "" {
		return ""
	}
	return fmt.Sprintf("\n\033[1;97;42m  FIRST-RUN SETUP REQUIRED  \033[0m\n\033[1;32m╔══════════════════════════════════════════════════════════════╗\n║ Open this one-time setup link in your browser:               ║\n╚══════════════════════════════════════════════════════════════╝\033[0m\n\033[1;97m%s\033[0m\n", link)
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		runtime := config.LoadRuntime()
		certificates, err := security.LoadServerCertificateManager(config.DataDirFromArgs(os.Args))
		if err != nil {
			log.Print("readiness check failed: HTTPS certificate is unavailable")
			os.Exit(1)
		}
		if err := checkHealth(config.AppURL("", runtime.HTTPPort)+"/api/ready", certificates.HealthClient(2*time.Second)); err != nil {
			log.Print(err)
			os.Exit(1)
		}
		return
	}
	args, err := config.LoadEnvFile(os.Args)
	if err != nil {
		log.Fatal(err)
	}
	os.Args = args
	runtime := config.LoadRuntime()
	for _, name := range config.DeprecatedEnvironmentNames() {
		log.Printf("deprecated environment setting %s is ignored; configure it in PocketBase or remove it", name)
	}
	args, err = config.WithServeAddress(os.Args, runtime.HTTPPort)
	if err != nil {
		log.Fatal(err)
	}
	os.Args = args
	if _, err = adapters.New("pocketbase", nil); err != nil {
		log.Fatal(err)
	}
	pbVersion, err := upgrade.PocketBaseVersion()
	if err != nil {
		log.Fatal(err)
	}
	if pbVersion != upgrade.RequiredPocketBaseVersion {
		log.Fatalf("binary uses PocketBase %s; this release requires %s", pbVersion, upgrade.RequiredPocketBaseVersion)
	}
	if handled, dispatchErr := dispatchUpgradeCommand(os.Args, "pocketbase", runtime.Environment, runtime.HTTPPort, runtime.EncryptionKey); handled {
		if dispatchErr != nil {
			log.Fatal(dispatchErr)
		}
		return
	}
	if safePocketBaseCommand(os.Args) {
		if err := pocketbase.New().Start(); err != nil {
			log.Fatal(err)
		}
		return
	}
	if _, err := security.DeriveKeyring(runtime.EncryptionKey); err != nil {
		log.Fatal(err)
	}
	setupURL := config.AppURL("", runtime.HTTPPort)
	manager, err := upgrade.New(config.DataDirFromArgs(os.Args))
	if err != nil {
		log.Fatal(err)
	}
	isServe := len(os.Args) > 1 && os.Args[1] == "serve"
	acquireErr := manager.Acquire()
	if acquireErr != nil && !errors.Is(acquireErr, upgrade.ErrRestoreRecovered) && !manager.HasLock() {
		log.Fatal(acquireErr)
	}
	var statusServer *upgrade.StatusServer
	var certificates *security.ServerCertificateManager
	if isServe {
		certificates, err = security.OpenServerCertificateManager(config.DataDirFromArgs(os.Args))
		if err != nil {
			_ = manager.Close()
			log.Fatal("cannot initialize HTTPS certificate: ", err)
		}
		address, addressErr := config.ServeHTTPSAddress(os.Args)
		if addressErr != nil {
			_ = manager.Close()
			log.Fatal(addressErr)
		}
		statusServer, err = upgrade.StartStatusServer(address, manager, certificates)
		if err != nil {
			_ = manager.Fail("maintenance_server_unavailable")
			_ = manager.Close()
			log.Fatal("cannot start maintenance status service: ", err)
		}
	}
	defer manager.Close()
	if statusServer != nil {
		defer statusServer.Close()
	}
	if errors.Is(acquireErr, upgrade.ErrRestoreRecovered) {
		log.Printf("SubFlow restore recovery completed; start the matching older binary before continuing")
		if statusServer != nil {
			waitForShutdown()
		}
		return
	}
	if acquireErr != nil {
		log.Printf("SubFlow restore recovery stopped in maintenance mode: %v", acquireErr)
		if statusServer != nil {
			waitForShutdown()
		} else {
			log.Fatal(acquireErr)
		}
		return
	}
	if err := manager.BeginStartup(); err != nil {
		_ = manager.Fail("upgrade_state_unavailable")
		_ = manager.Close()
		log.Fatal("cannot prepare upgrade status: ", err)
	}
	if err := manager.Prepare(upgrade.AppVersion(), pbVersion, false); err != nil {
		if manager.HasLock() {
			if auditErr := manager.Fail(problemCode(err)); auditErr != nil {
				log.Printf("could not persist upgrade failure: %v", auditErr)
			}
		}
		log.Printf("SubFlow upgrade stopped: %v", err)
		if statusServer != nil {
			waitForShutdown()
		}
		return
	}
	app := pocketbase.New()
	registerPocketBaseSettingsAudit(app, runtime.EncryptionKey)
	events := realtime.NewBus()
	setupLink := registerUpgradeBootstrap(app, manager, "pocketbase", setupURL, runtime.EncryptionKey)
	realtime.BindRecordEvents(app, events)
	app.OnRecordRequestPasswordResetRequest("users").BindFunc(func(e *core.RecordRequestPasswordResetRequestEvent) error {
		if runtime.Environment != "development" && !e.App.Settings().SMTP.Enabled {
			return errors.New("SMTP must be configured in PocketBase for password reset outside development")
		}
		return e.Next()
	})
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		e.Server.TLSConfig = certificates.TLSConfig()
		stores, err := adapters.New("pocketbase", e.App)
		if err != nil {
			return err
		}
		base := application.New(stores, runtime.EncryptionKey)
		base.Events = events
		base.Rates = exchange.NewOpenERAPIProvider()
		base.PocketBaseApplication = func() (string, string) {
			settings := e.App.Settings().Meta
			return settings.AppName, settings.AppURL
		}
		app.OnRecordRequestPasswordResetRequest("users").BindFunc(func(event *core.RecordRequestPasswordResetRequestEvent) error {
			ctx := application.WithAuditRequestMeta(event.Request.Context(), application.AuditRequestMeta{IP: event.RealIP(), UserAgent: event.Request.UserAgent()})
			if err := base.VerifyCaptcha(ctx, domain.CaptchaFlowPasswordReset, event.Request.Header.Get("X-SubFlow-Captcha"), event.RealIP()); err != nil {
				return event.BadRequestError("captcha_verification_failed", nil)
			}
			return event.Next()
		})
		app.OnRecordConfirmPasswordResetRequest("users").BindFunc(func(event *core.RecordConfirmPasswordResetRequestEvent) error {
			userID := event.Record.Id
			requestApp := event.App
			var notice *domain.Notification
			if err := requestApp.RunInTransaction(func(txApp core.App) error {
				event.App = txApp
				defer func() { event.App = requestApp }()
				if err := event.Next(); err != nil {
					return err
				}
				txStores, err := adapters.New("pocketbase", txApp)
				if err != nil {
					return err
				}
				now := base.Now().UTC()
				note := &domain.Notification{UserID: userID, Category: domain.NotificationCategorySecurity, Type: "password_changed", EventKey: fmt.Sprintf("password-changed:%s:%d", userID, now.UnixNano()), Title: "password_changed", Status: domain.NotificationStatusPending, CreatedAt: now, UpdatedAt: now}
				if err := txStores.Notifications.Create(event.Request.Context(), note); err != nil {
					return err
				}
				notice = note
				if err := txStores.NotificationEmails.Enqueue(event.Request.Context(), &domain.NotificationEmailJob{NotificationID: note.ID, UserID: userID, Status: domain.NotificationEmailQueued, NextAttemptAt: &now}); err != nil {
					return err
				}
				if err := application.New(txStores, runtime.EncryptionKey).RecordAudit(event.Request.Context(), userID, "", "account.password.changed", "user", userID, "success"); err != nil {
					return err
				}
				return nil
			}); err != nil {
				return err
			}
			if notice != nil {
				_ = events.Publish(event.Request.Context(), domain.Event{Type: "created", UserID: userID, Resource: "notifications", ResourceID: notice.ID, OccurredAt: base.Now().UTC()})
			}
			return nil
		})
		app.OnRecordRequestOTPRequest("users").BindFunc(func(event *core.RecordCreateOTPRequestEvent) error {
			ctx := application.WithAuditRequestMeta(event.Request.Context(), application.AuditRequestMeta{IP: event.RealIP(), UserAgent: event.Request.UserAgent()})
			if err := base.VerifyCaptcha(ctx, domain.CaptchaFlowOTPRequest, event.Request.Header.Get("X-SubFlow-Captcha"), event.RealIP()); err != nil {
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
			ctx := application.WithAuditRequestMeta(event.Request.Context(), application.AuditRequestMeta{IP: event.RealIP(), UserAgent: event.Request.UserAgent()})
			if err := base.VerifyCaptcha(ctx, domain.CaptchaFlowLogin, event.Request.Header.Get("X-SubFlow-Captcha"), event.RealIP()); err != nil {
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
			if postErr := base.PostDueRecurringIncomes(context.Background()); postErr != nil {
				app.Logger().Warn("recurring income posting failed", "operation", "post_due_recurring_incomes", "error", postErr)
			}
		})
		notificationDelivery := application.NewNotificationDeliveryService(stores.Notifications, stores.NotificationEmails, stores.Transactions, stores.Users, base.RecordAudit, &mailer.Native{App: e.App}, nil)
		app.Cron().MustAdd("subflow_notification_email_delivery", "* * * * *", func() {
			if deliveryErr := notificationDelivery.ProcessDue(context.Background()); deliveryErr != nil {
				app.Logger().Warn("notification email delivery failed", "operation", "deliver_notification_emails", "error", deliveryErr)
			}
		})
		go func() {
			if refreshErr := base.RefreshReferenceRates(context.Background()); refreshErr != nil {
				app.Logger().Warn("initial exchange rate refresh failed", "error", refreshErr)
			}
		}()
		(&httpapi.API{Service: base, Upgrades: manager, Certificates: certificates}).RegisterRoutes(e)
		(&httpapi.CollaborationAPI{Service: &application.CollaborationService{Base: base, Events: events, Mailer: &mailer.Native{App: e.App}, Environment: runtime.Environment, ApplicationURL: func() string { return e.App.Settings().Meta.AppURL }}}).RegisterRoutes(e)
		web.Register(e)
		fmt.Print(setupStartupNotice(*setupLink))
		if err := manager.Complete(func() error {
			return upgrade.ReconcileDurableEvents(context.Background(), e.App, "pocketbase", manager.Dir, runtime.EncryptionKey)
		}); err != nil {
			return err
		}
		if statusServer != nil {
			if err := statusServer.Close(); err != nil {
				return err
			}
		}
		return e.Next()
	})
	if err := app.Start(); err != nil {
		if manager.HasLock() {
			if auditErr := manager.Fail(problemCode(err)); auditErr != nil {
				log.Printf("could not persist upgrade failure: %v", auditErr)
			}
		}
		log.Printf("SubFlow stopped: %v", err)
		if statusServer != nil {
			waitForShutdown()
		}
		return
	}
	if !isServe {
		if err := manager.Complete(func() error {
			return upgrade.ReconcileDurableEvents(context.Background(), app, "pocketbase", manager.Dir, runtime.EncryptionKey)
		}); err != nil {
			_ = manager.Fail("upgrade_finalize_failed")
			log.Printf("SubFlow command completed but upgrade state could not be finalized: %v", err)
		}
	}
}

func safePocketBaseCommand(args []string) bool {
	for _, arg := range args[1:] {
		if arg == "--help" || arg == "-h" || arg == "--version" || arg == "version" || strings.HasPrefix(arg, "--help=") {
			return true
		}
	}
	return false
}

func waitForShutdown() {
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)
	<-shutdown
	signal.Stop(shutdown)
}
