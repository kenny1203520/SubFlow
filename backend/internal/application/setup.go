package application

import (
	"context"
	"crypto/subtle"
	"os"
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

func equalSecret(actual, supplied string) bool {
	if actual == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(actual), []byte(supplied)) == 1
}

func (s *Service) InitializeSetup(ctx context.Context, input domain.SetupInput) (*domain.User, error) {
	if !validSetup(input) {
		s.audit(ctx, "", "", "setup.initialize", "system", "", "failed")
		return nil, domain.ErrInvalid
	}
	if !equalSecret(os.Getenv("SUBFLOW_SETUP_SECRET"), input.Secret) {
		s.audit(ctx, "", "", "setup.initialize", "system", "", "failed")
		return nil, domain.ErrSetupSecret
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
		settings = domain.SystemSettings{Initialized: true, SiteName: input.SiteName, DefaultTimezone: input.DefaultTimezone, DefaultCurrency: input.DefaultCurrency, AllowRegistration: input.AllowRegistration}
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
	if !settings.Initialized || !settings.AllowRegistration {
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
	value.Initialized = true
	if err := s.Stores.Settings.Save(ctx, value); err != nil {
		return domain.SystemSettings{}, err
	}
	s.audit(ctx, userID, "", "system.settings.updated", "system_settings", "primary", "success")
	return value, nil
}

func (s *Service) GetSystemSettings(ctx context.Context, userID string) (domain.SystemSettings, error) {
	if err := s.systemPermission(ctx, userID, "system.settings.manage"); err != nil {
		return domain.SystemSettings{}, err
	}
	return s.Stores.Settings.Get(ctx)
}
