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
import { useAddURLogView } from '@/classes/use-add-ur-log-view'
import { useAddLantanaView } from '@/classes/use-add-lantana-view'
import { useAddTimeIsView } from '@/classes/use-add-time-is-view'
import { useAddKCView } from '@/classes/use-add-kc-view'

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

  // 連番の uuid（tx_id と行ごとの id を見分けるため。mock-api の既定は固定値）と、
  // 受け取った id をそのまま返す get_kyou にする
  function use_distinct_ids(): void {
    let n = 0
    props.gkill_api.generate_uuid.mockImplementation(() => `uuid-${++n}`)
    props.gkill_api.get_kyou.mockImplementation((req: { id: string }) =>
      Promise.resolve({ kyou_histories: [{ id: req.id }], messages: [], errors: [] }))
  }

  test('店名は空、品名と金額の行は1行・空で始まる', () => {
    const view = useAddNlogView({ props, emits })
    expect(view.nlog_shop_value.value).toBe('')
    expect(view.nlog_rows.value.length).toBe(1)
    expect(view.nlog_rows.value[0].title).toBe('')
    expect(view.nlog_rows.value[0].amount).toBe(0)
    expect(view.can_delete_row.value).toBe(false)
  })

  test('save() emits received_errors when title is blank', async () => {
    const view = useAddNlogView({ props, emits })
    view.nlog_rows.value[0].title = ''
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
    view.nlog_rows.value[0].title = 'テスト支出'
    view.nlog_shop_value.value = 'テスト店'
    view.nlog_rows.value[0].amount = 500
    await view.save()
    expect(props.gkill_api.add_nlog).toHaveBeenCalled()
  })

  test('行を足して消せる。最後の1行は消えず、reset で1行に戻る', () => {
    const view = useAddNlogView({ props, emits })
    view.add_row()
    view.add_row()
    expect(view.nlog_rows.value.length).toBe(3)
    expect(new Set(view.nlog_rows.value.map(row => row.row_key)).size).toBe(3)
    expect(view.can_delete_row.value).toBe(true)
    view.nlog_rows.value[1].title = '2行目'
    view.delete_row(0)
    expect(view.nlog_rows.value.map(row => row.title)).toEqual(['2行目', ''])
    view.delete_row(5)
    expect(view.nlog_rows.value.length).toBe(2)
    view.delete_row(0)
    view.delete_row(0)
    expect(view.nlog_rows.value.length).toBe(1)
    view.add_row()
    view.reset()
    expect(view.nlog_rows.value.length).toBe(1)
  })

  // メモ帳の支出と同じく、店名と関連時刻は全行で共有し、1行が1件になる。
  // 全行を1つの tx に積むので、1件でも失敗したら何も残らない
  test('2行なら同じ tx で add_nlog を2回呼び、店名と関連時刻は共通・id は別', async () => {
    use_distinct_ids()
    const view = useAddNlogView({ props, emits })
    view.nlog_shop_value.value = 'コンビニ'
    view.nlog_rows.value[0].title = 'おにぎり'
    view.nlog_rows.value[0].amount = '150'
    view.add_row()
    view.nlog_rows.value[1].title = 'お茶'
    view.nlog_rows.value[1].amount = 120

    await view.save()

    const reqs = props.gkill_api.add_nlog.mock.calls.map((call: unknown[]) => call[0] as {
      nlog: { id: string, shop: string, title: string, amount: number, related_time: Date }, tx_id: string
    })
    expect(reqs.length).toBe(2)
    expect(reqs[0].tx_id).toBe(reqs[1].tx_id)
    expect(reqs[0].nlog.id).not.toBe(reqs[1].nlog.id)
    expect(reqs.map(req => req.nlog.shop)).toEqual(['コンビニ', 'コンビニ'])
    expect(reqs.map(req => req.nlog.title)).toEqual(['おにぎり', 'お茶'])
    // 入力欄が返す文字列の金額は数値にして送る
    expect(reqs.map(req => req.nlog.amount)).toEqual([150, 120])
    expect(reqs[0].nlog.related_time.getTime()).toBe(reqs[1].nlog.related_time.getTime())
    expect(props.gkill_api.commit_tx).toHaveBeenCalledTimes(1)
    const registered = emits.mock.calls.filter((c: unknown[]) => c[0] === 'registered_kyou').map((c: unknown[]) => (c[1] as { id: string }).id)
    expect(registered).toEqual(reqs.map(req => req.nlog.id))
  })

  test('タグは全行に付け、registered_tag はタグ名ごとに1回、順序は add_nlog→add_tag→…→commit_tx→registered_kyou', async () => {
    use_distinct_ids()
    const call_order: string[] = []
    props.gkill_api.add_nlog.mockImplementation(() => {
      call_order.push('add_nlog')
      return Promise.resolve({ added_kyou: null, messages: [], errors: [] })
    })
    props.gkill_api.add_tag.mockImplementation(() => {
      call_order.push('add_tag')
      return Promise.resolve({ added_tag: null, messages: [], errors: [] })
    })
    props.gkill_api.commit_tx.mockImplementation(() => {
      call_order.push('commit_tx')
      return Promise.resolve({ committed: [], messages: [], errors: [] })
    })
    const ordered_emits = vi.fn((event: string) => {
      if (event === 'registered_kyou' || event === 'registered_tag') {
        call_order.push(event)
      }
    })
    const view = useAddNlogView({ props, emits: ordered_emits })
    view.nlog_shop_value.value = 'コンビニ'
    view.nlog_rows.value[0].title = 'おにぎり'
    view.nlog_rows.value[0].amount = 150
    view.add_row()
    view.nlog_rows.value[1].title = 'お茶'
    view.nlog_rows.value[1].amount = 120
    view.kyou_tags_view.value = { get_tag_names: () => ['食費'], reset: () => { } }
    props.application_config.tag_struct = { children: [{ tag_name: '食費', children: [] }] }

    await view.save()

    expect(call_order).toEqual([
      'add_nlog', 'add_tag', 'add_nlog', 'add_tag', 'commit_tx',
      'registered_tag', 'registered_kyou', 'registered_kyou',
    ])
  })

  test('2行目が不正なら何も書かない', async () => {
    const view = useAddNlogView({ props, emits })
    view.nlog_shop_value.value = 'コンビニ'
    view.nlog_rows.value[0].title = 'おにぎり'
    view.nlog_rows.value[0].amount = 150
    view.add_row()
    view.nlog_rows.value[1].title = ''

    await view.save()

    expect(props.gkill_api.add_nlog).not.toHaveBeenCalled()
    expect(emits.mock.calls.map((c: unknown[]) => c[0])).toContain('received_errors')
  })

  test('2行目の add_nlog が失敗したら discard_tx して registered_kyou を出さない（1行目も残らない）', async () => {
    use_distinct_ids()
    props.gkill_api.add_nlog
      .mockResolvedValueOnce({ added_kyou: null, messages: [], errors: [] })
      .mockResolvedValueOnce({ added_kyou: null, messages: [], errors: [{ error_code: 'ERR_TEST', error_message: 'ng' }] })
    const view = useAddNlogView({ props, emits })
    view.nlog_shop_value.value = 'コンビニ'
    view.nlog_rows.value[0].title = 'おにぎり'
    view.nlog_rows.value[0].amount = 150
    view.add_row()
    view.nlog_rows.value[1].title = 'お茶'
    view.nlog_rows.value[1].amount = 120

    await view.save()

    expect(props.gkill_api.commit_tx).not.toHaveBeenCalled()
    expect(props.gkill_api.discard_tx).toHaveBeenCalledTimes(1)
    const events = emits.mock.calls.map((c: unknown[]) => c[0])
    expect(events).toContain('received_errors')
    expect(events).not.toContain('registered_kyou')
  })

  test('returns expected interface', () => {
    const view = useAddNlogView({ props, emits })
    expect(typeof view.save).toBe('function')
    expect(typeof view.reset).toBe('function')
    expect(typeof view.add_row).toBe('function')
    expect(typeof view.delete_row).toBe('function')
  })
})

// ========== useAddURLogView ==========

describe('useAddURLogView', () => {
  let props: Record<string, unknown>
  let emits: ReturnType<typeof vi.fn>

  beforeEach(() => {
    vi.clearAllMocks()
    props = createBaseProps()
    emits = vi.fn()
  })

  test('initializes with empty URL', () => {
    const view = useAddURLogView({ props, emits })
    expect(view.url.value).toBe('')
  })

  test('save() emits received_errors when URL is blank', async () => {
    const view = useAddURLogView({ props, emits })
    view.url.value = ''
    await view.save()
    const errorCalls = emits.mock.calls.filter((c: unknown[]) => c[0] === 'received_errors')
    expect(errorCalls.length).toBeGreaterThan(0)
  })

  test('returns expected interface', () => {
    const view = useAddURLogView({ props, emits })
    expect(typeof view.save).toBe('function')
    expect(typeof view.reset).toBe('function')
  })

  // 局所挿入(use-registered-kyou-local-insert.ts)は渡されたKyouをそのまま使わず
  // refresh_kyou で引き直すので、その時点でサーバにタグが入っていれば
  // attached_tags 込みで差し込まれる。逆に registered_kyou を先に emit すると、
  // タグで絞り込んだ列が空のタグ列を見て「一致しない」と判定し、
  // エラーも出ないまま行が現れない。順序が唯一の防御線
  // 2026-09-15 からは本体とタグを1つの tx に積み、commit_tx が通ってから引き直して emit する。
  // 「タグが付いてから」の約束は「commit が終わってから」に読み替わった（ADR-0410）
  test('registered_kyou は add_tag と commit_tx が終わってから emit される', async () => {
    const call_order: string[] = []
    props.gkill_api.add_urlog.mockImplementation((req: { tx_id: string | null }) => {
      call_order.push('add_urlog')
      expect(req.tx_id).not.toBeNull()
      // tx 中は added_kyou が返らない（一時リポジトリにしか無い）
      return Promise.resolve({ added_kyou: null, messages: [], errors: [] })
    })
    // 遅延させて「先にemitしていないか」を確実に捕まえる
    props.gkill_api.add_tag.mockImplementation((req: { tag: { tag: string }, tx_id: string | null }) => new Promise(resolve => {
      setTimeout(() => {
        call_order.push('add_tag')
        expect(req.tx_id).not.toBeNull()
        resolve({ added_tag: null, messages: [], errors: [] })
      }, 10)
    }))
    props.gkill_api.commit_tx.mockImplementation(() => new Promise(resolve => {
      setTimeout(() => {
        call_order.push('commit_tx')
        resolve({ committed: [], messages: [], errors: [] })
      }, 10)
    }))
    props.gkill_api.get_kyou.mockResolvedValue({ kyou_histories: [{ id: 'new-urlog-id' }], messages: [], errors: [] })
    // emitされた瞬間に記録する。save()が返ってから mock.calls を読むと
    // 実際の順序に関わらず registered_kyou が最後に積まれて検査にならない
    const ordered_emits = vi.fn((event: string) => {
      if (event === 'registered_kyou') {
        call_order.push('registered_kyou')
      }
    })
    const view = useAddURLogView({ props, emits: ordered_emits })
    view.url.value = 'https://example.com/'
    // タグ欄の子ビューは親から見ると template ref。値を返すだけのスタブで十分
    view.kyou_tags_view.value = { get_tag_names: () => ['既知タグ'], reset: () => { } }
    // 未知タグ確認を挟ませないため、タグツリーに入れておく
    props.application_config.tag_struct = { children: [{ tag_name: '既知タグ', children: [] }] }

    await view.save()

    expect(call_order).toEqual(['add_urlog', 'add_tag', 'commit_tx', 'registered_kyou'])
    expect(props.gkill_api.discard_tx).not.toHaveBeenCalled()
  })

  test('タグの追加が失敗したら discard_tx して registered_kyou を出さない（何も保存されていない）', async () => {
    props.gkill_api.add_urlog.mockResolvedValue({ added_kyou: null, messages: [], errors: [] })
    props.gkill_api.add_tag.mockResolvedValue({ added_tag: null, messages: [], errors: [{ error_code: 'ERR_TEST', error_message: 'ng' }] })
    const view = useAddURLogView({ props, emits })
    view.url.value = 'https://example.com/'
    view.kyou_tags_view.value = { get_tag_names: () => ['既知タグ'], reset: () => { } }
    props.application_config.tag_struct = { children: [{ tag_name: '既知タグ', children: [] }] }

    await view.save()

    expect(props.gkill_api.commit_tx).not.toHaveBeenCalled()
    expect(props.gkill_api.discard_tx).toHaveBeenCalledTimes(1)
    const events = emits.mock.calls.map(call => call[0])
    expect(events).toContain('received_errors')
    expect(events).not.toContain('registered_kyou')
    expect(events).not.toContain('registered_tag')
  })

  test('commit_tx が失敗したら discard_tx して registered_kyou を出さない', async () => {
    props.gkill_api.add_urlog.mockResolvedValue({ added_kyou: null, messages: [], errors: [] })
    props.gkill_api.commit_tx.mockResolvedValue({ committed: [], messages: [], errors: [{ error_code: 'ERR000419', error_message: 'rolled back' }] })
    const view = useAddURLogView({ props, emits })
    view.url.value = 'https://example.com/'

    await view.save()

    expect(props.gkill_api.discard_tx).toHaveBeenCalledTimes(1)
    const events = emits.mock.calls.map(call => call[0])
    expect(events).toContain('received_errors')
    expect(events).not.toContain('registered_kyou')
  })

  test('タグ名が新しいときは保存せず確認ダイアログを開く', async () => {
    const view = useAddURLogView({ props, emits })
    view.url.value = 'https://example.com/'
    view.kyou_tags_view.value = { get_tag_names: () => ['新しいタグ'], reset: () => { } }

    await view.save()

    expect(view.unknown_tags.value).toEqual(['新しいタグ'])
    expect(props.gkill_api.add_urlog).not.toHaveBeenCalled()
    expect(props.gkill_api.add_tag).not.toHaveBeenCalled()
  })

  test('タグを書かなければ add_tag は呼ばれない', async () => {
    props.gkill_api.add_urlog.mockResolvedValue({ added_kyou: { id: 'new-urlog-id' }, messages: [], errors: [] })
    const view = useAddURLogView({ props, emits })
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

// ========== useAddKCView ==========

describe('useAddKCView', () => {
  let props: Record<string, unknown>
  let emits: ReturnType<typeof vi.fn>

  beforeEach(() => {
    vi.clearAllMocks()
    props = createBaseProps()
    emits = vi.fn()
  })

  test('initializes with empty title', () => {
    const view = useAddKCView({ props, emits })
    expect(view.title.value).toBe('')
  })

  test('save() emits received_errors when title is blank', async () => {
    const view = useAddKCView({ props, emits })
    view.title.value = ''
    await view.save()
    const errorCalls = emits.mock.calls.filter((c: unknown[]) => c[0] === 'received_errors')
    expect(errorCalls.length).toBeGreaterThan(0)
  })

  test('returns expected interface', () => {
    const view = useAddKCView({ props, emits })
    expect(typeof view.save).toBe('function')
    expect(typeof view.reset).toBe('function')
  })
})
