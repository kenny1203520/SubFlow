export type Currency = 'TWD' | 'USD' | 'JPY' | 'EUR'
export type Role = 'owner' | 'member'
export type BillingCycle = 'monthly' | 'quarterly' | 'yearly'
export type SubscriptionStatus = 'active' | 'paused' | 'cancelled'

export interface Group { id:string;name:string;description:string;currency:Currency;color:string;ownerId:string;createdAt:string;updatedAt:string }
export interface User { id:string;email:string;name:string;avatar?:string;timezone:string }
export interface Membership { id:string;groupId:string;userId:string;role:Role;user?:User;createdAt:string }
export interface Invitation { id:string;groupId:string;email:string;status:'pending'|'delivery_failed'|'accepted'|'revoked'|'expired';invitedBy:string;acceptedBy?:string;expiresAt:string;debugUrl?:string;createdAt:string;updatedAt:string }
export type SplitMode='equal'|'amount'|'percentage'
export interface ExpenseSplit { id?:string;expenseId?:string;userId:string;amountMinor:number;percentageBasisPoints?:number }
export interface Subscription { id:string;groupId?:string;ownerId?:string;paidBy:string;name:string;category:string;amountMinor:number;currency:Currency;billingCycle:BillingCycle;startsOn:string;endsOn?:string;nextBilling:string;status:SubscriptionStatus;lifecycleStatus?:'active'|'paused'|'ending'|'ended'|'cancelled';notes:string;createdAt:string;updatedAt:string }
export interface Expense { id:string;groupId?:string;ownerId?:string;title:string;category:string;amountMinor:number;currency:Currency;paidBy:string;incurredOn:string;notes:string;splitMode?:SplitMode;splits?:ExpenseSplit[];createdAt:string;updatedAt:string }
export interface Settlement { id:string;groupId:string;fromUserId:string;toUserId:string;createdBy:string;amountMinor:number;settledOn:string;notes:string;createdAt:string;updatedAt:string }
export interface CurrencyDashboard { currency:Currency;cashOutflowMinor:number;personalShareMinor:number;reimbursableMinor:number;monthlySubscriptionMinor:number;activeSubscriptions:number;chargeCount?:number }
export interface MemberBalance { userId:string;amountMinor:number }
export interface DashboardSummary { month?:string;monthlySubscriptionMinor:number;monthExpenseMinor:number;activeSubscriptions:number;upcoming:Subscription[];currencies?:CurrencyDashboard[];balances?:MemberBalance[] }
export interface BillingDates { dates:string[];nextCursor?:string }
export interface SubFlowEvent { type:string;groupId:string;resource:string;resourceId:string;occurredAt:string }
export interface Meta { page:number;perPage:number;totalItems:number;totalPages:number }
export interface Envelope<T> { data:T;meta?:Meta }
export interface ApiFailure { error:{code:string;message:string;fields?:Record<string,string>} }
