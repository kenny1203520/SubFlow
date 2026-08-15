import { describe, expect, it } from 'vitest'
import { roleLabel, roleSearchText } from './role'

const tr = (key: string) => `translated:${key}`

describe('roleLabel', () => {
  it('translates the protected owner/member keys instead of showing them raw', () => {
    expect(roleLabel({ key: 'owner', name: 'Owner' }, tr)).toBe('translated:owner')
    expect(roleLabel({ key: 'member', name: 'Member' }, tr)).toBe('translated:member')
  })

  it('shows a custom role name as-is, untranslated', () => {
    expect(roleLabel({ key: 'custom-role', name: 'Treasurer' }, tr)).toBe('Treasurer')
  })

  it('falls back to the key when a custom role has no name', () => {
    expect(roleLabel({ key: 'custom-role', name: '' }, tr)).toBe('custom-role')
  })

  it('returns an empty string when no role is given', () => {
    expect(roleLabel(undefined, tr)).toBe('')
  })
})

describe('roleSearchText', () => {
  it('joins the label, name, key, and category for filtering', () => {
    const text = roleSearchText({ key: 'custom-role', name: 'Treasurer', category: 'finance' }, tr)
    expect(text).toBe('Treasurer Treasurer custom-role finance')
  })

  it('omits blank fields instead of leaving stray whitespace', () => {
    const text = roleSearchText({ key: 'owner', name: '', category: '' }, tr)
    expect(text).toBe('translated:owner owner')
  })
})
