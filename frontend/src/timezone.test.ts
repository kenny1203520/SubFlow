import { describe, expect, it } from 'vitest'
import { timezoneLabel, timezoneOffset } from './timezone'

describe('timezone labels', () => {
  it('always includes an explicit UTC offset', () => {
    expect(timezoneLabel('Asia/Taipei', '2026-08-08T00:00:00Z')).toBe('Asia/Taipei (UTC+08:00)')
    expect(timezoneLabel('UTC', '2026-08-08T00:00:00Z')).toBe('UTC (UTC+00:00)')
  })

  it('reflects daylight-saving changes for the displayed date', () => {
    expect(timezoneOffset('America/New_York', '2026-01-15T00:00:00Z').label).toBe('UTC-05:00')
    expect(timezoneOffset('America/New_York', '2026-07-15T00:00:00Z').label).toBe('UTC-04:00')
  })
})
