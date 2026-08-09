import { computed, ref, watchEffect } from 'vue'
import { en } from './locales/en'
import { zhTW, type MessageKey } from './locales/zh-TW'
import { adminExtras } from './locales/admin'

export type Locale = 'zh-TW' | 'en'
const stored = typeof localStorage !== 'undefined' ? localStorage.getItem('subflow-locale') : null
export const locale = ref<Locale>(stored === 'en' ? 'en' : 'zh-TW')
const messages = { 'zh-TW': zhTW, en } as const

watchEffect(() => {
  if (typeof document !== 'undefined') document.documentElement.lang = locale.value
})

export function useI18n() {
  const t = computed(() => messages[locale.value])
  function tr(key: MessageKey | string, values: Record<string, string | number> = {}) {
    const text = (t.value as Record<string,string>)[key] ?? adminExtras[locale.value][key as keyof typeof adminExtras['zh-TW']] ?? key
    return text.replace(/\{(\w+)\}/g, (_, name: string) => String(values[name] ?? ''))
  }
  function setLocale(next: Locale) {
    locale.value = next
    if (typeof localStorage !== 'undefined') localStorage.setItem('subflow-locale', next)
  }
  function formatDate(value: string | Date, options: Intl.DateTimeFormatOptions = { dateStyle: 'medium' }) {
    return new Intl.DateTimeFormat(locale.value, options).format(new Date(value))
  }
  function formatMonth(value: string) {
    return new Intl.DateTimeFormat(locale.value, { year: 'numeric', month: 'long', timeZone: 'UTC' }).format(new Date(`${value}-01T00:00:00Z`))
  }
  function formatNumber(value: number, options?: Intl.NumberFormatOptions) {
    return new Intl.NumberFormat(locale.value, options).format(value)
  }
  return { locale, t, tr, setLocale, formatDate, formatMonth, formatNumber }
}

export type { MessageKey }
