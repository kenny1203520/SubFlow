package application

import (
	"crypto/rand"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"fmt"
	"strings"
	"time"

	"subflow/internal/domain"
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
		s.audit(ctx, "", "", "setup.initialize", "system", "", "failed")
		return nil, domain.ErrInvalid
	}
	_, validToken, settingsErr := s.ValidateSetupToken(ctx, input.Token)
	if settingsErr != nil || !validToken {
		s.audit(ctx, "", "", "setup.initialize", "system", "", "failed")
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
		settings = domain.SystemSettings{Initialized: true, SiteName: input.SiteName, DefaultTimezone: input.DefaultTimezone, DefaultCurrency: input.DefaultCurrency, AllowRegistration: allowPassword, AllowPasswordRegistration: allowPassword, AllowOIDCRegistration: allowOIDC}
		return s.Stores.Settings.Save(tx, settings)
	})
	if err != nil {
		s.audit(ctx, "", "", "setup.initialize", "system", "", "failed")
		return nil, err
	}
	s.audit(ctx, created.ID, "", "setup.initialize", "system", "", "success")
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
	s.audit(ctx, created.ID, "", "user.registered", "user", created.ID, "success")
	return created, nil
}

func (s *Service) UpdateSystemSettings(ctx context.Context, userID string, value domain.SystemSettings) (domain.SystemSettings, error) {
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
	if value.CaptchaProvider == "altcha" { value.CaptchaProvider = "altcha_community" }
	if value.CaptchaProvider != "" && value.CaptchaProvider != "recaptcha" && value.CaptchaProvider != "turnstile" && value.CaptchaProvider != "hcaptcha" && value.CaptchaProvider != "altcha_community" && value.CaptchaProvider != "altcha_sentinel" {
		return domain.SystemSettings{}, domain.ErrInvalid
	}
	if value.CaptchaProvider == "altcha_community" && value.CaptchaSecret == "" && current.CaptchaSecretCiphertext == "" {
		key := make([]byte, 32); if _, err := rand.Read(key); err != nil { return domain.SystemSettings{}, err }; value.CaptchaSecret = fmt.Sprintf("%x", key)
	}
	if value.CaptchaSecret != "" {
		if !s.Cipher.Available() {
			return domain.SystemSettings{}, domain.ErrInvalid
		}
		ciphertext, cipherErr := s.Cipher.Encrypt(value.CaptchaSecret)
		if cipherErr != nil {
			return domain.SystemSettings{}, cipherErr
		}
		value.CaptchaSecretCiphertext = ciphertext
	} else {
		value.CaptchaSecretCiphertext = current.CaptchaSecretCiphertext
	}
	value.CaptchaConfigured = value.CaptchaSecretCiphertext != ""
	value.CaptchaSecret = ""
	value.Initialized = true
	value.AllowRegistration = value.AllowPasswordRegistration
	if err := s.Stores.Settings.Save(ctx, value); err != nil {
		return domain.SystemSettings{}, err
	}
	s.audit(ctx, userID, "", "system.settings.updated", "system_settings", "primary", "success")
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
	value.CaptchaConfigured = value.CaptchaConfigured || value.CaptchaProvider != ""
	return value
}

func (s *Service) VerifyCaptcha(ctx context.Context, token, remoteIP string) error {
	settings, err := s.Stores.Settings.Get(ctx)
	if err != nil || settings.CaptchaProvider == "" {
		return err
	}
	secret, err := s.Cipher.Decrypt(settings.CaptchaSecretCiphertext)
	if err != nil {
		return err
	}
	return s.Captcha.Verify(ctx, settings.CaptchaProvider, secret, settings.CaptchaVerifyURL, token, remoteIP)
}

func (s *Service) CommunityCaptchaChallenge(ctx context.Context, flow string) (any, error) {
	settings, err := s.Stores.Settings.Get(ctx); if err != nil { return nil, err }
	if settings.CaptchaProvider != "altcha_community" && settings.CaptchaProvider != "altcha" { return nil, domain.ErrNotFound }
	secret, err := s.Cipher.Decrypt(settings.CaptchaSecretCiphertext); if err != nil { return nil, err }
	return s.Captcha.CreateCommunityChallenge(secret, flow)
}
