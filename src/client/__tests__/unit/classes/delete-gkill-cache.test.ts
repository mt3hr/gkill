/**
 * delete-gkill-cache.ts tests.
 * Mocks the Cache API (globalThis.caches).
 */
import { vi } from 'vitest'

// jsdom Request requires absolute URLs; patch to accept relative
const OriginalRequest = globalThis.Request
globalThis.Request = class extends OriginalRequest {
  constructor(input: RequestInfo | URL, init?: RequestInit) {
    const url = typeof input === 'string' && !input.startsWith('http') ? `http://localhost${input}` : input
    super(url, init)
  }
} as typeof Request

// Mock the Cache API
const mockCacheDelete = vi.fn().mockResolvedValue(true)
const mockCacheMatch = vi.fn().mockResolvedValue(undefined)
const mockCache = {
  delete: mockCacheDelete,
  match: mockCacheMatch,
  put: vi.fn(),
  add: vi.fn(),
  addAll: vi.fn(),
  keys: vi.fn(),
  matchAll: vi.fn(),
}
const mockCachesOpen = vi.fn().mockResolvedValue(mockCache)
const mockCachesDelete = vi.fn().mockResolvedValue(true)

Object.defineProperty(globalThis, 'caches', {
  value: {
    open: mockCachesOpen,
    delete: mockCachesDelete,
    has: vi.fn(),
    keys: vi.fn(),
    match: vi.fn(),
  },
  writable: true,
})

import delete_gkill_kyou_cache, { delete_gkill_config_cache } from '@/classes/delete-gkill-cache'

beforeEach(() => {
  vi.clearAllMocks()
})

describe('delete_gkill_kyou_cache', () => {
  test('deletes cache entries for all 14 data types when id provided', async () => {
    await delete_gkill_kyou_cache('test-id')
    expect(mockCachesOpen).toHaveBeenCalledWith('gkill-post-kyou-cache')
    expect(mockCacheDelete).toHaveBeenCalledTimes(14)
  })

  test('constructs correct cache key format /cache/api/{type}/{id}', async () => {
    await delete_gkill_kyou_cache('abc-123')
    const firstCallArg = mockCacheDelete.mock.calls[0][0]
    expect(firstCallArg).toBeInstanceOf(Request)
    expect(firstCallArg.url).toContain('/cache/api/kyou/abc-123')
  })

  test('deletes entire cache when id is null', async () => {
    await delete_gkill_kyou_cache(null)
    expect(mockCachesDelete).toHaveBeenCalledWith('gkill-post-kyou-cache')
    expect(mockCacheDelete).not.toHaveBeenCalled()
  })
})

describe('delete_gkill_config_cache', () => {
  test('deletes cache entries for all 4 config types', async () => {
    await delete_gkill_config_cache()
    expect(mockCacheDelete).toHaveBeenCalledTimes(4)
  })

  test('opens gkill-post-config-cache', async () => {
    await delete_gkill_config_cache()
    expect(mockCachesOpen).toHaveBeenCalledWith('gkill-post-config-cache')
  })

  test('constructs correct cache key format /cache/api/{type}', async () => {
    await delete_gkill_config_cache()
    const firstCallArg = mockCacheDelete.mock.calls[0][0]
    expect(firstCallArg).toBeInstanceOf(Request)
    expect(firstCallArg.url).toContain('/cache/api/application_config')
  })
})
