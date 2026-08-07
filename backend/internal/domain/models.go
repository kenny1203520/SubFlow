package domain

import "time"

type Currency string

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
	ID              string             `json:"id"`
	GroupID         string             `json:"groupId,omitempty"`
	OwnerID         string             `json:"ownerId,omitempty"`
	PaidBy          string             `json:"paidBy"`
	Name            string             `json:"name"`
	Category        string             `json:"category"`
	AmountMinor     int64              `json:"amountMinor"`
	Currency        Currency           `json:"currency"`
	BillingCycle    BillingCycle       `json:"billingCycle"`
	StartsOn        time.Time          `json:"startsOn"`
	EndsOn          *time.Time         `json:"endsOn,omitempty"`
	NextBilling     time.Time          `json:"nextBilling"`
	Status          SubscriptionStatus `json:"status"`
	LifecycleStatus string             `json:"lifecycleStatus,omitempty"`
	Notes           string             `json:"notes"`
	CreatedAt       time.Time          `json:"createdAt"`
	UpdatedAt       time.Time          `json:"updatedAt"`
}

type Expense struct {
	ID          string         `json:"id"`
	GroupID     string         `json:"groupId,omitempty"`
	OwnerID     string         `json:"ownerId,omitempty"`
	Title       string         `json:"title"`
	Category    string         `json:"category"`
	AmountMinor int64          `json:"amountMinor"`
	Currency    Currency       `json:"currency"`
	PaidBy      string         `json:"paidBy"`
	IncurredOn  time.Time      `json:"incurredOn"`
	Notes       string         `json:"notes"`
	SplitMode   SplitMode      `json:"splitMode,omitempty"`
	Splits      []ExpenseSplit `json:"splits,omitempty"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
}

type ExpenseSplit struct {
	ID                    string `json:"id,omitempty"`
	ExpenseID             string `json:"expenseId,omitempty"`
	UserID                string `json:"userId"`
	AmountMinor           int64  `json:"amountMinor"`
	PercentageBasisPoints int    `json:"percentageBasisPoints,omitempty"`
}

type Settlement struct {
	ID          string    `json:"id"`
	GroupID     string    `json:"groupId"`
	FromUserID  string    `json:"fromUserId"`
	ToUserID    string    `json:"toUserId"`
	CreatedBy   string    `json:"createdBy"`
	AmountMinor int64     `json:"amountMinor"`
	SettledOn   time.Time `json:"settledOn"`
	Notes       string    `json:"notes"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
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
}

type Event struct {
	Type       string    `json:"type"`
	GroupID    string    `json:"groupId"`
	UserID     string    `json:"userId,omitempty"`
	Resource   string    `json:"resource"`
	ResourceID string    `json:"resourceId"`
	OccurredAt time.Time `json:"occurredAt"`
}
