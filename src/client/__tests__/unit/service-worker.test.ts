/**
 * Service Worker utility function tests.
 * Tests should_cache_response and parse_bool_loose extracted to service-worker-utils.ts.
 */
import { should_cache_response, should_cache_for_session, parse_bool_loose } from '@/classes/service-worker-utils'

// Helper to create a mock Response
function mockResponse(body: object | string, ok = true): Response {
  const bodyStr = typeof body === 'string' ? body : JSON.stringify(body)
  return new Response(bodyStr, {
    status: ok ? 200 : 500,
    headers: { 'Content-Type': 'application/json' },
  })
}

// ========== should_cache_response ==========

describe('shouldCacheResponse', () => {
  test('returns false for non-ok response', async () => {
    const resp = mockResponse({}, false)
    expect(await should_cache_response(resp, false)).toBe(false)
  })

  test('returns false when errors array is non-empty', async () => {
    const resp = mockResponse({ errors: [{ error_code: 'ERR001', error_message: 'test' }] })
    expect(await should_cache_response(resp, false)).toBe(false)
  })

  test('returns true for ok response with empty errors', async () => {
    const resp = mockResponse({ errors: [], data: 'ok' })
    expect(await should_cache_response(resp, false)).toBe(true)
  })

  test('returns true when no errors field at all', async () => {
    const resp = mockResponse({ data: 'ok' })
    expect(await should_cache_response(resp, false)).toBe(true)
  })

  test('returns false when JSON parse fails', async () => {
    const resp = new Response('not json', { status: 200 })
    expect(await should_cache_response(resp, false)).toBe(false)
  })

  test('returns false when checkHistories=true and _histories is empty array', async () => {
    const resp = mockResponse({ errors: [], kmemo_histories: [] })
    expect(await should_cache_response(resp, true)).toBe(false)
  })

  test('returns true when checkHistories=true and _histories is non-empty', async () => {
    const resp = mockResponse({ errors: [], kmemo_histories: [{ id: '1' }] })
    expect(await should_cache_response(resp, true)).toBe(true)
  })

  test('returns true when checkHistories=false regardless of histories', async () => {
    const resp = mockResponse({ errors: [], kmemo_histories: [] })
    expect(await should_cache_response(resp, false)).toBe(true)
  })
})

// ========== parse_bool_loose ==========

describe('parseBoolLoose', () => {
  test('boolean true returns true', () => {
    expect(parse_bool_loose(true)).toBe(true)
  })

  test('boolean false returns false', () => {
    expect(parse_bool_loose(false)).toBe(false)
  })

  test('number 1 returns true', () => {
    expect(parse_bool_loose(1)).toBe(true)
  })

  test('number 0 returns false', () => {
    expect(parse_bool_loose(0)).toBe(false)
  })

  test('string "true"/"1"/"yes"/"y" return true', () => {
    expect(parse_bool_loose('true')).toBe(true)
    expect(parse_bool_loose('1')).toBe(true)
    expect(parse_bool_loose('yes')).toBe(true)
    expect(parse_bool_loose('y')).toBe(true)
  })

  test('string "false"/"0"/"no"/"n" return false', () => {
    expect(parse_bool_loose('false')).toBe(false)
    expect(parse_bool_loose('0')).toBe(false)
    expect(parse_bool_loose('no')).toBe(false)
    expect(parse_bool_loose('n')).toBe(false)
  })

  test('handles case-insensitive and trimmed strings', () => {
    expect(parse_bool_loose('  TRUE  ')).toBe(true)
    expect(parse_bool_loose('Yes')).toBe(true)
    expect(parse_bool_loose(' FALSE ')).toBe(false)
  })

  test('throws SyntaxError for invalid values', () => {
    expect(() => parse_bool_loose('maybe')).toThrow(SyntaxError)
    expect(() => parse_bool_loose(null)).toThrow(SyntaxError)
    expect(() => parse_bool_loose(undefined)).toThrow(SyntaxError)
  })
})

// ========== should_cache_for_session (M-9) ==========

describe('should_cache_for_session', () => {
  test('matching session caches', () => {
    expect(should_cache_for_session('sess-a', 'sess-a')).toBe(true)
  })
  test('mismatched session does not cache (account switch)', () => {
    expect(should_cache_for_session('sess-a', 'sess-b')).toBe(false)
  })
  test('unknown current session falls back to caching (e.g. Firefox no cookieStore)', () => {
    expect(should_cache_for_session('sess-a', undefined)).toBe(true)
    expect(should_cache_for_session('sess-a', null)).toBe(true)
    expect(should_cache_for_session('sess-a', '')).toBe(true)
  })
  test('missing body session falls back to caching', () => {
    expect(should_cache_for_session(undefined, 'sess-a')).toBe(true)
    expect(should_cache_for_session('', 'sess-a')).toBe(true)
  })
})
