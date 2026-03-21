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

export function makeReKyou(overrides: Record<string, unknown> = {}) {
  return {
    is_deleted: false,
    id: 'test-rekyou-id',
    rep_name: 'test-rep',
    related_time: '2025-03-15T09:00:00+09:00',
    data_type: 're_kyou',
    create_time: '2025-03-15T09:00:00+09:00',
    create_app: 'gkill',
    create_device: 'test-device',
    create_user: 'admin',
    update_time: '2025-03-15T09:00:00+09:00',
    update_app: 'gkill',
    update_device: 'test-device',
    update_user: 'admin',
    target_id: 'test-target-id',
    attached_kyou: null,
    attached_histories: [],
    ...overrides,
  }
}

export function makeInfoBase(overrides: Record<string, unknown> = {}) {
  return {
    is_deleted: false,
    id: 'test-info-id',
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
    attached_tags: [],
    attached_texts: [],
    attached_notifications: [],
    attached_timeis_kyou: [],
    is_checked_kyou: false,
    ...overrides,
  }
}

export function makeMetaInfoBase(overrides: Record<string, unknown> = {}) {
  return {
    is_deleted: false,
    id: 'test-meta-id',
    target_id: 'test-target-id',
    related_time: '2025-03-15T09:00:00+09:00',
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

export function makeTimeis(overrides: Record<string, unknown> = {}) {
  return {
    is_deleted: false,
    id: 'test-timeis-id',
    rep_name: 'test-rep',
    related_time: '2025-03-15T09:00:00+09:00',
    title: 'テストタイムイズ',
    start_time: '2025-03-15T09:00:00+09:00',
    end_time: null,
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

export function makeKc(overrides: Record<string, unknown> = {}) {
  return {
    is_deleted: false,
    id: 'test-kc-id',
    rep_name: 'test-rep',
    related_time: '2025-03-15T09:00:00+09:00',
    title: 'テストKC',
    num_value: 42,
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

export function makeGitCommitLog(overrides: Record<string, unknown> = {}) {
  return {
    is_deleted: false,
    id: 'test-git-commit-log-id',
    rep_name: 'test-rep',
    related_time: '2025-03-15T09:00:00+09:00',
    commit_message: 'test commit',
    addition: 10,
    deletion: 5,
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

/**
 * D-note test helpers: Kyou objects with typed_* fields populated.
 * related_time is a Date object (not string) as required by KeyGetters and Predicates.
 */
export function makeKyouWithKmemo(content = 'テストメモ', overrides: Record<string, unknown> = {}) {
  return {
    ...makeKyou({ data_type: 'kmemo', related_time: new Date('2025-03-15T09:00:00+09:00') }),
    typed_kmemo: makeKmemo({ content }),
    attached_tags: [],
    attached_texts: [],
    ...overrides,
  }
}

export function makeKyouWithNlog(shop = 'テスト店', title = 'テスト支出', amount = 1000, overrides: Record<string, unknown> = {}) {
  return {
    ...makeKyou({ data_type: 'nlog', related_time: new Date('2025-03-15T09:00:00+09:00') }),
    typed_nlog: makeNlog({ shop, title, amount }),
    attached_tags: [],
    attached_texts: [],
    ...overrides,
  }
}

export function makeKyouWithLantana(mood = 5, overrides: Record<string, unknown> = {}) {
  return {
    ...makeKyou({ data_type: 'lantana', related_time: new Date('2025-03-15T09:00:00+09:00') }),
    typed_lantana: makeLantana({ mood }),
    attached_tags: [],
    attached_texts: [],
    ...overrides,
  }
}

export function makeKyouWithKc(title = 'テストKC', numValue = 42, overrides: Record<string, unknown> = {}) {
  return {
    ...makeKyou({ data_type: 'kc', related_time: new Date('2025-03-15T09:00:00+09:00') }),
    typed_kc: makeKc({ title, num_value: numValue }),
    attached_tags: [],
    attached_texts: [],
    ...overrides,
  }
}

export function makeKyouWithMi(title = 'テストタスク', overrides: Record<string, unknown> = {}) {
  return {
    ...makeKyou({ data_type: 'mi', related_time: new Date('2025-03-15T09:00:00+09:00') }),
    typed_mi: makeMi({ title }),
    attached_tags: [],
    attached_texts: [],
    ...overrides,
  }
}

export function makeKyouWithTimeis(title = 'テストタイムイズ', overrides: Record<string, unknown> = {}) {
  return {
    ...makeKyou({ data_type: 'timeis', related_time: new Date('2025-03-15T09:00:00+09:00') }),
    typed_timeis: makeTimeis({ title }),
    attached_tags: [],
    attached_texts: [],
    ...overrides,
  }
}

export function makeKyouWithTags(tags: string[], overrides: Record<string, unknown> = {}) {
  return {
    ...makeKyou({ related_time: new Date('2025-03-15T09:00:00+09:00') }),
    attached_tags: tags.map((t, i) => makeTag({ id: `tag-${i}`, tag: t })),
    attached_texts: [],
    ...overrides,
  }
}

export function makeKyouWithGitCommitLog(message = 'test commit', addition = 10, deletion = 5, overrides: Record<string, unknown> = {}) {
  return {
    ...makeKyou({ data_type: 'git_commit_log', related_time: new Date('2025-03-15T09:00:00+09:00') }),
    typed_git_commit_log: makeGitCommitLog({ commit_message: message, addition, deletion }),
    attached_tags: [],
    attached_texts: [],
    ...overrides,
  }
}

export function makeShareKyousInfo(overrides: Record<string, unknown> = {}) {
  return {
    share_id: 'test-share-id',
    user_id: 'admin',
    device: 'test-device',
    share_title: 'Test Share',
    find_query_json: {},
    view_type: 'rykv',
    is_share_time_only: false,
    is_share_with_tags: false,
    is_share_with_texts: false,
    is_share_with_timeiss: false,
    is_share_with_locations: false,
    ...overrides,
  }
}
