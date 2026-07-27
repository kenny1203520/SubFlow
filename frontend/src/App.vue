<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { pb, messageFromError } from './pocketbase'
import type { Expense, Group, Member, Subscription } from './types'

type View = 'overview' | 'subscriptions' | 'expenses'
type Dialog = 'group' | 'subscription' | 'expense' | null

const user = ref<Member | null>(pb.authStore.record as Member | null)
const booting = ref(true)
const loading = ref(false)
const error = ref('')
const notice = ref('')
const authMode = ref<'login' | 'register'>('login')
const currentView = ref<View>('overview')
const dialog = ref<Dialog>(null)
const groups = ref<Group[]>([])
const subscriptions = ref<Subscription[]>([])
const expenses = ref<Expense[]>([])
const selectedGroupId = ref('')

const authForm = reactive({ name: '', email: '', password: '', passwordConfirm: '' })
const groupForm = reactive({ name: '', description: '', color: '#7357ff', currency: 'TWD' as Group['currency'] })
const subscriptionForm = reactive({ name: '', category: '影音娛樂', amount: 0, billing_cycle: 'monthly' as Subscription['billing_cycle'], next_billing: '', notes: '' })
const expenseForm = reactive({ title: '', category: '共用採購', amount: 0, expense_date: new Date().toISOString().slice(0, 10), notes: '' })

const selectedGroup = computed(() => groups.value.find((group) => group.id === selectedGroupId.value) || null)
const visibleSubscriptions = computed(() => subscriptions.value.filter((item) => item.group === selectedGroupId.value))
const visibleExpenses = computed(() => expenses.value.filter((item) => item.group === selectedGroupId.value))
const activeSubscriptions = computed(() => visibleSubscriptions.value.filter((item) => item.status === 'active'))
const monthlyTotal = computed(() => activeSubscriptions.value.reduce((sum, item) => {
    const divisor = item.billing_cycle === 'yearly' ? 12 : item.billing_cycle === 'quarterly' ? 3 : 1
    return sum + item.amount / divisor
}, 0))
const monthExpenseTotal = computed(() => {
    const month = new Date().toISOString().slice(0, 7)
    return visibleExpenses.value.filter((item) => item.expense_date.startsWith(month)).reduce((sum, item) => sum + item.amount, 0)
})
const upcoming = computed(() => [...activeSubscriptions.value].sort((a, b) => a.next_billing.localeCompare(b.next_billing)).slice(0, 4))

const cycleLabels = { monthly: '每月', quarterly: '每季', yearly: '每年' }
const viewLabels: Record<View, string> = { overview: '總覽', subscriptions: '訂閱服務', expenses: '共同支出' }

function money(value: number, currency = selectedGroup.value?.currency || 'TWD') {
    return new Intl.NumberFormat('zh-TW', { style: 'currency', currency, maximumFractionDigits: currency === 'TWD' || currency === 'JPY' ? 0 : 2 }).format(value)
}

function shortDate(value: string) {
    return new Intl.DateTimeFormat('zh-TW', { month: 'short', day: 'numeric' }).format(new Date(value))
}

function resetMessage() {
    error.value = ''
    notice.value = ''
}

async function submitAuth() {
    resetMessage()
    loading.value = true
    try {
        if (authMode.value === 'register') {
            await pb.collection('members').create({
                name: authForm.name,
                email: authForm.email,
                password: authForm.password,
                passwordConfirm: authForm.passwordConfirm,
            })
            notice.value = '帳號建立成功，歡迎加入 SubFlow。'
        }
        await pb.collection('members').authWithPassword(authForm.email, authForm.password)
        user.value = pb.authStore.record as Member
        await loadData()
    } catch (err) {
        error.value = messageFromError(err)
    } finally {
        loading.value = false
    }
}

function logout() {
    pb.authStore.clear()
    user.value = null
    groups.value = []
    subscriptions.value = []
    expenses.value = []
    selectedGroupId.value = ''
    void unsubscribeRealtime()
}

async function loadData() {
    if (!pb.authStore.isValid) return
    loading.value = true
    error.value = ''
    try {
        const [groupList, subscriptionList, expenseList] = await Promise.all([
            pb.collection('groups').getFullList<Group>({ sort: 'created' }),
            pb.collection('subscriptions').getFullList<Subscription>({ sort: 'next_billing' }),
            pb.collection('expenses').getFullList<Expense>({ sort: '-expense_date' }),
        ])
        groups.value = groupList
        subscriptions.value = subscriptionList
        expenses.value = expenseList
        if (!groups.value.some((item) => item.id === selectedGroupId.value)) selectedGroupId.value = groups.value[0]?.id || ''
        await subscribeRealtime()
    } catch (err) {
        error.value = messageFromError(err)
    } finally {
        loading.value = false
    }
}

async function subscribeRealtime() {
    await unsubscribeRealtime()
    for (const collection of ['groups', 'subscriptions', 'expenses']) {
        await pb.collection(collection).subscribe('*', () => void loadData())
    }
}

async function unsubscribeRealtime() {
    await Promise.all(['groups', 'subscriptions', 'expenses'].map((collection) => pb.collection(collection).unsubscribe()))
}

async function createGroup() {
    if (!user.value) return
    await runAction(async () => {
        const created = await pb.collection('groups').create<Group>({ ...groupForm, owner: user.value!.id, members: [user.value!.id] })
        dialog.value = null
        selectedGroupId.value = created.id
        groupForm.name = ''
        groupForm.description = ''
        await loadData()
    }, '群組已建立。')
}

async function createSubscription() {
    if (!selectedGroup.value) return
    await runAction(async () => {
        await pb.collection('subscriptions').create({
            ...subscriptionForm,
            group: selectedGroup.value!.id,
            currency: selectedGroup.value!.currency,
            status: 'active',
            next_billing: `${subscriptionForm.next_billing} 12:00:00.000Z`,
        })
        dialog.value = null
        Object.assign(subscriptionForm, { name: '', category: '影音娛樂', amount: 0, billing_cycle: 'monthly', next_billing: '', notes: '' })
        await loadData()
    }, '訂閱已新增。')
}

async function createExpense() {
    if (!selectedGroup.value || !user.value) return
    await runAction(async () => {
        await pb.collection('expenses').create({
            ...expenseForm,
            group: selectedGroup.value!.id,
            paid_by: user.value!.id,
            expense_date: `${expenseForm.expense_date} 12:00:00.000Z`,
        })
        dialog.value = null
        Object.assign(expenseForm, { title: '', category: '共用採購', amount: 0, expense_date: new Date().toISOString().slice(0, 10), notes: '' })
        await loadData()
    }, '支出已記錄。')
}

async function toggleSubscription(item: Subscription) {
    await runAction(async () => {
        await pb.collection('subscriptions').update(item.id, { status: item.status === 'active' ? 'paused' : 'active' })
        await loadData()
    }, item.status === 'active' ? '訂閱已暫停。' : '訂閱已恢復。')
}

async function removeRecord(collection: 'subscriptions' | 'expenses', id: string) {
    if (!window.confirm('確定要刪除這筆資料嗎？此動作無法復原。')) return
    await runAction(async () => {
        await pb.collection(collection).delete(id)
        await loadData()
    }, '資料已刪除。')
}

async function runAction(action: () => Promise<void>, success: string) {
    resetMessage()
    loading.value = true
    try {
        await action()
        notice.value = success
    } catch (err) {
        error.value = messageFromError(err)
    } finally {
        loading.value = false
    }
}

watch(selectedGroupId, () => resetMessage())

onMounted(async () => {
    try {
        if (pb.authStore.isValid) {
            await pb.collection('members').authRefresh()
            user.value = pb.authStore.record as Member
            await loadData()
        }
    } catch {
        pb.authStore.clear()
        user.value = null
    } finally {
        booting.value = false
    }
})

onBeforeUnmount(() => void unsubscribeRealtime())
</script>

<template>
    <div v-if="booting" class="splash"><span class="brand-mark">S</span><p>正在整理你的訂閱…</p></div>

    <main v-else-if="!user" class="auth-page">
        <section class="auth-story">
            <a class="brand" href="#"><span class="brand-mark">S</span><span>SubFlow</span></a>
            <div class="story-copy">
                <p class="eyebrow">共享生活，更有餘裕</p>
                <h1>每一筆訂閱，<br><em>都流向值得的地方。</em></h1>
                <p>把家庭、團隊與朋友的週期費用集中管理，告別漏繳、重複訂閱與難算的共同支出。</p>
            </div>
            <div class="story-card">
                <div><span>本月訂閱</span><strong>12 項</strong></div>
                <div><span>成功省下</span><strong>NT$ 1,240</strong></div>
                <div class="avatars"><i>J</i><i>M</i><i>Y</i><small>一起分擔</small></div>
            </div>
        </section>
        <section class="auth-panel">
            <form class="auth-card" @submit.prevent="submitAuth">
                <p class="eyebrow">{{ authMode === 'login' ? '歡迎回來' : '建立你的空間' }}</p>
                <h2>{{ authMode === 'login' ? '登入 SubFlow' : '開始管理訂閱' }}</h2>
                <p>{{ authMode === 'login' ? '繼續掌握每一筆共同費用。' : '只要一分鐘，讓固定支出變得清楚。' }}</p>
                <label v-if="authMode === 'register'">顯示名稱<input v-model.trim="authForm.name" required minlength="2" autocomplete="name" placeholder="你的名字"></label>
                <label>電子郵件<input v-model.trim="authForm.email" required type="email" autocomplete="email" placeholder="you@example.com"></label>
                <label>密碼<input v-model="authForm.password" required type="password" minlength="8" :autocomplete="authMode === 'login' ? 'current-password' : 'new-password'" placeholder="至少 8 個字元"></label>
                <label v-if="authMode === 'register'">確認密碼<input v-model="authForm.passwordConfirm" required type="password" minlength="8" autocomplete="new-password" placeholder="再次輸入密碼"></label>
                <p v-if="error" class="message error">{{ error }}</p>
                <p v-if="notice" class="message success">{{ notice }}</p>
                <button class="primary wide" :disabled="loading">{{ loading ? '處理中…' : authMode === 'login' ? '登入' : '免費建立帳號' }}</button>
                <button class="text-button" type="button" @click="authMode = authMode === 'login' ? 'register' : 'login'; resetMessage()">
                    {{ authMode === 'login' ? '還沒有帳號？立即註冊' : '已經有帳號？返回登入' }}
                </button>
            </form>
        </section>
    </main>

    <div v-else class="app-shell">
        <aside class="sidebar">
            <a class="brand" href="#"><span class="brand-mark">S</span><span>SubFlow</span></a>
            <nav>
                <button v-for="view in (Object.keys(viewLabels) as View[])" :key="view" :class="{ active: currentView === view }" @click="currentView = view">
                    <span class="nav-icon">{{ view === 'overview' ? '⌂' : view === 'subscriptions' ? '↻' : '↗' }}</span>{{ viewLabels[view] }}
                </button>
            </nav>
            <div class="sidebar-bottom">
                <div class="user-chip"><span>{{ user.name.slice(0, 1).toUpperCase() }}</span><div><strong>{{ user.name }}</strong><small>{{ user.email }}</small></div></div>
                <button class="logout" @click="logout">登出</button>
            </div>
        </aside>

        <main class="workspace">
            <header class="topbar">
                <div>
                    <p class="eyebrow">{{ selectedGroup ? selectedGroup.name : '你的共享空間' }}</p>
                    <h1>{{ viewLabels[currentView] }}</h1>
                </div>
                <div class="top-actions">
                    <select v-if="groups.length" v-model="selectedGroupId" aria-label="選擇群組">
                        <option v-for="group in groups" :key="group.id" :value="group.id">{{ group.name }}</option>
                    </select>
                    <button class="secondary" @click="dialog = 'group'">＋ 新增群組</button>
                    <button v-if="selectedGroup && currentView !== 'overview'" class="primary" @click="dialog = currentView === 'subscriptions' ? 'subscription' : 'expense'">＋ 新增{{ currentView === 'subscriptions' ? '訂閱' : '支出' }}</button>
                </div>
            </header>

            <p v-if="error" class="message error">{{ error }}</p>
            <p v-if="notice" class="message success">{{ notice }}</p>

            <section v-if="!selectedGroup" class="empty-state">
                <span>✦</span><h2>建立第一個共享群組</h2><p>你可以為家庭、工作室或朋友建立獨立空間，接著加入訂閱與支出。</p>
                <button class="primary" @click="dialog = 'group'">建立群組</button>
            </section>

            <template v-else-if="currentView === 'overview'">
                <section class="hero-card" :style="{ '--group-color': selectedGroup.color }">
                    <div><p>預估每月固定支出</p><strong>{{ money(monthlyTotal) }}</strong><small>{{ activeSubscriptions.length }} 項使用中的訂閱</small></div>
                    <div class="hero-orbit"><span>{{ Math.min(99, Math.round(monthlyTotal / 100)) }}</span><small>Flow 指數</small></div>
                </section>
                <section class="stat-grid">
                    <article><span class="stat-icon violet">↻</span><div><small>啟用訂閱</small><strong>{{ activeSubscriptions.length }}</strong></div></article>
                    <article><span class="stat-icon green">↗</span><div><small>本月共同支出</small><strong>{{ money(monthExpenseTotal) }}</strong></div></article>
                    <article><span class="stat-icon orange">◷</span><div><small>最近扣款</small><strong>{{ upcoming[0] ? shortDate(upcoming[0].next_billing) : '—' }}</strong></div></article>
                </section>
                <section class="content-grid">
                    <article class="panel">
                        <div class="panel-head"><div><p class="eyebrow">接下來</p><h2>即將扣款</h2></div><button class="text-button" @click="currentView = 'subscriptions'">查看全部</button></div>
                        <div v-if="upcoming.length" class="stack-list">
                            <div v-for="item in upcoming" :key="item.id" class="list-row"><span class="service-avatar">{{ item.name.slice(0, 1) }}</span><div class="grow"><strong>{{ item.name }}</strong><small>{{ item.category }} · {{ cycleLabels[item.billing_cycle] }}</small></div><div class="align-right"><strong>{{ money(item.amount, item.currency) }}</strong><small>{{ shortDate(item.next_billing) }}</small></div></div>
                        </div>
                        <p v-else class="muted">目前沒有即將扣款的訂閱。</p>
                    </article>
                    <article class="panel accent-panel"><p class="eyebrow">小提醒</p><h2>讓每一筆費用<br>保持在視線內。</h2><p>固定檢視不常使用的服務，暫停一項小額訂閱，也能累積更多生活彈性。</p><button class="secondary light" @click="currentView = 'subscriptions'">整理訂閱</button></article>
                </section>
            </template>

            <section v-else-if="currentView === 'subscriptions'" class="panel table-panel">
                <div class="panel-head"><div><p class="eyebrow">{{ selectedGroup.name }}</p><h2>所有訂閱</h2></div><span class="count-pill">{{ visibleSubscriptions.length }} 項</span></div>
                <div v-if="visibleSubscriptions.length" class="record-list">
                    <article v-for="item in visibleSubscriptions" :key="item.id" class="record-row">
                        <span class="service-avatar large">{{ item.name.slice(0, 1) }}</span><div class="record-main"><strong>{{ item.name }}</strong><small>{{ item.category }} · {{ cycleLabels[item.billing_cycle] }} · 下次 {{ shortDate(item.next_billing) }}</small></div>
                        <span :class="['status', item.status]">{{ item.status === 'active' ? '使用中' : item.status === 'paused' ? '已暫停' : '已取消' }}</span>
                        <strong class="record-amount">{{ money(item.amount, item.currency) }}</strong>
                        <button class="icon-button" :title="item.status === 'active' ? '暫停' : '恢復'" @click="toggleSubscription(item)">{{ item.status === 'active' ? 'Ⅱ' : '▶' }}</button>
                        <button class="icon-button danger" title="刪除" @click="removeRecord('subscriptions', item.id)">×</button>
                    </article>
                </div>
                <div v-else class="inline-empty"><span>↻</span><p>還沒有訂閱。新增第一項固定服務，開始掌握每月流向。</p><button class="primary" @click="dialog = 'subscription'">新增訂閱</button></div>
            </section>

            <section v-else class="panel table-panel">
                <div class="panel-head"><div><p class="eyebrow">{{ selectedGroup.name }}</p><h2>支出紀錄</h2></div><span class="count-pill">本月 {{ money(monthExpenseTotal) }}</span></div>
                <div v-if="visibleExpenses.length" class="record-list">
                    <article v-for="item in visibleExpenses" :key="item.id" class="record-row">
                        <span class="service-avatar large expense">↗</span><div class="record-main"><strong>{{ item.title }}</strong><small>{{ item.category }} · {{ shortDate(item.expense_date) }}</small></div>
                        <strong class="record-amount">{{ money(item.amount) }}</strong>
                        <button class="icon-button danger" title="刪除" @click="removeRecord('expenses', item.id)">×</button>
                    </article>
                </div>
                <div v-else class="inline-empty"><span>↗</span><p>目前沒有共同支出，新增一筆讓群組帳目保持同步。</p><button class="primary" @click="dialog = 'expense'">記錄支出</button></div>
            </section>
        </main>
    </div>

    <div v-if="dialog" class="modal-backdrop" @click.self="dialog = null">
        <form v-if="dialog === 'group'" class="modal" @submit.prevent="createGroup"><button class="modal-close" type="button" @click="dialog = null">×</button><p class="eyebrow">新的空間</p><h2>建立共享群組</h2><label>群組名稱<input v-model.trim="groupForm.name" required minlength="2" placeholder="例如：溫暖的家"></label><label>簡短說明<textarea v-model.trim="groupForm.description" rows="3" placeholder="這個群組用來管理什麼？"></textarea></label><div class="form-grid"><label>主題色<input v-model="groupForm.color" type="color"></label><label>主要幣別<select v-model="groupForm.currency"><option>TWD</option><option>USD</option><option>JPY</option><option>EUR</option></select></label></div><button class="primary wide" :disabled="loading">建立群組</button></form>
        <form v-else-if="dialog === 'subscription'" class="modal" @submit.prevent="createSubscription"><button class="modal-close" type="button" @click="dialog = null">×</button><p class="eyebrow">固定費用</p><h2>新增訂閱</h2><label>服務名稱<input v-model.trim="subscriptionForm.name" required minlength="2" placeholder="例如：Netflix"></label><div class="form-grid"><label>分類<input v-model.trim="subscriptionForm.category" required></label><label>金額<input v-model.number="subscriptionForm.amount" required min="0" step="0.01" type="number"></label><label>計費週期<select v-model="subscriptionForm.billing_cycle"><option value="monthly">每月</option><option value="quarterly">每季</option><option value="yearly">每年</option></select></label><label>下次扣款<input v-model="subscriptionForm.next_billing" required type="date"></label></div><label>備註<textarea v-model.trim="subscriptionForm.notes" rows="2"></textarea></label><button class="primary wide" :disabled="loading">新增訂閱</button></form>
        <form v-else class="modal" @submit.prevent="createExpense"><button class="modal-close" type="button" @click="dialog = null">×</button><p class="eyebrow">共同帳目</p><h2>記錄支出</h2><label>支出項目<input v-model.trim="expenseForm.title" required minlength="2" placeholder="例如：本月網路費"></label><div class="form-grid"><label>分類<input v-model.trim="expenseForm.category" required></label><label>金額<input v-model.number="expenseForm.amount" required min="0" step="0.01" type="number"></label><label>日期<input v-model="expenseForm.expense_date" required type="date"></label></div><label>備註<textarea v-model.trim="expenseForm.notes" rows="2"></textarea></label><button class="primary wide" :disabled="loading">記錄支出</button></form>
    </div>
</template>
