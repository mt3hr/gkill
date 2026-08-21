import { describe, test, expect, vi } from 'vitest'

// Must import i18n helper before the module under test, so i18n.global is initialised
import { i18n } from '../../helpers/setup-i18n'

// Mock the @/i18n module to use our test i18n instance
vi.mock('@/i18n', () => ({ i18n }))

import { format_time, format_duration, format_time_of_day, to_single_line, DURATION_LINE_SEPARATOR } from '@/classes/format-date-time'

const HOUR = 60 * 60 * 1000
const MINUTE = 60 * 1000

describe('format_time_of_day', () => {
  test('formats milliseconds from midnight as HH:mm', () => {
    expect(format_time_of_day(0)).toBe('00:00')
    expect(format_time_of_day(9 * HOUR + 5 * MINUTE)).toBe('09:05')
    expect(format_time_of_day(23 * HOUR + 59 * MINUTE)).toBe('23:59')
  })

  test('returns empty string for null and non-finite values', () => {
    expect(format_time_of_day(null)).toBe('')
    expect(format_time_of_day(NaN)).toBe('')
  })

  test('wraps back to 00:00 instead of showing 24:00', () => {
    // 23:59:45 は分に丸めると 24:00 になる
    expect(format_time_of_day(23 * HOUR + 59 * MINUTE + 45_000)).toBe('00:00')
    expect(format_time_of_day(24 * HOUR)).toBe('00:00')
  })
})

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

  // 表示文字列に HTML タグを埋めてはいけない。
  // 表示側は {{ }} 補間なので Vue がエスケープし、剥がし忘れた画面では
  // <br> がそのまま文字として見える（Dnoteの集計リストと相関グラフで実際に出ていた）。
  // 区切りは本物の改行にしてあるので、剥がし忘れても white-space 既定なら空白1個へ畳まれる。
  test('does not embed an HTML tag', () => {
    const result = format_duration(83160000)
    expect(result).not.toContain('<')
    expect(result).not.toContain('>')
  })

  test('separates the trimmed hours with a real newline', () => {
    // 23時間6分 = 83160000ms
    const result = format_duration(83160000)
    expect(result).toContain(DURATION_LINE_SEPARATOR)
    expect(result.split(DURATION_LINE_SEPARATOR)).toEqual(['23時間 6分', '（23.1時間）'])
  })

  test('has no separator when there is nothing to separate', () => {
    expect(format_duration(0)).not.toContain(DURATION_LINE_SEPARATOR)
  })
})

describe('to_single_line', () => {
  test('folds the duration separator into a single space', () => {
    expect(to_single_line(format_duration(83160000))).toBe('23時間 6分 （23.1時間）')
  })

  test('leaves a string without a separator untouched', () => {
    expect(to_single_line('1000')).toBe('1000')
  })

  // .replace(文字列, …) は最初の1個しか置換しないので、区切りが増えたときに黙って壊れる
  test('folds every separator, not just the first', () => {
    expect(to_single_line(`a${DURATION_LINE_SEPARATOR}b${DURATION_LINE_SEPARATOR}c`)).toBe('a b c')
  })
})
