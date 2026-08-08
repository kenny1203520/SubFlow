import { describe, expect, it } from 'vitest'
import { categoryLabel } from './category'
import { zhTW } from './locales/zh-TW'

const tr = (key: keyof typeof zhTW) => zhTW[key]

describe('categoryLabel', () => {
  it('translates known system categories and preserves custom names', () => {
    expect(categoryLabel({ systemKey: 'food_dining' }, '', tr)).toBe('餐飲')
    expect(categoryLabel({ customName: '咖啡豆' }, '', tr)).toBe('咖啡豆')
  })

  it('never throws for a category introduced by a later migration', () => {
    expect(categoryLabel({ systemKey: 'future_category' }, 'Legacy value', tr)).toBe('Legacy value')
    expect(categoryLabel({ systemKey: 'future_category' }, '', tr)).toBe('未分類')
  })
})
