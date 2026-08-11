package mailer

import (
	"context"
	"fmt"
	"net/mail"

	"github.com/pocketbase/pocketbase/core"
	pbmailer "github.com/pocketbase/pocketbase/tools/mailer"

	"subflow/internal/domain"
)

// Native sends invitation email through PocketBase's own configured mailer
// (Settings.SMTP, set via the Admin UI or synced from SUBFLOW_SMTP_* through
// ConfigurePocketBase), the same client PocketBase itself uses for
// password-reset and verification email — so invitations stop requiring a
// second, separately-configured SMTP client.
type Native struct{ App core.App }

func (m *Native) Configured() bool { return m.App.Settings().SMTP.Enabled }

func (m *Native) SendInvitation(ctx context.Context, inv domain.Invitation, group domain.Group, url string) error {
	settings := m.App.Settings()
	message := &pbmailer.Message{
		From:    mail.Address{Name: settings.Meta.SenderName, Address: settings.Meta.SenderAddress},
		To:      []mail.Address{{Address: inv.Email}},
		Subject: "SubFlow 群組邀請",
		Text:    fmt.Sprintf("你已受邀加入 %s。\r\n\r\n%s\r\n", group.Name, url),
	}
	return m.App.NewMailClient().Send(message)
}
