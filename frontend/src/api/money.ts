import type { Currency } from './types'

export function currencyDigits(currency: Currency | string = 'TWD') {
  if (['BIF','CLP','DJF','GNF','ISK','JPY','KMF','KRW','PYG','RWF','UGX','VND','VUV','XAF','XOF','XPF'].includes(currency)) return 0
  if (['BHD','IQD','JOD','KWD','LYD','OMR','TND'].includes(currency)) return 3
  try { return new Intl.NumberFormat('en', { style: 'currency', currency }).resolvedOptions().maximumFractionDigits ?? 2 } catch { return 2 }
}

export function majorToMinor(value: string | number, currency: Currency | string = 'TWD') {
  return Math.round(Number(value) * 10 ** currencyDigits(currency))
}

export function minorToMajor(value: number, currency: Currency | string = 'TWD') {
  return value / 10 ** currencyDigits(currency)
}

export function minorToInput(value: number, currency: Currency | string = 'TWD') {
  return minorToMajor(value, currency).toFixed(currencyDigits(currency))
}

// Shared with MoneyValue.vue so any other place that needs a plain-text
// formatted amount (e.g. the audit log, which renders inside a <li> rather
// than a component) doesn't have to duplicate the Intl.NumberFormat/fallback
// dance.
export function formatMoney(amountMinor: number, currency: Currency | string = 'TWD', locale = 'zh-TW') {
  try { return new Intl.NumberFormat(locale, { style: 'currency', currency }).format(minorToMajor(amountMinor, currency)) }
  catch { return `${minorToMajor(amountMinor, currency)} ${currency}` }
}

// The smallest positive amount an <input type=number> should step by/allow
// for a given currency: "1" for zero-decimal currencies (JPY, KRW…), "0.001"
// for three-decimal ones (BHD, KWD…), "0.01" otherwise. A hardcoded 0.01
// offers meaningless sub-unit precision for the former and silently rejects
// valid values for the latter.
export function amountStep(currency: Currency | string = 'TWD') {
  const digits = currencyDigits(currency)
  return digits === 0 ? '1' : (10 ** -digits).toFixed(digits)
}

