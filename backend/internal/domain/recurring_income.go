package domain

import "time"

// RecurringIncome owns a schedule, never a posted balance. Only the Income
// linked from a posted occurrence contributes to accounting totals.
type RecurringIncome struct {
	ID          string       `json:"id"`
	OwnerID     string       `json:"ownerId,omitempty"`
	GroupID     string       `json:"groupId,omitempty"`
	Template    Income       `json:"template"`
	Timezone    string       `json:"timezone"`
	Cycle       BillingCycle `json:"cycle"`
	Interval    int          `json:"interval"`
	StartsOn    time.Time    `json:"startsOn"`
	EndsOn      *time.Time   `json:"endsOn,omitempty"`
	NextOn      time.Time    `json:"nextOn"`
	PostingMode string       `json:"postingMode"`
	Status      string       `json:"status"`
	CreatedAt   time.Time    `json:"createdAt"`
	UpdatedAt   time.Time    `json:"updatedAt"`
}

type RecurringIncomeRevision struct {
	ID                string     `json:"id"`
	RecurringIncomeID string     `json:"recurringIncomeId"`
	EffectiveOn       time.Time  `json:"effectiveOn"`
	Through           *time.Time `json:"through,omitempty"`
	Scope             string     `json:"scope"`
	Template          Income     `json:"template"`
	PostingMode       string     `json:"postingMode"`
	CreatedAt         time.Time  `json:"createdAt"`
}

type IncomeOccurrence struct {
	ID                string    `json:"id"`
	RecurringIncomeID string    `json:"recurringIncomeId"`
	RevisionID        string    `json:"revisionId"`
	ScheduledOn       time.Time `json:"scheduledOn"`
	Status            string    `json:"status"`
	PostingMode       string    `json:"postingMode,omitempty"`
	IncomeID          string    `json:"incomeId,omitempty"`
	Template          Income    `json:"template"`
	ManualOverride    bool      `json:"manualOverride"`
	Reason            string    `json:"reason,omitempty"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

type IncomePeriodPreview struct {
	Periods []IncomeOccurrence `json:"periods"`
	Total   int                `json:"total"`
	Token   string             `json:"token"`
}
