import type { Currency } from './types'

export function currencyDigits(currency: Currency | string = 'TWD') {
  return new Intl.NumberFormat('en', { style: 'currency', currency }).resolvedOptions().maximumFractionDigits ?? 2
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
