package domain

import "time"

type Currency string

const ExchangeRateScale int64 = 100000000

const (
	CurrencyTWD Currency = "TWD"
	CurrencyUSD Currency = "USD"
	CurrencyJPY Currency = "JPY"
	CurrencyEUR Currency = "EUR"
)

type MemberRole string

const (
	RoleOwner  MemberRole = "owner"
	RoleMember MemberRole = "member"
)

type InvitationStatus string

const (
	InvitationPending        InvitationStatus = "pending"
	InvitationDeliveryFailed InvitationStatus = "delivery_failed"
	InvitationAccepted       InvitationStatus = "accepted"
	InvitationRevoked        InvitationStatus = "revoked"
	InvitationExpired        InvitationStatus = "expired"
)

type SubscriptionStatus string

const (
	SubscriptionActive    SubscriptionStatus = "active"
	SubscriptionPaused    SubscriptionStatus = "paused"
	SubscriptionCancelled SubscriptionStatus = "cancelled"
)

type BillingCycle string

const (
	BillingMonthly   BillingCycle = "monthly"
	BillingQuarterly BillingCycle = "quarterly"
	BillingYearly    BillingCycle = "yearly"
)

type SplitMode string

const (
	SplitEqual      SplitMode = "equal"
	SplitAmount     SplitMode = "amount"
	SplitPercentage SplitMode = "percentage"
)

type RateMode string

const (
	RateAutomatic RateMode = "automatic"
	RateManual    RateMode = "manual"
)

type Category struct {
	ID         string    `json:"id"`
	Scope      string    `json:"scope"`
	OwnerID    string    `json:"ownerId,omitempty"`
	GroupID    string    `json:"groupId,omitempty"`
	SystemKey  string    `json:"systemKey,omitempty"`
	CustomName string    `json:"customName,omitempty"`
	CreatedBy  string    `json:"createdBy,omitempty"`
	Archived   bool      `json:"archived"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type CurrencyInfo struct {
	Code   Currency `json:"code"`
	Digits int      `json:"digits"`
}

type ExchangeRate struct {
	ID            string    `json:"id,omitempty"`
	BaseCurrency  Currency  `json:"baseCurrency"`
	QuoteCurrency Currency  `json:"quoteCurrency"`
	RateScaled    int64     `json:"-"`
	Rate          string    `json:"rate"`
	EffectiveDate time.Time `json:"effectiveDate"`
	Provider      string    `json:"provider"`
	FetchedAt     time.Time `json:"fetchedAt"`
	Stale         bool      `json:"stale,omitempty"`
}

type User struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	Avatar   string `json:"avatar,omitempty"`
	Timezone string `json:"timezone"`
}

type Group struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Currency    Currency  `json:"currency"`
	Timezone    string    `json:"timezone"`
	Color       string    `json:"color"`
	OwnerID     string    `json:"ownerId"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type Membership struct {
	ID        string     `json:"id"`
	GroupID   string     `json:"groupId"`
	UserID    string     `json:"userId"`
	Role      MemberRole `json:"role"`
	User      *User      `json:"user,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
}

type Invitation struct {
	ID         string           `json:"id"`
	GroupID    string           `json:"groupId"`
	Email      string           `json:"email"`
	TokenHash  string           `json:"-"`
	Status     InvitationStatus `json:"status"`
	InvitedBy  string           `json:"invitedBy"`
	AcceptedBy string           `json:"acceptedBy,omitempty"`
	ExpiresAt  time.Time        `json:"expiresAt"`
	CreatedAt  time.Time        `json:"createdAt"`
	UpdatedAt  time.Time        `json:"updatedAt"`
	DebugURL   string           `json:"debugUrl,omitempty"`
}

type Subscription struct {
	ID               string             `json:"id"`
	GroupID          string             `json:"groupId,omitempty"`
	OwnerID          string             `json:"ownerId,omitempty"`
	PaidBy           string             `json:"paidBy"`
	Name             string             `json:"name"`
	Category         string             `json:"category"`
	CategoryID       string             `json:"categoryId,omitempty"`
	CategoryInfo     *Category          `json:"categoryInfo,omitempty"`
	AmountMinor      int64              `json:"amountMinor"`
	Currency         Currency           `json:"currency"`
	BaseCurrency     Currency           `json:"baseCurrency"`
	BaseAmountMinor  int64              `json:"baseAmountMinor"`
	ExchangeRate     string             `json:"exchangeRate"`
	RateScaled       int64              `json:"-"`
	ExchangeRateDate time.Time          `json:"exchangeRateDate"`
	RateMode         RateMode           `json:"rateMode"`
	BillingCycle     BillingCycle       `json:"billingCycle"`
	StartsOn         time.Time          `json:"startsOn"`
	EndsOn           *time.Time         `json:"endsOn,omitempty"`
	NextBilling      time.Time          `json:"nextBilling"`
	Status           SubscriptionStatus `json:"status"`
	LifecycleStatus  string             `json:"lifecycleStatus,omitempty"`
	Notes            string             `json:"notes"`
	CreatedAt        time.Time          `json:"createdAt"`
	UpdatedAt        time.Time          `json:"updatedAt"`
}

type Expense struct {
	ID               string         `json:"id"`
	GroupID          string         `json:"groupId,omitempty"`
	OwnerID          string         `json:"ownerId,omitempty"`
	Title            string         `json:"title"`
	Category         string         `json:"category"`
	CategoryID       string         `json:"categoryId,omitempty"`
	CategoryInfo     *Category      `json:"categoryInfo,omitempty"`
	AmountMinor      int64          `json:"amountMinor"`
	Currency         Currency       `json:"currency"`
	BaseCurrency     Currency       `json:"baseCurrency"`
	BaseAmountMinor  int64          `json:"baseAmountMinor"`
	ExchangeRate     string         `json:"exchangeRate"`
	RateScaled       int64          `json:"-"`
	ExchangeRateDate time.Time      `json:"exchangeRateDate"`
	RateMode         RateMode       `json:"rateMode"`
	PaidBy           string         `json:"paidBy"`
	IncurredOn       time.Time      `json:"incurredOn"`
	Notes            string         `json:"notes"`
	SplitMode        SplitMode      `json:"splitMode,omitempty"`
	Splits           []ExpenseSplit `json:"splits,omitempty"`
	CreatedAt        time.Time      `json:"createdAt"`
	UpdatedAt        time.Time      `json:"updatedAt"`
}

type ExpenseSplit struct {
	ID                    string `json:"id,omitempty"`
	ExpenseID             string `json:"expenseId,omitempty"`
	UserID                string `json:"userId"`
	AmountMinor           int64  `json:"amountMinor"`
	BaseAmountMinor       int64  `json:"baseAmountMinor"`
	PercentageBasisPoints int    `json:"percentageBasisPoints,omitempty"`
}

type Settlement struct {
	ID               string    `json:"id"`
	GroupID          string    `json:"groupId"`
	FromUserID       string    `json:"fromUserId"`
	ToUserID         string    `json:"toUserId"`
	CreatedBy        string    `json:"createdBy"`
	AmountMinor      int64     `json:"amountMinor"`
	Currency         Currency  `json:"currency"`
	BaseCurrency     Currency  `json:"baseCurrency"`
	BaseAmountMinor  int64     `json:"baseAmountMinor"`
	ExchangeRate     string    `json:"exchangeRate"`
	RateScaled       int64     `json:"-"`
	ExchangeRateDate time.Time `json:"exchangeRateDate"`
	SettledOn        time.Time `json:"settledOn"`
	Notes            string    `json:"notes"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

type CurrencyDashboard struct {
	Currency                 Currency `json:"currency"`
	CashOutflowMinor         int64    `json:"cashOutflowMinor"`
	PersonalShareMinor       int64    `json:"personalShareMinor"`
	ReimbursableMinor        int64    `json:"reimbursableMinor"`
	MonthlySubscriptionMinor int64    `json:"monthlySubscriptionMinor"`
	ActiveSubscriptions      int      `json:"activeSubscriptions"`
	ChargeCount              int      `json:"chargeCount"`
}

type MemberBalance struct {
	UserID      string `json:"userId"`
	AmountMinor int64  `json:"amountMinor"`
}

type DashboardSummary struct {
	Month                    string              `json:"month,omitempty"`
	MonthlySubscriptionMinor int64               `json:"monthlySubscriptionMinor"`
	MonthExpenseMinor        int64               `json:"monthExpenseMinor"`
	ActiveSubscriptions      int                 `json:"activeSubscriptions"`
	Upcoming                 []Subscription      `json:"upcoming"`
	Currencies               []CurrencyDashboard `json:"currencies,omitempty"`
	Balances                 []MemberBalance     `json:"balances,omitempty"`
	ReportingCurrency        Currency            `json:"reportingCurrency,omitempty"`
	OriginalCurrencies       []CurrencyDashboard `json:"originalCurrencies,omitempty"`
}

type CurrencyChangeMissing struct {
	Resource string   `json:"resource"`
	ID       string   `json:"id"`
	From     Currency `json:"from"`
	To       Currency `json:"to"`
	Date     string   `json:"date"`
}

type CurrencyChangePreview struct {
	From     Currency                `json:"from"`
	To       Currency                `json:"to"`
	Affected int                     `json:"affected"`
	Missing  []CurrencyChangeMissing `json:"missing"`
}

type Event struct {
	Type       string    `json:"type"`
	GroupID    string    `json:"groupId"`
	UserID     string    `json:"userId,omitempty"`
	Resource   string    `json:"resource"`
	ResourceID string    `json:"resourceId"`
	OccurredAt time.Time `json:"occurredAt"`
}
