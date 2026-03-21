/**
 * Add View Composable tests.
 * Tests validation logic and API call behavior for add operations.
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
      generate_uuid: vi.fn(() => 'test-uuid-' + Math.random().toString(36).slice(2, 8)),
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
import { useAddMiView } from '@/classes/use-add-mi-view'
import { useAddTagView } from '@/classes/use-add-tag-view'
import { useAddNlogView } from '@/classes/use-add-nlog-view'
import { useAddUrlogView } from '@/classes/use-add-urlog-view'
import { useAddLantanaView } from '@/classes/use-add-lantana-view'
import { useAddTimeisView } from '@/classes/use-add-timeis-view'
import { useAddKcView } from '@/classes/use-add-kc-view'

function createBaseProps() {
  return {
    gkill_api: createMockGkillAPI() as any,
    application_config: {
      device: 'test-device',
      user_id: 'admin',
      mi_default_board: 'Inbox',
      tag_struct: { children: [] },
    } as any,
  }
}

// ========== useAddMiView ==========

describe('useAddMiView', () => {
  let props: ReturnType<typeof createBaseProps>
  let emits: ReturnType<typeof vi.fn>

  beforeEach(() => {
    vi.clearAllMocks()
    props = createBaseProps()
    props.gkill_api.get_mi_board_list.mockResolvedValue({
      boards: ['Inbox', 'Work'],
      messages: [],
      errors: [],
    })
    emits = vi.fn()
  })

  test('initializes with default board from application_config', () => {
    const view = useAddMiView({ props, emits })
    expect(view.mi_board_name.value).toBe('Inbox')
  })

  test('initializes with empty title', () => {
    const view = useAddMiView({ props, emits })
    expect(view.mi_title.value).toBe('')
  })

  test('save() emits received_errors when title is blank', async () => {
    const view = useAddMiView({ props, emits })
    view.mi_title.value = ''
    await view.save()
    const errorCalls = emits.mock.calls.filter((c: any[]) => c[0] === 'received_errors')
    expect(errorCalls.length).toBeGreaterThan(0)
  })

  test('save() calls add_mi API on valid input', async () => {
    props.gkill_api.add_mi.mockResolvedValue({
      messages: [{ message_code: 'OK' }],
      errors: [],
      added_mi: { id: 'new-id' },
    })
    const view = useAddMiView({ props, emits })
    view.mi_title.value = 'テストタスク'
    await view.save()
    expect(props.gkill_api.add_mi).toHaveBeenCalled()
  })

  test('reset() clears title', () => {
    const view = useAddMiView({ props, emits })
    view.mi_title.value = 'something'
    view.reset()
    expect(view.mi_title.value).toBe('')
  })

  test('load_mi_board_names() calls get_mi_board_list API', async () => {
    const view = useAddMiView({ props, emits })
    await view.load_mi_board_names()
    expect(props.gkill_api.get_mi_board_list).toHaveBeenCalled()
  })

  test('returns expected interface', () => {
    const view = useAddMiView({ props, emits })
    expect(view.mi_title).toBeDefined()
    expect(view.mi_board_name).toBeDefined()
    expect(typeof view.save).toBe('function')
    expect(typeof view.reset).toBe('function')
    expect(typeof view.load_mi_board_names).toBe('function')
  })
})

// ========== useAddTagView ==========

describe('useAddTagView', () => {
  let props: any
  let emits: ReturnType<typeof vi.fn>

  beforeEach(() => {
    vi.clearAllMocks()
    const base = createBaseProps()
    props = {
      ...base,
      kyou: {
        id: 'target-kyou-id',
        related_time: new Date(),
        clone: () => ({ id: 'target-kyou-id', related_time: new Date() }),
      },
    }
    emits = vi.fn()
  })

  test('initializes with empty tag name', () => {
    const view = useAddTagView({ props, emits })
    expect(view.tag_name.value).toBe('')
  })

  test('save() emits received_errors when tag text empty', async () => {
    const view = useAddTagView({ props, emits })
    view.tag_name.value = ''
    await view.save()
    const errorCalls = emits.mock.calls.filter((c: any[]) => c[0] === 'received_errors')
    expect(errorCalls.length).toBeGreaterThan(0)
  })

  test('save() with valid input does not emit errors (may show confirmation dialog)', async () => {
    props.gkill_api.add_tag.mockResolvedValue({
      messages: [{ message_code: 'OK' }],
      errors: [],
      added_tag: { tag: 'テスト' },
    })
    const view = useAddTagView({ props, emits })
    view.tag_name.value = 'テストタグ'
    await view.save()
    // save() may show a confirmation dialog for unknown tags before calling API
    // Verify no errors emitted for valid input
    const errorCalls = emits.mock.calls.filter((c: any[]) => c[0] === 'received_errors')
    expect(errorCalls.length).toBe(0)
  })

  test('returns expected interface', () => {
    const view = useAddTagView({ props, emits })
    expect(view.tag_name).toBeDefined()
    expect(typeof view.save).toBe('function')
    expect(view.show_kyou).toBeDefined()
  })
})

// ========== useAddNlogView ==========

describe('useAddNlogView', () => {
  let props: ReturnType<typeof createBaseProps>
  let emits: ReturnType<typeof vi.fn>

  beforeEach(() => {
    vi.clearAllMocks()
    props = createBaseProps()
    emits = vi.fn()
  })

  test('initializes with empty title and zero amount', () => {
    const view = useAddNlogView({ props, emits })
    expect(view.nlog_title_value.value).toBe('')
    expect(view.nlog_amount_value.value).toBe(0)
    expect(view.nlog_shop_value.value).toBe('')
  })

  test('save() emits received_errors when title is blank', async () => {
    const view = useAddNlogView({ props, emits })
    view.nlog_title_value.value = ''
    await view.save()
    const errorCalls = emits.mock.calls.filter((c: any[]) => c[0] === 'received_errors')
    expect(errorCalls.length).toBeGreaterThan(0)
  })

  test('save() calls add_nlog API on valid input', async () => {
    props.gkill_api.add_nlog.mockResolvedValue({
      messages: [{ message_code: 'OK' }],
      errors: [],
    })
    const view = useAddNlogView({ props, emits })
    view.nlog_title_value.value = 'テスト支出'
    view.nlog_shop_value.value = 'テスト店'
    view.nlog_amount_value.value = 500
    await view.save()
    expect(props.gkill_api.add_nlog).toHaveBeenCalled()
  })

  test('returns expected interface', () => {
    const view = useAddNlogView({ props, emits })
    expect(typeof view.save).toBe('function')
    expect(typeof view.reset).toBe('function')
  })
})

// ========== useAddURLogView ==========

describe('useAddUrlogView', () => {
  let props: ReturnType<typeof createBaseProps>
  let emits: ReturnType<typeof vi.fn>

  beforeEach(() => {
    vi.clearAllMocks()
    props = createBaseProps()
    emits = vi.fn()
  })

  test('initializes with empty URL', () => {
    const view = useAddUrlogView({ props, emits })
    expect(view.url.value).toBe('')
  })

  test('save() emits received_errors when URL is blank', async () => {
    const view = useAddUrlogView({ props, emits })
    view.url.value = ''
    await view.save()
    const errorCalls = emits.mock.calls.filter((c: any[]) => c[0] === 'received_errors')
    expect(errorCalls.length).toBeGreaterThan(0)
  })

  test('returns expected interface', () => {
    const view = useAddUrlogView({ props, emits })
    expect(typeof view.save).toBe('function')
    expect(typeof view.reset).toBe('function')
  })
})

// ========== useAddLantanaView ==========

describe('useAddLantanaView', () => {
  let props: ReturnType<typeof createBaseProps>
  let emits: ReturnType<typeof vi.fn>

  beforeEach(() => {
    vi.clearAllMocks()
    props = createBaseProps()
    emits = vi.fn()
  })

  test('initializes with default mood value', () => {
    const view = useAddLantanaView({ props, emits })
    expect(view.mood).toBeDefined()
  })

  test('returns expected interface', () => {
    const view = useAddLantanaView({ props, emits })
    expect(typeof view.save).toBe('function')
    expect(typeof view.reset).toBe('function')
  })
})

// ========== useAddTimeisView ==========

describe('useAddTimeisView', () => {
  let props: ReturnType<typeof createBaseProps>
  let emits: ReturnType<typeof vi.fn>

  beforeEach(() => {
    vi.clearAllMocks()
    props = createBaseProps()
    emits = vi.fn()
  })

  test('initializes with empty title', () => {
    const view = useAddTimeisView({ props, emits })
    expect(view.timeis_title.value).toBe('')
  })

  test('save() emits received_errors when title is blank', async () => {
    const view = useAddTimeisView({ props, emits })
    view.timeis_title.value = ''
    await view.save()
    const errorCalls = emits.mock.calls.filter((c: any[]) => c[0] === 'received_errors')
    expect(errorCalls.length).toBeGreaterThan(0)
  })

  test('returns expected interface', () => {
    const view = useAddTimeisView({ props, emits })
    expect(typeof view.save).toBe('function')
    expect(typeof view.reset).toBe('function')
  })
})

// ========== useAddKcView ==========

describe('useAddKcView', () => {
  let props: ReturnType<typeof createBaseProps>
  let emits: ReturnType<typeof vi.fn>

  beforeEach(() => {
    vi.clearAllMocks()
    props = createBaseProps()
    emits = vi.fn()
  })

  test('initializes with empty title', () => {
    const view = useAddKcView({ props, emits })
    expect(view.title.value).toBe('')
  })

  test('save() emits received_errors when title is blank', async () => {
    const view = useAddKcView({ props, emits })
    view.title.value = ''
    await view.save()
    const errorCalls = emits.mock.calls.filter((c: any[]) => c[0] === 'received_errors')
    expect(errorCalls.length).toBeGreaterThan(0)
  })

  test('returns expected interface', () => {
    const view = useAddKcView({ props, emits })
    expect(typeof view.save).toBe('function')
    expect(typeof view.reset).toBe('function')
  })
})
