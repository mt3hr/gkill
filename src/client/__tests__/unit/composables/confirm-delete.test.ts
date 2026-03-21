/**
 * Confirm Delete Composable tests.
 */
import { vi } from 'vitest'

vi.mock('@/i18n', () => ({
  default: { global: { t: (key: string) => key, locale: 'ja' } },
  i18n: { global: { t: (key: string) => key, locale: 'ja' } },
}))

vi.mock('@/classes/api/gkill-api', () => ({
  GkillAPI: {
    get_instance: vi.fn(() => ({
      get_session_id: vi.fn(() => 'mock-session'),
      generate_uuid: vi.fn(() => 'mock-uuid'),
    })),
    get_gkill_api: vi.fn(() => ({
      get_session_id: vi.fn(() => 'mock-session'),
    })),
  },
}))

vi.mock('@/classes/delete-gkill-cache', () => ({
  default: vi.fn().mockResolvedValue(undefined),
  delete_gkill_config_cache: vi.fn().mockResolvedValue(undefined),
}))

import { createMockGkillAPI } from '../../helpers/mock-api'
import { makeKyouWithKmemo } from '../../helpers/factory'

// Dynamic import to test which confirm-delete composables exist
let useConfirmDeleteKyouView: any
let useConfirmDeleteTagView: any

try {
  const mod1 = await import('@/classes/use-confirm-delete-kyou-view')
  useConfirmDeleteKyouView = mod1.useConfirmDeleteKyouView
} catch { /* may not exist */ }

try {
  const mod2 = await import('@/classes/use-confirm-delete-tag-view')
  useConfirmDeleteTagView = mod2.useConfirmDeleteTagView
} catch { /* may not exist */ }

function createMockProps() {
  return {
    gkill_api: createMockGkillAPI() as any,
    application_config: {
      device: 'test-device',
      user_id: 'admin',
    } as any,
    kyou: {
      ...makeKyouWithKmemo('テスト'),
      clone: () => ({ ...makeKyouWithKmemo('テスト') }),
    } as any,
    highlight_targets: [],
    enable_context_menu: true,
    enable_dialog: true,
  }
}

describe('Confirm Delete Composables', () => {
  test('useConfirmDeleteKyouView is importable and has delete_kyou method', () => {
    expect(useConfirmDeleteKyouView).toBeDefined()
    const props = createMockProps()
    const emits = vi.fn()
    const view = useConfirmDeleteKyouView({ props, emits })
    expect(typeof view.delete_kyou).toBe('function')
  })

  test('useConfirmDeleteTagView is importable and has delete_tag method', () => {
    expect(useConfirmDeleteTagView).toBeDefined()
    const props = createMockProps()
    const emits = vi.fn()
    const view = useConfirmDeleteTagView({ props, emits })
    expect(typeof view.delete_tag).toBe('function')
  })

  test('mock API has delete methods', () => {
    const api = createMockGkillAPI()
    expect(typeof api.delete_kmemo).toBe('function')
    expect(typeof api.delete_mi).toBe('function')
    expect(typeof api.delete_tag).toBe('function')
    expect(typeof api.delete_text).toBe('function')
  })

  test('mock API delete methods return expected structure', async () => {
    const api = createMockGkillAPI()
    const result = await api.delete_kmemo()
    expect(result).toHaveProperty('messages')
    expect(result).toHaveProperty('errors')
  })
})
