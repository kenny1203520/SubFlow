export type Currency = string
export type RateMode = 'automatic'|'manual'
export interface CurrencyInfo { code:Currency;digits:number }
export interface Category { id:string;scope:'system'|'personal'|'group';ownerId?:string;groupId?:string;systemKey?:string;customName?:string;iconKey?:string;createdBy?:string;archived:boolean;createdAt:string;updatedAt:string }
export interface ExchangeRate { baseCurrency:Currency;quoteCurrency:Currency;rate:string;effectiveDate:string;provider:string;fetchedAt:string;stale?:boolean }
export interface CurrencyChangePreview { from:Currency;to:Currency;affected:number;missing:{resource:string;id:string;from:Currency;to:Currency;date:string}[] }
export type Role = 'owner' | 'member'
export type BillingCycle = 'monthly' | 'quarterly' | 'yearly'
export type SubscriptionStatus = 'active' | 'paused' | 'cancelled'

export interface Group { id:string;name:string;description:string;currency:Currency;timezone:string;color:string;ownerId:string;createdAt:string;updatedAt:string }
export interface User { id:string;email:string;name:string;avatar?:string;timezone:string;defaultCurrency?:Currency;systemRoleId?:string }
export interface Membership { id:string;groupId:string;userId:string;role:Role;roleId?:string;roleName?:string;user?:User;createdAt:string }
export interface AccessRole { id:string;scope:'system'|'group';groupId?:string;name:string;key:string;permissions:string[];protected:boolean;createdBy?:string;createdAt:string;updatedAt:string }
export interface AuditLog { id:string;actorId:string;groupId?:string;scope:'system'|'group';action:string;resource:string;resourceId:string;outcome:'success'|'failure';summary?:string;ip?:string;userAgent?:string;createdAt:string }
export interface Invitation { id:string;groupId:string;email:string;status:'pending'|'delivery_failed'|'accepted'|'revoked'|'expired';invitedBy:string;acceptedBy?:string;expiresAt:string;debugUrl?:string;createdAt:string;updatedAt:string }
export type SplitMode='equal'|'amount'|'percentage'
export interface ExpenseSplit { id?:string;expenseId?:string;userId:string;amountMinor:number;baseAmountMinor?:number;percentageBasisPoints?:number }
interface ConvertedRecord { categoryId?:string;categoryInfo?:Category;baseCurrency:Currency;baseAmountMinor:number;exchangeRate:string;exchangeRateDate:string;rateMode:RateMode }
export interface Subscription extends ConvertedRecord { id:string;groupId?:string;ownerId?:string;paidBy:string;name:string;category:string;amountMinor:number;currency:Currency;billingCycle:BillingCycle;startsOn:string;endsOn?:string;nextBilling:string;status:SubscriptionStatus;lifecycleStatus?:'active'|'paused'|'ending'|'ended'|'cancelled';notes:string;createdAt:string;updatedAt:string }
export interface Expense extends ConvertedRecord { id:string;groupId?:string;ownerId?:string;title:string;category:string;amountMinor:number;currency:Currency;paidBy:string;incurredOn:string;notes:string;splitMode?:SplitMode;splits?:ExpenseSplit[];createdAt:string;updatedAt:string }
export interface Settlement { id:string;groupId:string;fromUserId:string;toUserId:string;createdBy:string;amountMinor:number;currency:Currency;baseCurrency:Currency;baseAmountMinor:number;exchangeRate:string;exchangeRateDate:string;settledOn:string;notes:string;createdAt:string;updatedAt:string }
export interface CurrencyDashboard { currency:Currency;cashOutflowMinor:number;personalShareMinor:number;reimbursableMinor:number;monthlySubscriptionMinor:number;activeSubscriptions:number;chargeCount?:number }
export interface MemberBalance { userId:string;amountMinor:number }
export interface DashboardSummary { month?:string;monthlySubscriptionMinor:number;monthExpenseMinor:number;activeSubscriptions:number;upcoming:Subscription[];currencies?:CurrencyDashboard[];balances?:MemberBalance[];reportingCurrency?:Currency;originalCurrencies?:CurrencyDashboard[] }
export interface BillingDates { dates:string[];nextCursor?:string }
export interface SubFlowEvent { type:string;groupId:string;resource:string;resourceId:string;occurredAt:string }
export interface Meta { page:number;perPage:number;totalItems:number;totalPages:number }
export interface Envelope<T> { data:T;meta?:Meta }
export interface ApiFailure { error:{code:string;message:string;fields?:Record<string,string>} }
