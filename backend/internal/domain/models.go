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
	ID           string             `json:"id"`
	GroupID      string             `json:"groupId"`
	Name         string             `json:"name"`
	Category     string             `json:"category"`
	AmountMinor  int64              `json:"amountMinor"`
	Currency     Currency           `json:"currency"`
	BillingCycle BillingCycle       `json:"billingCycle"`
	NextBilling  time.Time          `json:"nextBilling"`
	Status       SubscriptionStatus `json:"status"`
	Notes        string             `json:"notes"`
	CreatedAt    time.Time          `json:"createdAt"`
	UpdatedAt    time.Time          `json:"updatedAt"`
}

type Expense struct {
	ID          string    `json:"id"`
	GroupID     string    `json:"groupId"`
	Title       string    `json:"title"`
	Category    string    `json:"category"`
	AmountMinor int64     `json:"amountMinor"`
	PaidBy      string    `json:"paidBy"`
	IncurredOn  time.Time `json:"incurredOn"`
	Notes       string    `json:"notes"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type DashboardSummary struct {
	MonthlySubscriptionMinor int64          `json:"monthlySubscriptionMinor"`
	MonthExpenseMinor        int64          `json:"monthExpenseMinor"`
	ActiveSubscriptions      int            `json:"activeSubscriptions"`
	Upcoming                 []Subscription `json:"upcoming"`
}

type Event struct {
	Type       string    `json:"type"`
	GroupID    string    `json:"groupId"`
	Resource   string    `json:"resource"`
	ResourceID string    `json:"resourceId"`
	OccurredAt time.Time `json:"occurredAt"`
}
