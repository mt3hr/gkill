import { is_url } from '@/classes/looks-like-url'

describe('isUrl', () => {
  test('valid http URL returns true', () => {
    expect(is_url('http://example.com')).toBe(true)
  })

  test('valid https URL returns true', () => {
    expect(is_url('https://example.com')).toBe(true)
    expect(is_url('https://example.com/path?query=1#hash')).toBe(true)
  })

  test('URL with leading/trailing whitespace is accepted', () => {
    expect(is_url('  https://example.com  ')).toBe(true)
  })

  test('ftp URL returns false (only http/https)', () => {
    expect(is_url('ftp://example.com')).toBe(false)
  })

  test('plain text returns false', () => {
    expect(is_url('hello world')).toBe(false)
    expect(is_url('not a url')).toBe(false)
  })

  test('empty string returns false', () => {
    expect(is_url('')).toBe(false)
  })

  test('null returns false', () => {
    expect(is_url(null)).toBe(false)
  })

  test('undefined returns false', () => {
    expect(is_url(undefined)).toBe(false)
  })

  test('string without protocol returns false', () => {
    expect(is_url('example.com')).toBe(false)
  })
})
