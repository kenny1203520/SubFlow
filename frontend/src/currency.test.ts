import { describe, expect, it } from 'vitest'
import { currencyLabel } from './currency'

describe('currency labels',()=>{
  it('includes localized name and stable code',()=>{
    expect(currencyLabel('TWD','zh-TW')).toContain('(TWD)')
    expect(currencyLabel('TWD','en')).toMatch(/New Taiwan dollars? \(TWD\)/i)
  })
  it('falls back safely for an unknown code',()=>expect(currencyLabel('ZZZ','en')).toBe('ZZZ (ZZZ)'))
})
