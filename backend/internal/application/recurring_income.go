package application

import (
	"context"
	"errors"
	"log/slog"
	"sort"
	"strings"
	"subflow/internal/domain"
	"subflow/internal/ports"
	"time"
)

const maxIncomeBackfillPeriods = 600

func (s *Service) AuthorizeRecurringIncome(ctx context.Context, actor string, v *domain.RecurringIncome, action string) error {
	return s.recurringPermission(ctx, actor, v, action)
}
func (s *Service) RecurringIncomeRequestFailure(ctx context.Context, actor string, v *domain.RecurringIncome, action string, cause error) error {
	if strings.HasPrefix(action, "ledger.") {
		action = "recurring_income.access_denied"
	}
	if !strings.HasPrefix(action, "recurring_income.") && !strings.HasPrefix(action, "income_occurrence.") {
		action = "recurring_income." + action
	}
	return s.recurringIncomeFailure(ctx, actor, v, action, cause)
}

func (s *Service) recurringPermission(ctx context.Context, actor string, v *domain.RecurringIncome, action string) error {
	if v.GroupID != "" {
		if err := s.groupPermission(ctx, actor, v.GroupID, action); err != nil {
			return err
		}
	} else if v.OwnerID != actor {
		return domain.ErrForbidden
	}
	return nil
}
func (s *Service) normalizeRecurringIncome(ctx context.Context, actor string, v *domain.RecurringIncome) error {
	if v.Interval < 1 || v.StartsOn.IsZero() || v.NextOn.IsZero() || !validBillingCycle(v.Cycle) || v.PostingMode != "automatic" && v.PostingMode != "manual" {
		return domain.ErrInvalid
	}
	if v.Status == "" {
		v.Status = "active"
	}
	if v.Status != "active" && v.Status != "paused" && v.Status != "ended" {
		return domain.ErrInvalid
	}
	v.Template.ReceivedOn = v.StartsOn
	if v.GroupID != "" {
		v.OwnerID = ""
		v.Template.GroupID = v.GroupID
		if err := s.prepareGroupIncome(ctx, actor, &v.Template, nil); err != nil {
			return err
		}
	} else {
		v.GroupID = ""
		v.OwnerID = actor
		v.Template.OwnerID = actor
		if err := s.preparePersonalIncome(ctx, actor, &v.Template); err != nil {
			return err
		}
	}
	if v.Timezone == "" {
		v.Timezone = "UTC"
	}
	if _, err := time.LoadLocation(v.Timezone); err != nil {
		return domain.ErrInvalid
	}
	loc, _ := time.LoadLocation(v.Timezone)
	v.StartsOn = v.StartsOn.In(loc)
	v.NextOn = v.NextOn.In(loc)
	if v.EndsOn != nil {
		end := v.EndsOn.In(loc)
		v.EndsOn = &end
		if end.Before(v.StartsOn) {
			return domain.ErrInvalid
		}
	}
	return nil
}
func validBillingCycle(v domain.BillingCycle) bool {
	switch v {
	case domain.BillingDaily, domain.BillingEveryNDays, domain.BillingWeekly, domain.BillingEveryNWeeks, domain.BillingEveryNHours, domain.BillingMonthly, domain.BillingQuarterly, domain.BillingYearly:
		return true
	}
	return false
}
func (s *Service) CreateRecurringIncome(ctx context.Context, actor string, v domain.RecurringIncome) (*domain.RecurringIncome, error) {
	if err := s.normalizeRecurringIncome(ctx, actor, &v); err != nil {
		return nil, s.recurringIncomeFailure(ctx, actor, &v, "recurring_income.created", err)
	}
	err := s.Stores.Transactions.Within(ctx, func(tx context.Context) error {
		if err := s.Stores.RecurringIncomes.Create(tx, &v); err != nil {
			return err
		}
		rev := domain.RecurringIncomeRevision{RecurringIncomeID: v.ID, EffectiveOn: v.StartsOn, Scope: "future", Template: v.Template, PostingMode: v.PostingMode}
		if err := s.Stores.RecurringIncomes.CreateRevision(tx, &rev); err != nil {
			return err
		}
		return s.audit(tx, actor, v.GroupID, "recurring_income.created", "recurring_income", v.ID, "success", encodeAuditSummary(map[string]any{"cycle": v.Cycle, "interval": v.Interval, "posting_mode": v.PostingMode}, nil))
	})
	if err != nil {
		return nil, s.recurringIncomeFailure(ctx, actor, &v, "recurring_income.created", err)
	}
	return &v, nil
}
func (s *Service) UpdateRecurringIncome(ctx context.Context, actor, id string, v domain.RecurringIncome) (*domain.RecurringIncome, error) {
	current, err := s.Stores.RecurringIncomes.Get(ctx, id)
	if err != nil {
		return nil, s.recurringIncomeFailure(ctx, actor, nil, "recurring_income.updated", err)
	}
	if err = s.recurringPermission(ctx, actor, current, "ledger.incomes.write"); err != nil {
		return nil, s.recurringIncomeFailure(ctx, actor, current, "recurring_income.updated", err)
	}
	v.ID = id
	v.OwnerID = current.OwnerID
	v.GroupID = current.GroupID
	v.CreatedAt = current.CreatedAt
	if err = s.normalizeRecurringIncome(ctx, actor, &v); err != nil {
		return nil, s.recurringIncomeFailure(ctx, actor, current, "recurring_income.updated", err)
	}
	if v.StartsOn.Before(current.StartsOn) {
		return nil, s.recurringIncomeFailure(ctx, actor, current, "recurring_income.updated", domain.ErrInvalid)
	}
	v.NextOn = current.NextOn
	err = s.Stores.Transactions.Within(ctx, func(tx context.Context) error {
		rev := domain.RecurringIncomeRevision{RecurringIncomeID: id, EffectiveOn: v.StartsOn, Scope: "future", Template: v.Template, PostingMode: v.PostingMode}
		if err := s.Stores.RecurringIncomes.CreateRevision(tx, &rev); err != nil {
			return err
		}
		if err := s.Stores.RecurringIncomes.Update(tx, &v); err != nil {
			return err
		}
		return s.audit(tx, actor, v.GroupID, "recurring_income.updated", "recurring_income", id, "success", encodeAuditSummary(map[string]any{"effective_on": v.StartsOn.Format(time.RFC3339), "cycle": v.Cycle, "interval": v.Interval, "posting_mode": v.PostingMode}, nil))
	})
	if err != nil {
		return nil, s.recurringIncomeFailure(ctx, actor, current, "recurring_income.updated", err)
	}
	return &v, nil
}
func (s *Service) ListRecurringIncomes(ctx context.Context, actor, group string, p ports.PageRequest) (ports.Page[domain.RecurringIncome], error) {
	if group != "" {
		if err := s.groupPermission(ctx, actor, group, "ledger.incomes.read"); err != nil {
			return ports.Page[domain.RecurringIncome]{}, err
		}
	}
	return s.Stores.RecurringIncomes.List(ctx, actor, group, p)
}
func (s *Service) ListIncomeOccurrences(ctx context.Context, actor, id string, p ports.PageRequest) (ports.Page[domain.IncomeOccurrence], error) {
	series, err := s.Stores.RecurringIncomes.Get(ctx, id)
	if err != nil {
		return ports.Page[domain.IncomeOccurrence]{}, err
	}
	if err = s.recurringPermission(ctx, actor, series, "ledger.incomes.read"); err != nil {
		return ports.Page[domain.IncomeOccurrence]{}, err
	}
	return s.Stores.RecurringIncomes.ListOccurrences(ctx, id, p)
}
func (s *Service) PreviewRecurringIncomeBackfill(ctx context.Context, actor, id string, through time.Time) (*domain.IncomePeriodPreview, error) {
	series, err := s.Stores.RecurringIncomes.Get(ctx, id)
	if err != nil {
		return nil, s.recurringIncomeFailure(ctx, actor, nil, "recurring_income.backfill_previewed", err)
	}
	if err = s.recurringPermission(ctx, actor, series, "ledger.incomes.write"); err != nil {
		return nil, s.recurringIncomeFailure(ctx, actor, series, "recurring_income.backfill_previewed", err)
	}
	if through.IsZero() || through.After(s.Now()) {
		return nil, s.recurringIncomeFailure(ctx, actor, series, "recurring_income.backfill_previewed", domain.ErrInvalid)
	}
	from := series.NextOn
	if from.Before(series.StartsOn) {
		from = series.StartsOn
	}
	if series.GroupID != "" {
		monthStart, _, rangeErr := s.monthRange(ctx, actor, series.GroupID, "")
		if rangeErr != nil {
			return nil, s.recurringIncomeFailure(ctx, actor, series, "recurring_income.backfill_previewed", rangeErr)
		}
		if from.Before(monthStart) {
			if err = s.groupPermission(ctx, actor, series.GroupID, "ledger.records.historical_write"); err != nil {
				return nil, s.recurringIncomeFailure(ctx, actor, series, "recurring_income.backfill_previewed", err)
			}
		}
	}
	loc, _ := time.LoadLocation(series.Timezone)
	dates, err := billingDatesBetween(series.StartsOn.In(loc), series.Cycle, series.Interval, from.In(loc), through.In(loc).Add(time.Nanosecond))
	if err != nil {
		return nil, s.recurringIncomeFailure(ctx, actor, series, "recurring_income.backfill_previewed", err)
	}
	if len(dates) > maxIncomeBackfillPeriods {
		dates = dates[:maxIncomeBackfillPeriods]
	}
	revisions, err := s.Stores.RecurringIncomes.ListRevisions(ctx, id)
	if err != nil {
		return nil, s.recurringIncomeFailure(ctx, actor, series, "recurring_income.backfill_previewed", err)
	}
	sort.Slice(revisions, func(i, j int) bool { return revisions[i].EffectiveOn.Before(revisions[j].EffectiveOn) })
	preview := &domain.IncomePeriodPreview{Periods: make([]domain.IncomeOccurrence, 0, len(dates)), Token: id + ":" + through.UTC().Format(time.RFC3339), Total: len(dates)}
	for _, date := range dates {
		if series.EndsOn != nil && date.After(*series.EndsOn) {
			continue
		}
		existing, findErr := s.Stores.RecurringIncomes.GetOccurrence(ctx, id, date)
		if findErr == nil && existing != nil {
			preview.Periods = append(preview.Periods, *existing)
			continue
		}
		if findErr != nil && !errors.Is(findErr, domain.ErrNotFound) {
			return nil, s.recurringIncomeFailure(ctx, actor, series, "recurring_income.backfill_previewed", findErr)
		}
		template, posting, revisionID := series.Template, series.PostingMode, ""
		for _, revision := range revisions {
			if !date.Before(revision.EffectiveOn) && (revision.Through == nil || !date.After(*revision.Through)) {
				template, posting, revisionID = revision.Template, revision.PostingMode, revision.ID
			}
		}
		preview.Periods = append(preview.Periods, domain.IncomeOccurrence{RecurringIncomeID: id, RevisionID: revisionID, ScheduledOn: date, Status: "pending", PostingMode: posting, Template: template})
	}
	preview.Total = len(preview.Periods)
	if err = s.audit(ctx, actor, series.GroupID, "recurring_income.backfill_previewed", "recurring_income", id, "success", encodeAuditSummary(map[string]any{"period_count": preview.Total, "through": through.Format(time.RFC3339)}, nil)); err != nil {
		return nil, errors.Join(domain.ErrAuditUnavailable, err)
	}
	return preview, nil
}
func (s *Service) BackfillRecurringIncome(ctx context.Context, actor, id string, through time.Time) error {
	preview, err := s.PreviewRecurringIncomeBackfill(ctx, actor, id, through)
	if err != nil {
		return err
	}
	series, err := s.Stores.RecurringIncomes.Get(ctx, id)
	if err != nil {
		return s.recurringIncomeFailure(ctx, actor, nil, "recurring_income.backfilled", err)
	}
	err = s.postRecurringSeries(ctx, series, through)
	if err != nil {
		return s.recurringIncomeFailure(ctx, actor, series, "recurring_income.backfilled", err)
	}
	if err = s.audit(ctx, actor, series.GroupID, "recurring_income.backfilled", "recurring_income", id, "success", encodeAuditSummary(map[string]any{"period_count": preview.Total, "through": through.Format(time.RFC3339)}, nil)); err != nil {
		return errors.Join(domain.ErrAuditUnavailable, err)
	}
	return nil
}
func (s *Service) SetRecurringIncomeStatus(ctx context.Context, actor, id, status string, end *time.Time) error {
	v, err := s.Stores.RecurringIncomes.Get(ctx, id)
	if err != nil {
		return s.recurringIncomeFailure(ctx, actor, nil, "recurring_income.status_changed", err)
	}
	if err = s.recurringPermission(ctx, actor, v, "ledger.incomes.write"); err != nil {
		return s.recurringIncomeFailure(ctx, actor, v, "recurring_income.status_changed", err)
	}
	if status != "active" && status != "paused" && status != "ended" {
		return s.recurringIncomeFailure(ctx, actor, v, "recurring_income.status_changed", domain.ErrInvalid)
	}
	before := v.Status
	if status == "active" && before == "paused" && v.NextOn.Before(s.Now()) {
		loc, _ := time.LoadLocation(v.Timezone)
		start := s.Now().In(loc)
		dates, e := billingDatesBetween(v.StartsOn.In(loc), v.Cycle, v.Interval, start, start.AddDate(10, 0, 0))
		if e != nil {
			return s.recurringIncomeFailure(ctx, actor, v, "recurring_income.status_changed", e)
		}
		v.NextOn = start
		if len(dates) > 0 {
			v.NextOn = dates[0]
		}
	}
	v.Status = status
	if end != nil {
		loc, _ := time.LoadLocation(v.Timezone)
		x := end.In(loc)
		v.EndsOn = &x
	}
	err = s.Stores.Transactions.Within(ctx, func(tx context.Context) error {
		if err := s.Stores.RecurringIncomes.Update(tx, v); err != nil {
			return err
		}
		return s.audit(tx, actor, v.GroupID, "recurring_income.status_changed", "recurring_income", id, "success", encodeAuditSummary(map[string]any{"before": before, "after": status, "ends_on": func() string {
			if v.EndsOn == nil {
				return ""
			}
			return v.EndsOn.Format(time.RFC3339)
		}()}, nil))
	})
	if err != nil {
		return s.recurringIncomeFailure(ctx, actor, v, "recurring_income.status_changed", err)
	}
	return nil
}
func (s *Service) SkipIncomeOccurrence(ctx context.Context, actor, id string, on time.Time, reason string) error {
	series, err := s.Stores.RecurringIncomes.Get(ctx, id)
	if err != nil {
		return s.recurringIncomeFailure(ctx, actor, nil, "income_occurrence.skipped", err)
	}
	if err = s.recurringPermission(ctx, actor, series, "ledger.incomes.write"); err != nil {
		return s.recurringIncomeFailure(ctx, actor, series, "income_occurrence.skipped", err)
	}
	if strings.TrimSpace(reason) == "" {
		return s.recurringIncomeFailure(ctx, actor, series, "income_occurrence.skipped", domain.ErrInvalid)
	}
	old, _ := s.Stores.RecurringIncomes.GetOccurrence(ctx, id, on)
	if old != nil && (old.Status == "posted" || old.Status == "deleted") {
		return s.recurringIncomeFailure(ctx, actor, series, "income_occurrence.skipped", domain.ErrConflict)
	}
	occ := domain.IncomeOccurrence{RecurringIncomeID: id, ScheduledOn: on, Status: "skipped", Template: series.Template, Reason: strings.TrimSpace(reason)}
	if old != nil {
		occ.ID = old.ID
	}
	err = s.Stores.Transactions.Within(ctx, func(tx context.Context) error {
		if err := s.Stores.RecurringIncomes.SaveOccurrence(tx, &occ); err != nil {
			return err
		}
		return s.audit(tx, actor, series.GroupID, "income_occurrence.skipped", "income_occurrence", occ.ID, "success", encodeAuditSummary(map[string]any{"scheduled_on": on.Format(time.RFC3339), "reason": occ.Reason}, nil))
	})
	if err != nil {
		return s.recurringIncomeFailure(ctx, actor, series, "income_occurrence.skipped", err)
	}
	return nil
}

func (s *Service) ConfirmIncomeOccurrence(ctx context.Context, actor, id string, on time.Time, actual domain.Income) (*domain.Income, error) {
	series, err := s.Stores.RecurringIncomes.Get(ctx, id)
	if err != nil {
		return nil, s.recurringIncomeFailure(ctx, actor, nil, "income_occurrence.confirmed", err)
	}
	if err = s.recurringPermission(ctx, actor, series, "ledger.incomes.write"); err != nil {
		return nil, s.recurringIncomeFailure(ctx, actor, series, "income_occurrence.confirmed", err)
	}
	occ, err := s.Stores.RecurringIncomes.GetOccurrence(ctx, id, on)
	if err != nil {
		return nil, s.recurringIncomeFailure(ctx, actor, series, "income_occurrence.confirmed", err)
	}
	if occ.Status != "pending" || occ.IncomeID != "" {
		return nil, s.recurringIncomeFailure(ctx, actor, series, "income_occurrence.confirmed", domain.ErrConflict)
	}
	if actual.Title == "" {
		actual = occ.Template
	}
	actual.ID = ""
	actual.GroupID = series.GroupID
	actual.OwnerID = series.OwnerID
	if actual.ReceivedOn.IsZero() {
		actual.ReceivedOn = on
	}
	if actual.EarnedBy == "" {
		actual.EarnedBy = occ.Template.EarnedBy
	}
	if series.GroupID != "" {
		err = s.prepareGroupIncome(ctx, actor, &actual, nil)
	} else {
		err = s.preparePersonalIncome(ctx, actor, &actual)
	}
	if err != nil {
		return nil, s.recurringIncomeFailure(ctx, actor, series, "income_occurrence.confirmed", err)
	}
	old := *occ
	occ.Template = actual
	occ.Status = "posted"
	occ.IncomeID = ""
	occ.ManualOverride = actual.AmountMinor != old.Template.AmountMinor || !actual.ReceivedOn.Equal(on)
	err = s.Stores.Transactions.Within(ctx, func(tx context.Context) error {
		if err := s.Stores.Incomes.Create(tx, &actual); err != nil {
			return err
		}
		if err := s.Stores.Incomes.ReplaceSplits(tx, actual.ID, actual.Splits); err != nil {
			return err
		}
		occ.IncomeID = actual.ID
		if err := s.Stores.RecurringIncomes.SaveOccurrence(tx, occ); err != nil {
			return err
		}
		return s.audit(tx, actor, series.GroupID, "income_occurrence.confirmed", "income_occurrence", occ.ID, "success", encodeAuditSummary(map[string]any{"income_id": actual.ID, "amount_minor": actual.AmountMinor, "scheduled_on": on.Format(time.RFC3339)}, nil))
	})
	if err != nil {
		return nil, s.recurringIncomeFailure(ctx, actor, series, "income_occurrence.confirmed", err)
	}
	return &actual, nil
}

func (s *Service) RetryIncomeOccurrence(ctx context.Context, actor, id string, on time.Time) error {
	series, err := s.Stores.RecurringIncomes.Get(ctx, id)
	if err != nil {
		return s.recurringIncomeFailure(ctx, actor, nil, "income_occurrence.retried", err)
	}
	if err = s.recurringPermission(ctx, actor, series, "ledger.incomes.write"); err != nil {
		return s.recurringIncomeFailure(ctx, actor, series, "income_occurrence.retried", err)
	}
	occ, err := s.Stores.RecurringIncomes.GetOccurrence(ctx, id, on)
	if err != nil {
		return s.recurringIncomeFailure(ctx, actor, series, "income_occurrence.retried", err)
	}
	if occ.Status != "failed" {
		return s.recurringIncomeFailure(ctx, actor, series, "income_occurrence.retried", domain.ErrConflict)
	}
	next := nextIncomeOccurrence(series, on, func() *time.Location { loc, _ := time.LoadLocation(series.Timezone); return loc }())
	occ.Status = "pending"
	occ.Reason = ""
	err = s.Stores.Transactions.Within(ctx, func(tx context.Context) error {
		if err := s.Stores.RecurringIncomes.SaveOccurrence(tx, occ); err != nil {
			return err
		}
		return s.audit(tx, actor, series.GroupID, "income_occurrence.retried", "income_occurrence", occ.ID, "success", encodeAuditSummary(map[string]any{"scheduled_on": on.Format(time.RFC3339)}, nil))
	})
	if err != nil {
		return s.recurringIncomeFailure(ctx, actor, series, "income_occurrence.retried", err)
	}
	if series.PostingMode == "manual" {
		return nil
	}
	if err := s.postAutomaticIncomeOccurrence(ctx, series, occ, on, next); err != nil {
		if markErr := s.markIncomeOccurrenceFailed(ctx, actor, series, occ, err); markErr != nil {
			return errors.Join(err, markErr)
		}
		return err
	}
	return nil
}

func (s *Service) PostDueRecurringIncomes(ctx context.Context) error {
	series, err := s.Stores.RecurringIncomes.ListDue(ctx, s.Now())
	if err != nil {
		return err
	}
	var failures []error
	for i := range series {
		if err := s.postRecurringSeries(ctx, &series[i], s.Now()); err != nil {
			failures = append(failures, err)
		}
	}
	return errors.Join(failures...)
}
func (s *Service) postRecurringSeries(ctx context.Context, v *domain.RecurringIncome, until time.Time) error {
	loc, err := time.LoadLocation(v.Timezone)
	if err != nil {
		return err
	}
	dates, err := billingDatesBetween(v.StartsOn.In(loc), v.Cycle, v.Interval, v.NextOn.In(loc), until.In(loc).Add(time.Nanosecond))
	if err != nil {
		return err
	}
	if len(dates) > maxIncomeBackfillPeriods {
		dates = dates[:maxIncomeBackfillPeriods]
	}
	revisions, err := s.Stores.RecurringIncomes.ListRevisions(ctx, v.ID)
	if err != nil {
		return err
	}
	sort.Slice(revisions, func(i, j int) bool { return revisions[i].EffectiveOn.Before(revisions[j].EffectiveOn) })
	for _, date := range dates {
		if v.EndsOn != nil && date.After(*v.EndsOn) {
			v.Status = "ended"
			break
		}
		existing, findErr := s.Stores.RecurringIncomes.GetOccurrence(ctx, v.ID, date)
		if findErr == nil && existing != nil {
			v.NextOn = nextIncomeOccurrence(v, date, loc)
			if updateErr := s.Stores.Transactions.Within(ctx, func(tx context.Context) error {
				if err := s.Stores.RecurringIncomes.Update(tx, v); err != nil {
					return err
				}
				return s.audit(tx, recurringActor(v), v.GroupID, "income_occurrence.reconciled", "income_occurrence", existing.ID, "success", encodeAuditSummary(map[string]any{"scheduled_on": date.Format(time.RFC3339)}, nil))
			}); updateErr != nil {
				return s.recurringIncomeFailure(ctx, recurringActor(v), v, "income_occurrence.reconciled", updateErr)
			}
			continue
		}
		if findErr != nil && !errors.Is(findErr, domain.ErrNotFound) {
			return s.recurringIncomeFailure(ctx, recurringActor(v), v, "income_occurrence.posted", findErr)
		}
		template := v.Template
		posting := v.PostingMode
		revisionID := ""
		for _, revision := range revisions {
			if !date.Before(revision.EffectiveOn) && (revision.Through == nil || !date.After(*revision.Through)) {
				template = revision.Template
				posting = revision.PostingMode
				revisionID = revision.ID
			}
		}
		occ := domain.IncomeOccurrence{RecurringIncomeID: v.ID, RevisionID: revisionID, ScheduledOn: date, Template: template, Status: "pending", PostingMode: posting}
		next := nextIncomeOccurrence(v, date, loc)
		if next.After(until) {
			v.NextOn = next
		}
		if posting == "manual" {
			if err = s.Stores.Transactions.Within(ctx, func(tx context.Context) error {
				if err := s.Stores.RecurringIncomes.SaveOccurrence(tx, &occ); err != nil {
					return err
				}
				if err := s.Stores.RecurringIncomes.Update(tx, v); err != nil {
					return err
				}
				return s.audit(tx, recurringActor(v), v.GroupID, "income_occurrence.pending", "income_occurrence", occ.ID, "success", encodeAuditSummary(map[string]any{"scheduled_on": date.Format(time.RFC3339)}, nil))
			}); err != nil {
				return err
			}
			continue
		}
		err = s.postAutomaticIncomeOccurrence(ctx, v, &occ, date, next)
		if err != nil {
			if markErr := s.markIncomeOccurrenceFailed(ctx, recurringActor(v), v, &occ, err); markErr != nil {
				return errors.Join(err, markErr)
			}
			return err
		}
	}
	return nil
}
func recurringActor(v *domain.RecurringIncome) string {
	if v.OwnerID != "" {
		return v.OwnerID
	}
	return v.Template.EarnedBy
}
func (s *Service) markIncomeOccurrenceFailed(ctx context.Context, actor string, series *domain.RecurringIncome, occurrence *domain.IncomeOccurrence, cause error) error {
	occurrence.Status = "failed"
	occurrence.Reason = "posting_failed"
	var notification *domain.Notification
	err := s.Stores.Transactions.Within(ctx, func(tx context.Context) error {
		if saveErr := s.Stores.RecurringIncomes.SaveOccurrence(tx, occurrence); saveErr != nil {
			return saveErr
		}
		canNotify := true
		if series.GroupID != "" {
			if _, roleErr := s.Stores.Memberships.GetRole(tx, series.GroupID, actor); errors.Is(roleErr, domain.ErrNotFound) {
				canNotify = false
			} else if roleErr != nil {
				return roleErr
			}
		}
		if canNotify {
			eventKey := "income_occurrence:" + occurrence.ID
			if _, findErr := s.Stores.Notifications.FindByEventKey(tx, actor, eventKey); errors.Is(findErr, domain.ErrNotFound) {
				var createErr error
				notification, createErr = s.createNotification(tx, actor, domain.NotificationCategoryException, eventKey, "recurring_income_post_failed", series.GroupID, series.ID)
				if createErr != nil {
					return createErr
				}
			} else if findErr != nil {
				return findErr
			}
		}
		return s.audit(tx, actor, series.GroupID, "income_occurrence.post_failed", "income_occurrence", occurrence.ID, "failure", encodeAuditSummary(map[string]any{"scheduled_on": occurrence.ScheduledOn.Format(time.RFC3339), "reason": "posting_failed"}, nil))
	})
	if err != nil {
		if auditErr := s.audit(ctx, actor, series.GroupID, "income_occurrence.post_failed", "income_occurrence", occurrence.ID, "failure", encodeAuditSummary(map[string]any{"scheduled_on": occurrence.ScheduledOn.Format(time.RFC3339), "reason": "posting_failed"}, nil)); auditErr != nil {
			return errors.Join(domain.ErrAuditUnavailable, auditErr, cause)
		}
		return errors.Join(err, cause)
	}
	s.publishNotification(ctx, notification, "created")
	return nil
}
func nextIncomeOccurrence(v *domain.RecurringIncome, date time.Time, loc *time.Location) time.Time {
	dates, err := billingDatesBetween(v.StartsOn.In(loc), v.Cycle, v.Interval, date.Add(time.Nanosecond), date.AddDate(10, 0, 0))
	if err == nil && len(dates) > 0 {
		return dates[0]
	}
	return date.AddDate(10, 0, 0)
}
func (s *Service) postAutomaticIncomeOccurrence(ctx context.Context, series *domain.RecurringIncome, occ *domain.IncomeOccurrence, date, next time.Time) error {
	actor := series.OwnerID
	if actor == "" {
		actor = series.Template.EarnedBy
	}
	template := occ.Template
	template.ReceivedOn = date
	err := s.Stores.Transactions.Within(ctx, func(tx context.Context) error {
		if series.GroupID != "" {
			if err := s.prepareGroupIncome(tx, actor, &template, nil); err != nil {
				return err
			}
		} else if err := s.preparePersonalIncome(tx, actor, &template); err != nil {
			return err
		}
		if err := s.Stores.Incomes.Create(tx, &template); err != nil {
			return err
		}
		if err := s.Stores.Incomes.ReplaceSplits(tx, template.ID, template.Splits); err != nil {
			return err
		}
		occ.Template = template
		occ.Status = "posted"
		occ.IncomeID = template.ID
		if err := s.Stores.RecurringIncomes.SaveOccurrence(tx, occ); err != nil {
			return err
		}
		if err := s.Stores.Notifications.SetEventStatus(tx, actor, "income_occurrence:"+occ.ID, domain.NotificationStatusResolved, "posted", s.Now().UTC()); err != nil {
			return err
		}
		series.NextOn = next
		if series.EndsOn != nil && next.After(*series.EndsOn) {
			series.Status = "ended"
		}
		if err := s.Stores.RecurringIncomes.Update(tx, series); err != nil {
			return err
		}
		return s.audit(tx, actor, series.GroupID, "income_occurrence.posted", "income_occurrence", occ.ID, "success", encodeAuditSummary(map[string]any{"income_id": template.ID, "scheduled_on": date.Format(time.RFC3339), "amount_minor": template.AmountMinor}, nil))
	})
	if err != nil {
		return s.recurringIncomeFailure(ctx, actor, series, "income_occurrence.posted", err)
	}
	return nil
}
func (s *Service) recurringIncomeFailure(ctx context.Context, actor string, v *domain.RecurringIncome, action string, cause error) error {
	group := ""
	if v != nil {
		group = v.GroupID
	}
	if err := s.audit(ctx, actor, group, action, "recurring_income", func() string {
		if v != nil {
			return v.ID
		}
		return ""
	}(), "failure", encodeAuditSummary(map[string]any{"reason": "recurring_income_operation_failed"}, nil)); err != nil {
		slog.Error("recurring income failure audit could not be persisted", "action", action, "audit_error", err)
		return errors.Join(domain.ErrAuditUnavailable, err, cause)
	}
	return cause
}
