/**
 * Service Worker utility function tests.
 * Tests shouldCacheResponse and parseBoolLoose extracted to service-worker-utils.ts.
 */
import { shouldCacheResponse, parseBoolLoose } from '@/classes/service-worker-utils'

// Helper to create a mock Response
function mockResponse(body: object | string, ok = true): Response {
  const bodyStr = typeof body === 'string' ? body : JSON.stringify(body)
  return new Response(bodyStr, {
    status: ok ? 200 : 500,
    headers: { 'Content-Type': 'application/json' },
  })
}

// ========== shouldCacheResponse ==========

describe('shouldCacheResponse', () => {
  test('returns false for non-ok response', async () => {
    const resp = mockResponse({}, false)
    expect(await shouldCacheResponse(resp, false)).toBe(false)
  })

  test('returns false when errors array is non-empty', async () => {
    const resp = mockResponse({ errors: [{ error_code: 'ERR001', error_message: 'test' }] })
    expect(await shouldCacheResponse(resp, false)).toBe(false)
  })

  test('returns true for ok response with empty errors', async () => {
    const resp = mockResponse({ errors: [], data: 'ok' })
    expect(await shouldCacheResponse(resp, false)).toBe(true)
  })

  test('returns true when no errors field at all', async () => {
    const resp = mockResponse({ data: 'ok' })
    expect(await shouldCacheResponse(resp, false)).toBe(true)
  })

  test('returns false when JSON parse fails', async () => {
    const resp = new Response('not json', { status: 200 })
    expect(await shouldCacheResponse(resp, false)).toBe(false)
  })

  test('returns false when checkHistories=true and _histories is empty array', async () => {
    const resp = mockResponse({ errors: [], kmemo_histories: [] })
    expect(await shouldCacheResponse(resp, true)).toBe(false)
  })

  test('returns true when checkHistories=true and _histories is non-empty', async () => {
    const resp = mockResponse({ errors: [], kmemo_histories: [{ id: '1' }] })
    expect(await shouldCacheResponse(resp, true)).toBe(true)
  })

  test('returns true when checkHistories=false regardless of histories', async () => {
    const resp = mockResponse({ errors: [], kmemo_histories: [] })
    expect(await shouldCacheResponse(resp, false)).toBe(true)
  })
})

// ========== parseBoolLoose ==========

describe('parseBoolLoose', () => {
  test('boolean true returns true', () => {
    expect(parseBoolLoose(true)).toBe(true)
  })

  test('boolean false returns false', () => {
    expect(parseBoolLoose(false)).toBe(false)
  })

  test('number 1 returns true', () => {
    expect(parseBoolLoose(1)).toBe(true)
  })

  test('number 0 returns false', () => {
    expect(parseBoolLoose(0)).toBe(false)
  })

  test('string "true"/"1"/"yes"/"y" return true', () => {
    expect(parseBoolLoose('true')).toBe(true)
    expect(parseBoolLoose('1')).toBe(true)
    expect(parseBoolLoose('yes')).toBe(true)
    expect(parseBoolLoose('y')).toBe(true)
  })

  test('string "false"/"0"/"no"/"n" return false', () => {
    expect(parseBoolLoose('false')).toBe(false)
    expect(parseBoolLoose('0')).toBe(false)
    expect(parseBoolLoose('no')).toBe(false)
    expect(parseBoolLoose('n')).toBe(false)
  })

  test('handles case-insensitive and trimmed strings', () => {
    expect(parseBoolLoose('  TRUE  ')).toBe(true)
    expect(parseBoolLoose('Yes')).toBe(true)
    expect(parseBoolLoose(' FALSE ')).toBe(false)
  })

  test('throws SyntaxError for invalid values', () => {
    expect(() => parseBoolLoose('maybe')).toThrow(SyntaxError)
    expect(() => parseBoolLoose(null)).toThrow(SyntaxError)
    expect(() => parseBoolLoose(undefined)).toThrow(SyntaxError)
  })
})
