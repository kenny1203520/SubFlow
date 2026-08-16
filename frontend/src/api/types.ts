export type Currency = string
export type RateMode = 'automatic'|'manual'
export interface CurrencyInfo { code:Currency;digits:number }
export interface Category { id:string;scope:'system'|'personal'|'group';ownerId?:string;groupId?:string;systemKey?:string;customName?:string;iconKey?:string;createdBy?:string;archived:boolean;createdAt:string;updatedAt:string }
export interface ExchangeRate { baseCurrency:Currency;quoteCurrency:Currency;rate:string;effectiveDate:string;provider:string;fetchedAt:string;stale?:boolean }
export interface CurrencyChangePreview { from:Currency;to:Currency;affected:number;missing:{resource:string;id:string;from:Currency;to:Currency;date:string}[] }
export type Role = 'owner' | 'member'
export type BillingCycle = 'daily' | 'every_n_days' | 'weekly' | 'every_n_weeks' | 'every_n_hours' | 'monthly' | 'quarterly' | 'yearly'
export type SubscriptionStatus = 'active' | 'paused' | 'cancelled'

export interface Group { id:string;name:string;description:string;currency:Currency;timezone:string;color:string;ownerId:string;createdAt:string;updatedAt:string }
export interface User { id:string;email:string;name:string;avatar?:string;timezone:string;defaultCurrency?:Currency;systemRoleId?:string;placeholder?:boolean;linkedUserId?:string }
export interface SystemAccess { permissions:string[] }
export interface GroupAccess { permissions:string[] }
export interface Membership { id:string;groupId:string;userId:string;role:Role;roleId?:string;roleName?:string;user?:User;createdAt:string }
export interface AccessRole { id:string;scope:'system'|'group';groupId?:string;name:string;category:string;key:string;permissions:string[];protected:boolean;createdBy?:string;createdAt:string;updatedAt:string }
export interface AuditLog { id:string;actorId:string;actorName?:string;groupId?:string;scope:'system'|'group';action:string;resource:string;resourceId:string;outcome:'success'|'failure';summary?:string;ip?:string;userAgent?:string;createdAt:string }
export interface Invitation { id:string;groupId:string;email:string;status:'pending'|'delivery_failed'|'accepted'|'declined'|'revoked'|'expired';invitedBy:string;acceptedBy?:string;expiresAt:string;debugUrl?:string;groupInfo?:Group;createdAt:string;updatedAt:string }
export interface OwnershipTransfer { id:string;groupId:string;fromUserId:string;toUserId:string;status:'pending'|'accepted'|'declined'|'cancelled';createdAt:string;updatedAt:string }
export interface MemberTransfer { id:string;groupId:string;fromUserId:string;toUserId:string;status:'pending'|'accepted'|'declined'|'cancelled';createdAt:string;updatedAt:string }
export type CaptchaTrigger = 'load'|'submit'
export type CaptchaMode = 'invisible'|'interactive'
export interface CaptchaFlowConfig { enabled:boolean; trigger:CaptchaTrigger; mode:CaptchaMode }
export interface CaptchaFlowSettings { register:CaptchaFlowConfig; passwordReset:CaptchaFlowConfig; otpRequest:CaptchaFlowConfig; login:CaptchaFlowConfig }
export interface Notification { id:string;userId:string;type:string;groupId?:string;resourceId?:string;readAt?:string;createdAt:string;updatedAt:string }
export type SplitMode='equal'|'amount'|'percentage'
export interface ExpenseSplit { id?:string;expenseId?:string;userId:string;amountMinor:number;baseAmountMinor?:number;percentageBasisPoints?:number }
export interface SubscriptionRevision { id:string;subscriptionId:string;scope:'future'|'one_off';effectiveBillingAt:string;endBillingAt?:string;name:string;category:string;categoryId?:string;amountMinor:number;currency:Currency;baseCurrency:Currency;baseAmountMinor:number;exchangeRate:string;rateScaled:number;exchangeRateDate:string;rateMode:RateMode;paidBy:string;splitMode:SplitMode;splits:ExpenseSplit[];createdAt:string }
export interface SubscriptionOccurrence { id:string;subscriptionId:string;revisionId:string;expenseId?:string;billingAt:string;status:'pending'|'posted'|'failed';error?:string;createdAt:string;updatedAt:string }
interface ConvertedRecord { categoryId?:string;categoryInfo?:Category;baseCurrency:Currency;baseAmountMinor:number;exchangeRate:string;exchangeRateDate:string;rateMode:RateMode }
// pendingSync/syncError exist only on the client: set on a record that was
// created/changed/deleted locally while offline and hasn't round-tripped to
// the server yet (or failed to). The server never sends these fields.
export interface OfflineState { pendingSync?:boolean;syncError?:string }
export interface Subscription extends ConvertedRecord, OfflineState { id:string;groupId?:string;ownerId?:string;paidBy:string;name:string;category:string;amountMinor:number;currency:Currency;billingCycle:BillingCycle;billingInterval?:number;startsOn:string;endsOn?:string;nextBilling:string;status:SubscriptionStatus;lifecycleStatus?:'active'|'paused'|'ending'|'ended'|'cancelled';notes:string;splitMode?:SplitMode;splits?:ExpenseSplit[];revisionScope?:'future'|'one_off';effectiveBillingAt?:string;endBillingAt?:string;revisions?:SubscriptionRevision[];occurrences?:SubscriptionOccurrence[];createdAt:string;updatedAt:string }
export interface Expense extends ConvertedRecord, OfflineState { id:string;groupId?:string;ownerId?:string;title:string;category:string;amountMinor:number;currency:Currency;paidBy:string;incurredOn:string;notes:string;splitMode?:SplitMode;splits?:ExpenseSplit[];createdAt:string;updatedAt:string }
export interface Settlement extends OfflineState { id:string;groupId:string;fromUserId:string;toUserId:string;createdBy:string;amountMinor:number;currency:Currency;baseCurrency:Currency;baseAmountMinor:number;exchangeRate:string;exchangeRateDate:string;settledOn:string;notes:string;createdAt:string;updatedAt:string }
export interface CurrencyDashboard { currency:Currency;cashOutflowMinor:number;personalShareMinor:number;reimbursableMinor:number;monthlySubscriptionMinor:number;personalMonthlySubscriptionMinor:number;activeSubscriptions:number;chargeCount?:number }
export interface MemberBalance { userId:string;amountMinor:number }
export interface DashboardSummary { month?:string;monthlySubscriptionMinor:number;monthExpenseMinor:number;activeSubscriptions:number;upcoming:Subscription[];currencies?:CurrencyDashboard[];balances?:MemberBalance[];reportingCurrency?:Currency;originalCurrencies?:CurrencyDashboard[] }
export interface BillingDates { dates:string[];nextCursor?:string }
export interface SubscriptionPeriod { billingAt:string;amountMinor:number;currency:Currency;baseAmountMinor:number;baseCurrency:Currency;paidBy:string;splits?:ExpenseSplit[];status:'pending'|'posted'|'failed';expenseId?:string;error?:string }
export interface SubscriptionPeriods { periods:SubscriptionPeriod[];nextCursor?:string }
export interface SubFlowEvent { type:string;groupId:string;resource:string;resourceId:string;occurredAt:string }
export interface Meta { page:number;perPage:number;totalItems:number;totalPages:number }
export interface Envelope<T> { data:T;meta?:Meta }
export interface ApiFailure { error:{code:string;message:string;fields?:Record<string,string>} }

