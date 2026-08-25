package application

import (
	"context"
	"net/mail"
	"strings"

	"subflow/internal/domain"
	"subflow/internal/ports"
)

type ContactInput struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

func normalizeContactInput(input ContactInput) (domain.Contact, error) {
	name := strings.TrimSpace(input.Name)
	email := domain.NormalizeEmail(input.Email)
	parsed, err := mail.ParseAddress(email)
	if name == "" || err != nil || parsed.Address != email {
		return domain.Contact{}, domain.ErrInvalid
	}
	return domain.Contact{Name: name, Email: email}, nil
}

func (s *Service) ListContacts(ctx context.Context, userID string, request ports.PageRequest) (ports.Page[domain.Contact], error) {
	return s.Stores.Contacts.List(ctx, userID, request)
}

func (s *Service) CreateContact(ctx context.Context, userID string, input ContactInput) (*domain.Contact, error) {
	value, err := normalizeContactInput(input)
	if err != nil {
		_ = s.audit(ctx, userID, "", "contact.created", "contact", "", "failure", encodeAuditSummary(map[string]any{"reason": "invalid_input"}, nil))
		return nil, err
	}
	value.OwnerID = userID
	err = s.Stores.Transactions.Within(ctx, func(tx context.Context) error {
		if createErr := s.Stores.Contacts.Create(tx, &value); createErr != nil {
			return createErr
		}
		return s.audit(tx, userID, "", "contact.created", "contact", value.ID, "success", encodeAuditSummary(map[string]any{"name": value.Name}, nil))
	})
	if err != nil {
		_ = s.audit(ctx, userID, "", "contact.created", "contact", "", "failure", encodeAuditSummary(map[string]any{"reason": "write_failed"}, nil))
		return nil, err
	}
	return &value, nil
}

func (s *Service) UpdateContact(ctx context.Context, userID, id string, input ContactInput) (*domain.Contact, error) {
	current, err := s.Stores.Contacts.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if current.OwnerID != userID {
		return nil, domain.ErrForbidden
	}
	value, err := normalizeContactInput(input)
	if err != nil {
		_ = s.audit(ctx, userID, "", "contact.updated", "contact", id, "failure", encodeAuditSummary(map[string]any{"reason": "invalid_input"}, nil))
		return nil, err
	}
	value.ID, value.OwnerID, value.CreatedAt = current.ID, current.OwnerID, current.CreatedAt
	err = s.Stores.Transactions.Within(ctx, func(tx context.Context) error {
		if updateErr := s.Stores.Contacts.Update(tx, &value); updateErr != nil {
			return updateErr
		}
		return s.audit(tx, userID, "", "contact.updated", "contact", value.ID, "success", encodeAuditSummary(map[string]any{"name": value.Name}, nil))
	})
	if err != nil {
		_ = s.audit(ctx, userID, "", "contact.updated", "contact", id, "failure", encodeAuditSummary(map[string]any{"reason": "write_failed"}, nil))
		return nil, err
	}
	return &value, nil
}

func (s *Service) DeleteContact(ctx context.Context, userID, id string) error {
	current, err := s.Stores.Contacts.Get(ctx, id)
	if err != nil {
		return err
	}
	if current.OwnerID != userID {
		return domain.ErrForbidden
	}
	err = s.Stores.Transactions.Within(ctx, func(tx context.Context) error {
		if deleteErr := s.Stores.Contacts.Delete(tx, id); deleteErr != nil {
			return deleteErr
		}
		return s.audit(tx, userID, "", "contact.deleted", "contact", id, "success")
	})
	if err != nil {
		_ = s.audit(ctx, userID, "", "contact.deleted", "contact", id, "failure", encodeAuditSummary(map[string]any{"reason": "write_failed"}, nil))
	}
	return err
}
