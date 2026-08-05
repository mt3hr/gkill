/**
 * Router configuration tests.
 * We mock Vue component imports to avoid CSS/Vuetify resolution issues in jsdom.
 */
import { vi } from 'vitest'

// Mock all page components to avoid CSS/Vuetify import chain
vi.mock('@/pages/login-page.vue', () => ({ default: { name: 'login-page' } }))
vi.mock('@/pages/kftl-page.vue', () => ({ default: { name: 'kftl-page' } }))
vi.mock('@/pages/mi-page.vue', () => ({ default: { name: 'mi-page' } }))
vi.mock('@/pages/rykv-page.vue', () => ({ default: { name: 'rykv-page' } }))
vi.mock('@/pages/kyou-page.vue', () => ({ default: { name: 'kyou-page' } }))
vi.mock('@/pages/saihate-page.vue', () => ({ default: { name: 'saihate-page' } }))
vi.mock('@/pages/set-new-password-page.vue', () => ({ default: { name: 'set-new-password-page' } }))
vi.mock('@/pages/shared-page.vue', () => ({ default: { name: 'shared-page' } }))
vi.mock('@/pages/old-shared-mi-page.vue', () => ({ default: { name: 'old-shared-mi-page' } }))
vi.mock('@/pages/plaing-time-is-page.vue', () => ({ default: { name: 'plaing-time-is-page' } }))
vi.mock('@/pages/mkfl-page.vue', () => ({ default: { name: 'mkfl-page' } }))
vi.mock('@/pages/register-first-account-page.vue', () => ({ default: { name: 'register-first-account-page' } }))
vi.mock('@/pages/dashboard-page.vue', () => ({ default: { name: 'dashboard-page' } }))

import router from '@/router/index'

describe('router', () => {
  const routes = router.getRoutes()
  // リダイレクト専用ルート（コンポーネントを持たない旧パス）は画面ルートと分けて数える
  const page_routes = routes.filter(r => r.components)
  const redirect_routes = routes.filter(r => !r.components)

  test('defines exactly 13 page routes', () => {
    expect(page_routes.length).toBe(13)
  })

  test('all route names match expected set', () => {
    const names = page_routes.map(r => r.name).sort()
    const expected = [
      'dashboard', 'kftl', 'kyou', 'login', 'mi', 'mkfl', 'plaing',
      'register_first_account', 'rykv', 'saihate',
      'set_new_password', 'shared_mi', 'shared_page',
    ].sort()
    expect(names).toEqual(expected)
  })

  test('login route is at path /', () => {
    const login = routes.find(r => r.name === 'login')
    expect(login).toBeDefined()
    expect(login!.path).toBe('/')
  })

  test('old regist_first_account path redirects to register_first_account', () => {
    const legacy = redirect_routes.find(r => r.path === '/regist_first_account')
    expect(legacy).toBeDefined()
    expect(typeof legacy!.redirect).toBe('function')
    // reset_token クエリを落とすと初回セットアップが通らない
    const redirect = legacy!.redirect as (to: { query: Record<string, string> }) => { path: string, query: Record<string, string> }
    const target = redirect({ query: { reset_token: 'tkn' } })
    expect(target.path).toBe('/register_first_account')
    expect(target.query.reset_token).toBe('tkn')
  })

  test('no duplicate paths', () => {
    const paths = routes.map(r => r.path)
    expect(new Set(paths).size).toBe(paths.length)
  })

  test('no duplicate names', () => {
    const names = routes.map(r => r.name).filter(Boolean)
    expect(new Set(names).size).toBe(names.length)
  })
})
