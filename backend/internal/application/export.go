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
// history. locale selects which language the written content (headers,
// labels, category names) is translated into; anything other than "en"
// falls back to zh-TW, mirroring the frontend's own default.
func (s *Service) ExportLedger(ctx context.Context, userID, groupID, locale string) ([]byte, string, error) {
	msgs := exportMessagesFor(locale)
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
	categories := map[string]*domain.Category{}
	lookupCategory := func(id string) *domain.Category {
		if id == "" {
			return nil
		}
		if v, ok := categories[id]; ok {
			return v
		}
		v, _ := s.Stores.Categories.Get(ctx, id)
		categories[id] = v
		return v
	}

	var buf bytes.Buffer
	buf.WriteString("\xEF\xBB\xBF") // UTF-8 BOM so Excel opens CJK text correctly.
	w := csv.NewWriter(&buf)
	tzNote := fmt.Sprintf(msgs.tzNotePersonal, loc.String())
	if groupID != "" {
		tzNote = fmt.Sprintf(msgs.tzNoteGroup, loc.String())
	}
	// Padded to the same column count as every other row: csv.Reader (and
	// some spreadsheet tools) treat a short first record as fixing the
	// field count for the whole file, which would otherwise misparse every
	// subsequent 10-column row as "too many fields".
	noteRow := make([]string, len(msgs.headers))
	noteRow[0] = tzNote
	_ = w.Write(noteRow)
	_ = w.Write(msgs.headers)

	if groupID == "" {
		expenses, err := listAllPersonalExpenses(ctx, s, userID)
		if err != nil {
			return nil, "", err
		}
		for _, v := range expenses {
			_ = w.Write(expenseRow(v, loc, lookupName, msgs))
		}
		subs, err := listAllPersonalSubscriptions(ctx, s, userID)
		if err != nil {
			return nil, "", err
		}
		for _, v := range subs {
			for _, row := range subscriptionRows(v, loc, lookupName, lookupCategory, msgs) {
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
			_ = w.Write(expenseRow(v, loc, lookupName, msgs))
		}
		subs, err := listAllSubscriptions(ctx, s, userID, groupID)
		if err != nil {
			return nil, "", err
		}
		for _, v := range subs {
			for _, row := range subscriptionRows(v, loc, lookupName, lookupCategory, msgs) {
				_ = w.Write(row)
			}
		}
		settlements, err := listAllSettlements(ctx, s, userID, groupID)
		if err != nil {
			return nil, "", err
		}
		for _, v := range settlements {
			_ = w.Write(settlementRow(v, loc, lookupName, msgs))
		}
	}

	w.Flush()
	if err := w.Error(); err != nil {
		return nil, "", err
	}

	return buf.Bytes(), s.exportFilename(ctx, userID, groupID), nil
}

// exportLocation resolves which timezone dates should be shown in: the
// group's accounting timezone first (matching how the app already computes
// billing/month boundaries in group time), falling back to the viewer's own
// timezone, and finally UTC if neither is set or valid. This is the
// timezone the CSV *content* uses, and is called out in a note row so it
// isn't ambiguous once the file leaves the app.
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

// exportFilename always stamps the export in the *exporting user's own*
// timezone, independent of exportLocation's content timezone (which for a
// group export is the group's accounting timezone): the filename records
// when the viewer personally triggered the download, not when the group's
// books are dated.
func (s *Service) exportFilename(ctx context.Context, userID, groupID string) string {
	site := "SubFlow"
	if settings, err := s.Stores.Settings.Get(ctx); err == nil {
		if sanitized := sanitizeFilenamePart(settings.SiteName); sanitized != "" {
			site = sanitized
		}
	}
	scope, id := "personal", userID
	if groupID != "" {
		scope, id = "group", groupID
	}
	userLoc := time.UTC
	if user, err := s.Stores.Users.Get(ctx, userID); err == nil {
		if l, locErr := time.LoadLocation(user.Timezone); locErr == nil {
			userLoc = l
		}
	}
	stamp := s.Now().In(userLoc).Format("20060102-150405-0700")
	return fmt.Sprintf("%s-ledger-%s-%s-%s.csv", site, scope, id, stamp)
}

// sanitizeFilenamePart keeps a filename segment portable across OSes and
// browsers by dropping everything but ASCII letters/digits and collapsing
// any run of other characters (spaces, CJK, punctuation) into a single
// hyphen. A site name that sanitizes to nothing (e.g. a purely CJK name)
// falls back to the default "SubFlow" in the caller.
func sanitizeFilenamePart(v string) string {
	var b strings.Builder
	lastDash := false
	for _, r := range strings.TrimSpace(v) {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash && b.Len() > 0 {
				b.WriteRune('-')
				lastDash = true
			}
		}
	}
	return strings.TrimRight(b.String(), "-")
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

func listAllGroups(ctx context.Context, s *Service, userID string) ([]domain.Group, error) {
	var all []domain.Group
	for page := 1; ; page++ {
		result, err := s.ListGroups(ctx, userID, ports.PageRequest{Page: page, PerPage: exportPageSize})
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

func expenseRow(v domain.Expense, loc *time.Location, lookupName func(string) string, msgs exportMessages) []string {
	return []string{msgs.typeExpense, formatDate(v.IncurredOn, loc), v.Title, formatAmount(v.AmountMinor), string(v.Currency), msgs.categoryLabel(v.CategoryInfo, v.Category), lookupName(v.PaidBy), "", splitDetail(v.Splits, lookupName), v.Notes}
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
// synthetic placeholder row (using the subscription's own current settings)
// is only added when the subscription has *never* billed at all — i.e.
// v.Occurrences is empty. If occurrences exist but every one of them is
// "posted" (the normal steady state once a subscription is caught up, and
// always true right after a historical backfill), every period already has
// its own expense row, and adding this placeholder would double-count the
// first period under a second "訂閱" row dated on StartsOn.
func subscriptionRows(v domain.Subscription, loc *time.Location, lookupName func(string) string, lookupCategory func(string) *domain.Category, msgs exportMessages) [][]string {
	revisionByID := make(map[string]domain.SubscriptionRevision, len(v.Revisions))
	for _, revision := range v.Revisions {
		revisionByID[revision.ID] = revision
	}

	var rows [][]string
	for _, occurrence := range v.Occurrences {
		if occurrence.Status == "posted" {
			continue
		}
		name, amount, currency, paidBy := v.Name, v.AmountMinor, v.Currency, v.PaidBy
		categoryInfo, categoryLegacy := v.CategoryInfo, v.Category
		if revision, ok := revisionByID[occurrence.RevisionID]; ok {
			name, amount, currency, paidBy = revision.Name, revision.AmountMinor, revision.Currency, revision.PaidBy
			categoryInfo, categoryLegacy = lookupCategory(revision.CategoryID), revision.Category
		}
		rows = append(rows, []string{msgs.typeSubscription, formatDate(occurrence.BillingAt, loc), name, formatAmount(amount), string(currency), msgs.categoryLabel(categoryInfo, categoryLegacy), lookupName(paidBy), msgs.occurrenceStatusLabel(occurrence.Status), "", v.Notes})
	}
	if len(v.Occurrences) == 0 {
		rows = append(rows, []string{msgs.typeSubscription, formatDate(v.StartsOn, loc), v.Name, formatAmount(v.AmountMinor), string(v.Currency), msgs.categoryLabel(v.CategoryInfo, v.Category), lookupName(v.PaidBy), msgs.subscriptionStatusLabel(v.Status), "", v.Notes})
	}
	return rows
}

func settlementRow(v domain.Settlement, loc *time.Location, lookupName func(string) string, msgs exportMessages) []string {
	party := fmt.Sprintf("%s → %s", lookupName(v.FromUserID), lookupName(v.ToUserID))
	return []string{msgs.typeSettlement, formatDate(v.SettledOn, loc), "", formatAmount(v.AmountMinor), string(v.Currency), "", party, "", "", v.Notes}
}

// exportMessages holds every piece of written text the export CSV needs,
// translated up front for the requested locale so row-building code never
// writes raw literals directly into the file.
type exportMessages struct {
	headers                                                                  []string
	typeExpense, typeSubscription, typeSettlement                            string
	statusActive, statusPaused, statusCancelled, statusPending, statusFailed string
	uncategorized                                                            string
	category                                                                 map[string]string
	tzNoteGroup, tzNotePersonal                                              string
}

func (m exportMessages) categoryLabel(info *domain.Category, legacy string) string {
	if info != nil {
		if info.SystemKey != "" {
			if label, ok := m.category[info.SystemKey]; ok {
				return label
			}
		}
		if info.CustomName != "" {
			return info.CustomName
		}
	}
	if legacy != "" {
		return legacy
	}
	return m.uncategorized
}

func (m exportMessages) subscriptionStatusLabel(status domain.SubscriptionStatus) string {
	switch status {
	case domain.SubscriptionActive:
		return m.statusActive
	case domain.SubscriptionPaused:
		return m.statusPaused
	case domain.SubscriptionCancelled:
		return m.statusCancelled
	default:
		return string(status)
	}
}

func (m exportMessages) occurrenceStatusLabel(status string) string {
	switch status {
	case "pending":
		return m.statusPending
	case "failed":
		return m.statusFailed
	default:
		return status
	}
}

// exportMessagesFor mirrors frontend/src/locales/{zh-TW,en}.ts and
// frontend/src/category.ts's systemKeys map so the export reads the same
// language as the rest of the app. Anything other than "en" defaults to
// zh-TW, matching the frontend's own locale fallback in i18n.ts.
func exportMessagesFor(locale string) exportMessages {
	if locale == "en" {
		return exportMessages{
			headers:          []string{"Type", "Date", "Name", "Amount", "Currency", "Category", "Payer / Counterparty", "Status", "Split detail", "Notes"},
			typeExpense:      "Expense",
			typeSubscription: "Subscription",
			typeSettlement:   "Settlement",
			statusActive:     "Active",
			statusPaused:     "Paused",
			statusCancelled:  "Cancelled",
			statusPending:    "Not yet billed",
			statusFailed:     "Posting failed",
			uncategorized:    "Uncategorized",
			category: map[string]string{
				"food_dining": "Food & dining", "transport": "Transport", "housing": "Housing", "utilities": "Utilities & bills",
				"shopping": "Shopping", "entertainment": "Entertainment", "health": "Health", "education": "Education",
				"travel": "Travel", "insurance": "Insurance", "software_digital": "Software & digital services",
				"memberships": "Memberships", "taxes_fees": "Taxes & fees", "gifts_donations": "Gifts & donations", "other": "Other",
			},
			tzNoteGroup:    "Export timezone: %s (the group's accounting time zone)",
			tzNotePersonal: "Export timezone: %s (your personal time zone)",
		}
	}
	return exportMessages{
		headers:          []string{"類型", "日期", "名稱", "金額", "幣別", "分類", "付款人/對象", "狀態", "分帳明細", "備註"},
		typeExpense:      "支出",
		typeSubscription: "訂閱",
		typeSettlement:   "還款",
		statusActive:     "啟用",
		statusPaused:     "暫停",
		statusCancelled:  "已取消",
		statusPending:    "尚未入帳",
		statusFailed:     "入帳失敗",
		uncategorized:    "未分類",
		category: map[string]string{
			"food_dining": "餐飲", "transport": "交通", "housing": "住家", "utilities": "水電與帳單",
			"shopping": "購物", "entertainment": "娛樂", "health": "健康", "education": "教育",
			"travel": "旅行", "insurance": "保險", "software_digital": "軟體與數位服務",
			"memberships": "會員", "taxes_fees": "稅務與手續費", "gifts_donations": "禮物與捐款", "other": "其他",
		},
		tzNoteGroup:    "匯出時區：%s（群組記帳時區）",
		tzNotePersonal: "匯出時區：%s（你的個人時區）",
	}
}
