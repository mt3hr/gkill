import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest'
import { i18n } from '../../helpers/setup-i18n'

// Mock @/i18n before importing GkillAPI
vi.mock('@/i18n', () => ({ i18n }))

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
      await api.login(req as any)

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
      const result = await api.login({ user_id: 'u', password_sha256: 'p' } as any)
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
      expect(typeof (api as any)[method]).toBe('function')
    })
  })
})
