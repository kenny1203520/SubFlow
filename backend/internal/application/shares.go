package application

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"subflow/internal/domain"
	"subflow/internal/ports"
)

type ShareInput struct {
	domain.Share
	ViewerEmails []string `json:"viewerEmails"`
	Password     string   `json:"password"`
}

type CreatedShare struct {
	*domain.Share
	URL string `json:"url,omitempty"`
}

func shareToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}
func shareTokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func (s *Service) sharePermission(ctx context.Context, userID, groupID string) error {
	if groupID == "" {
		return nil
	}
	return s.groupPermission(ctx, userID, groupID, "ledger.share.manage")
}

func validShare(value domain.Share) bool {
	if strings.TrimSpace(value.Name) == "" || (value.GroupID == "" && value.OwnerID == "") || (value.GroupID != "" && value.OwnerID != "") {
		return false
	}
	if value.AccessMode != "link" && value.AccessMode != "password" && value.AccessMode != "accounts" {
		return false
	}
	if value.RangeMode != "all" && value.RangeMode != "rolling" && value.RangeMode != "fixed" {
		return false
	}
	if value.RangeMode == "rolling" && value.RollingDays != 30 && value.RollingDays != 90 && value.RollingDays != 365 {
		return false
	}
	if value.RangeMode == "fixed" && (value.StartsOn.IsZero() || value.EndsOn.IsZero() || value.StartsOn.After(value.EndsOn)) {
		return false
	}
	if value.OwnerID != "" {
		return value.ShowSummary || value.ShowExpenses || value.ShowSubscriptions
	}
	return value.ShowSummary || value.ShowExpenses || value.ShowSubscriptions || value.ShowSettlements
}

func (s *Service) resolveShareViewers(ctx context.Context, emails []string) ([]domain.ShareViewer, error) {
	result, seen := make([]domain.ShareViewer, 0, len(emails)), map[string]bool{}
	for _, email := range emails {
		email = strings.ToLower(strings.TrimSpace(email))
		if email == "" {
			continue
		}
		user, err := s.Stores.Users.FindByEmail(ctx, email)
		if err != nil || user.Placeholder {
			return nil, domain.ErrInvalid
		}
		if !seen[user.ID] {
			result, seen[user.ID] = append(result, domain.ShareViewer{UserID: user.ID, Email: user.Email, Name: user.Name}), true
		}
	}
	return result, nil
}

func (s *Service) CreateShare(ctx context.Context, userID, groupID string, input ShareInput) (*CreatedShare, error) {
	if err := s.sharePermission(ctx, userID, groupID); err != nil {
		return nil, err
	}
	value := input.Share
	value.GroupID, value.OwnerID = groupID, ""
	if groupID == "" {
		value.OwnerID = userID
	}
	if !validShare(value) || (value.AccessMode == "password" && len(strings.TrimSpace(input.Password)) < 8) {
		return nil, domain.ErrInvalid
	}
	viewers, err := s.resolveShareViewers(ctx, input.ViewerEmails)
	if err != nil {
		return nil, err
	}
	if value.AccessMode == "accounts" && len(viewers) == 0 {
		return nil, domain.ErrInvalid
	}
	token, err := shareToken()
	if err != nil {
		return nil, err
	}
	value.TokenHash, value.Enabled, value.AccessVersion = shareTokenHash(token), true, 1
	if value.AccessMode == "password" {
		hash, hashErr := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
		if hashErr != nil {
			return nil, hashErr
		}
		value.PasswordHash = string(hash)
	}
	err = s.Stores.Transactions.Within(ctx, func(tx context.Context) error {
		if createErr := s.Stores.Shares.Create(tx, &value); createErr != nil {
			return createErr
		}
		if viewerErr := s.Stores.Shares.ReplaceViewers(tx, value.ID, viewers); viewerErr != nil {
			return viewerErr
		}
		return s.audit(tx, userID, groupID, "share.created", "share", value.ID, "success", encodeAuditSummary(map[string]any{"mode": value.AccessMode}, nil))
	})
	if err != nil {
		return nil, err
	}
	value.Viewers = viewers
	return &CreatedShare{Share: &value, URL: "/share/" + token}, nil
}

func (s *Service) ListShares(ctx context.Context, userID, groupID string, page ports.PageRequest) (ports.Page[domain.Share], error) {
	if err := s.sharePermission(ctx, userID, groupID); err != nil {
		return ports.Page[domain.Share]{}, err
	}
	ownerID := ""
	if groupID == "" {
		ownerID = userID
	}
	result, err := s.Stores.Shares.List(ctx, groupID, ownerID, page)
	if err != nil {
		return result, err
	}
	for i := range result.Items {
		result.Items[i].Viewers, _ = s.Stores.Shares.ListViewers(ctx, result.Items[i].ID)
	}
	return result, nil
}

func (s *Service) managedShare(ctx context.Context, userID, id string) (*domain.Share, error) {
	value, err := s.Stores.Shares.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if value.GroupID != "" {
		if err = s.sharePermission(ctx, userID, value.GroupID); err != nil {
			return nil, err
		}
	} else if value.OwnerID != userID {
		return nil, domain.ErrForbidden
	}
	return value, nil
}

func (s *Service) UpdateShare(ctx context.Context, userID, id string, input ShareInput) (*domain.Share, error) {
	current, err := s.managedShare(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	value := input.Share
	value.ID, value.GroupID, value.OwnerID, value.TokenHash, value.CreatedAt = current.ID, current.GroupID, current.OwnerID, current.TokenHash, current.CreatedAt
	value.PasswordHash, value.AccessVersion = current.PasswordHash, current.AccessVersion
	if !validShare(value) {
		return nil, domain.ErrInvalid
	}
	if value.AccessMode == "password" {
		if input.Password != "" {
			hash, hashErr := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
			if hashErr != nil {
				return nil, hashErr
			}
			value.PasswordHash, value.AccessVersion = string(hash), current.AccessVersion+1
		}
		if value.PasswordHash == "" {
			return nil, domain.ErrInvalid
		}
	} else {
		value.PasswordHash = ""
	}
	viewers, err := s.resolveShareViewers(ctx, input.ViewerEmails)
	if err != nil {
		return nil, err
	}
	if value.AccessMode == "accounts" && len(viewers) == 0 {
		return nil, domain.ErrInvalid
	}
	err = s.Stores.Transactions.Within(ctx, func(tx context.Context) error {
		if updateErr := s.Stores.Shares.Update(tx, &value); updateErr != nil {
			return updateErr
		}
		if viewerErr := s.Stores.Shares.ReplaceViewers(tx, value.ID, viewers); viewerErr != nil {
			return viewerErr
		}
		return s.audit(tx, userID, value.GroupID, "share.updated", "share", value.ID, "success", encodeAuditSummary(map[string]any{"mode": value.AccessMode, "enabled": value.Enabled}, nil))
	})
	if err != nil {
		return nil, err
	}
	value.Viewers = viewers
	return &value, nil
}

func (s *Service) RotateShare(ctx context.Context, userID, id string) (*CreatedShare, error) {
	value, err := s.managedShare(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	token, err := shareToken()
	if err != nil {
		return nil, err
	}
	value.TokenHash, value.AccessVersion = shareTokenHash(token), value.AccessVersion+1
	err = s.Stores.Transactions.Within(ctx, func(tx context.Context) error {
		if updateErr := s.Stores.Shares.Update(tx, value); updateErr != nil {
			return updateErr
		}
		return s.audit(tx, userID, value.GroupID, "share.rotated", "share", value.ID, "success", encodeAuditSummary(map[string]any{"mode": value.AccessMode}, nil))
	})
	if err != nil {
		return nil, err
	}
	return &CreatedShare{Share: value, URL: "/share/" + token}, nil
}

func (s *Service) DeleteShare(ctx context.Context, userID, id string) error {
	value, err := s.managedShare(ctx, userID, id)
	if err != nil {
		return err
	}
	return s.Stores.Transactions.Within(ctx, func(tx context.Context) error {
		if deleteErr := s.Stores.Shares.Delete(tx, id); deleteErr != nil {
			return deleteErr
		}
		return s.audit(tx, userID, value.GroupID, "share.deleted", "share", id, "success", encodeAuditSummary(map[string]any{"mode": value.AccessMode}, nil))
	})
}

func shareDateConfigured(value time.Time) bool {
	return !value.IsZero() && value.Year() > 1
}
func (s *Service) FindAvailableShare(ctx context.Context, token string) (*domain.Share, error) {
	value, err := s.Stores.Shares.GetByTokenHash(ctx, shareTokenHash(token))
	if err != nil || !value.Enabled || (shareDateConfigured(value.ExpiresAt) && !value.ExpiresAt.After(s.Now())) {
		return nil, domain.ErrNotFound
	}
	return value, nil
}

func (s *Service) AuditShareAccess(ctx context.Context, value *domain.Share, actor, outcome, reason string) error {
	return s.audit(ctx, actor, value.GroupID, "share.view", "share", value.ID, outcome, encodeAuditSummary(map[string]any{"mode": value.AccessMode, "reason": reason}, nil))
}

func (s *Service) AuditMissingShareAccess(ctx context.Context, actor, reason string) error {
	return s.audit(ctx, actor, "", "share.view", "share", "", "failure", encodeAuditSummary(map[string]any{"reason": reason}, nil))
}

func (s *Service) VerifySharePassword(ctx context.Context, value *domain.Share, password string) error {
	if value.AccessMode != "password" || bcrypt.CompareHashAndPassword([]byte(value.PasswordHash), []byte(password)) != nil {
		return domain.ErrNotFound
	}
	return nil
}

func (s *Service) AuthorizeShareViewer(ctx context.Context, value *domain.Share, userID string) error {
	if value.AccessMode == "link" {
		return nil
	}
	if value.AccessMode == "password" {
		return domain.ErrForbidden
	}
	viewers, err := s.Stores.Shares.ListViewers(ctx, value.ID)
	if err != nil {
		return err
	}
	for _, viewer := range viewers {
		if viewer.UserID == userID {
			return nil
		}
	}
	return domain.ErrNotFound
}

type SharePageOptions struct {
	Page    int
	PerPage int
	Section string
	Query   string
	Sort    string
}

type shareRecord struct {
	kind       string
	occurredOn time.Time
	row        map[string]any
}

func (s *Service) SharePage(ctx context.Context, value *domain.Share, pageNumber, perPage int) (*domain.SharePage, error) {
	return s.SharePageWithOptions(ctx, value, SharePageOptions{Page: pageNumber, PerPage: perPage})
}

func (s *Service) SharePageWithOptions(ctx context.Context, value *domain.Share, options SharePageOptions) (*domain.SharePage, error) {
	pageNumber := options.Page
	if pageNumber < 1 {
		pageNumber = 1
	}
	perPage := options.PerPage
	if perPage != 5 && perPage != 10 && perPage != 15 && perPage != 25 {
		perPage = 25
	}
	section := options.Section
	if section != "expenses" && section != "subscriptions" && section != "settlements" {
		section = "all"
	}
	sortNewest := options.Sort != "oldest"

	start, end, label, err := s.shareRange(ctx, value)
	if err != nil {
		return nil, err
	}
	page := &domain.SharePage{
		Name: value.Name, RangeLabel: label, ShowSummary: value.ShowSummary,
		Page: pageNumber, PerPage: perPage,
	}
	if value.GroupID != "" {
		group, groupErr := s.Stores.Groups.Get(ctx, value.GroupID)
		if groupErr != nil {
			return nil, groupErr
		}
		page.Currency = group.Currency
	} else if user, userErr := s.Stores.Users.Get(ctx, value.OwnerID); userErr == nil {
		page.Currency = user.DefaultCurrency
	}

	filteredExpenses := make([]domain.Expense, 0)
	if value.ShowExpenses || value.ShowSummary {
		expenses, err := s.allShareExpenses(ctx, value)
		if err != nil {
			return nil, err
		}
		for _, item := range expenses {
			if inRange(item.IncurredOn, start, end) {
				filteredExpenses = append(filteredExpenses, item)
			}
		}
	}
	filteredSubs := make([]domain.Subscription, 0)
	if value.ShowSubscriptions || value.ShowSummary {
		subs, err := s.allShareSubscriptions(ctx, value)
		if err != nil {
			return nil, err
		}
		for _, item := range subs {
			if subscriptionInShareRange(item, start, end) {
				filteredSubs = append(filteredSubs, item)
			}
		}
	}
	settlements := []domain.Settlement{}
	if value.GroupID != "" && (value.ShowSettlements || value.ShowSummary) {
		settlements, err = s.allShareSettlements(ctx, value.GroupID)
		if err != nil {
			return nil, err
		}
		settlements = filterSettlements(settlements, start, end)
	}
	if value.ShowSummary {
		totals := map[string]int64{}
		for _, item := range filteredExpenses {
			totals[string(item.Currency)] += item.AmountMinor
		}
		page.Summary = map[string]any{
			"expenseCount":      len(filteredExpenses),
			"subscriptionCount": len(filteredSubs),
			"settlementCount":   len(settlements),
			"expenseTotals":     totals,
		}
	}

	records := make([]shareRecord, 0, len(filteredExpenses)+len(filteredSubs)+len(settlements))
	if value.ShowExpenses {
		for _, item := range filteredExpenses {
			row := s.shareExpenseRow(ctx, value, item)
			row["type"] = "expense"
			row["occurredOn"] = item.IncurredOn
			records = append(records, shareRecord{kind: "expenses", occurredOn: item.IncurredOn, row: row})
		}
	}
	if value.ShowSubscriptions {
		for _, item := range filteredSubs {
			occurredOn := item.NextBilling
			if occurredOn.IsZero() {
				occurredOn = item.StartsOn
			}
			row := s.shareSubscriptionRow(ctx, value, item)
			row["type"] = "subscription"
			row["occurredOn"] = occurredOn
			records = append(records, shareRecord{kind: "subscriptions", occurredOn: occurredOn, row: row})
		}
	}
	if value.ShowSettlements && value.GroupID != "" {
		for _, item := range settlements {
			row := s.shareSettlementRow(ctx, value, item)
			row["type"] = "settlement"
			row["occurredOn"] = item.SettledOn
			records = append(records, shareRecord{kind: "settlements", occurredOn: item.SettledOn, row: row})
		}
	}

	query := strings.ToLower(strings.TrimSpace(options.Query))
	if query != "" {
		filtered := records[:0]
		for _, item := range records {
			if strings.Contains(shareRecordSearchText(item.row), query) {
				filtered = append(filtered, item)
			}
		}
		records = filtered
	}
	sort.SliceStable(records, func(i, j int) bool {
		if records[i].occurredOn.Equal(records[j].occurredOn) {
			return records[i].kind < records[j].kind
		}
		if sortNewest {
			return records[i].occurredOn.After(records[j].occurredOn)
		}
		return records[i].occurredOn.Before(records[j].occurredOn)
	})

	counts := map[string]int{"all": len(records), "expenses": 0, "subscriptions": 0, "settlements": 0}
	for _, item := range records {
		counts[item.kind]++
	}
	page.RecordCounts = counts

	selected, totalItems, pageNumber, totalPagesValue, nextPage := paginateShareRecords(records, section, pageNumber, perPage)
	page.Page = pageNumber
	page.TotalItems = totalItems
	page.TotalPages = totalPagesValue
	for _, item := range selected {
		page.Records = append(page.Records, item.row)
		switch item.kind {
		case "expenses":
			page.Expenses = append(page.Expenses, item.row)
		case "subscriptions":
			page.Subscriptions = append(page.Subscriptions, item.row)
		case "settlements":
			page.Settlements = append(page.Settlements, item.row)
		}
	}
	if nextPage > 0 {
		page.NextPage = nextPage
	}

	if value.GroupID != "" && value.ShowSettlements {
		balanceExpenses := append([]domain.Expense(nil), filteredExpenses...)
		for i := range balanceExpenses {
			splits, splitErr := s.Stores.Expenses.ListSplits(ctx, balanceExpenses[i].ID)
			if splitErr != nil {
				return nil, splitErr
			}
			balanceExpenses[i].Splits = splits
		}
		for index, balance := range domain.MemberBalances(balanceExpenses, settlements) {
			if balance.AmountMinor == 0 {
				continue
			}
			row := s.shareBalanceRow(ctx, value, balance, index)
			row["currency"] = page.Currency
			page.Balances = append(page.Balances, row)
		}
	}
	return page, nil
}

func paginateShareRecords(records []shareRecord, section string, pageNumber, perPage int) ([]shareRecord, int, int, int, int) {
	selected := records
	if section != "all" {
		selected = make([]shareRecord, 0, len(records))
		for _, item := range records {
			if item.kind == section {
				selected = append(selected, item)
			}
		}
	}
	totalItems := len(selected)
	totalPagesValue := totalPages(totalItems, perPage)
	if totalPagesValue < 1 {
		totalPagesValue = 1
	}
	if pageNumber < 1 {
		pageNumber = 1
	}
	if pageNumber > totalPagesValue {
		pageNumber = totalPagesValue
	}
	start := (pageNumber - 1) * perPage
	if start > totalItems {
		start = totalItems
	}
	end := minInt(start+perPage, totalItems)
	nextPage := 0
	if pageNumber < totalPagesValue {
		nextPage = pageNumber + 1
	}
	return selected[start:end], totalItems, pageNumber, totalPagesValue, nextPage
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}

func shareRecordSearchText(row map[string]any) string {
	keys := make([]string, 0, len(row))
	for key := range row {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var builder strings.Builder
	for _, key := range keys {
		value := row[key]
		switch typed := value.(type) {
		case time.Time:
			builder.WriteString(typed.Format(time.RFC3339))
		default:
			_, _ = fmt.Fprint(&builder, value)
		}
		builder.WriteByte(' ')
	}
	return strings.ToLower(builder.String())
}

func (s *Service) shareRange(ctx context.Context, value *domain.Share) (time.Time, time.Time, string, error) {
	location := time.UTC
	if value.GroupID != "" {
		if group, err := s.Stores.Groups.Get(ctx, value.GroupID); err == nil {
			location, _ = time.LoadLocation(group.Timezone)
		}
	} else if user, err := s.Stores.Users.Get(ctx, value.OwnerID); err == nil {
		location, _ = time.LoadLocation(user.Timezone)
	}
	if location == nil {
		location = time.UTC
	}
	now := s.Now().In(location)
	if value.RangeMode == "all" {
		return time.Time{}, now.AddDate(100, 0, 0), "all", nil
	}
	if value.RangeMode == "rolling" {
		day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, location)
		return day.AddDate(0, 0, -value.RollingDays+1), day.AddDate(0, 0, 1), "last " + strconv.Itoa(value.RollingDays) + " days", nil
	}
	return value.StartsOn.In(location), value.EndsOn.In(location).AddDate(0, 0, 1), value.StartsOn.Format("2006-01-02") + " – " + value.EndsOn.Format("2006-01-02"), nil
}

func inRange(value, start, end time.Time) bool {
	return (start.IsZero() || !value.Before(start)) && value.Before(end)
}
func filterSettlements(values []domain.Settlement, start, end time.Time) []domain.Settlement {
	result := []domain.Settlement{}
	for _, item := range values {
		if inRange(item.SettledOn, start, end) {
			result = append(result, item)
		}
	}
	return result
}
func totalPages(length, perPage int) int {
	if length == 0 {
		return 0
	}
	return (length + perPage - 1) / perPage
}
func subscriptionInShareRange(item domain.Subscription, start, end time.Time) bool {
	return item.StartsOn.Before(end) && (item.EndsOn == nil || !item.EndsOn.Before(start))
}

func (s *Service) allShareExpenses(ctx context.Context, value *domain.Share) ([]domain.Expense, error) {
	result := []domain.Expense{}
	for p := 1; ; p++ {
		var page ports.Page[domain.Expense]
		var err error
		if value.GroupID != "" {
			page, err = s.Stores.Expenses.List(ctx, value.GroupID, ports.PageRequest{Page: p, PerPage: 100, Sort: "-incurred_on"})
		} else {
			page, err = s.Stores.Expenses.ListPersonal(ctx, value.OwnerID, ports.PageRequest{Page: p, PerPage: 100, Sort: "-incurred_on"})
		}
		if err != nil {
			return nil, err
		}
		result = append(result, page.Items...)
		if page.TotalPages == 0 || p >= page.TotalPages {
			return result, nil
		}
	}
}
func (s *Service) allShareSubscriptions(ctx context.Context, value *domain.Share) ([]domain.Subscription, error) {
	result := []domain.Subscription{}
	for p := 1; ; p++ {
		var page ports.Page[domain.Subscription]
		var err error
		if value.GroupID != "" {
			page, err = s.Stores.Subscriptions.List(ctx, value.GroupID, ports.PageRequest{Page: p, PerPage: 100, Sort: "next_billing"})
		} else {
			page, err = s.Stores.Subscriptions.ListPersonal(ctx, value.OwnerID, ports.PageRequest{Page: p, PerPage: 100, Sort: "next_billing"})
		}
		if err != nil {
			return nil, err
		}
		result = append(result, page.Items...)
		if page.TotalPages == 0 || p >= page.TotalPages {
			return result, nil
		}
	}
}
func (s *Service) allShareSettlements(ctx context.Context, groupID string) ([]domain.Settlement, error) {
	result := []domain.Settlement{}
	for p := 1; ; p++ {
		page, err := s.Stores.Settlements.List(ctx, groupID, ports.SettlementQuery{PageRequest: ports.PageRequest{Page: p, PerPage: 100, Sort: "-settled_on"}})
		if err != nil {
			return nil, err
		}
		result = append(result, page.Items...)
		if page.TotalPages == 0 || p >= page.TotalPages {
			return result, nil
		}
	}
}

func (s *Service) displayName(ctx context.Context, id string) string {
	user, err := s.Stores.Users.Get(ctx, id)
	if err != nil {
		return ""
	}
	return user.Name
}
func (s *Service) shareExpenseRow(ctx context.Context, share *domain.Share, v domain.Expense) map[string]any {
	row := map[string]any{"title": v.Title, "category": v.Category, "amountMinor": v.AmountMinor, "currency": v.Currency, "incurredOn": v.IncurredOn}
	if share.ShowIdentities {
		row["paidBy"] = s.displayName(ctx, v.PaidBy)
	}
	if share.ShowNotes {
		row["notes"] = v.Notes
	}
	return row
}
func (s *Service) shareSubscriptionRow(ctx context.Context, share *domain.Share, v domain.Subscription) map[string]any {
	row := map[string]any{"name": v.Name, "category": v.Category, "amountMinor": v.AmountMinor, "currency": v.Currency, "status": v.Status, "billingCycle": v.BillingCycle, "nextBilling": v.NextBilling}
	if share.ShowIdentities {
		row["paidBy"] = s.displayName(ctx, v.PaidBy)
	}
	if share.ShowNotes {
		row["notes"] = v.Notes
	}
	return row
}
func (s *Service) shareBalanceRow(ctx context.Context, share *domain.Share, v domain.MemberBalance, index int) map[string]any {
	label := fmt.Sprintf("Member %d", index+1)
	if share.ShowIdentities {
		if name := s.displayName(ctx, v.UserID); name != "" {
			label = name
		}
	}
	return map[string]any{"member": label, "amountMinor": v.AmountMinor}
}

func (s *Service) shareSettlementRow(ctx context.Context, share *domain.Share, v domain.Settlement) map[string]any {
	row := map[string]any{"amountMinor": v.AmountMinor, "currency": v.Currency, "settledOn": v.SettledOn}
	if share.ShowIdentities {
		row["from"] = s.displayName(ctx, v.FromUserID)
		row["to"] = s.displayName(ctx, v.ToUserID)
	}
	if share.ShowNotes {
		row["notes"] = v.Notes
	}
	return row
}
