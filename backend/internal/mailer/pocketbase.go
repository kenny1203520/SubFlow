package mailer

import (
	"strconv"

	"github.com/pocketbase/pocketbase/core"
)

// ConfigurePocketBase makes native users password-reset/verification emails
// use the same SMTP environment as SubFlow invitation mail.
func ConfigurePocketBase(app core.App, smtp *SMTP) error {
	if smtp == nil || !smtp.Configured() {
		return nil
	}
	port, err := strconv.Atoi(smtp.Port)
	if err != nil {
		return err
	}
	settings := app.Settings()
	settings.SMTP.Enabled = true
	settings.SMTP.Host = smtp.Host
	settings.SMTP.Port = port
	settings.SMTP.Username = smtp.User
	settings.SMTP.Password = smtp.Password
	settings.SMTP.AuthMethod = "PLAIN"
	settings.SMTP.TLS = port == 465
	settings.Meta.SenderName = "SubFlow"
	settings.Meta.SenderAddress = smtp.From
	return app.Save(settings)
}
