import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { fromDateInput, fromDateTimeInput, toDateInput, toDateTimeInput, todayInput } from './dateInput'

describe('toDateInput / toDateTimeInput', () => {
  it('never string-slices the ISO instant -- the viewer timezone can push the local date to the other side of the UTC day boundary', () => {
    // 23:30 UTC is already the next calendar day in Taipei (UTC+8), and still
    // the same day in New York (UTC-5 in January) -- a naive .slice(0,10) on
    // the ISO string would get both of these wrong.
    expect(toDateInput('2026-01-15T23:30:00Z', 'Asia/Taipei')).toBe('2026-01-16')
    expect(toDateInput('2026-01-15T23:30:00Z', 'America/New_York')).toBe('2026-01-15')
  })

  it('formats the time-of-day alongside the date for datetime-local inputs', () => {
    expect(toDateTimeInput('2026-01-15T23:30:00Z', 'Asia/Taipei')).toBe('2026-01-16T07:30')
  })

  it('returns an empty string for an empty input instead of throwing', () => {
    expect(toDateInput('', 'UTC')).toBe('')
    expect(toDateTimeInput('', 'UTC')).toBe('')
  })
})

describe('fromDateInput / fromDateTimeInput', () => {
  it('resolves a local wall-clock date to the correct UTC instant', () => {
    // Local midnight 2026-01-16 in Taipei (UTC+8) is 2026-01-15T16:00 UTC.
    expect(fromDateInput('2026-01-16', 'Asia/Taipei')).toBe('2026-01-15T16:00:00.000Z')
  })

  it('accounts for the timezone offset at that specific instant, not the current offset', () => {
    // America/New_York is UTC-5 in January (EST) and UTC-4 in July (EDT) --
    // the same "09:00 local" wall-clock time must resolve to different UTC
    // instants depending on which side of the DST transition it falls on.
    expect(fromDateTimeInput('2026-01-15T09:00', 'America/New_York')).toBe('2026-01-15T14:00:00.000Z')
    expect(fromDateTimeInput('2026-07-15T09:00', 'America/New_York')).toBe('2026-07-15T13:00:00.000Z')
  })

  it('returns an empty string for an empty input instead of throwing', () => {
    expect(fromDateInput('', 'UTC')).toBe('')
    expect(fromDateTimeInput('', 'UTC')).toBe('')
  })
})

describe('round trip', () => {
  it('toDateTimeInput(fromDateTimeInput(x)) reproduces the original wall-clock value across DST boundaries', () => {
    for (const [value, tz] of [['2026-01-15T09:15', 'America/New_York'], ['2026-07-15T09:15', 'America/New_York'], ['2026-08-08T14:45', 'Asia/Taipei']] as const) {
      const iso = fromDateTimeInput(value, tz)
      expect(toDateTimeInput(iso, tz)).toBe(value)
    }
  })
})

describe('todayInput', () => {
  beforeEach(() => vi.useFakeTimers())
  afterEach(() => vi.useRealTimers())

  it("reflects the viewer's local calendar day even when it differs from UTC's", () => {
    vi.setSystemTime(new Date('2026-01-15T23:30:00Z'))
    expect(todayInput('Asia/Taipei')).toBe('2026-01-16')
    expect(todayInput('America/New_York')).toBe('2026-01-15')
  })
})
