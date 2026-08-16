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
import { useAddUrlogView } from '@/classes/use-add-ur-log-view'
import { useAddLantanaView } from '@/classes/use-add-lantana-view'
import { useAddTimeIsView } from '@/classes/use-add-time-is-view'
import { useAddKcView } from '@/classes/use-add-kc-view'

function createBaseProps() {
  return {
    gkill_api: createMockGkillAPI() as never,
    application_config: {
      device: 'test-device',
      user_id: 'admin',
      mi_default_board: 'Inbox',
      tag_struct: { children: [] },
      // 板の実在確認(use-confirm-unknown-mi-board)が参照する板ツリー
      mi_board_struct: {
        board_name: '',
        children: [
          { board_name: 'Inbox', children: [] },
          { board_name: 'Work', children: [] },
        ],
      },
    } as never,
  }
}

// ========== useAddMiView ==========

describe('useAddMiView', () => {
  let props: Record<string, unknown>
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
    const errorCalls = emits.mock.calls.filter((c: unknown[]) => c[0] === 'received_errors')
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

  // サーバの板一覧はマップ反復順で返るので、並び順は ApplicationConfig の板ツリーが正
  test('板名は ApplicationConfig の設定順に並ぶ', async () => {
    props.gkill_api.get_mi_board_list.mockResolvedValue({
      boards: ['Work', 'Inbox'],
      messages: [],
      errors: [],
    })
    const view = useAddMiView({ props, emits })
    await view.load_mi_board_names()
    expect(view.mi_board_names.value).toStrictEqual(['Inbox', 'Work'])
  })

  test('設定に無い板はAPIの順のまま末尾へ', async () => {
    props.gkill_api.get_mi_board_list.mockResolvedValue({
      boards: ['新板B', 'Work', '新板A', 'Inbox'],
      messages: [],
      errors: [],
    })
    const view = useAddMiView({ props, emits })
    await view.load_mi_board_names()
    expect(view.mi_board_names.value).toStrictEqual(['Inbox', 'Work', '新板B', '新板A'])
  })

  // ＋ボタンで作った板はまだ板ツリーに無いので末尾に出るが、候補には必ず入る
  test('update_board_name() で足した板が候補に入り選択される', async () => {
    const view = useAddMiView({ props, emits })
    await view.load_mi_board_names()
    view.update_board_name('新しい板')
    expect(view.mi_board_names.value).toStrictEqual(['Inbox', 'Work', '新しい板'])
    expect(view.mi_board_name.value).toBe('新しい板')
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
  let props: Record<string, unknown>
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
    const errorCalls = emits.mock.calls.filter((c: unknown[]) => c[0] === 'received_errors')
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
    const errorCalls = emits.mock.calls.filter((c: unknown[]) => c[0] === 'received_errors')
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
  let props: Record<string, unknown>
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
    const errorCalls = emits.mock.calls.filter((c: unknown[]) => c[0] === 'received_errors')
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
  let props: Record<string, unknown>
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
    const errorCalls = emits.mock.calls.filter((c: unknown[]) => c[0] === 'received_errors')
    expect(errorCalls.length).toBeGreaterThan(0)
  })

  test('returns expected interface', () => {
    const view = useAddUrlogView({ props, emits })
    expect(typeof view.save).toBe('function')
    expect(typeof view.reset).toBe('function')
  })

  // 局所挿入(use-registered-kyou-local-insert.ts)は渡されたKyouをそのまま使わず
  // refresh_kyou で引き直すので、その時点でサーバにタグが入っていれば
  // attached_tags 込みで差し込まれる。逆に registered_kyou を先に emit すると、
  // タグで絞り込んだ列が空のタグ列を見て「一致しない」と判定し、
  // エラーも出ないまま行が現れない。順序が唯一の防御線
  test('registered_kyou は add_tag が終わってから emit される', async () => {
    const call_order: string[] = []
    props.gkill_api.add_urlog.mockImplementation(() => {
      call_order.push('add_urlog')
      return Promise.resolve({ added_kyou: { id: 'new-urlog-id' }, messages: [], errors: [] })
    })
    // 遅延させて「先にemitしていないか」を確実に捕まえる
    props.gkill_api.add_tag.mockImplementation((req: { tag: { tag: string } }) => new Promise(resolve => {
      setTimeout(() => {
        call_order.push('add_tag')
        resolve({ added_tag: req.tag, messages: [], errors: [] })
      }, 10)
    }))
    // emitされた瞬間に記録する。save()が返ってから mock.calls を読むと
    // 実際の順序に関わらず registered_kyou が最後に積まれて検査にならない
    const ordered_emits = vi.fn((event: string) => {
      if (event === 'registered_kyou') {
        call_order.push('registered_kyou')
      }
    })
    const view = useAddUrlogView({ props, emits: ordered_emits })
    view.url.value = 'https://example.com/'
    // タグ欄の子ビューは親から見ると template ref。値を返すだけのスタブで十分
    view.kyou_tags_view.value = { get_tag_names: () => ['既知タグ'], reset: () => { } }
    // 未知タグ確認を挟ませないため、タグツリーに入れておく
    props.application_config.tag_struct = { children: [{ tag_name: '既知タグ', children: [] }] }

    await view.save()

    expect(call_order).toEqual(['add_urlog', 'add_tag', 'registered_kyou'])
  })

  test('タグ名が新しいときは保存せず確認ダイアログを開く', async () => {
    const view = useAddUrlogView({ props, emits })
    view.url.value = 'https://example.com/'
    view.kyou_tags_view.value = { get_tag_names: () => ['新しいタグ'], reset: () => { } }

    await view.save()

    expect(view.unknown_tags.value).toEqual(['新しいタグ'])
    expect(props.gkill_api.add_urlog).not.toHaveBeenCalled()
    expect(props.gkill_api.add_tag).not.toHaveBeenCalled()
  })

  test('タグを書かなければ add_tag は呼ばれない', async () => {
    props.gkill_api.add_urlog.mockResolvedValue({ added_kyou: { id: 'new-urlog-id' }, messages: [], errors: [] })
    const view = useAddUrlogView({ props, emits })
    view.url.value = 'https://example.com/'
    view.kyou_tags_view.value = { get_tag_names: () => [], reset: () => { } }

    await view.save()

    expect(props.gkill_api.add_urlog).toHaveBeenCalledTimes(1)
    expect(props.gkill_api.add_tag).not.toHaveBeenCalled()
  })
})

// ========== useAddLantanaView ==========

describe('useAddLantanaView', () => {
  let props: Record<string, unknown>
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

// ========== useAddTimeIsView ==========

describe('useAddTimeIsView', () => {
  let props: Record<string, unknown>
  let emits: ReturnType<typeof vi.fn>

  beforeEach(() => {
    vi.clearAllMocks()
    props = createBaseProps()
    emits = vi.fn()
  })

  test('initializes with empty title', () => {
    const view = useAddTimeIsView({ props, emits })
    expect(view.timeis_title.value).toBe('')
  })

  test('save() emits received_errors when title is blank', async () => {
    const view = useAddTimeIsView({ props, emits })
    view.timeis_title.value = ''
    await view.save()
    const errorCalls = emits.mock.calls.filter((c: unknown[]) => c[0] === 'received_errors')
    expect(errorCalls.length).toBeGreaterThan(0)
  })

  test('returns expected interface', () => {
    const view = useAddTimeIsView({ props, emits })
    expect(typeof view.save).toBe('function')
    expect(typeof view.reset).toBe('function')
  })
})

// ========== useAddKcView ==========

describe('useAddKcView', () => {
  let props: Record<string, unknown>
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
    const errorCalls = emits.mock.calls.filter((c: unknown[]) => c[0] === 'received_errors')
    expect(errorCalls.length).toBeGreaterThan(0)
  })

  test('returns expected interface', () => {
    const view = useAddKcView({ props, emits })
    expect(typeof view.save).toBe('function')
    expect(typeof view.reset).toBe('function')
  })
})
