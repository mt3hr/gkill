import { isUrl } from '@/classes/looks-like-url'

describe('isUrl', () => {
  test('valid http URL returns true', () => {
    expect(isUrl('http://example.com')).toBe(true)
  })

  test('valid https URL returns true', () => {
    expect(isUrl('https://example.com')).toBe(true)
    expect(isUrl('https://example.com/path?query=1#hash')).toBe(true)
  })

  test('URL with leading/trailing whitespace is accepted', () => {
    expect(isUrl('  https://example.com  ')).toBe(true)
  })

  test('ftp URL returns false (only http/https)', () => {
    expect(isUrl('ftp://example.com')).toBe(false)
  })

  test('plain text returns false', () => {
    expect(isUrl('hello world')).toBe(false)
    expect(isUrl('not a url')).toBe(false)
  })

  test('empty string returns false', () => {
    expect(isUrl('')).toBe(false)
  })

  test('null returns false', () => {
    expect(isUrl(null)).toBe(false)
  })

  test('undefined returns false', () => {
    expect(isUrl(undefined)).toBe(false)
  })

  test('string without protocol returns false', () => {
    expect(isUrl('example.com')).toBe(false)
  })
})
