package application

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"strings"
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
	loc := s.exportLocation(ctx, userID, groupID)

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
	_ = w.Write([]string{"類型", "日期", "名稱", "金額", "幣別", "分類", "付款人/對象", "狀態", "分帳明細", "備註"})

	if groupID == "" {
		expenses, err := listAllPersonalExpenses(ctx, s, userID)
		if err != nil {
			return nil, "", err
		}
		for _, v := range expenses {
			_ = w.Write(expenseRow(v, loc, lookupName))
		}
		subs, err := listAllPersonalSubscriptions(ctx, s, userID)
		if err != nil {
			return nil, "", err
		}
		for _, v := range subs {
			for _, row := range subscriptionRows(v, loc, lookupName) {
				_ = w.Write(row)
			}
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
			_ = w.Write(expenseRow(v, loc, lookupName))
		}
		subs, err := listAllSubscriptions(ctx, s, userID, groupID)
		if err != nil {
			return nil, "", err
		}
		for _, v := range subs {
			for _, row := range subscriptionRows(v, loc, lookupName) {
				_ = w.Write(row)
			}
		}
		settlements, err := listAllSettlements(ctx, s, userID, groupID)
		if err != nil {
			return nil, "", err
		}
		for _, v := range settlements {
			_ = w.Write(settlementRow(v, loc, lookupName))
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

// exportLocation resolves which timezone dates should be shown in: the
// group's accounting timezone first (matching how the app already computes
// billing/month boundaries in group time), falling back to the viewer's own
// timezone, and finally UTC if neither is set or valid.
func (s *Service) exportLocation(ctx context.Context, userID, groupID string) *time.Location {
	if groupID != "" {
		if group, err := s.Stores.Groups.Get(ctx, groupID); err == nil {
			if loc, locErr := time.LoadLocation(group.Timezone); locErr == nil {
				return loc
			}
		}
	}
	if user, err := s.Stores.Users.Get(ctx, userID); err == nil {
		if loc, locErr := time.LoadLocation(user.Timezone); locErr == nil {
			return loc
		}
	}
	return time.UTC
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
func formatDate(t time.Time, loc *time.Location) string {
	if t.IsZero() {
		return ""
	}
	return t.In(loc).Format("2006-01-02")
}

func expenseRow(v domain.Expense, loc *time.Location, lookupName func(string) string) []string {
	return []string{"支出", formatDate(v.IncurredOn, loc), v.Title, formatAmount(v.AmountMinor), string(v.Currency), v.Category, lookupName(v.PaidBy), "", splitDetail(v.Splits, lookupName), v.Notes}
}

// splitDetail renders every participant's share as "姓名:金額" pairs so the
// export fully reflects how an expense was divided, not just its total.
func splitDetail(splits []domain.ExpenseSplit, lookupName func(string) string) string {
	if len(splits) == 0 {
		return ""
	}
	parts := make([]string, 0, len(splits))
	for _, split := range splits {
		parts = append(parts, fmt.Sprintf("%s:%s", lookupName(split.UserID), formatAmount(split.AmountMinor)))
	}
	return strings.Join(parts, "; ")
}

// subscriptionRows returns one row per billing period that isn't already
// represented by an expense row. A "posted" occurrence always has a matching
// Expense (created by postSubscriptionOccurrence), so including it here too
// would double the amount in the exported ledger; a "failed" occurrence never
// gets an Expense, so it's the only period-level detail worth surfacing. A
// subscription with no occurrences yet (nothing has billed) still gets one
// row so it isn't silently missing from the export.
func subscriptionRows(v domain.Subscription, loc *time.Location, lookupName func(string) string) [][]string {
	revisionByID := make(map[string]domain.SubscriptionRevision, len(v.Revisions))
	for _, revision := range v.Revisions {
		revisionByID[revision.ID] = revision
	}

	var rows [][]string
	for _, occurrence := range v.Occurrences {
		if occurrence.Status == "posted" {
			continue
		}
		name, category, amount, currency, paidBy := v.Name, v.Category, v.AmountMinor, v.Currency, v.PaidBy
		if revision, ok := revisionByID[occurrence.RevisionID]; ok {
			name, category, amount, currency, paidBy = revision.Name, revision.Category, revision.AmountMinor, revision.Currency, revision.PaidBy
		}
		rows = append(rows, []string{"訂閱", formatDate(occurrence.BillingAt, loc), name, formatAmount(amount), string(currency), category, lookupName(paidBy), occurrence.Status, "", v.Notes})
	}
	if len(rows) == 0 {
		rows = append(rows, []string{"訂閱", formatDate(v.StartsOn, loc), v.Name, formatAmount(v.AmountMinor), string(v.Currency), v.Category, lookupName(v.PaidBy), string(v.Status), "", v.Notes})
	}
	return rows
}

func settlementRow(v domain.Settlement, loc *time.Location, lookupName func(string) string) []string {
	party := fmt.Sprintf("%s → %s", lookupName(v.FromUserID), lookupName(v.ToUserID))
	return []string{"還款", formatDate(v.SettledOn, loc), "", formatAmount(v.AmountMinor), string(v.Currency), "", party, "", "", v.Notes}
}
