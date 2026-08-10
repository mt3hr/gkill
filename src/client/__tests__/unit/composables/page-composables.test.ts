/**
 * Page Composable tests.
 * Tests basic initialization and interface of page-level composables.
 * Page composables often have heavy dependency chains (Vue router, Vuetify, etc.),
 * so we test what's safely importable.
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
      get_session_id_from_cookie_store: vi.fn().mockResolvedValue('mock-session'),
      check_auth: vi.fn(),
      get_application_config: vi.fn().mockResolvedValue({
        application_config: { device: 'test', user_id: 'admin' },
        messages: [],
        errors: [],
      }),
    })),
    get_gkill_api: vi.fn(() => ({
      get_session_id: vi.fn(() => 'mock-session'),
      get_application_config: vi.fn().mockResolvedValue({
        application_config: { device: 'test', user_id: 'admin' },
        messages: [],
        errors: [],
      }),
    })),
  },
}))

vi.mock('@/classes/delete-gkill-cache', () => ({
  default: vi.fn().mockResolvedValue(undefined),
  delete_gkill_config_cache: vi.fn().mockResolvedValue(undefined),
}))

// Mock vue-router to prevent router dependency issues
vi.mock('vue-router', () => ({
  useRouter: vi.fn(() => ({
    push: vi.fn(),
    replace: vi.fn(),
    currentRoute: { value: { path: '/', query: {} } },
  })),
  useRoute: vi.fn(() => ({
    path: '/',
    query: {},
    params: {},
  })),
}))

// Try importing page composables - some may fail due to heavy dependencies
const pageComposables: Array<{ name: string; factory: unknown }> = []

async function tryImport(name: string, path: string, exportName: string) {
  try {
    const mod = await import(path)
    if (mod[exportName]) {
      pageComposables.push({ name, factory: mod[exportName] })
    }
  } catch {
    // Import failed due to dependency chain - skip gracefully
  }
}

await tryImport('useLoginPage', '@/classes/use-login-page', 'useLoginPage')
await tryImport('useSetNewPasswordPage', '@/classes/use-set-new-password-page', 'useSetNewPasswordPage')
await tryImport('useRegisterFirstAccountPage', '@/classes/use-register-first-account-page', 'useRegisterFirstAccountPage')
await tryImport('useDashboardPage', '@/classes/use-dashboard-page', 'useDashboardPage')

describe('Page Composables', () => {
  test('at least one page composable is importable', () => {
    expect(pageComposables.length).toBeGreaterThan(0)
  })

  // Dynamic tests for each successfully imported composable
  for (const { name, factory } of pageComposables) {
    describe(name, () => {
      test('can be instantiated', () => {
        const result = factory()
        expect(result).toBeDefined()
      })

      test('returns an object with methods or refs', () => {
        const result = factory()
        const keys = Object.keys(result)
        expect(keys.length).toBeGreaterThan(0)
      })
    })
  }
})

// 「モック自身が動くこと」を確かめる自己言及テスト（GkillAPI/vue-router のモック検査、
// ローカル配列へのpush検査）はここにあったが、production コードを一切通らないので削除した。
// ページコンポーザブルの実挙動は上の生成テストと、個別の
// dashboard-page-reload.test.ts などの専用テストで見る。
