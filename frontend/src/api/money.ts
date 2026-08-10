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

