export type Currency = 'TWD' | 'USD' | 'JPY' | 'EUR'
export type Role = 'owner' | 'member'
export type BillingCycle = 'monthly' | 'quarterly' | 'yearly'
export type SubscriptionStatus = 'active' | 'paused' | 'cancelled'

export interface Group { id:string;name:string;description:string;currency:Currency;color:string;ownerId:string;createdAt:string;updatedAt:string }
export interface User { id:string;email:string;name:string;avatar?:string;timezone:string }
export interface Membership { id:string;groupId:string;userId:string;role:Role;user?:User;createdAt:string }
export interface Invitation { id:string;groupId:string;email:string;status:'pending'|'delivery_failed'|'accepted'|'revoked'|'expired';invitedBy:string;acceptedBy?:string;expiresAt:string;debugUrl?:string;createdAt:string;updatedAt:string }
export interface Subscription { id:string;groupId:string;name:string;category:string;amountMinor:number;currency:Currency;billingCycle:BillingCycle;nextBilling:string;status:SubscriptionStatus;notes:string;createdAt:string;updatedAt:string }
export interface Expense { id:string;groupId:string;title:string;category:string;amountMinor:number;paidBy:string;incurredOn:string;notes:string;createdAt:string;updatedAt:string }
export interface DashboardSummary { monthlySubscriptionMinor:number;monthExpenseMinor:number;activeSubscriptions:number;upcoming:Subscription[] }
export interface SubFlowEvent { type:string;groupId:string;resource:string;resourceId:string;occurredAt:string }
export interface Meta { page:number;perPage:number;totalItems:number;totalPages:number }
export interface Envelope<T> { data:T;meta?:Meta }
export interface ApiFailure { error:{code:string;message:string;fields?:Record<string,string>} }
