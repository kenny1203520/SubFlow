import type { RecordModel } from 'pocketbase'

export interface Member extends RecordModel {
    name: string
    email: string
    avatar: string
}

export interface Group extends RecordModel {
    name: string
    description: string
    color: string
    currency: 'TWD' | 'USD' | 'JPY' | 'EUR'
    owner: string
    members: string[]
}

export interface Subscription extends RecordModel {
    group: string
    name: string
    category: string
    amount: number
    currency: Group['currency']
    billing_cycle: 'monthly' | 'quarterly' | 'yearly'
    next_billing: string
    status: 'active' | 'paused' | 'cancelled'
    notes: string
}

export interface Expense extends RecordModel {
    group: string
    title: string
    amount: number
    paid_by: string
    expense_date: string
    category: string
    notes: string
}
