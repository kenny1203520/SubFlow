package application

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"fmt"
	"strings"
	"time"

	"subflow/internal/captcha"
	"subflow/internal/domain"
	"subflow/internal/publicurl"
)

func (s *Service) SetupStatus(ctx context.Context) (domain.SystemSettings, error) {
	return s.Stores.Settings.Get(ctx)
}

func validSetup(input domain.SetupInput) bool {
	if strings.TrimSpace(input.AdminName) == "" || strings.TrimSpace(input.Email) == "" || len(input.Password) < 8 || strings.TrimSpace(input.SiteName) == "" || !domain.IsCurrency(input.DefaultCurrency) {
		return false
	}
	_, err := time.LoadLocation(input.DefaultTimezone)
	return err == nil
}

func equalSetupToken(expected, supplied string) bool {
	sum := sha256.Sum256([]byte(supplied))
	return expected != "" && subtle.ConstantTimeCompare([]byte(expected), []byte(fmt.Sprintf("%x", sum))) == 1
}

func (s *Service) ValidateSetupToken(ctx context.Context, token string) (domain.SystemSettings, bool, error) {
	settings, err := s.Stores.Settings.Get(ctx)
	if err != nil || settings.Initialized {
		return settings, false, err
	}
	return settings, settings.SetupTokenIssued && equalSetupToken(settings.SetupTokenHash, token), nil
}

func (s *Service) InitializeSetup(ctx context.Context, input domain.SetupInput) (*domain.User, error) {
	if !validSetup(input) {
		s.audit(ctx, "", "", "setup.initialize", "system", "", "failure", encodeAuditSummary(map[string]any{"reason": "invalid_input"}, nil))
		return nil, domain.ErrInvalid
	}
	_, validToken, settingsErr := s.ValidateSetupToken(ctx, input.Token)
	if settingsErr != nil || !validToken {
		s.audit(ctx, "", "", "setup.initialize", "system", "", "failure", encodeAuditSummary(map[string]any{"reason": "invalid_token"}, nil))
		return nil, domain.ErrSetupToken
	}
	var created *domain.User
	err := s.Stores.Transactions.Within(ctx, func(tx context.Context) error {
		settings, err := s.Stores.Settings.Get(tx)
		if err != nil {
			return err
		}
		if settings.Initialized {
			return domain.ErrSetupDisabled
		}
		roles, err := s.Stores.Roles.List(tx, "system", "")
		if err != nil {
			return err
		}
		var admin domain.Role
		for _, role := range roles {
			if role.Key == "admin" {
				admin = role
				break
			}
		}
		if admin.ID == "" {
			return domain.ErrNotFound
		}
		count, err := s.Stores.Users.CountBySystemRole(tx, admin.ID)
		if err != nil {
			return err
		}
		if count != 0 {
			return domain.ErrSetupDisabled
		}
		input.AdminName = strings.TrimSpace(input.AdminName)
		input.Email = strings.TrimSpace(strings.ToLower(input.Email))
		input.SiteName = strings.TrimSpace(input.SiteName)
		created, err = s.Stores.Users.Create(tx, input)
		if err != nil {
			return err
		}
		if err = s.Stores.Users.SetSystemRole(tx, created.ID, admin.ID); err != nil {
			return err
		}
		allowPassword, allowOIDC := input.AllowPasswordRegistration, input.AllowOIDCRegistration
		if !allowPassword && !allowOIDC && input.AllowRegistration {
			allowPassword, allowOIDC = true, true
		}
		settings = domain.SystemSettings{Initialized: true, SiteName: input.SiteName, DefaultTimezone: input.DefaultTimezone, DefaultCurrency: input.DefaultCurrency, AllowRegistration: allowPassword, AllowPasswordRegistration: allowPassword, AllowOIDCRegistration: allowOIDC, AttachmentDefaultMaxSize: 20 * 1024 * 1024, AttachmentDefaultMaxCount: 10, AttachmentLimitMaxSize: 20 * 1024 * 1024, AttachmentLimitMaxCount: 10}
		return s.Stores.Settings.Save(tx, settings)
	})
	if err != nil {
		s.audit(ctx, "", "", "setup.initialize", "system", "", "failure", encodeAuditSummary(map[string]any{"reason": err.Error()}, nil))
		return nil, err
	}
	s.audit(ctx, created.ID, "", "setup.initialize", "system", "", "success", encodeAuditSummary(map[string]any{"site_name": input.SiteName, "admin_email": created.Email}, nil))
	return created, nil
}

func (s *Service) Register(ctx context.Context, input domain.SetupInput) (*domain.User, error) {
	settings, err := s.Stores.Settings.Get(ctx)
	if err != nil {
		return nil, err
	}
	if !settings.Initialized || !settings.AllowPasswordRegistration {
		return nil, domain.ErrForbidden
	}
	input.AdminName = strings.TrimSpace(input.AdminName)
	input.Email = strings.TrimSpace(strings.ToLower(input.Email))
	input.DefaultTimezone = settings.DefaultTimezone
	input.DefaultCurrency = settings.DefaultCurrency
	if input.AdminName == "" || input.Email == "" || len(input.Password) < 8 {
		return nil, domain.ErrInvalid
	}
	created, err := s.Stores.Users.Create(ctx, input)
	if err != nil {
		return nil, err
	}
	roles, err := s.Stores.Roles.List(ctx, "system", "")
	if err != nil {
		return nil, err
	}
	for _, role := range roles {
		if role.Key == "user" {
			_ = s.Stores.Users.SetSystemRole(ctx, created.ID, role.ID)
			break
		}
	}
	s.audit(ctx, created.ID, "", "user.registered", "user", created.ID, "success", encodeAuditSummary(map[string]any{"email": created.Email}, nil))
	return created, nil
}

func (s *Service) UpdateSystemSettings(ctx context.Context, userID string, value domain.SystemSettings) (result domain.SystemSettings, retErr error) {
	defer func() {
		if retErr != nil {
			if auditErr := s.audit(ctx, userID, "", "system.settings.updated", "system_settings", "primary", "failure", encodeAuditSummary(map[string]any{"reason": "update_failed"}, nil)); auditErr != nil {
				retErr = fmt.Errorf("settings update failed and failure audit could not be written: %w", auditErr)
			}
		}
	}()
	if err := s.systemPermission(ctx, userID, "system.settings.manage"); err != nil {
		return domain.SystemSettings{}, err
	}
	if strings.TrimSpace(value.SiteName) == "" || !domain.IsCurrency(value.DefaultCurrency) {
		return domain.SystemSettings{}, domain.ErrInvalid
	}
	if _, err := time.LoadLocation(value.DefaultTimezone); err != nil {
		return domain.SystemSettings{}, domain.ErrInvalid
	}
	current, err := s.Stores.Settings.Get(ctx)
	if err != nil {
		return domain.SystemSettings{}, err
	}
	// Preserve attachment settings for older clients that omit the new fields.
	if value.AttachmentDefaultMaxSize == 0 {
		value.AttachmentDefaultMaxSize = current.AttachmentDefaultMaxSize
	}
	if value.AttachmentDefaultMaxCount == 0 {
		value.AttachmentDefaultMaxCount = current.AttachmentDefaultMaxCount
	}
	if value.AttachmentLimitMaxSize == 0 {
		value.AttachmentLimitMaxSize = current.AttachmentLimitMaxSize
	}
	if value.AttachmentLimitMaxCount == 0 {
		value.AttachmentLimitMaxCount = current.AttachmentLimitMaxCount
	}
	if value.AttachmentDefaultMaxSize <= 0 || value.AttachmentDefaultMaxCount <= 0 || value.AttachmentLimitMaxSize < value.AttachmentDefaultMaxSize || value.AttachmentLimitMaxCount < value.AttachmentDefaultMaxCount || value.AttachmentLimitMaxSize > 1024*1024*1024 || value.AttachmentLimitMaxCount > 1000 {
		return domain.SystemSettings{}, domain.ErrInvalid
	}
	if value.CaptchaProvider == "altcha" {
		value.CaptchaProvider = "altcha_community"
	}
	if value.CaptchaProvider != "" && value.CaptchaProvider != "recaptcha" && value.CaptchaProvider != "turnstile" && value.CaptchaProvider != "hcaptcha" && value.CaptchaProvider != "altcha_community" && value.CaptchaProvider != "altcha_sentinel" {
		return domain.SystemSettings{}, domain.ErrInvalid
	}
	profiles := current.CaptchaProviders
	if profiles == nil {
		profiles = map[string]domain.CaptchaProviderSettings{}
	}
	if value.CaptchaProviders != nil {
		for provider, requested := range value.CaptchaProviders {
			if requested.Clear {
				delete(profiles, provider)
				continue
			}
			previous := profiles[provider]
			if requested.Secret != "" {
				if captchaProviderRequiresEncryptedSecret(provider) && !s.Cipher.Available() {
					return domain.SystemSettings{}, domain.ErrInvalid
				}
				ciphertext := "plain:" + requested.Secret
				if s.Cipher.Available() {
					encrypted, encryptErr := s.Cipher.Encrypt(requested.Secret)
					if encryptErr != nil {
						return domain.SystemSettings{}, encryptErr
					}
					ciphertext = encrypted
				}
				previous.Secret, previous.Configured = ciphertext, true
			}
			previous.SiteKey, previous.ChallengeURL, previous.VerifyURL = requested.SiteKey, requested.ChallengeURL, requested.VerifyURL
			profiles[provider] = previous
		}
	} else if value.CaptchaProvider != "" { // compatibility with older clients
		profiles[value.CaptchaProvider] = domain.CaptchaProviderSettings{SiteKey: value.CaptchaSiteKey, ChallengeURL: value.CaptchaChallengeURL, VerifyURL: value.CaptchaVerifyURL, Secret: current.CaptchaSecretCiphertext, Configured: current.CaptchaConfigured}
	}
	selected := profiles[value.CaptchaProvider]
	if value.CaptchaEnabled && value.CaptchaProvider == "altcha_community" && selected.Secret == "" {
		key := make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			return domain.SystemSettings{}, err
		}
		generated := fmt.Sprintf("%x", key)
		selected.Secret = "plain:" + generated
		if s.Cipher.Available() {
			ciphertext, encryptErr := s.Cipher.Encrypt(generated)
			if encryptErr != nil {
				return domain.SystemSettings{}, encryptErr
			}
			selected.Secret = ciphertext
		}
		selected.Configured = true
	}
	profiles[value.CaptchaProvider] = selected
	value.CaptchaProviders = profiles
	value.CaptchaSiteKey, value.CaptchaChallengeURL, value.CaptchaVerifyURL = selected.SiteKey, selected.ChallengeURL, selected.VerifyURL
	value.CaptchaSecretCiphertext, value.CaptchaConfigured = selected.Secret, selected.Configured
	if value.CaptchaEnabled && captchaFlowsEnabled(value.CaptchaFlows) && captchaProviderRequiresSecret(value.CaptchaProvider) && selected.Secret == "" {
		return domain.SystemSettings{}, domain.ErrInvalid
	}
	if value.CaptchaEnabled && captchaFlowsEnabled(value.CaptchaFlows) && captchaProviderRequiresSiteKey(value.CaptchaProvider) && strings.TrimSpace(value.CaptchaSiteKey) == "" {
		return domain.SystemSettings{}, domain.ErrInvalid
	}
	value.CaptchaConfigured = value.CaptchaSecretCiphertext != ""
	value.CaptchaSecret = ""
	value.Initialized = true
	value.AllowRegistration = value.AllowPasswordRegistration
	// Never put the captcha secret itself in the audit trail; only whether it changed.
	var settingsChanges changeSet
	settingsChanges.addString("site_name", current.SiteName, value.SiteName)
	settingsChanges.addString("default_timezone", current.DefaultTimezone, value.DefaultTimezone)
	settingsChanges.addString("default_currency", string(current.DefaultCurrency), string(value.DefaultCurrency))
	settingsChanges.addBool("allow_password_registration", current.AllowPasswordRegistration, value.AllowPasswordRegistration)
	settingsChanges.addBool("allow_oidc_registration", current.AllowOIDCRegistration, value.AllowOIDCRegistration)
	settingsChanges.addString("captcha_provider", current.CaptchaProvider, value.CaptchaProvider)
	settingsChanges.addBool("captcha_enabled", current.CaptchaEnabled, value.CaptchaEnabled)
	settingsChanges.addBool("captcha_flows_changed", false, current.CaptchaFlows != value.CaptchaFlows)
	settingsChanges.addBool("captcha_secret_changed", false, current.CaptchaSecretCiphertext != value.CaptchaSecretCiphertext)
	settingsChanges.addInt64("attachment_default_max_size", current.AttachmentDefaultMaxSize, value.AttachmentDefaultMaxSize)
	settingsChanges.addInt("attachment_default_max_count", current.AttachmentDefaultMaxCount, value.AttachmentDefaultMaxCount)
	settingsChanges.addInt64("attachment_limit_max_size", current.AttachmentLimitMaxSize, value.AttachmentLimitMaxSize)
	settingsChanges.addInt("attachment_limit_max_count", current.AttachmentLimitMaxCount, value.AttachmentLimitMaxCount)
	if err = s.Stores.Transactions.Within(ctx, func(tx context.Context) error {
		if saveErr := s.Stores.Settings.Save(tx, value); saveErr != nil {
			return saveErr
		}
		return s.audit(tx, userID, "", "system.settings.updated", "system_settings", "primary", "success", encodeAuditSummary(nil, settingsChanges))
	}); err != nil {
		return domain.SystemSettings{}, err
	}
	return s.sanitiseSettings(value), nil
}

func (s *Service) GetSystemSettings(ctx context.Context, userID string) (domain.SystemSettings, error) {
	if err := s.systemPermission(ctx, userID, "system.settings.manage"); err != nil {
		return domain.SystemSettings{}, err
	}
	value, err := s.Stores.Settings.Get(ctx)
	return s.sanitiseSettings(value), err
}

func (s *Service) sanitiseSettings(value domain.SystemSettings) domain.SystemSettings {
	value.CaptchaSecret = ""
	value.CaptchaSecretCiphertext = ""
	for provider, profile := range value.CaptchaProviders {
		profile.Secret = ""
		profile.Clear = false
		value.CaptchaProviders[provider] = profile
	}
	return value
}

func captchaFlowsEnabled(flows domain.CaptchaFlowSettings) bool {
	return flows.Register.Enabled || flows.PasswordReset.Enabled || flows.OTPRequest.Enabled || flows.Login.Enabled
}

func captchaProviderRequiresSecret(provider string) bool {
	return provider != "" && provider != "altcha_community"
}

func captchaProviderRequiresEncryptedSecret(provider string) bool {
	return provider == "recaptcha" || provider == "turnstile" || provider == "hcaptcha" || provider == "altcha_sentinel"
}

func captchaProviderRequiresSiteKey(provider string) bool {
	return provider == "recaptcha" || provider == "turnstile" || provider == "hcaptcha"
}

func captchaTurnstileAction(flow string) string {
	switch flow {
	case domain.CaptchaFlowRegister:
		return "register"
	case domain.CaptchaFlowPasswordReset:
		return "password_reset"
	case domain.CaptchaFlowOTPRequest:
		return "otp_request"
	case domain.CaptchaFlowLogin:
		return "login"
	default:
		return ""
	}
}

func (s *Service) captchaSecretValue(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}
	if strings.HasPrefix(ciphertext, "plain:") {
		return strings.TrimPrefix(ciphertext, "plain:"), nil
	}
	secret, err := s.Cipher.Decrypt(ciphertext)
	if err == nil {
		return secret, nil
	}
	if !s.Cipher.Available() {
		return ciphertext, nil
	}
	return "", err
}

// VerifyCaptcha checks a submitted captcha token for one gate-able flow (one
// of the domain.CaptchaFlow* constants). A flow left disabled in
// CaptchaFlowSettings is skipped entirely -- even if a provider is globally
// configured -- which is the per-flow granularity layered on top of the
// pre-existing global on/off (settings.CaptchaProvider == "").
func (s *Service) VerifyCaptcha(ctx context.Context, flow, token, remoteIP string) error {
	settings, err := s.Stores.Settings.Get(ctx)
	if err != nil || settings.CaptchaProvider == "" || !settings.CaptchaEnabled {
		return err
	}
	if !settings.CaptchaFlows.For(flow).Enabled {
		return nil
	}
	secret, err := s.captchaSecretValue(settings.CaptchaSecretCiphertext)
	if err != nil {
		return err
	}
	appName, appURL := s.CaptchaAppName, s.CaptchaAppURL
	if s.PocketBaseApplication != nil {
		appName, appURL = s.PocketBaseApplication()
	}
	identity := captcha.ApplicationIdentity{
		Name:    appName,
		URL:     appURL,
		SiteKey: settings.CaptchaSiteKey,
	}
	switch settings.CaptchaProvider {
	case "recaptcha", "turnstile", "hcaptcha":
		parsedURL, parseErr := publicurl.Parse(appURL)
		if parseErr != nil {
			err = fmt.Errorf("%w: PocketBase Application URL must be a valid root HTTP or HTTPS URL", domain.ErrConfiguration)
		} else {
			identity.URL = parsedURL.String()
			err = s.Captcha.Verify(ctx, settings.CaptchaProvider, secret, settings.CaptchaVerifyURL, token, remoteIP, captchaTurnstileAction(flow), identity)
		}
	default:
		err = s.Captcha.Verify(ctx, settings.CaptchaProvider, secret, settings.CaptchaVerifyURL, token, remoteIP, captchaTurnstileAction(flow), identity)
	}
	if s.Stores.Audits != nil {
		outcome := "success"
		reason := "verified"
		if err != nil {
			outcome, reason = "failure", captcha.FailureReason(err)
			if errors.Is(err, domain.ErrConfiguration) {
				reason = "application_url_invalid"
			}
		}
		auditErr := s.audit(ctx, "", "", "captcha.verified", "captcha", flow, outcome, encodeAuditSummary(map[string]any{"flow": flow, "provider": settings.CaptchaProvider, "reason": reason}, nil))
		if auditErr != nil {
			return auditErr
		}
	}
	return err
}

func (s *Service) CommunityCaptchaChallenge(ctx context.Context, flow string) (any, error) {
	settings, err := s.Stores.Settings.Get(ctx)
	if err != nil {
		return nil, err
	}
	if !settings.CaptchaEnabled || !settings.CaptchaFlows.For(flow).Enabled || settings.CaptchaProvider != "altcha_community" && settings.CaptchaProvider != "altcha" {
		return nil, domain.ErrNotFound
	}
	secret, err := s.captchaSecretValue(settings.CaptchaSecretCiphertext)
	if err != nil {
		return nil, err
	}
	return s.Captcha.CreateCommunityChallenge(secret, flow)
}
