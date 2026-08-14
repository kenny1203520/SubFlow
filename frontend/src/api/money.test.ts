import { describe, expect, it } from 'vitest'
import { amountStep, majorToMinor, minorToInput, minorToMajor } from './money'

describe('currency minor units', () => {
  it('uses two fraction digits for TWD, USD and EUR', () => {
    expect(majorToMinor('12.34', 'TWD')).toBe(1234)
    expect(minorToMajor(1234, 'USD')).toBe(12.34)
    expect(minorToInput(1234, 'EUR')).toBe('12.34')
  })

  it('uses zero fraction digits for JPY', () => {
    expect(majorToMinor('1234', 'JPY')).toBe(1234)
    expect(minorToMajor(1234, 'JPY')).toBe(1234)
    expect(minorToInput(1234, 'JPY')).toBe('1234')
  })
})

// A hardcoded step="0.01" offers meaningless sub-unit precision for
// zero-decimal currencies and silently rejects valid values for
// three-decimal ones, so amount inputs must derive their step from the
// selected currency instead.
describe('amountStep', () => {
  it('steps by whole units for zero-decimal currencies', () => {
    expect(amountStep('JPY')).toBe('1')
    expect(amountStep('KRW')).toBe('1')
  })

  it('steps by cents for ordinary two-decimal currencies', () => {
    expect(amountStep('TWD')).toBe('0.01')
    expect(amountStep('USD')).toBe('0.01')
  })

  it('steps by mils for three-decimal currencies', () => {
    expect(amountStep('BHD')).toBe('0.001')
    expect(amountStep('KWD')).toBe('0.001')
  })
})

