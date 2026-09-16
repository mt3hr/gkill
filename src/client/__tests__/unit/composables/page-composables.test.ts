/**
 * ページ用コンポーザブルの生成テスト。
 *
 * 以前は動的 import を try/catch で包み、import に失敗したコンポーザブルを黙ってスキップしていた
 * （「1本でも import できれば緑」）。依存の循環や壊れた import が起きても検出できない形だったので、
 * 4本を静的に import して、生成できることと返り値の形をそれぞれ固定する。
 * ページの実挙動は dashboard-page-reload.test.ts などの専用テストが見る。
 * useDashboardPage は vuetify の useTheme と router を引き込む（router は全ページ→CSS まで辿る）ので、
 * dashboard-page-reload.test.ts と同じ形で差し替える。
 */
import { describe, expect, test, vi } from 'vitest'

vi.mock('vuetify', () => ({
  useTheme: () => ({ global: { name: { value: 'gkill_theme' } } }),
}))

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
      set_use_dark_theme: vi.fn(),
      set_saved_application_config: vi.fn(),
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
  delete_gkill_all_tag_names_cache: vi.fn().mockResolvedValue(undefined),
  delete_gkill_attached_datas_cache: vi.fn().mockResolvedValue(undefined),
}))

vi.mock('@/classes/use-dialog-history-stack', () => ({
  reset_dialog_history: vi.fn().mockResolvedValue(undefined),
}))

// router は全ページを引き込むので、ページが使う replace / push だけ差し替える
vi.mock('@/router', () => ({
  default: { replace: vi.fn(), push: vi.fn() },
}))

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

import { useLoginPage } from '@/classes/use-login-page'
import { useSetNewPasswordPage } from '@/classes/use-set-new-password-page'
import { useRegisterFirstAccountPage } from '@/classes/use-register-first-account-page'
import { useDashboardPage } from '@/classes/use-dashboard-page'

const page_composables: Array<[string, () => Record<string, unknown>]> = [
  ['useLoginPage', () => useLoginPage() as unknown as Record<string, unknown>],
  ['useSetNewPasswordPage', () => useSetNewPasswordPage() as unknown as Record<string, unknown>],
  ['useRegisterFirstAccountPage', () => useRegisterFirstAccountPage() as unknown as Record<string, unknown>],
  ['useDashboardPage', () => useDashboardPage() as unknown as Record<string, unknown>],
]

describe('Page Composables', () => {
  test.each(page_composables)('%s は生成でき、state か handler を1つ以上返す', (_name, factory) => {
    const result = factory()
    expect(result).toBeDefined()
    expect(Object.keys(result).length).toBeGreaterThan(0)
  })
})
