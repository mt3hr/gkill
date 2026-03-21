/**
 * Test data factories.
 *
 * Data model classes (Kyou, Kmemo, Tag, Mi, etc.) have circular import chains
 * that cause "Class extends value undefined" errors in the jsdom test environment.
 * These factories return plain objects matching the expected shapes instead.
 */

export function makeKyou(overrides: Record<string, unknown> = {}) {
  return {
    is_deleted: false,
    id: 'test-kyou-id',
    rep_name: 'test-rep',
    related_time: '2025-03-15T09:00:00+09:00',
    data_type: 'kmemo',
    create_time: '2025-03-15T09:00:00+09:00',
    create_app: 'gkill',
    create_device: 'test-device',
    create_user: 'admin',
    update_time: '2025-03-15T09:00:00+09:00',
    update_app: 'gkill',
    update_device: 'test-device',
    update_user: 'admin',
    image_source: '',
    typed_kmemo: null,
    typed_urlog: null,
    typed_nlog: null,
    typed_timeis: null,
    typed_mi: null,
    typed_lantana: null,
    typed_kc: null,
    typed_idf_kyou: null,
    typed_git_commit_log: null,
    ...overrides,
  }
}

export function makeKmemo(overrides: Record<string, unknown> = {}) {
  return {
    is_deleted: false,
    id: 'test-kmemo-id',
    rep_name: 'test-rep',
    related_time: '2025-03-15T09:00:00+09:00',
    content: 'テストメモ',
    create_time: '2025-03-15T09:00:00+09:00',
    create_app: 'gkill',
    create_device: 'test-device',
    create_user: 'admin',
    update_time: '2025-03-15T09:00:00+09:00',
    update_app: 'gkill',
    update_device: 'test-device',
    update_user: 'admin',
    ...overrides,
  }
}

export function makeTag(overrides: Record<string, unknown> = {}) {
  return {
    is_deleted: false,
    id: 'test-tag-id',
    rep_name: 'test-rep',
    related_time: '2025-03-15T09:00:00+09:00',
    target_id: 'test-target-id',
    tag: 'test-tag',
    create_time: '2025-03-15T09:00:00+09:00',
    create_app: 'gkill',
    create_device: 'test-device',
    create_user: 'admin',
    update_time: '2025-03-15T09:00:00+09:00',
    update_app: 'gkill',
    update_device: 'test-device',
    update_user: 'admin',
    ...overrides,
  }
}

export function makeMi(overrides: Record<string, unknown> = {}) {
  return {
    is_deleted: false,
    id: 'test-mi-id',
    rep_name: 'test-rep',
    related_time: '2025-03-15T09:00:00+09:00',
    title: 'テストタスク',
    is_checked: false,
    board_name: 'default',
    limit_time: null,
    estimate_start_time: null,
    estimate_end_time: null,
    create_time: '2025-03-15T09:00:00+09:00',
    create_app: 'gkill',
    create_device: 'test-device',
    create_user: 'admin',
    update_time: '2025-03-15T09:00:00+09:00',
    update_app: 'gkill',
    update_device: 'test-device',
    update_user: 'admin',
    ...overrides,
  }
}

export function makeText(overrides: Record<string, unknown> = {}) {
  return {
    is_deleted: false,
    id: 'test-text-id',
    rep_name: 'test-rep',
    related_time: '2025-03-15T09:00:00+09:00',
    target_id: 'test-target-id',
    text: 'テストテキスト',
    create_time: '2025-03-15T09:00:00+09:00',
    create_app: 'gkill',
    create_device: 'test-device',
    create_user: 'admin',
    update_time: '2025-03-15T09:00:00+09:00',
    update_app: 'gkill',
    update_device: 'test-device',
    update_user: 'admin',
    ...overrides,
  }
}

export function makeURLog(overrides: Record<string, unknown> = {}) {
  return {
    is_deleted: false,
    id: 'test-urlog-id',
    rep_name: 'test-rep',
    related_time: '2025-03-15T09:00:00+09:00',
    url: 'https://example.com',
    title: 'Example',
    favicon_image: '',
    thumbnail_image: '',
    create_time: '2025-03-15T09:00:00+09:00',
    create_app: 'gkill',
    create_device: 'test-device',
    create_user: 'admin',
    update_time: '2025-03-15T09:00:00+09:00',
    update_app: 'gkill',
    update_device: 'test-device',
    update_user: 'admin',
    ...overrides,
  }
}

export function makeNlog(overrides: Record<string, unknown> = {}) {
  return {
    is_deleted: false,
    id: 'test-nlog-id',
    rep_name: 'test-rep',
    related_time: '2025-03-15T09:00:00+09:00',
    shop: 'テスト店',
    title: 'テスト支出',
    amount: 1000,
    create_time: '2025-03-15T09:00:00+09:00',
    create_app: 'gkill',
    create_device: 'test-device',
    create_user: 'admin',
    update_time: '2025-03-15T09:00:00+09:00',
    update_app: 'gkill',
    update_device: 'test-device',
    update_user: 'admin',
    ...overrides,
  }
}

export function makeLantana(overrides: Record<string, unknown> = {}) {
  return {
    is_deleted: false,
    id: 'test-lantana-id',
    rep_name: 'test-rep',
    related_time: '2025-03-15T09:00:00+09:00',
    mood: 5,
    create_time: '2025-03-15T09:00:00+09:00',
    create_app: 'gkill',
    create_device: 'test-device',
    create_user: 'admin',
    update_time: '2025-03-15T09:00:00+09:00',
    update_app: 'gkill',
    update_device: 'test-device',
    update_user: 'admin',
    ...overrides,
  }
}
