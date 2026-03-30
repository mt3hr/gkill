import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest'
import { i18n } from '../../helpers/setup-i18n'
import { makeKmemo, makeMi, makeTag, makeURLog, makeNlog, makeLantana, makeText, makeShareKyousInfo } from '../../helpers/factory'

// Mock @/i18n before importing GkillAPI
vi.mock('@/i18n', () => ({ i18n }))

// Mock delete-gkill-cache to avoid Cache API / Request issues in jsdom
vi.mock('@/classes/delete-gkill-cache', () => ({
  default: vi.fn().mockResolvedValue(undefined),
  delete_gkill_config_cache: vi.fn().mockResolvedValue(undefined),
}))

import { GkillAPI } from '@/classes/api/gkill-api'

describe('GkillAPI', () => {
  describe('singleton access', () => {
    test('get_instance returns a GkillAPI instance', () => {
      const api = GkillAPI.get_instance()
      expect(api).toBeInstanceOf(GkillAPI)
    })

    test('get_gkill_api returns a GkillAPI instance', () => {
      const api = GkillAPI.get_gkill_api()
      expect(api).toBeInstanceOf(GkillAPI)
    })

    test('get_instance returns the same instance on multiple calls', () => {
      const a = GkillAPI.get_instance()
      const b = GkillAPI.get_instance()
      expect(a).toBe(b)
    })
  })

  describe('endpoint addresses', () => {
    test('login_address is /api/login', () => {
      const api = GkillAPI.get_instance()
      expect(api.login_address).toBe('/api/login')
    })

    test('get_kyous_address is /api/get_kyous', () => {
      const api = GkillAPI.get_instance()
      expect(api.get_kyous_address).toBe('/api/get_kyous')
    })

    test('add_kmemo_address is /api/add_kmemo', () => {
      const api = GkillAPI.get_instance()
      expect(api.add_kmemo_address).toBe('/api/add_kmemo')
    })
  })

  describe('generate_uuid', () => {
    test('returns a string', () => {
      const api = GkillAPI.get_instance()
      const uuid = api.generate_uuid()
      expect(typeof uuid).toBe('string')
    })

    test('returns UUID v4 format', () => {
      const api = GkillAPI.get_instance()
      const uuid = api.generate_uuid()
      // UUID v4 format: 8-4-4-4-12 hex characters
      expect(uuid).toMatch(/^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/)
    })

    test('generates unique UUIDs', () => {
      const api = GkillAPI.get_instance()
      const uuids = new Set<string>()
      for (let i = 0; i < 100; i++) {
        uuids.add(api.generate_uuid())
      }
      expect(uuids.size).toBe(100)
    })
  })

  describe('login method with mocked fetch', () => {
    const originalFetch = globalThis.fetch

    beforeEach(() => {
      globalThis.fetch = vi.fn()
    })

    afterEach(() => {
      globalThis.fetch = originalFetch
    })

    test('calls fetch with correct URL and method', async () => {
      const mockResponse = {
        messages: [],
        errors: [],
      }
      ;(globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue({
        json: () => Promise.resolve(mockResponse),
      })

      const api = GkillAPI.get_instance()
      const req = {
        user_id: 'test_user',
        password_sha256: 'abc123',
      }
      await api.login(req as never)

      expect(globalThis.fetch).toHaveBeenCalledWith(
        '/api/login',
        expect.objectContaining({
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(req),
        })
      )
    })

    test('returns parsed JSON response', async () => {
      const mockResponse = {
        session_id: 'test-session-123',
        messages: [],
        errors: [],
      }
      ;(globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue({
        json: () => Promise.resolve(mockResponse),
      })

      const api = GkillAPI.get_instance()
      const result = await api.login({ user_id: 'u', password_sha256: 'p' } as never)
      expect(result.session_id).toBe('test-session-123')
    })
  })

  describe('key methods exist', () => {
    const api = GkillAPI.get_instance()

    test.each([
      'login',
      'logout',
      'reset_password',
      'generate_uuid',
      'get_session_id',
      'set_session_id',
      'check_auth',
    ])('%s is a function', (method) => {
      expect(typeof (api as Record<string, unknown>)[method]).toBe('function')
    })
  })

  describe('data write methods', () => {
    const originalFetch = globalThis.fetch

    beforeEach(() => {
      globalThis.fetch = vi.fn()
    })

    afterEach(() => {
      globalThis.fetch = originalFetch
    })

    test('add_kmemo sends POST to /api/add_kmemo with session_id and kmemo', async () => {
      const mockResponse = { messages: [], errors: [] }
      ;(globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue({
        json: () => Promise.resolve(mockResponse),
      })

      const api = GkillAPI.get_instance()
      const kmemo = makeKmemo()
      const req = {
        session_id: 'test-session',
        kmemo,
        tx_id: null,
        abort_controller: new AbortController(),
        force_reget: false,
        locale_name: 'ja',
      }
      await api.add_kmemo(req as never)

      expect(globalThis.fetch).toHaveBeenCalledWith(
        '/api/add_kmemo',
        expect.objectContaining({
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(req),
        })
      )
    })

    test('add_mi sends POST to /api/add_mi with session_id and mi', async () => {
      const mockResponse = { messages: [], errors: [] }
      ;(globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue({
        json: () => Promise.resolve(mockResponse),
      })

      const api = GkillAPI.get_instance()
      const mi = makeMi()
      const req = {
        session_id: 'test-session',
        mi,
        tx_id: null,
        want_response_kyou: false,
        added_kyou: null,
        abort_controller: new AbortController(),
        force_reget: false,
        locale_name: 'ja',
      }
      await api.add_mi(req as never)

      expect(globalThis.fetch).toHaveBeenCalledWith(
        '/api/add_mi',
        expect.objectContaining({
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(req),
        })
      )
    })

    test('add_tag sends POST to /api/add_tag with session_id and tag', async () => {
      const mockResponse = { messages: [], errors: [] }
      ;(globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue({
        json: () => Promise.resolve(mockResponse),
      })

      const api = GkillAPI.get_instance()
      const tag = makeTag()
      const req = {
        session_id: 'test-session',
        tag,
        tx_id: null,
        abort_controller: new AbortController(),
        force_reget: false,
        locale_name: 'ja',
      }
      await api.add_tag(req as never)

      expect(globalThis.fetch).toHaveBeenCalledWith(
        '/api/add_tag',
        expect.objectContaining({
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(req),
        })
      )
    })
  })

  describe('data read methods', () => {
    const originalFetch = globalThis.fetch

    beforeEach(() => {
      globalThis.fetch = vi.fn()
    })

    afterEach(() => {
      globalThis.fetch = originalFetch
    })

    test('get_kyous sends POST to /api/get_kyous with session_id and query', async () => {
      const mockResponse = { kyous: [], messages: [], errors: [] }
      ;(globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue({
        json: () => Promise.resolve(mockResponse),
      })

      const api = GkillAPI.get_instance()
      const req = {
        session_id: 'test-session',
        query: { words: [], tags: [], use_plaing: false },
        abort_controller: new AbortController(),
        force_reget: false,
        locale_name: 'ja',
      }
      await api.get_kyous(req as never)

      expect(globalThis.fetch).toHaveBeenCalledWith(
        '/api/get_kyous',
        expect.objectContaining({
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(req),
        })
      )
    })

    test('get_kmemo sends POST to /api/get_kmemo with session_id and id', async () => {
      const mockResponse = { kmemo_histories: [], messages: [], errors: [] }
      ;(globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue({
        json: () => Promise.resolve(mockResponse),
      })

      const api = GkillAPI.get_instance()
      const req = {
        session_id: 'test-session',
        id: 'test-kmemo-id',
        update_time: null,
        abort_controller: new AbortController(),
        force_reget: false,
        locale_name: 'ja',
      }
      await api.get_kmemo(req as never)

      expect(globalThis.fetch).toHaveBeenCalledWith(
        '/api/get_kmemo',
        expect.objectContaining({
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(req),
        })
      )
    })

    test('get_mi sends POST to /api/get_mi with session_id and id', async () => {
      const mockResponse = { mi_histories: [], messages: [], errors: [] }
      ;(globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue({
        json: () => Promise.resolve(mockResponse),
      })

      const api = GkillAPI.get_instance()
      const req = {
        session_id: 'test-session',
        id: 'test-mi-id',
        update_time: null,
        abort_controller: new AbortController(),
        force_reget: false,
        locale_name: 'ja',
      }
      await api.get_mi(req as never)

      expect(globalThis.fetch).toHaveBeenCalledWith(
        '/api/get_mi',
        expect.objectContaining({
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(req),
        })
      )
    })
  })

  describe('update methods', () => {
    const originalFetch = globalThis.fetch

    beforeEach(() => {
      globalThis.fetch = vi.fn()
    })

    afterEach(() => {
      globalThis.fetch = originalFetch
    })

    test('update_kmemo sends POST to /api/update_kmemo with session_id and kmemo', async () => {
      const mockResponse = { messages: [], errors: [] }
      ;(globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue({
        json: () => Promise.resolve(mockResponse),
      })

      const api = GkillAPI.get_instance()
      const kmemo = makeKmemo({ content: 'updated content' })
      const req = {
        session_id: 'test-session',
        kmemo,
        tx_id: null,
        want_response_kyou: false,
        updated_kyou: null,
        abort_controller: new AbortController(),
        force_reget: false,
        locale_name: 'ja',
      }
      await api.update_kmemo(req as never)

      expect(globalThis.fetch).toHaveBeenCalledWith(
        '/api/update_kmemo',
        expect.objectContaining({
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(req),
        })
      )
    })

    test('update_mi sends POST to /api/update_mi with session_id and mi', async () => {
      const mockResponse = { messages: [], errors: [] }
      ;(globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue({
        json: () => Promise.resolve(mockResponse),
      })

      const api = GkillAPI.get_instance()
      const mi = makeMi({ title: 'updated task', is_checked: true })
      const req = {
        session_id: 'test-session',
        mi,
        tx_id: null,
        want_response_kyou: false,
        updated_kyou: null,
        abort_controller: new AbortController(),
        force_reget: false,
        locale_name: 'ja',
      }
      await api.update_mi(req as never)

      expect(globalThis.fetch).toHaveBeenCalledWith(
        '/api/update_mi',
        expect.objectContaining({
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(req),
        })
      )
    })
  })

  describe('error handling', () => {
    const originalFetch = globalThis.fetch

    beforeEach(() => {
      globalThis.fetch = vi.fn()
    })

    afterEach(() => {
      globalThis.fetch = originalFetch
    })

    test('network failure (fetch throws Error) propagates the error', async () => {
      ;(globalThis.fetch as ReturnType<typeof vi.fn>).mockRejectedValue(
        new Error('Network error')
      )

      const api = GkillAPI.get_instance()
      const req = {
        session_id: 'test-session',
        id: 'test-id',
        update_time: null,
        abort_controller: new AbortController(),
        force_reget: false,
        locale_name: 'ja',
      }

      await expect(api.get_kmemo(req as never)).rejects.toThrow('Network error')
    })

    test('HTTP 500 response still attempts to parse JSON', async () => {
      const errorResponse = {
        messages: [],
        errors: [{ error_code: 'INTERNAL', error_message: 'Server error' }],
      }
      ;(globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue({
        status: 500,
        json: () => Promise.resolve(errorResponse),
      })

      const api = GkillAPI.get_instance()
      const req = {
        session_id: 'test-session',
        id: 'test-id',
        update_time: null,
        abort_controller: new AbortController(),
        force_reget: false,
        locale_name: 'ja',
      }

      // The API does not check HTTP status; it parses JSON regardless
      const result = await api.get_kmemo(req as never)
      expect(result.errors).toEqual([
        { error_code: 'INTERNAL', error_message: 'Server error' },
      ])
    })

    test('malformed JSON response causes an error', async () => {
      ;(globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue({
        json: () => Promise.reject(new SyntaxError('Unexpected token')),
      })

      const api = GkillAPI.get_instance()
      const req = {
        session_id: 'test-session',
        id: 'test-id',
        update_time: null,
        abort_controller: new AbortController(),
        force_reget: false,
        locale_name: 'ja',
      }

      await expect(api.get_kmemo(req as never)).rejects.toThrow(SyntaxError)
    })
  })

  describe('session management', () => {
    test('set_session_id / get_session_id round-trip', () => {
      const api = GkillAPI.get_instance()
      api.set_session_id('round-trip-session-123')
      const result = api.get_session_id()
      expect(result).toBe('round-trip-session-123')

      // Clean up
      api.set_session_id('')
    })

    test('set_session_id with empty string clears the session', () => {
      const api = GkillAPI.get_instance()
      api.set_session_id('some-session')
      api.set_session_id('')
      const result = api.get_session_id()
      expect(result).toBe('')
    })

    test('logout clears session by calling the logout endpoint', async () => {
      const originalFetch = globalThis.fetch
      globalThis.fetch = vi.fn()

      // Mock the Cache API (used by delete_gkill_cache inside logout)
      const originalCaches = globalThis.caches
      const mockCache = { keys: vi.fn().mockResolvedValue([]), delete: vi.fn().mockResolvedValue(true) }
      globalThis.caches = {
        open: vi.fn().mockResolvedValue(mockCache),
        keys: vi.fn().mockResolvedValue([]),
        delete: vi.fn().mockResolvedValue(true),
        has: vi.fn().mockResolvedValue(false),
        match: vi.fn().mockResolvedValue(undefined),
      } as CacheStorage

      const mockResponse = { messages: [], errors: [] }
      ;(globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue({
        json: () => Promise.resolve(mockResponse),
      })

      const api = GkillAPI.get_instance()
      api.set_session_id('session-to-clear')

      const req = {
        session_id: 'session-to-clear',
        close_database: false,
        abort_controller: new AbortController(),
        force_reget: false,
        locale_name: 'ja',
      }
      const result = await api.logout(req as never)

      expect(globalThis.fetch).toHaveBeenCalledWith(
        '/api/logout',
        expect.objectContaining({
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(req),
        })
      )
      expect(result.messages).toEqual([])
      expect(result.errors).toEqual([])

      // Clean up
      api.set_session_id('')
      globalThis.fetch = originalFetch
      if (originalCaches) {
        globalThis.caches = originalCaches
      }
    })
  })

  describe('additional write methods', () => {
    const originalFetch = globalThis.fetch

    beforeEach(() => {
      globalThis.fetch = vi.fn()
    })

    afterEach(() => {
      globalThis.fetch = originalFetch
    })

    test('add_timeis sends POST to /api/add_timeis with session_id and timeis', async () => {
      const mockResponse = { messages: [], errors: [] }
      ;(globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue({
        json: () => Promise.resolve(mockResponse),
      })

      const api = GkillAPI.get_instance()
      const timeis = {
        is_deleted: false,
        id: 'test-timeis-id',
        rep_name: 'test-rep',
        related_time: '2025-03-15T09:00:00+09:00',
        start_time: '2025-03-15T09:00:00+09:00',
        end_time: '2025-03-15T10:00:00+09:00',
        create_time: '2025-03-15T09:00:00+09:00',
        create_app: 'gkill',
        create_device: 'test-device',
        create_user: 'admin',
        update_time: '2025-03-15T09:00:00+09:00',
        update_app: 'gkill',
        update_device: 'test-device',
        update_user: 'admin',
      }
      const req = {
        session_id: 'test-session',
        timeis,
        tx_id: null,
        abort_controller: new AbortController(),
        force_reget: false,
        locale_name: 'ja',
      }
      await api.add_timeis(req as never)

      expect(globalThis.fetch).toHaveBeenCalledWith(
        '/api/add_timeis',
        expect.objectContaining({
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(req),
        })
      )
    })

    test('add_lantana sends POST to /api/add_lantana with session_id and lantana', async () => {
      const mockResponse = { messages: [], errors: [] }
      ;(globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue({
        json: () => Promise.resolve(mockResponse),
      })

      const api = GkillAPI.get_instance()
      const lantana = makeLantana()
      const req = {
        session_id: 'test-session',
        lantana,
        tx_id: null,
        abort_controller: new AbortController(),
        force_reget: false,
        locale_name: 'ja',
      }
      await api.add_lantana(req as never)

      expect(globalThis.fetch).toHaveBeenCalledWith(
        '/api/add_lantana',
        expect.objectContaining({
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(req),
        })
      )
    })

    test('add_nlog sends POST to /api/add_nlog with session_id and nlog', async () => {
      const mockResponse = { messages: [], errors: [] }
      ;(globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue({
        json: () => Promise.resolve(mockResponse),
      })

      const api = GkillAPI.get_instance()
      const nlog = makeNlog()
      const req = {
        session_id: 'test-session',
        nlog,
        tx_id: null,
        abort_controller: new AbortController(),
        force_reget: false,
        locale_name: 'ja',
      }
      await api.add_nlog(req as never)

      expect(globalThis.fetch).toHaveBeenCalledWith(
        '/api/add_nlog',
        expect.objectContaining({
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(req),
        })
      )
    })

    test('add_urlog sends POST to /api/add_urlog with session_id and urlog', async () => {
      const mockResponse = { messages: [], errors: [] }
      ;(globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue({
        json: () => Promise.resolve(mockResponse),
      })

      const api = GkillAPI.get_instance()
      const urlog = makeURLog()
      const req = {
        session_id: 'test-session',
        urlog,
        tx_id: null,
        abort_controller: new AbortController(),
        force_reget: false,
        locale_name: 'ja',
      }
      await api.add_urlog(req as never)

      expect(globalThis.fetch).toHaveBeenCalledWith(
        '/api/add_urlog',
        expect.objectContaining({
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(req),
        })
      )
    })

    test('add_kc sends POST to /api/add_kc with session_id and kc', async () => {
      const mockResponse = { messages: [], errors: [] }
      ;(globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue({
        json: () => Promise.resolve(mockResponse),
      })

      const api = GkillAPI.get_instance()
      const kc = {
        is_deleted: false,
        id: 'test-kc-id',
        rep_name: 'test-rep',
        related_time: '2025-03-15T09:00:00+09:00',
        value: 42,
        create_time: '2025-03-15T09:00:00+09:00',
        create_app: 'gkill',
        create_device: 'test-device',
        create_user: 'admin',
        update_time: '2025-03-15T09:00:00+09:00',
        update_app: 'gkill',
        update_device: 'test-device',
        update_user: 'admin',
      }
      const req = {
        session_id: 'test-session',
        kc,
        tx_id: null,
        abort_controller: new AbortController(),
        force_reget: false,
        locale_name: 'ja',
      }
      await api.add_kc(req as never)

      expect(globalThis.fetch).toHaveBeenCalledWith(
        '/api/add_kc',
        expect.objectContaining({
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(req),
        })
      )
    })

    test('add_text sends POST to /api/add_text with session_id and text', async () => {
      const mockResponse = { messages: [], errors: [] }
      ;(globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue({
        json: () => Promise.resolve(mockResponse),
      })

      const api = GkillAPI.get_instance()
      const text = makeText()
      const req = {
        session_id: 'test-session',
        text,
        tx_id: null,
        abort_controller: new AbortController(),
        force_reget: false,
        locale_name: 'ja',
      }
      await api.add_text(req as never)

      expect(globalThis.fetch).toHaveBeenCalledWith(
        '/api/add_text',
        expect.objectContaining({
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(req),
        })
      )
    })

    test('add_rekyou sends POST to /api/add_rekyou with session_id and re_kyou', async () => {
      const mockResponse = { messages: [], errors: [] }
      ;(globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue({
        json: () => Promise.resolve(mockResponse),
      })

      const api = GkillAPI.get_instance()
      const re_kyou = {
        is_deleted: false,
        id: 'test-rekyou-id',
        rep_name: 'test-rep',
        related_time: '2025-03-15T09:00:00+09:00',
        target_id: 'test-target-id',
        create_time: '2025-03-15T09:00:00+09:00',
        create_app: 'gkill',
        create_device: 'test-device',
        create_user: 'admin',
        update_time: '2025-03-15T09:00:00+09:00',
        update_app: 'gkill',
        update_device: 'test-device',
        update_user: 'admin',
      }
      const req = {
        session_id: 'test-session',
        re_kyou,
        tx_id: null,
        abort_controller: new AbortController(),
        force_reget: false,
        locale_name: 'ja',
      }
      await api.add_rekyou(req as never)

      expect(globalThis.fetch).toHaveBeenCalledWith(
        '/api/add_rekyou',
        expect.objectContaining({
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(req),
        })
      )
    })
  })

  describe('additional read methods', () => {
    const originalFetch = globalThis.fetch

    beforeEach(() => {
      globalThis.fetch = vi.fn()
    })

    afterEach(() => {
      globalThis.fetch = originalFetch
    })

    test('get_kyou sends POST to /api/get_kyou with session_id and id', async () => {
      const mockResponse = { kyou_histories: [], messages: [], errors: [] }
      ;(globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue({
        json: () => Promise.resolve(mockResponse),
      })

      const api = GkillAPI.get_instance()
      const req = {
        session_id: 'test-session',
        id: 'test-kyou-id',
        update_time: null,
        abort_controller: new AbortController(),
        force_reget: false,
        locale_name: 'ja',
      }
      await api.get_kyou(req as never)

      expect(globalThis.fetch).toHaveBeenCalledWith(
        '/api/get_kyou',
        expect.objectContaining({
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(req),
        })
      )
    })

    test('get_tags_by_target_id sends POST to /api/get_tags_by_id with session_id and target_id', async () => {
      const mockResponse = { tags: [], messages: [], errors: [] }
      ;(globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue({
        json: () => Promise.resolve(mockResponse),
      })

      const api = GkillAPI.get_instance()
      const req = {
        session_id: 'test-session',
        target_id: 'test-target-id',
        abort_controller: new AbortController(),
        force_reget: false,
        locale_name: 'ja',
      }
      await api.get_tags_by_target_id(req as never)

      expect(globalThis.fetch).toHaveBeenCalledWith(
        '/api/get_tags_by_id',
        expect.objectContaining({
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(req),
        })
      )
    })

    test('get_texts_by_target_id sends POST to /api/get_texts_by_id with session_id and target_id', async () => {
      const mockResponse = { texts: [], messages: [], errors: [] }
      ;(globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue({
        json: () => Promise.resolve(mockResponse),
      })

      const api = GkillAPI.get_instance()
      const req = {
        session_id: 'test-session',
        target_id: 'test-target-id',
        abort_controller: new AbortController(),
        force_reget: false,
        locale_name: 'ja',
      }
      await api.get_texts_by_target_id(req as never)

      expect(globalThis.fetch).toHaveBeenCalledWith(
        '/api/get_texts_by_id',
        expect.objectContaining({
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(req),
        })
      )
    })

    test('get_mi_board_list sends POST to /api/get_mi_board_list with session_id', async () => {
      const mockResponse = { mi_board_list: [], messages: [], errors: [] }
      ;(globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue({
        json: () => Promise.resolve(mockResponse),
      })

      const api = GkillAPI.get_instance()
      const req = {
        session_id: 'test-session',
        abort_controller: new AbortController(),
        force_reget: false,
        locale_name: 'ja',
      }
      await api.get_mi_board_list(req as never)

      expect(globalThis.fetch).toHaveBeenCalledWith(
        '/api/get_mi_board_list',
        expect.objectContaining({
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(req),
        })
      )
    })

    test('get_all_tag_names sends POST to /api/get_all_tag_names with session_id', async () => {
      const mockResponse = { tag_names: [], messages: [], errors: [] }
      ;(globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue({
        json: () => Promise.resolve(mockResponse),
      })

      const api = GkillAPI.get_instance()
      const req = {
        session_id: 'test-session',
        abort_controller: new AbortController(),
        force_reget: false,
        locale_name: 'ja',
      }
      await api.get_all_tag_names(req as never)

      expect(globalThis.fetch).toHaveBeenCalledWith(
        '/api/get_all_tag_names',
        expect.objectContaining({
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(req),
        })
      )
    })
  })

  describe('additional update methods', () => {
    const originalFetch = globalThis.fetch

    beforeEach(() => {
      globalThis.fetch = vi.fn()
    })

    afterEach(() => {
      globalThis.fetch = originalFetch
    })

    test('update_tag sends POST to /api/update_tag with session_id and tag', async () => {
      const mockResponse = { messages: [], errors: [] }
      ;(globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue({
        json: () => Promise.resolve(mockResponse),
      })

      const api = GkillAPI.get_instance()
      const tag = makeTag({ tag: 'updated-tag' })
      const req = {
        session_id: 'test-session',
        tag,
        tx_id: null,
        abort_controller: new AbortController(),
        force_reget: false,
        locale_name: 'ja',
      }
      await api.update_tag(req as never)

      expect(globalThis.fetch).toHaveBeenCalledWith(
        '/api/update_tag',
        expect.objectContaining({
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(req),
        })
      )
    })

    test('update_timeis sends POST to /api/update_timeis with session_id and timeis', async () => {
      const mockResponse = { messages: [], errors: [] }
      ;(globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue({
        json: () => Promise.resolve(mockResponse),
      })

      const api = GkillAPI.get_instance()
      const timeis = {
        is_deleted: false,
        id: 'test-timeis-id',
        rep_name: 'test-rep',
        related_time: '2025-03-15T09:00:00+09:00',
        start_time: '2025-03-15T09:00:00+09:00',
        end_time: '2025-03-15T11:00:00+09:00',
        create_time: '2025-03-15T09:00:00+09:00',
        create_app: 'gkill',
        create_device: 'test-device',
        create_user: 'admin',
        update_time: '2025-03-15T10:00:00+09:00',
        update_app: 'gkill',
        update_device: 'test-device',
        update_user: 'admin',
      }
      const req = {
        session_id: 'test-session',
        timeis,
        tx_id: null,
        want_response_kyou: false,
        updated_kyou: null,
        abort_controller: new AbortController(),
        force_reget: false,
        locale_name: 'ja',
      }
      await api.update_timeis(req as never)

      expect(globalThis.fetch).toHaveBeenCalledWith(
        '/api/update_timeis',
        expect.objectContaining({
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(req),
        })
      )
    })

    test('update_lantana sends POST to /api/update_lantana with session_id and lantana', async () => {
      const mockResponse = { messages: [], errors: [] }
      ;(globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue({
        json: () => Promise.resolve(mockResponse),
      })

      const api = GkillAPI.get_instance()
      const lantana = makeLantana({ mood: 8 })
      const req = {
        session_id: 'test-session',
        lantana,
        tx_id: null,
        want_response_kyou: false,
        updated_kyou: null,
        abort_controller: new AbortController(),
        force_reget: false,
        locale_name: 'ja',
      }
      await api.update_lantana(req as never)

      expect(globalThis.fetch).toHaveBeenCalledWith(
        '/api/update_lantana',
        expect.objectContaining({
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(req),
        })
      )
    })

    test('update_nlog sends POST to /api/update_nlog with session_id and nlog', async () => {
      const mockResponse = { messages: [], errors: [] }
      ;(globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue({
        json: () => Promise.resolve(mockResponse),
      })

      const api = GkillAPI.get_instance()
      const nlog = makeNlog({ amount: 2500, title: 'updated expense' })
      const req = {
        session_id: 'test-session',
        nlog,
        tx_id: null,
        want_response_kyou: false,
        updated_kyou: null,
        abort_controller: new AbortController(),
        force_reget: false,
        locale_name: 'ja',
      }
      await api.update_nlog(req as never)

      expect(globalThis.fetch).toHaveBeenCalledWith(
        '/api/update_nlog',
        expect.objectContaining({
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(req),
        })
      )
    })
  })

  describe('HTTP 403 error handling', () => {
    const originalFetch = globalThis.fetch

    beforeEach(() => {
      globalThis.fetch = vi.fn()
    })

    afterEach(() => {
      globalThis.fetch = originalFetch
    })

    test('HTTP 403 response returns error in errors array', async () => {
      const forbiddenResponse = {
        messages: [],
        errors: [{ error_code: 'ERR000013', error_message: 'Account session not found' }],
      }
      ;(globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue({
        status: 403,
        json: () => Promise.resolve(forbiddenResponse),
      })

      const api = GkillAPI.get_instance()
      const req = {
        session_id: 'invalid-session',
        id: 'test-id',
        update_time: null,
        abort_controller: new AbortController(),
        force_reget: false,
        locale_name: 'ja',
      }

      const result = await api.get_kmemo(req as never)
      expect(result.errors).toEqual([
        { error_code: 'ERR000013', error_message: 'Account session not found' },
      ])
    })
  })

  describe('additional get methods', () => {
    const originalFetch = globalThis.fetch

    beforeEach(() => {
      globalThis.fetch = vi.fn()
    })

    afterEach(() => {
      globalThis.fetch = originalFetch
    })

    test('get_kc sends POST to /api/get_kc with session_id and id', async () => {
      const mockResponse = { kc_histories: [], messages: [], errors: [] }
      ;(globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue({
        json: () => Promise.resolve(mockResponse),
      })

      const api = GkillAPI.get_instance()
      const req = {
        session_id: 'test-session',
        id: 'test-kc-id',
        update_time: null,
        abort_controller: new AbortController(),
        force_reget: false,
        locale_name: 'ja',
      }
      await api.get_kc(req as never)

      expect(globalThis.fetch).toHaveBeenCalledWith(
        '/api/get_kc',
        expect.objectContaining({
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(req),
        })
      )
    })

    test('get_urlog sends POST to /api/get_urlog with session_id and id', async () => {
      const mockResponse = { urlog_histories: [], messages: [], errors: [] }
      ;(globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue({
        json: () => Promise.resolve(mockResponse),
      })

      const api = GkillAPI.get_instance()
      const req = {
        session_id: 'test-session',
        id: 'test-urlog-id',
        update_time: null,
        abort_controller: new AbortController(),
        force_reget: false,
        locale_name: 'ja',
      }
      await api.get_urlog(req as never)

      expect(globalThis.fetch).toHaveBeenCalledWith(
        '/api/get_urlog',
        expect.objectContaining({
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(req),
        })
      )
    })

    test('get_nlog sends POST to /api/get_nlog with session_id and id', async () => {
      const mockResponse = { nlog_histories: [], messages: [], errors: [] }
      ;(globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue({
        json: () => Promise.resolve(mockResponse),
      })

      const api = GkillAPI.get_instance()
      const req = {
        session_id: 'test-session',
        id: 'test-nlog-id',
        update_time: null,
        abort_controller: new AbortController(),
        force_reget: false,
        locale_name: 'ja',
      }
      await api.get_nlog(req as never)

      expect(globalThis.fetch).toHaveBeenCalledWith(
        '/api/get_nlog',
        expect.objectContaining({
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(req),
        })
      )
    })

    test('get_timeis sends POST to /api/get_timeis with session_id and id', async () => {
      const mockResponse = { timeis_histories: [], messages: [], errors: [] }
      ;(globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue({
        json: () => Promise.resolve(mockResponse),
      })

      const api = GkillAPI.get_instance()
      const req = {
        session_id: 'test-session',
        id: 'test-timeis-id',
        update_time: null,
        abort_controller: new AbortController(),
        force_reget: false,
        locale_name: 'ja',
      }
      await api.get_timeis(req as never)

      expect(globalThis.fetch).toHaveBeenCalledWith(
        '/api/get_timeis',
        expect.objectContaining({
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(req),
        })
      )
    })

    test('get_lantana sends POST to /api/get_lantana with session_id and id', async () => {
      const mockResponse = { lantana_histories: [], messages: [], errors: [] }
      ;(globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue({
        json: () => Promise.resolve(mockResponse),
      })

      const api = GkillAPI.get_instance()
      const req = {
        session_id: 'test-session',
        id: 'test-lantana-id',
        update_time: null,
        abort_controller: new AbortController(),
        force_reget: false,
        locale_name: 'ja',
      }
      await api.get_lantana(req as never)

      expect(globalThis.fetch).toHaveBeenCalledWith(
        '/api/get_lantana',
        expect.objectContaining({
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(req),
        })
      )
    })

    test('get_git_commit_log sends POST to /api/get_git_commit_log with session_id and id', async () => {
      const mockResponse = { git_commit_log_histories: [], messages: [], errors: [] }
      ;(globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue({
        json: () => Promise.resolve(mockResponse),
      })

      const api = GkillAPI.get_instance()
      const req = {
        session_id: 'test-session',
        id: 'test-gcl-id',
        update_time: null,
        abort_controller: new AbortController(),
        force_reget: false,
        locale_name: 'ja',
      }
      await api.get_git_commit_log(req as never)

      expect(globalThis.fetch).toHaveBeenCalledWith(
        '/api/get_git_commit_log',
        expect.objectContaining({
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(req),
        })
      )
    })

    test('get_idf_kyou sends POST to /api/get_idf_kyou with session_id and id', async () => {
      const mockResponse = { idf_kyou_histories: [], messages: [], errors: [] }
      ;(globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue({
        json: () => Promise.resolve(mockResponse),
      })

      const api = GkillAPI.get_instance()
      const req = {
        session_id: 'test-session',
        id: 'test-idf-id',
        update_time: null,
        abort_controller: new AbortController(),
        force_reget: false,
        locale_name: 'ja',
      }
      await api.get_idf_kyou(req as never)

      expect(globalThis.fetch).toHaveBeenCalledWith(
        '/api/get_idf_kyou',
        expect.objectContaining({
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(req),
        })
      )
    })

    test('get_rekyou sends POST to /api/get_rekyou with session_id and id', async () => {
      const mockResponse = { rekyou_histories: [], messages: [], errors: [] }
      ;(globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue({
        json: () => Promise.resolve(mockResponse),
      })

      const api = GkillAPI.get_instance()
      const req = {
        session_id: 'test-session',
        id: 'test-rekyou-id',
        update_time: null,
        abort_controller: new AbortController(),
        force_reget: false,
        locale_name: 'ja',
      }
      await api.get_rekyou(req as never)

      expect(globalThis.fetch).toHaveBeenCalledWith(
        '/api/get_rekyou',
        expect.objectContaining({
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(req),
        })
      )
    })
  })

  describe('history methods', () => {
    const originalFetch = globalThis.fetch

    beforeEach(() => {
      globalThis.fetch = vi.fn()
    })

    afterEach(() => {
      globalThis.fetch = originalFetch
    })

    test('get_tag_histories_by_tag_id sends POST to /api/get_tag_histories_by_tag_id', async () => {
      const mockResponse = { tag_histories: [], messages: [], errors: [] }
      ;(globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue({
        json: () => Promise.resolve(mockResponse),
      })

      const api = GkillAPI.get_instance()
      const req = {
        session_id: 'test-session',
        id: 'test-tag-id',
        abort_controller: new AbortController(),
        force_reget: false,
        locale_name: 'ja',
      }
      await api.get_tag_histories_by_tag_id(req as never)

      expect(globalThis.fetch).toHaveBeenCalledWith(
        '/api/get_tag_histories_by_tag_id',
        expect.objectContaining({
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(req),
        })
      )
    })

    test('get_text_history_by_text_id sends POST to /api/get_text_histories_by_text_id', async () => {
      const mockResponse = { text_histories: [], messages: [], errors: [] }
      ;(globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue({
        json: () => Promise.resolve(mockResponse),
      })

      const api = GkillAPI.get_instance()
      const req = {
        session_id: 'test-session',
        id: 'test-text-id',
        abort_controller: new AbortController(),
        force_reget: false,
        locale_name: 'ja',
      }
      await api.get_text_history_by_text_id(req as never)

      expect(globalThis.fetch).toHaveBeenCalledWith(
        '/api/get_text_histories_by_text_id',
        expect.objectContaining({
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(req),
        })
      )
    })

    test('get_notification_history_by_notification_id sends POST to /api/get_gkill_notification_histories_by_notification_id', async () => {
      const mockResponse = { notification_histories: [], messages: [], errors: [] }
      ;(globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue({
        json: () => Promise.resolve(mockResponse),
      })

      const api = GkillAPI.get_instance()
      const req = {
        session_id: 'test-session',
        id: 'test-notif-id',
        abort_controller: new AbortController(),
        force_reget: false,
        locale_name: 'ja',
      }
      await api.get_notification_history_by_notification_id(req as never)

      expect(globalThis.fetch).toHaveBeenCalledWith(
        '/api/get_gkill_notification_histories_by_notification_id',
        expect.objectContaining({
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(req),
        })
      )
    })
  })

  describe('server config and application config methods', () => {
    const originalFetch = globalThis.fetch

    beforeEach(() => {
      globalThis.fetch = vi.fn()
    })

    afterEach(() => {
      globalThis.fetch = originalFetch
    })

    test('get_server_configs sends POST to /api/get_server_configs', async () => {
      const mockResponse = { server_configs: [], messages: [], errors: [] }
      ;(globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue({
        json: () => Promise.resolve(mockResponse),
      })

      const api = GkillAPI.get_instance()
      const req = {
        session_id: 'test-session',
        abort_controller: new AbortController(),
        force_reget: false,
        locale_name: 'ja',
      }
      await api.get_server_configs(req as never)

      expect(globalThis.fetch).toHaveBeenCalledWith(
        '/api/get_server_configs',
        expect.objectContaining({
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(req),
        })
      )
    })

    test('update_application_config sends POST to /api/update_application_config', async () => {
      const mockResponse = { messages: [], errors: [] }
      ;(globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue({
        json: () => Promise.resolve(mockResponse),
      })

      const api = GkillAPI.get_instance()
      const req = {
        session_id: 'test-session',
        application_config: { use_dark_theme: false },
        abort_controller: new AbortController(),
        force_reget: false,
        locale_name: 'ja',
      }
      await api.update_application_config(req as never)

      expect(globalThis.fetch).toHaveBeenCalledWith(
        '/api/update_application_config',
        expect.objectContaining({
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(req),
        })
      )
    })

    test('update_user_reps sends POST to /api/update_user_reps', async () => {
      const mockResponse = { messages: [], errors: [] }
      ;(globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue({
        json: () => Promise.resolve(mockResponse),
      })

      const api = GkillAPI.get_instance()
      const req = {
        session_id: 'test-session',
        user_reps: [],
        abort_controller: new AbortController(),
        force_reget: false,
        locale_name: 'ja',
      }
      await api.update_user_reps(req as never)

      expect(globalThis.fetch).toHaveBeenCalledWith(
        '/api/update_user_reps',
        expect.objectContaining({
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(req),
        })
      )
    })

    test('update_server_config sends POST to /api/update_server_configs', async () => {
      const mockResponse = { messages: [], errors: [] }
      ;(globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue({
        json: () => Promise.resolve(mockResponse),
      })

      const api = GkillAPI.get_instance()
      const req = {
        session_id: 'test-session',
        server_configs: [],
        abort_controller: new AbortController(),
        force_reget: false,
        locale_name: 'ja',
      }
      await api.update_server_config(req as never)

      expect(globalThis.fetch).toHaveBeenCalledWith(
        '/api/update_server_configs',
        expect.objectContaining({
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(req),
        })
      )
    })
  })

  describe('repository and upload methods', () => {
    const originalFetch = globalThis.fetch

    beforeEach(() => {
      globalThis.fetch = vi.fn()
    })

    afterEach(() => {
      globalThis.fetch = originalFetch
    })

    test('get_repositories sends POST to /api/get_repositories', async () => {
      const mockResponse = { repositories: [], messages: [], errors: [] }
      ;(globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue({
        json: () => Promise.resolve(mockResponse),
      })

      const api = GkillAPI.get_instance()
      const req = {
        session_id: 'test-session',
        abort_controller: new AbortController(),
        force_reget: false,
        locale_name: 'ja',
      }
      await api.get_repositories(req as never)

      expect(globalThis.fetch).toHaveBeenCalledWith(
        '/api/get_repositories',
        expect.objectContaining({
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(req),
        })
      )
    })

    test('upload_files sends POST to /api/upload_files', async () => {
      const mockResponse = { uploaded_kyous: [], messages: [], errors: [] }
      ;(globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue({
        json: () => Promise.resolve(mockResponse),
      })

      const api = GkillAPI.get_instance()
      const req = {
        session_id: 'test-session',
        files: [],
        abort_controller: new AbortController(),
        force_reget: false,
        locale_name: 'ja',
      }
      await api.upload_files(req as never)

      expect(globalThis.fetch).toHaveBeenCalledWith(
        '/api/upload_files',
        expect.objectContaining({
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(req),
        })
      )
    })
  })

  describe('share methods', () => {
    const originalFetch = globalThis.fetch

    beforeEach(() => {
      globalThis.fetch = vi.fn()
    })

    afterEach(() => {
      globalThis.fetch = originalFetch
    })

    test('get_share_kyou_list_infos sends POST to /api/get_share_kyou_list_infos', async () => {
      const mockResponse = { share_kyou_list_infos: [], messages: [], errors: [] }
      ;(globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue({
        json: () => Promise.resolve(mockResponse),
      })

      const api = GkillAPI.get_instance()
      const req = {
        session_id: 'test-session',
        abort_controller: new AbortController(),
        force_reget: false,
        locale_name: 'ja',
      }
      await api.get_share_kyou_list_infos(req as never)

      expect(globalThis.fetch).toHaveBeenCalledWith(
        '/api/get_share_kyou_list_infos',
        expect.objectContaining({
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(req),
        })
      )
    })

    test('add_share_kyou_list_info sends POST to /api/add_share_kyou_list_info', async () => {
      const mockResponse = { share_kyou_list_info: {}, messages: [], errors: [] }
      ;(globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue({
        json: () => Promise.resolve(mockResponse),
      })

      const api = GkillAPI.get_instance()
      const share_kyou_list_info = makeShareKyousInfo()
      const req = {
        session_id: 'test-session',
        share_kyou_list_info,
        abort_controller: new AbortController(),
        force_reget: false,
        locale_name: 'ja',
      }
      await api.add_share_kyou_list_info(req as never)

      expect(globalThis.fetch).toHaveBeenCalledWith(
        '/api/add_share_kyou_list_info',
        expect.objectContaining({
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(req),
        })
      )
    })

    test('delete_share_kyou_list_infos sends POST to /api/delete_share_kyou_list_infos', async () => {
      const mockResponse = { messages: [], errors: [] }
      ;(globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue({
        json: () => Promise.resolve(mockResponse),
      })

      const api = GkillAPI.get_instance()
      const req = {
        session_id: 'test-session',
        share_kyou_list_infos: [makeShareKyousInfo()],
        abort_controller: new AbortController(),
        force_reget: false,
        locale_name: 'ja',
      }
      await api.delete_share_kyou_list_infos(req as never)

      expect(globalThis.fetch).toHaveBeenCalledWith(
        '/api/delete_share_kyou_list_infos',
        expect.objectContaining({
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(req),
        })
      )
    })
  })

  describe('transaction methods', () => {
    const originalFetch = globalThis.fetch

    beforeEach(() => {
      globalThis.fetch = vi.fn()
    })

    afterEach(() => {
      globalThis.fetch = originalFetch
    })

    test('commit_tx sends POST to /api/commit_tx', async () => {
      const mockResponse = { messages: [], errors: [] }
      ;(globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue({
        json: () => Promise.resolve(mockResponse),
      })

      const api = GkillAPI.get_instance()
      const req = {
        session_id: 'test-session',
        tx_id: 'test-tx-id',
        abort_controller: new AbortController(),
        force_reget: false,
        locale_name: 'ja',
      }
      await api.commit_tx(req as never)

      expect(globalThis.fetch).toHaveBeenCalledWith(
        '/api/commit_tx',
        expect.objectContaining({
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(req),
        })
      )
    })

    test('discard_tx sends POST to /api/discard_tx', async () => {
      const mockResponse = { messages: [], errors: [] }
      ;(globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue({
        json: () => Promise.resolve(mockResponse),
      })

      const api = GkillAPI.get_instance()
      const req = {
        session_id: 'test-session',
        tx_id: 'test-tx-id',
        abort_controller: new AbortController(),
        force_reget: false,
        locale_name: 'ja',
      }
      await api.discard_tx(req as never)

      expect(globalThis.fetch).toHaveBeenCalledWith(
        '/api/discard_tx',
        expect.objectContaining({
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(req),
        })
      )
    })
  })

  describe('notification registration methods', () => {
    const originalFetch = globalThis.fetch

    beforeEach(() => {
      globalThis.fetch = vi.fn()
    })

    afterEach(() => {
      globalThis.fetch = originalFetch
    })

    test('get_gkill_notification_public_key sends POST to /api/get_gkill_notification_public_key', async () => {
      const mockResponse = { public_key: '', messages: [], errors: [] }
      ;(globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue({
        json: () => Promise.resolve(mockResponse),
      })

      const api = GkillAPI.get_instance()
      const req = {
        session_id: 'test-session',
        abort_controller: new AbortController(),
        force_reget: false,
        locale_name: 'ja',
      }
      await api.get_gkill_notification_public_key(req as never)

      expect(globalThis.fetch).toHaveBeenCalledWith(
        '/api/get_gkill_notification_public_key',
        expect.objectContaining({
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(req),
        })
      )
    })

    test('register_gkill_notification sends POST to /api/register_gkill_notification', async () => {
      const mockResponse = { messages: [], errors: [] }
      ;(globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue({
        json: () => Promise.resolve(mockResponse),
      })

      const api = GkillAPI.get_instance()
      const req = {
        session_id: 'test-session',
        subscription: {},
        abort_controller: new AbortController(),
        force_reget: false,
        locale_name: 'ja',
      }
      await api.register_gkill_notification(req as never)

      expect(globalThis.fetch).toHaveBeenCalledWith(
        '/api/register_gkill_notification',
        expect.objectContaining({
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(req),
        })
      )
    })
  })

  describe('additional endpoint addresses', () => {
    const api = GkillAPI.get_instance()

    test('get_kc_address is /api/get_kc', () => {
      expect(api.get_kc_address).toBe('/api/get_kc')
    })

    test('get_urlog_address is /api/get_urlog', () => {
      expect(api.get_urlog_address).toBe('/api/get_urlog')
    })

    test('get_nlog_address is /api/get_nlog', () => {
      expect(api.get_nlog_address).toBe('/api/get_nlog')
    })

    test('get_timeis_address is /api/get_timeis', () => {
      expect(api.get_timeis_address).toBe('/api/get_timeis')
    })

    test('get_lantana_address is /api/get_lantana', () => {
      expect(api.get_lantana_address).toBe('/api/get_lantana')
    })

    test('get_rekyou_address is /api/get_rekyou', () => {
      expect(api.get_rekyou_address).toBe('/api/get_rekyou')
    })

    test('get_git_commit_log_address is /api/get_git_commit_log', () => {
      expect(api.get_git_commit_log_address).toBe('/api/get_git_commit_log')
    })

    test('get_idf_kyou_address is /api/get_idf_kyou', () => {
      expect(api.get_idf_kyou_address).toBe('/api/get_idf_kyou')
    })

    test('get_tag_histories_by_tag_id_address is /api/get_tag_histories_by_tag_id', () => {
      expect(api.get_tag_histories_by_tag_id_address).toBe('/api/get_tag_histories_by_tag_id')
    })

    test('get_text_histories_by_text_id_address is /api/get_text_histories_by_text_id', () => {
      expect(api.get_text_histories_by_text_id_address).toBe('/api/get_text_histories_by_text_id')
    })

    test('get_notification_histories_by_notification_id_address is /api/get_gkill_notification_histories_by_notification_id', () => {
      expect(api.get_notification_histories_by_notification_id_address).toBe('/api/get_gkill_notification_histories_by_notification_id')
    })

    test('get_server_configs_address is /api/get_server_configs', () => {
      expect(api.get_server_configs_address).toBe('/api/get_server_configs')
    })

    test('update_application_config_address is /api/update_application_config', () => {
      expect(api.update_application_config_address).toBe('/api/update_application_config')
    })

    test('update_user_reps_address is /api/update_user_reps', () => {
      expect(api.update_user_reps_address).toBe('/api/update_user_reps')
    })

    test('update_server_configs_address is /api/update_server_configs', () => {
      expect(api.update_server_configs_address).toBe('/api/update_server_configs')
    })

    test('upload_files_address is /api/upload_files', () => {
      expect(api.upload_files_address).toBe('/api/upload_files')
    })

    test('get_repositories_address is /api/get_repositories', () => {
      expect(api.get_repositories_address).toBe('/api/get_repositories')
    })

    test('get_share_kyou_list_infos_address is /api/get_share_kyou_list_infos', () => {
      expect(api.get_share_kyou_list_infos_address).toBe('/api/get_share_kyou_list_infos')
    })

    test('add_share_kyou_list_info_address is /api/add_share_kyou_list_info', () => {
      expect(api.add_share_kyou_list_info_address).toBe('/api/add_share_kyou_list_info')
    })

    test('delete_share_kyou_list_infos_address is /api/delete_share_kyou_list_infos', () => {
      expect(api.delete_share_kyou_list_infos_address).toBe('/api/delete_share_kyou_list_infos')
    })

    test('commit_tx_address is /api/commit_tx', () => {
      expect(api.commit_tx_address).toBe('/api/commit_tx')
    })

    test('discard_tx_address is /api/discard_tx', () => {
      expect(api.discard_tx_address).toBe('/api/discard_tx')
    })

    test('get_gkill_notification_public_key_address is /api/get_gkill_notification_public_key', () => {
      expect(api.get_gkill_notification_public_key_address).toBe('/api/get_gkill_notification_public_key')
    })

    test('register_gkill_notification_address is /api/register_gkill_notification', () => {
      expect(api.register_gkill_notification_address).toBe('/api/register_gkill_notification')
    })
  })

  describe('additional method existence checks', () => {
    const api = GkillAPI.get_instance()

    test.each([
      'get_kc',
      'get_urlog',
      'get_nlog',
      'get_timeis',
      'get_lantana',
      'get_git_commit_log',
      'get_idf_kyou',
      'get_rekyou',
      'get_tag_histories_by_tag_id',
      'get_text_history_by_text_id',
      'get_notification_history_by_notification_id',
      'get_server_configs',
      'update_application_config',
      'update_user_reps',
      'update_server_config',
      'get_repositories',
      'upload_files',
      'get_share_kyou_list_infos',
      'add_share_kyou_list_info',
      'delete_share_kyou_list_infos',
      'commit_tx',
      'discard_tx',
      'get_gkill_notification_public_key',
      'register_gkill_notification',
    ])('%s is a function', (method) => {
      expect(typeof (api as Record<string, unknown>)[method]).toBe('function')
    })
  })
})
