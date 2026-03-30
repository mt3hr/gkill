/**
 * Edit View Composable tests.
 * Tests loading, validation, and update behavior.
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
      generate_uuid: vi.fn(() => 'test-uuid'),
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
import { useEditKmemoView } from '@/classes/use-edit-kmemo-view'
import { useEditMiView } from '@/classes/use-edit-mi-view'
import { useEditNlogView } from '@/classes/use-edit-nlog-view'
import { useEditUrLogView } from '@/classes/use-edit-ur-log-view'
import { useEditTimeIsView } from '@/classes/use-edit-time-is-view'
import { useEditLantanaView } from '@/classes/use-edit-lantana-view'
import { useEditKCView } from '@/classes/use-edit-kc-view'

function createMockKyou(data: Record<string, unknown> = {}) {
  const base = {
    id: 'test-kyou-id',
    rep_name: 'test-rep',
    data_type: 'kmemo',
    related_time: new Date('2025-03-15T09:00:00+09:00'),
    is_deleted: false,
    create_time: new Date(),
    create_app: 'gkill',
    create_device: 'test-device',
    create_user: 'admin',
    update_time: new Date(),
    update_app: 'gkill',
    update_device: 'test-device',
    update_user: 'admin',
    typed_kmemo: (() => {
      const km: Record<string, unknown> = {
        content: 'テストメモ',
        id: 'test-kmemo-id',
        related_time: new Date('2025-03-15T09:00:00+09:00'),
        create_time: new Date(),
        create_app: 'gkill',
        create_device: 'test-device',
        create_user: 'admin',
        update_time: new Date(),
        update_app: 'gkill',
        update_device: 'test-device',
        update_user: 'admin',
      }
      km.clone = () => ({ ...km, clone: km.clone })
      return km
    })(),
    typed_mi: null,
    attached_tags: [],
    attached_texts: [],
    attached_notifications: [],
    attached_timeis_kyou: [],
    abort_controller: new AbortController(),
    ...data,
  }
  base.clone = () => ({
    ...base,
    clone: base.clone,
    reload: vi.fn().mockResolvedValue(undefined),
    load_typed_datas: vi.fn().mockResolvedValue(undefined),
    load_all: vi.fn().mockResolvedValue(undefined),
    load_attached_tags: vi.fn().mockResolvedValue(undefined),
    load_attached_texts: vi.fn().mockResolvedValue(undefined),
  })
  base.reload = vi.fn().mockResolvedValue(undefined)
  base.load_typed_datas = vi.fn().mockResolvedValue(undefined)
  base.load_all = vi.fn().mockResolvedValue(undefined)
  base.load_attached_tags = vi.fn().mockResolvedValue(undefined)
  base.load_attached_texts = vi.fn().mockResolvedValue(undefined)
  return base
}

function createBaseProps(kyouData = {}) {
  return {
    gkill_api: createMockGkillAPI() as never,
    application_config: {
      device: 'test-device',
      user_id: 'admin',
      mi_default_board: 'Inbox',
    } as never,
    kyou: createMockKyou(kyouData) as never,
    highlight_targets: [],
    enable_context_menu: true,
    enable_dialog: true,
  }
}

// ========== useEditKmemoView ==========

describe('useEditKmemoView', () => {
  let props: ReturnType<typeof createBaseProps>
  let emits: ReturnType<typeof vi.fn>

  beforeEach(() => {
    vi.clearAllMocks()
    props = createBaseProps()
    emits = vi.fn()
  })

  test('initializes form fields from kyou prop', () => {
    const view = useEditKmemoView({ props, emits })
    expect(view.kmemo_value.value).toBe('テストメモ')
  })

  test('save() emits received_errors when content is blank', async () => {
    const view = useEditKmemoView({ props, emits })
    view.kmemo_value.value = ''
    await view.save()
    const errorCalls = emits.mock.calls.filter((c: unknown[]) => c[0] === 'received_errors')
    expect(errorCalls.length).toBeGreaterThan(0)
  })

  test('save() calls update_kmemo with modified content', async () => {
    props.gkill_api.update_kmemo.mockResolvedValue({
      messages: [{ message_code: 'OK' }],
      errors: [],
      updated_kmemo: { content: '更新メモ' },
    })
    const view = useEditKmemoView({ props, emits })
    view.kmemo_value.value = '更新メモ'
    await view.save()
    expect(props.gkill_api.update_kmemo).toHaveBeenCalled()
  })

  test('save() emits received_errors on API failure', async () => {
    props.gkill_api.update_kmemo.mockResolvedValue({
      messages: [],
      errors: [{ error_code: 'ERR001', error_message: 'failed' }],
    })
    const view = useEditKmemoView({ props, emits })
    view.kmemo_value.value = '更新メモ'
    await view.save()
    const errorCalls = emits.mock.calls.filter((c: unknown[]) => c[0] === 'received_errors')
    expect(errorCalls.length).toBeGreaterThan(0)
  })

  test('returns expected interface', () => {
    const view = useEditKmemoView({ props, emits })
    expect(view.kmemo_value).toBeDefined()
    expect(typeof view.save).toBe('function')
  })
})

// ========== useEditMiView ==========

describe('useEditMiView', () => {
  let props: ReturnType<typeof createBaseProps>
  let emits: ReturnType<typeof vi.fn>

  beforeEach(() => {
    vi.clearAllMocks()
    props = createBaseProps({
      data_type: 'mi',
      typed_mi: (() => {
        const mi: Record<string, unknown> = {
          id: 'test-mi-id',
          title: 'テストタスク',
          board_name: 'Inbox',
          is_checked: false,
          limit_time: null,
          estimate_start_time: null,
          estimate_end_time: null,
          related_time: new Date('2025-03-15T09:00:00+09:00'),
          create_time: new Date(),
          create_app: 'gkill',
          create_device: 'test-device',
          create_user: 'admin',
          update_time: new Date(),
          update_app: 'gkill',
          update_device: 'test-device',
          update_user: 'admin',
        }
        mi.clone = () => ({ ...mi, clone: mi.clone })
        return mi
      })(),
      typed_kmemo: null,
    })
    props.gkill_api.get_mi_board_list.mockResolvedValue({
      boards: ['Inbox', 'Work'],
      messages: [],
      errors: [],
    })
    emits = vi.fn()
  })

  test('initializes fields from mi data', () => {
    const view = useEditMiView({ props, emits })
    expect(view.mi_title.value).toBe('テストタスク')
    expect(view.mi_board_name.value).toBe('Inbox')
  })

  test('save() validates title not blank', async () => {
    const view = useEditMiView({ props, emits })
    view.mi_title.value = ''
    await view.save()
    const errorCalls = emits.mock.calls.filter((c: unknown[]) => c[0] === 'received_errors')
    expect(errorCalls.length).toBeGreaterThan(0)
  })

  test('save() calls update_mi API with modified data', async () => {
    props.gkill_api.update_mi.mockResolvedValue({
      messages: [{ message_code: 'OK' }],
      errors: [],
      updated_mi: { title: '更新タスク' },
    })
    const view = useEditMiView({ props, emits })
    view.mi_title.value = '更新タスク'
    await view.save()
    expect(props.gkill_api.update_mi).toHaveBeenCalled()
  })

  test('returns expected interface', () => {
    const view = useEditMiView({ props, emits })
    expect(view.mi_title).toBeDefined()
    expect(view.mi_board_name).toBeDefined()
    expect(typeof view.save).toBe('function')
  })
})

// ========== Helper for other edit views ==========

function createTypedKyouProps(dataType: string, typedField: string, typedData: Record<string, unknown>) {
  const td = { ...typedData }
  td.clone = () => ({ ...td, clone: td.clone })
  return createBaseProps({
    data_type: dataType,
    [typedField]: td,
    typed_kmemo: typedField === 'typed_kmemo' ? td : null,
    typed_mi: typedField === 'typed_mi' ? td : null,
  })
}

// ========== useEditNlogView ==========

describe('useEditNlogView', () => {
  test('initializes from nlog data', () => {
    const props = createTypedKyouProps('nlog', 'typed_nlog', {
      id: 'nlog-1', shop: 'テスト店', title: 'テスト支出', amount: 500,
      related_time: new Date(), create_time: new Date(), update_time: new Date(),
      create_app: 'gkill', create_device: 'test', create_user: 'admin',
      update_app: 'gkill', update_device: 'test', update_user: 'admin',
    })
    const emits = vi.fn()
    const view = useEditNlogView({ props, emits })
    expect(view.nlog_title_value.value).toBe('テスト支出')
    expect(view.nlog_shop_value.value).toBe('テスト店')
  })

  test('returns expected interface', () => {
    const props = createTypedKyouProps('nlog', 'typed_nlog', {
      id: 'nlog-1', shop: '', title: '', amount: 0,
      related_time: new Date(), create_time: new Date(), update_time: new Date(),
      create_app: 'gkill', create_device: 'test', create_user: 'admin',
      update_app: 'gkill', update_device: 'test', update_user: 'admin',
    })
    const emits = vi.fn()
    const view = useEditNlogView({ props, emits })
    expect(typeof view.save).toBe('function')
  })
})

// ========== useEditUrLogView ==========

describe('useEditUrLogView', () => {
  test('initializes from urlog data', () => {
    const props = createTypedKyouProps('urlog', 'typed_urlog', {
      id: 'urlog-1', url: 'https://example.com', title: 'Example',
      favicon_image: '', thumbnail_image: '',
      related_time: new Date(), create_time: new Date(), update_time: new Date(),
      create_app: 'gkill', create_device: 'test', create_user: 'admin',
      update_app: 'gkill', update_device: 'test', update_user: 'admin',
    })
    const emits = vi.fn()
    const view = useEditUrLogView({ props, emits })
    expect(view.url.value).toBe('https://example.com')
  })

  test('returns expected interface', () => {
    const props = createTypedKyouProps('urlog', 'typed_urlog', {
      id: 'urlog-1', url: '', title: '', favicon_image: '', thumbnail_image: '',
      related_time: new Date(), create_time: new Date(), update_time: new Date(),
      create_app: 'gkill', create_device: 'test', create_user: 'admin',
      update_app: 'gkill', update_device: 'test', update_user: 'admin',
    })
    const emits = vi.fn()
    const view = useEditUrLogView({ props, emits })
    expect(typeof view.save).toBe('function')
  })
})

// ========== useEditTimeIsView ==========

describe('useEditTimeIsView', () => {
  test('initializes from timeis data', () => {
    const props = createTypedKyouProps('timeis', 'typed_timeis', {
      id: 'timeis-1', title: 'テスト時間',
      start_time: new Date('2025-03-15T09:00:00'), end_time: null,
      related_time: new Date(), create_time: new Date(), update_time: new Date(),
      create_app: 'gkill', create_device: 'test', create_user: 'admin',
      update_app: 'gkill', update_device: 'test', update_user: 'admin',
    })
    const emits = vi.fn()
    const view = useEditTimeIsView({ props, emits })
    expect(view.timeis_title.value).toBe('テスト時間')
  })

  test('returns expected interface', () => {
    const props = createTypedKyouProps('timeis', 'typed_timeis', {
      id: 'timeis-1', title: '',
      start_time: new Date(), end_time: null,
      related_time: new Date(), create_time: new Date(), update_time: new Date(),
      create_app: 'gkill', create_device: 'test', create_user: 'admin',
      update_app: 'gkill', update_device: 'test', update_user: 'admin',
    })
    const emits = vi.fn()
    const view = useEditTimeIsView({ props, emits })
    expect(typeof view.save).toBe('function')
  })
})

// ========== useEditLantanaView ==========

describe('useEditLantanaView', () => {
  test('initializes mood from lantana data', () => {
    const props = createTypedKyouProps('lantana', 'typed_lantana', {
      id: 'lantana-1', mood: 7,
      related_time: new Date(), create_time: new Date(), update_time: new Date(),
      create_app: 'gkill', create_device: 'test', create_user: 'admin',
      update_app: 'gkill', update_device: 'test', update_user: 'admin',
    })
    const emits = vi.fn()
    const view = useEditLantanaView({ props, emits })
    expect(view.mood.value).toBe(7)
  })

  test('returns expected interface', () => {
    const props = createTypedKyouProps('lantana', 'typed_lantana', {
      id: 'lantana-1', mood: 0,
      related_time: new Date(), create_time: new Date(), update_time: new Date(),
      create_app: 'gkill', create_device: 'test', create_user: 'admin',
      update_app: 'gkill', update_device: 'test', update_user: 'admin',
    })
    const emits = vi.fn()
    const view = useEditLantanaView({ props, emits })
    expect(typeof view.save).toBe('function')
  })
})

// ========== useEditKCView ==========

describe('useEditKCView', () => {
  test('initializes from kc data', () => {
    const props = createTypedKyouProps('kc', 'typed_kc', {
      id: 'kc-1', title: 'テストKC', num_value: 42,
      related_time: new Date(), create_time: new Date(), update_time: new Date(),
      create_app: 'gkill', create_device: 'test', create_user: 'admin',
      update_app: 'gkill', update_device: 'test', update_user: 'admin',
    })
    const emits = vi.fn()
    const view = useEditKCView({ props, emits })
    expect(view.title.value).toBe('テストKC')
  })

  test('returns expected interface', () => {
    const props = createTypedKyouProps('kc', 'typed_kc', {
      id: 'kc-1', title: '', num_value: 0,
      related_time: new Date(), create_time: new Date(), update_time: new Date(),
      create_app: 'gkill', create_device: 'test', create_user: 'admin',
      update_app: 'gkill', update_device: 'test', update_user: 'admin',
    })
    const emits = vi.fn()
    const view = useEditKCView({ props, emits })
    expect(typeof view.save).toBe('function')
  })
})
