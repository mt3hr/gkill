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
vi.mock('@/pages/plaing-timeis-page.vue', () => ({ default: { name: 'plaing-timeis-page' } }))
vi.mock('@/pages/mkfl-page.vue', () => ({ default: { name: 'mkfl-page' } }))
vi.mock('@/pages/regist-first-account-page.vue', () => ({ default: { name: 'regist-first-account-page' } }))

import router from '@/router/index'

describe('router', () => {
  const routes = router.getRoutes()

  test('defines exactly 12 routes', () => {
    expect(routes.length).toBe(12)
  })

  test('all route names match expected set', () => {
    const names = routes.map(r => r.name).sort()
    const expected = [
      'kftl', 'kyou', 'login', 'mi', 'mkfl', 'plaing',
      'regist_first_account', 'rykv', 'saihate',
      'set_new_password', 'shared_mi', 'shared_page',
    ].sort()
    expect(names).toEqual(expected)
  })

  test('login route is at path /', () => {
    const login = routes.find(r => r.name === 'login')
    expect(login).toBeDefined()
    expect(login!.path).toBe('/')
  })

  test('all routes have components defined', () => {
    for (const route of routes) {
      expect(route.components).toBeDefined()
    }
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
