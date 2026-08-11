package mailer

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"io"
	"net/mail"

	"github.com/pocketbase/pocketbase/core"
	pbmailer "github.com/pocketbase/pocketbase/tools/mailer"

	"subflow/internal/domain"
)

//go:embed assets/logo.png
var invitationLogoPNG []byte

// Native sends invitation email through PocketBase's own configured mailer
// (Settings.SMTP, set via the Admin UI or synced from SUBFLOW_SMTP_* through
// ConfigurePocketBase), the same client PocketBase itself uses for
// password-reset and verification email — so invitations stop requiring a
// second, separately-configured SMTP client.
type Native struct{ App core.App }

func (m *Native) Configured() bool { return m.App.Settings().SMTP.Enabled }

// The layout (white background, #16161a text, Source Sans Pro stack, black
// rounded button) mirrors PocketBase's own built-in auth email templates
// (mails/templates/layout.go) so this reads as visually consistent with the
// password-reset/verification mail recipients may already have seen from
// this app, rather than a distinct, less trustworthy-looking message.
func (m *Native) SendInvitation(ctx context.Context, inv domain.Invitation, group domain.Group, inviterName, url string) error {
	settings := m.App.Settings()
	senderName := settings.Meta.SenderName
	if senderName == "" {
		senderName = "SubFlow"
	}

	intro := fmt.Sprintf("你已受邀加入群組「%s」，一起管理共同支出、訂閱與分帳。", escapeHTML(group.Name))
	introText := fmt.Sprintf("你已受邀加入群組「%s」，一起管理共同支出、訂閱與分帳。", group.Name)
	if inviterName != "" {
		intro = fmt.Sprintf("<strong>%s</strong> 邀請你加入群組「%s」，一起管理共同支出、訂閱與分帳。", escapeHTML(inviterName), escapeHTML(group.Name))
		introText = fmt.Sprintf("%s 邀請你加入群組「%s」，一起管理共同支出、訂閱與分帳。", inviterName, group.Name)
	}

	descriptionHTML := ""
	descriptionText := ""
	if group.Description != "" {
		descriptionHTML = fmt.Sprintf(`<p style="margin:0 0 20px;padding:12px 16px;background:#f5f5f7;border-radius:6px;color:#4a4a52;font-size:13px;">%s</p>`, escapeHTML(group.Description))
		descriptionText = fmt.Sprintf("\r\n%s\r\n", group.Description)
	}

	expiresHTML := ""
	expiresText := ""
	if !inv.ExpiresAt.IsZero() {
		expires := inv.ExpiresAt.Format("2006-01-02")
		expiresHTML = fmt.Sprintf(`<p style="margin:20px 0 0;color:#8a8a94;font-size:12px;">此邀請將於 %s 前失效。</p>`, expires)
		expiresText = fmt.Sprintf("\r\n此邀請將於 %s 前失效。\r\n", expires)
	}

	html := fmt.Sprintf(`<!doctype html>
<html>
<body style="margin:0;padding:32px 16px;background:#f5f5f7;font-family:'Source Sans Pro',Helvetica,Arial,sans-serif;">
<table role="presentation" width="100%%" style="max-width:480px;margin:0 auto;background:#ffffff;border-radius:10px;padding:32px;">
<tr><td>
<div style="display:flex;align-items:center;gap:10px;margin-bottom:24px;">
<img src="cid:logo.png" width="32" height="32" alt="%s" style="border-radius:8px;">
<strong style="font-size:16px;color:#16161a;">%s</strong>
</div>
<p style="margin:0 0 20px;color:#16161a;font-size:14px;line-height:20px;">%s</p>
%s
<p style="margin:0 0 24px;">
<a href="%s" style="display:inline-block;min-width:150px;padding:0 16px;line-height:40px;background:#16161a;color:#ffffff;text-decoration:none;font-size:14px;font-weight:600;border-radius:6px;text-align:center;">加入群組</a>
</p>
<p style="margin:0;color:#8a8a94;font-size:12px;line-height:18px;"><i>如果你不認識寄件者，或不想加入這個群組，可以忽略這封信件。</i></p>
%s
<p style="margin:24px 0 0;color:#8a8a94;font-size:12px;">Thanks,<br/>%s 團隊</p>
</td></tr>
</table>
</body>
</html>`, escapeHTML(senderName), escapeHTML(senderName), intro, descriptionHTML, url, expiresHTML, escapeHTML(senderName))

	text := fmt.Sprintf("%s\r\n%s\r\n%s\r\n%s\r\n如果你不認識寄件者，或不想加入這個群組，可以忽略這封信件。\r\n\r\nThanks,\r\n%s 團隊\r\n", introText, descriptionText, url, expiresText, senderName)

	message := &pbmailer.Message{
		From:              mail.Address{Name: senderName, Address: settings.Meta.SenderAddress},
		To:                []mail.Address{{Address: inv.Email}},
		Subject:           fmt.Sprintf("邀請你加入「%s」— %s", group.Name, senderName),
		HTML:              html,
		Text:              text,
		InlineAttachments: map[string]io.Reader{"logo.png": bytes.NewReader(invitationLogoPNG)},
	}
	return m.App.NewMailClient().Send(message)
}

func escapeHTML(value string) string {
	var buf bytes.Buffer
	for _, r := range value {
		switch r {
		case '&':
			buf.WriteString("&amp;")
		case '<':
			buf.WriteString("&lt;")
		case '>':
			buf.WriteString("&gt;")
		case '"':
			buf.WriteString("&quot;")
		default:
			buf.WriteRune(r)
		}
	}
	return buf.String()
}
