import { describe, test, expect, vi } from 'vitest'

// Must import i18n helper before the module under test, so i18n.global is initialised
import { i18n } from '../../helpers/setup-i18n'

// Mock the @/i18n module to use our test i18n instance
vi.mock('@/i18n', () => ({ i18n }))

import { format_time, format_duration } from '@/classes/format-date-time'

describe('format_time', () => {
  test('formats a known date correctly', () => {
    // 2025-03-15 09:05:07 (Saturday)
    const date = new Date(2025, 2, 15, 9, 5, 7)
    const result = format_time(date)
    // Intl.DateTimeFormat locale-based format with day-of-week appended
    expect(result).toContain('2025')
    expect(result).toContain('09:05:07')
    expect(result).toContain('(土)')
  })

  test('pads single-digit months and days', () => {
    // 2025-01-02 00:00:00 (Thursday)
    const date = new Date(2025, 0, 2, 0, 0, 0)
    const result = format_time(date)
    expect(result).toContain('2025/01/02')
    expect(result).toContain('00:00:00')
  })

  test('includes day of week from i18n', () => {
    // Sunday
    const sunday = new Date(2025, 2, 16, 12, 0, 0)
    const result = format_time(sunday)
    expect(result).toContain('(日)')
  })

  test('handles midnight correctly', () => {
    const date = new Date(2025, 5, 1, 0, 0, 0)
    const result = format_time(date)
    expect(result).toContain('00:00:00')
  })

  test('handles end of day correctly', () => {
    const date = new Date(2025, 5, 1, 23, 59, 59)
    const result = format_time(date)
    expect(result).toContain('23:59:59')
  })
})

describe('format_duration', () => {
  test('returns empty string for null', () => {
    expect(format_duration(null)).toBe('')
  })

  test('returns empty string for 0', () => {
    expect(format_duration(0)).toBe('')
  })

  test('returns empty string for undefined', () => {
    expect(format_duration(undefined)).toBe('')
  })

  test('formats seconds only (under 1 minute)', () => {
    // 30 seconds = 30000ms
    const result = format_duration(30000)
    expect(result).toContain('30秒')
  })

  test('formats minutes', () => {
    // 5 minutes = 300000ms
    const result = format_duration(300000)
    expect(result).toContain('5分')
  })

  test('formats hours and minutes', () => {
    // 1 hour 30 minutes = 5400000ms
    const result = format_duration(5400000)
    expect(result).toContain('1時間')
    expect(result).toContain('30分')
  })

  test('formats days, hours, minutes', () => {
    // 1 day 2 hours 30 minutes = 95400000ms
    const result = format_duration(95400000)
    expect(result).toContain('1日')
    expect(result).toContain('2時間')
    expect(result).toContain('30分')
  })

  test('includes trimmed hours in parentheses', () => {
    // 2 hours = 7200000ms -> 2時間
    const result = format_duration(7200000)
    expect(result).toContain('（2時間）')
  })

  test('exactly 1 minute has no seconds component', () => {
    // 60 seconds = 60000ms
    const result = format_duration(60000)
    expect(result).toContain('1分')
    expect(result).not.toContain('秒')
  })
})
