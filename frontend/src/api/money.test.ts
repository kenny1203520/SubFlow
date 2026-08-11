import { describe, expect, it } from 'vitest'
import { majorToMinor, minorToInput, minorToMajor } from './money'

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

