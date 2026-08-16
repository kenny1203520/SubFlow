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
	InvitationDeclined       InvitationStatus = "declined"
)

type SubscriptionStatus string

const (
	SubscriptionActive    SubscriptionStatus = "active"
	SubscriptionPaused    SubscriptionStatus = "paused"
	SubscriptionCancelled SubscriptionStatus = "cancelled"
)

type BillingCycle string

const (
	BillingDaily       BillingCycle = "daily"
	BillingEveryNDays  BillingCycle = "every_n_days"
	BillingWeekly      BillingCycle = "weekly"
	BillingEveryNWeeks BillingCycle = "every_n_weeks"
	BillingEveryNHours BillingCycle = "every_n_hours"
	BillingMonthly     BillingCycle = "monthly"
	BillingQuarterly   BillingCycle = "quarterly"
	BillingYearly      BillingCycle = "yearly"
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
	IconKey    string    `json:"iconKey,omitempty"`
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
	ID              string   `json:"id"`
	Email           string   `json:"email"`
	Name            string   `json:"name"`
	Avatar          string   `json:"avatar,omitempty"`
	Timezone        string   `json:"timezone"`
	DefaultCurrency Currency `json:"defaultCurrency"`
	SystemRoleID    string   `json:"systemRoleId,omitempty"`
	// Placeholder marks a login-incapable stand-in identity created for a
	// group member who hasn't joined yet (see Service.CreateTempMember).
	Placeholder bool `json:"placeholder,omitempty"`
	// LinkedUserID is set once a placeholder has been bound to a real
	// account (see CollaborationService.accept), which eagerly repoints every
	// historical expense_split/subscription/settlement reference to the real
	// account and deletes the placeholder row (see Service.repointUserReferences).
	// A placeholder found with this set is legacy data from before that
	// rewrite existed; Service.resolvePlaceholderAliases exists only to keep
	// such pre-existing rows readable, not as steady-state behavior.
	LinkedUserID string `json:"linkedUserId,omitempty"`
}

// LinkedProvider is an OAuth2 provider linked to a user's account (see
// Service.ListLinkedProviders / UnlinkProvider), letting them sign in with
// that provider directly instead of email/password.
type LinkedProvider struct {
	Provider string    `json:"provider"`
	Created  time.Time `json:"created"`
}

// Flow identifiers for CaptchaFlowSettings/Service.VerifyCaptcha. OIDC login
// is deliberately not one of these -- the OAuth provider already handles its
// own bot defense, so SubFlow never gates it with a captcha of its own.
const (
	CaptchaFlowRegister      = "register"
	CaptchaFlowPasswordReset = "passwordReset"
	CaptchaFlowOTPRequest    = "otpRequest"
	CaptchaFlowLogin         = "login"
)

// CaptchaFlowConfig controls one authentication flow's captcha gate
// independently: whether it's gated at all, when the widget mounts and
// starts solving (Trigger: "load" as soon as the form appears, or "submit"
// only once the user clicks through), and whether it solves silently
// (Mode: "invisible") or requires visible interaction ("interactive"),
// wherever the configured CaptchaProvider supports both.
type CaptchaFlowConfig struct {
	Enabled bool   `json:"enabled"`
	Trigger string `json:"trigger"` // "load" | "submit"
	Mode    string `json:"mode"`    // "invisible" | "interactive"
}

// CaptchaFlowSettings holds one CaptchaFlowConfig per gate-able flow. All
// flows share the single global CaptchaProvider/credentials on
// SystemSettings -- only whether/when/how each flow applies that provider is
// per-flow.
type CaptchaFlowSettings struct {
	Register      CaptchaFlowConfig `json:"register"`
	PasswordReset CaptchaFlowConfig `json:"passwordReset"`
	OTPRequest    CaptchaFlowConfig `json:"otpRequest"`
	Login         CaptchaFlowConfig `json:"login"`
}

// For looks up the config for a flow identifier (one of the CaptchaFlow*
// constants). An unrecognized flow returns a disabled config, so an unknown
// or mistyped flow name fails safe (never gates) rather than failing open.
func (s CaptchaFlowSettings) For(flow string) CaptchaFlowConfig {
	switch flow {
	case CaptchaFlowRegister:
		return s.Register
	case CaptchaFlowPasswordReset:
		return s.PasswordReset
	case CaptchaFlowOTPRequest:
		return s.OTPRequest
	case CaptchaFlowLogin:
		return s.Login
	default:
		return CaptchaFlowConfig{}
	}
}

// SystemSettings is the single installation-wide configuration record. It is
// intentionally separate from a user profile so the initial setup can be
// completed exactly once.
type SystemSettings struct {
	Initialized               bool                `json:"initialized"`
	SiteName                  string              `json:"siteName"`
	DefaultTimezone           string              `json:"defaultTimezone"`
	DefaultCurrency           Currency            `json:"defaultCurrency"`
	AllowRegistration         bool                `json:"allowRegistration"` // legacy response compatibility
	AllowPasswordRegistration bool                `json:"allowPasswordRegistration"`
	AllowOIDCRegistration     bool                `json:"allowOidcRegistration"`
	CaptchaProvider           string              `json:"captchaProvider,omitempty"`
	CaptchaSiteKey            string              `json:"captchaSiteKey,omitempty"`
	CaptchaChallengeURL       string              `json:"captchaChallengeUrl,omitempty"`
	CaptchaVerifyURL          string              `json:"captchaVerifyUrl,omitempty"`
	CaptchaSecret             string              `json:"captchaSecret,omitempty"`
	CaptchaConfigured         bool                `json:"captchaConfigured"`
	CaptchaFlows              CaptchaFlowSettings `json:"captchaFlows"`
	CaptchaSecretCiphertext   string              `json:"-"`
	SetupTokenHash            string              `json:"-"`
	SetupTokenIssued          bool                `json:"-"`
}

type SetupInput struct {
	AdminName                 string   `json:"adminName"`
	Email                     string   `json:"email"`
	Password                  string   `json:"password"`
	SiteName                  string   `json:"siteName"`
	DefaultTimezone           string   `json:"defaultTimezone"`
	DefaultCurrency           Currency `json:"defaultCurrency"`
	AllowRegistration         bool     `json:"allowRegistration"`
	AllowPasswordRegistration bool     `json:"allowPasswordRegistration"`
	AllowOIDCRegistration     bool     `json:"allowOidcRegistration"`
	CaptchaToken              string   `json:"captchaToken"`
	Token                     string   `json:"token"`
}

type Role struct {
	ID          string    `json:"id"`
	Scope       string    `json:"scope"`
	GroupID     string    `json:"groupId,omitempty"`
	Name        string    `json:"name"`
	Category    string    `json:"category,omitempty"`
	Key         string    `json:"key"`
	Permissions []string  `json:"permissions"`
	Protected   bool      `json:"protected"`
	CreatedBy   string    `json:"createdBy,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type AuditLog struct {
	ID         string    `json:"id"`
	ActorID    string    `json:"actorId,omitempty"`
	ActorName  string    `json:"actorName,omitempty"`
	GroupID    string    `json:"groupId,omitempty"`
	Scope      string    `json:"scope"`
	Action     string    `json:"action"`
	Resource   string    `json:"resource"`
	ResourceID string    `json:"resourceId,omitempty"`
	Outcome    string    `json:"outcome"`
	Summary    string    `json:"summary,omitempty"`
	IP         string    `json:"ip,omitempty"`
	UserAgent  string    `json:"userAgent,omitempty"`
	Hash       string    `json:"hash"`
	CreatedAt  time.Time `json:"createdAt"`
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
	RoleID    string     `json:"roleId,omitempty"`
	RoleName  string     `json:"roleName,omitempty"`
	User      *User      `json:"user,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
}

type OwnershipTransferStatus string

const (
	OwnershipTransferPending   OwnershipTransferStatus = "pending"
	OwnershipTransferAccepted  OwnershipTransferStatus = "accepted"
	OwnershipTransferDeclined  OwnershipTransferStatus = "declined"
	OwnershipTransferCancelled OwnershipTransferStatus = "cancelled"
)

// OwnershipTransfer is an invite-and-accept handoff of a group's single
// owner slot: the current owner proposes a target member, and only that
// member accepting actually moves Group.OwnerID (see
// CollaborationService-style flow in application.Service). A group can have
// at most one pending transfer at a time.
type OwnershipTransfer struct {
	ID         string                  `json:"id"`
	GroupID    string                  `json:"groupId"`
	FromUserID string                  `json:"fromUserId"`
	ToUserID   string                  `json:"toUserId"`
	Status     OwnershipTransferStatus `json:"status"`
	CreatedAt  time.Time               `json:"createdAt"`
	UpdatedAt  time.Time               `json:"updatedAt"`
}

type MemberTransferStatus string

const (
	MemberTransferPending   MemberTransferStatus = "pending"
	MemberTransferAccepted  MemberTransferStatus = "accepted"
	MemberTransferDeclined  MemberTransferStatus = "declined"
	MemberTransferCancelled MemberTransferStatus = "cancelled"
)

// MemberTransfer is an invite-and-accept handoff of one real member's entire
// group-scoped history (expenses, subscriptions incl. every revision,
// settlements, group-scope categories) to another real member, followed by
// removing FromUserID from the group -- a full identity handover, conceptually
// the same repointing a temp-member bind does (see
// Service.repointUserReferences), just between two already-real accounts and
// requiring ToUserID's consent. Unlike OwnershipTransfer, a group may have
// several of these pending at once for different member pairs, so pending
// conflicts are scoped per FromUserID rather than per group.
type MemberTransfer struct {
	ID         string               `json:"id"`
	GroupID    string               `json:"groupId"`
	FromUserID string               `json:"fromUserId"`
	ToUserID   string               `json:"toUserId"`
	Status     MemberTransferStatus `json:"status"`
	CreatedAt  time.Time            `json:"createdAt"`
	UpdatedAt  time.Time            `json:"updatedAt"`
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
	Group      *Group           `json:"groupInfo,omitempty"`
	// TargetPlaceholderID, when set, is the user ID of a placeholder "temp
	// member" (see Service.CreateTempMember). Accepting this invitation
	// binds that placeholder to the accepting user rather than only
	// creating a fresh membership.
	TargetPlaceholderID string `json:"targetPlaceholderId,omitempty"`
}

// Notification is deliberately small and resource-oriented so its displayed
// wording remains localised by the client.  The first notification producer is
// group invitations, while the type can be extended without changing the UI
// contract.
type Notification struct {
	ID         string     `json:"id"`
	UserID     string     `json:"userId"`
	Type       string     `json:"type"`
	GroupID    string     `json:"groupId,omitempty"`
	ResourceID string     `json:"resourceId,omitempty"`
	ReadAt     *time.Time `json:"readAt,omitempty"`
	CreatedAt  time.Time  `json:"createdAt"`
	UpdatedAt  time.Time  `json:"updatedAt"`
}

type Subscription struct {
	ID                 string             `json:"id"`
	GroupID            string             `json:"groupId,omitempty"`
	OwnerID            string             `json:"ownerId,omitempty"`
	PaidBy             string             `json:"paidBy"`
	Name               string             `json:"name"`
	Category           string             `json:"category"`
	CategoryID         string             `json:"categoryId,omitempty"`
	CategoryInfo       *Category          `json:"categoryInfo,omitempty"`
	AmountMinor        int64              `json:"amountMinor"`
	Currency           Currency           `json:"currency"`
	BaseCurrency       Currency           `json:"baseCurrency"`
	BaseAmountMinor    int64              `json:"baseAmountMinor"`
	ExchangeRate       string             `json:"exchangeRate"`
	RateScaled         int64              `json:"-"`
	ExchangeRateDate   time.Time          `json:"exchangeRateDate"`
	RateMode           RateMode           `json:"rateMode"`
	BillingCycle       BillingCycle       `json:"billingCycle"`
	BillingInterval    int                `json:"billingInterval"`
	StartsOn           time.Time          `json:"startsOn"`
	EndsOn             *time.Time         `json:"endsOn,omitempty"`
	NextBilling        time.Time          `json:"nextBilling"`
	Status             SubscriptionStatus `json:"status"`
	LifecycleStatus    string             `json:"lifecycleStatus,omitempty"`
	Notes              string             `json:"notes"`
	SplitMode          SplitMode          `json:"splitMode,omitempty"`
	Splits             []ExpenseSplit     `json:"splits,omitempty"`
	RevisionScope      string             `json:"revisionScope,omitempty"`
	EffectiveBillingAt time.Time          `json:"effectiveBillingAt,omitempty"`
	// EndBillingAt, when set alongside RevisionScope "future", bounds the
	// edit to a closed range (EffectiveBillingAt through EndBillingAt
	// inclusive) instead of applying from EffectiveBillingAt onward
	// indefinitely. See Service.UpdateSubscription.
	EndBillingAt *time.Time               `json:"endBillingAt,omitempty"`
	Revisions    []SubscriptionRevision   `json:"revisions,omitempty"`
	Occurrences  []SubscriptionOccurrence `json:"occurrences,omitempty"`
	CreatedAt    time.Time                `json:"createdAt"`
	UpdatedAt    time.Time                `json:"updatedAt"`
}

// SubscriptionRevision is an immutable accounting snapshot.  A future
// revision applies from EffectiveBillingAt onward, while an one_off revision
// only applies to that exact legal billing date. When EndBillingAt is set, a
// future-scoped revision only applies through that billing date (a closed
// A-through-B range) instead of indefinitely.
type SubscriptionRevision struct {
	ID                 string         `json:"id"`
	SubscriptionID     string         `json:"subscriptionId"`
	Scope              string         `json:"scope"`
	EffectiveBillingAt time.Time      `json:"effectiveBillingAt"`
	EndBillingAt       *time.Time     `json:"endBillingAt,omitempty"`
	Name               string         `json:"name"`
	Category           string         `json:"category"`
	CategoryID         string         `json:"categoryId,omitempty"`
	AmountMinor        int64          `json:"amountMinor"`
	Currency           Currency       `json:"currency"`
	BaseCurrency       Currency       `json:"baseCurrency"`
	BaseAmountMinor    int64          `json:"baseAmountMinor"`
	ExchangeRate       string         `json:"exchangeRate"`
	RateScaled         int64          `json:"-"`
	ExchangeRateDate   time.Time      `json:"exchangeRateDate"`
	RateMode           RateMode       `json:"rateMode"`
	PaidBy             string         `json:"paidBy"`
	SplitMode          SplitMode      `json:"splitMode"`
	Splits             []ExpenseSplit `json:"splits,omitempty"`
	Notes              string         `json:"notes,omitempty"`
	CreatedAt          time.Time      `json:"createdAt"`
}

type SubscriptionOccurrence struct {
	ID             string    `json:"id"`
	SubscriptionID string    `json:"subscriptionId"`
	RevisionID     string    `json:"revisionId"`
	ExpenseID      string    `json:"expenseId,omitempty"`
	BillingAt      time.Time `json:"billingAt"`
	Status         string    `json:"status"`
	Error          string    `json:"error,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
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
	SubscriptionID   string         `json:"subscriptionId,omitempty"`
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
	Currency                         Currency `json:"currency"`
	CashOutflowMinor                 int64    `json:"cashOutflowMinor"`
	PersonalShareMinor               int64    `json:"personalShareMinor"`
	ReimbursableMinor                int64    `json:"reimbursableMinor"`
	MonthlySubscriptionMinor         int64    `json:"monthlySubscriptionMinor"`
	PersonalMonthlySubscriptionMinor int64    `json:"personalMonthlySubscriptionMinor"`
	ActiveSubscriptions              int      `json:"activeSubscriptions"`
	ChargeCount                      int      `json:"chargeCount"`
}

type MemberBalance struct {
	UserID      string `json:"userId"`
	AmountMinor int64  `json:"amountMinor"`
}

type DashboardSummary struct {
	Month                            string              `json:"month,omitempty"`
	MonthlySubscriptionMinor         int64               `json:"monthlySubscriptionMinor"`
	PersonalMonthlySubscriptionMinor int64               `json:"personalMonthlySubscriptionMinor"`
	MonthExpenseMinor                int64               `json:"monthExpenseMinor"`
	ActiveSubscriptions              int                 `json:"activeSubscriptions"`
	Upcoming                         []Subscription      `json:"upcoming"`
	Currencies                       []CurrencyDashboard `json:"currencies,omitempty"`
	Balances                         []MemberBalance     `json:"balances,omitempty"`
	ReportingCurrency                Currency            `json:"reportingCurrency,omitempty"`
	OriginalCurrencies               []CurrencyDashboard `json:"originalCurrencies,omitempty"`
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
