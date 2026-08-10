package application

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"time"

	"subflow/internal/domain"
	"subflow/internal/ports"
)

const exportPageSize = 100

// ExportLedger builds a single CSV combining every expense, subscription and
// settlement the caller can see, either for their personal ledger
// (groupID=="") or for a group's shared ledger. There is no date-range
// filter: the existing list queries have none, and the user asked for full
// history.
func (s *Service) ExportLedger(ctx context.Context, userID, groupID string) ([]byte, string, error) {
	names := map[string]string{}
	lookupName := func(id string) string {
		if id == "" {
			return ""
		}
		if name, ok := names[id]; ok {
			return name
		}
		name := id
		if user, err := s.Stores.Users.Get(ctx, id); err == nil {
			name = user.Name
		}
		names[id] = name
		return name
	}

	var buf bytes.Buffer
	buf.WriteString("\xEF\xBB\xBF") // UTF-8 BOM so Excel opens CJK text correctly.
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"類型", "日期", "名稱", "金額", "幣別", "分類", "付款人/對象", "狀態", "備註"})

	if groupID == "" {
		expenses, err := listAllPersonalExpenses(ctx, s, userID)
		if err != nil {
			return nil, "", err
		}
		for _, v := range expenses {
			_ = w.Write(expenseRow(v, lookupName))
		}
		subs, err := listAllPersonalSubscriptions(ctx, s, userID)
		if err != nil {
			return nil, "", err
		}
		for _, v := range subs {
			_ = w.Write(subscriptionRow(v, lookupName))
		}
	} else {
		if err := s.role(ctx, groupID, userID, false); err != nil {
			return nil, "", err
		}
		expenses, err := listAllExpenses(ctx, s, userID, groupID)
		if err != nil {
			return nil, "", err
		}
		for _, v := range expenses {
			_ = w.Write(expenseRow(v, lookupName))
		}
		subs, err := listAllSubscriptions(ctx, s, userID, groupID)
		if err != nil {
			return nil, "", err
		}
		for _, v := range subs {
			_ = w.Write(subscriptionRow(v, lookupName))
		}
		settlements, err := listAllSettlements(ctx, s, userID, groupID)
		if err != nil {
			return nil, "", err
		}
		for _, v := range settlements {
			_ = w.Write(settlementRow(v, lookupName))
		}
	}

	w.Flush()
	if err := w.Error(); err != nil {
		return nil, "", err
	}

	scope := "personal"
	if groupID != "" {
		scope = groupID
	}
	filename := fmt.Sprintf("subflow-ledger-%s-%s.csv", scope, s.Now().UTC().Format("20060102-150405"))
	return buf.Bytes(), filename, nil
}

func listAllPersonalExpenses(ctx context.Context, s *Service, userID string) ([]domain.Expense, error) {
	var all []domain.Expense
	for page := 1; ; page++ {
		result, err := s.ListPersonalExpenses(ctx, userID, ports.PageRequest{Page: page, PerPage: exportPageSize, Sort: "-incurred_on"})
		if err != nil {
			return nil, err
		}
		all = append(all, result.Items...)
		if page >= result.TotalPages || len(result.Items) == 0 {
			return all, nil
		}
	}
}

func listAllExpenses(ctx context.Context, s *Service, userID, groupID string) ([]domain.Expense, error) {
	var all []domain.Expense
	for page := 1; ; page++ {
		result, err := s.ListExpenses(ctx, userID, groupID, ports.PageRequest{Page: page, PerPage: exportPageSize, Sort: "-incurred_on"})
		if err != nil {
			return nil, err
		}
		all = append(all, result.Items...)
		if page >= result.TotalPages || len(result.Items) == 0 {
			return all, nil
		}
	}
}

func listAllPersonalSubscriptions(ctx context.Context, s *Service, userID string) ([]domain.Subscription, error) {
	var all []domain.Subscription
	for page := 1; ; page++ {
		result, err := s.ListPersonalSubscriptions(ctx, userID, ports.PageRequest{Page: page, PerPage: exportPageSize, Sort: "-created"})
		if err != nil {
			return nil, err
		}
		all = append(all, result.Items...)
		if page >= result.TotalPages || len(result.Items) == 0 {
			return all, nil
		}
	}
}

func listAllSubscriptions(ctx context.Context, s *Service, userID, groupID string) ([]domain.Subscription, error) {
	var all []domain.Subscription
	for page := 1; ; page++ {
		result, err := s.ListSubscriptions(ctx, userID, groupID, ports.PageRequest{Page: page, PerPage: exportPageSize, Sort: "-created"})
		if err != nil {
			return nil, err
		}
		all = append(all, result.Items...)
		if page >= result.TotalPages || len(result.Items) == 0 {
			return all, nil
		}
	}
}

func listAllSettlements(ctx context.Context, s *Service, userID, groupID string) ([]domain.Settlement, error) {
	var all []domain.Settlement
	for page := 1; ; page++ {
		result, err := s.ListSettlements(ctx, userID, groupID, ports.PageRequest{Page: page, PerPage: exportPageSize, Sort: "-settled_on"})
		if err != nil {
			return nil, err
		}
		all = append(all, result.Items...)
		if page >= result.TotalPages || len(result.Items) == 0 {
			return all, nil
		}
	}
}

func formatAmount(minor int64) string { return fmt.Sprintf("%.2f", float64(minor)/100) }
func formatDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02")
}

func expenseRow(v domain.Expense, lookupName func(string) string) []string {
	return []string{"支出", formatDate(v.IncurredOn), v.Title, formatAmount(v.AmountMinor), string(v.Currency), v.Category, lookupName(v.PaidBy), "", v.Notes}
}

func subscriptionRow(v domain.Subscription, lookupName func(string) string) []string {
	return []string{"訂閱", formatDate(v.StartsOn), v.Name, formatAmount(v.AmountMinor), string(v.Currency), v.Category, lookupName(v.PaidBy), string(v.Status), v.Notes}
}

func settlementRow(v domain.Settlement, lookupName func(string) string) []string {
	party := fmt.Sprintf("%s → %s", lookupName(v.FromUserID), lookupName(v.ToUserID))
	return []string{"還款", formatDate(v.SettledOn), "", formatAmount(v.AmountMinor), string(v.Currency), "", party, "", v.Notes}
}
