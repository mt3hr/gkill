/**
 * tx で束ねる保存経路のうち、どのテストからも呼ばれていなかった5画面の直接テスト。
 *
 * 追加/編集画面は本体とタグを1つの tx_id で temp rep に積み、commit_tx で確定する（ADR-0410）。
 * tx 中の add_* / update_* は応答に Kyou を載せられないので、実体は commit 後に引き直して
 * registered_kyou / updated_kyou を emit する。1本でも失敗したら discard_tx して何も残さない。
 * 順序（書き込み → commit → emit）と失敗時の後始末を、URLog / Kmemo と同じ形で固定する。
 * 網羅（18画面が tx を通ること）は tx-bundle-source-scan.test.ts が機械検査する。
 */
import { describe, expect, test, vi } from 'vitest'

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

import { useConfirmReKyouView } from '@/classes/use-confirm-re-kyou-view'
import { useAddMiReKyouView } from '@/classes/use-add-mi-re-kyou-view'
import { useEditReKyouView } from '@/classes/use-edit-re-kyou-view'
import { useEditIDFKyouView } from '@/classes/use-edit-idf-kyou-view'
import { useEditMiReKyouView } from '@/classes/use-edit-mi-re-kyou-view'

const ok = { messages: [], errors: [] }
const failed = { messages: [], errors: [{ error_code: 'ERR_TEST', error_message: 'ng' }] }

/** 呼ばれた順を記録する API モック。書き込み系は tx_id が付いていることも見る */
function make_api(call_order: string[]) {
  const record = (name: string) => vi.fn((req?: { tx_id?: string | null }) => {
    call_order.push(name)
    if (req && 'tx_id' in req) {
      expect(req.tx_id, `${name} に tx_id が無い`).toBeTruthy()
    }
    return Promise.resolve({ ...ok })
  })
  return {
    generate_uuid: vi.fn(() => 'id-' + Math.random().toString(36).slice(2, 8)),
    add_rekyou: record('add_rekyou'),
    add_mirekyou: record('add_mirekyou'),
    update_rekyou: record('update_rekyou'),
    update_idf_kyou: record('update_idf_kyou'),
    update_mirekyou: record('update_mirekyou'),
    add_tag: record('add_tag'),
    commit_tx: vi.fn(() => { call_order.push('commit_tx'); return Promise.resolve({ committed: [], ...ok }) }),
    discard_tx: vi.fn(() => { call_order.push('discard_tx'); return Promise.resolve({ ...ok }) }),
    get_kyou: vi.fn(() => Promise.resolve({ kyou_histories: [{ id: 'fetched-kyou' }], ...ok })),
    get_mi_board_list: vi.fn().mockResolvedValue({ boards: [], ...ok }),
    set_saved_last_added_tag: vi.fn(),
    push_tag_to_history: vi.fn(),
  }
}

function make_application_config() {
  return {
    device: 'test-device',
    user_id: 'admin',
    mi_default_board: 'Inbox',
    tag_struct: { children: [] },
    mi_board_struct: {
      board_name: '',
      children: [
        { board_name: 'Inbox', children: [] },
        { board_name: 'Work', children: [] },
      ],
    },
  }
}

/** emit された瞬間に順序へ積む（save() が返ってから mock.calls を読むと順序の検査にならない） */
function make_emits(call_order: string[]) {
  return vi.fn((event: string) => {
    if (event === 'registered_kyou' || event === 'updated_kyou' || event === 'received_errors') {
      call_order.push(event)
    }
  })
}

function emitted_events(emits: ReturnType<typeof vi.fn>): string[] {
  return emits.mock.calls.map(call => call[0] as string)
}

/** 編集画面の props.kyou。clone() と load_typed_datas() を持つ構造フェイク */
function make_editable_kyou(typed_key: string, typed: Record<string, unknown>) {
  const typed_with_clone: Record<string, unknown> = { ...typed }
  typed_with_clone.clone = () => ({ ...typed_with_clone })
  const kyou: Record<string, unknown> = {
    id: 'kyou-1',
    related_time: new Date('2025-03-15T09:00:00+09:00'),
    abort_controller: new AbortController(),
    load_typed_datas: vi.fn().mockResolvedValue([]),
    [typed_key]: typed_with_clone,
  }
  kyou.clone = () => ({ ...kyou, abort_controller: new AbortController(), clone: kyou.clone })
  return kyou
}

async function flush(): Promise<void> {
  for (let i = 0; i < 6; i++) {
    await Promise.resolve()
  }
}

describe('useConfirmReKyouView（リポスト作成）', () => {
  test('add_rekyou → commit_tx → registered_kyou の順で、閉じる要求まで出す', async () => {
    const order: string[] = []
    const api = make_api(order)
    const emits = make_emits(order)
    const view = useConfirmReKyouView({
      props: { gkill_api: api, application_config: make_application_config(), kyou: { id: 'target-1' } } as never,
      emits: emits as never,
    })

    await view.rekyou()

    expect(order).toEqual(['add_rekyou', 'commit_tx', 'registered_kyou'])
    expect(api.discard_tx).not.toHaveBeenCalled()
    expect(emitted_events(emits)).toContain('requested_close_dialog')
  })

  test('add_rekyou が失敗したら discard_tx して registered_kyou を出さない', async () => {
    const order: string[] = []
    const api = make_api(order)
    api.add_rekyou.mockImplementation(() => Promise.resolve({ ...failed }))
    const emits = make_emits(order)
    const view = useConfirmReKyouView({
      props: { gkill_api: api, application_config: make_application_config(), kyou: { id: 'target-1' } } as never,
      emits: emits as never,
    })

    await view.rekyou()

    expect(api.commit_tx).not.toHaveBeenCalled()
    expect(api.discard_tx).toHaveBeenCalledTimes(1)
    expect(emitted_events(emits)).toContain('received_errors')
    expect(emitted_events(emits)).not.toContain('registered_kyou')
  })
})

describe('useAddMiReKyouView（リポストタスク追加）', () => {
  test('add_mirekyou → commit_tx → registered_kyou の順', async () => {
    const order: string[] = []
    const api = make_api(order)
    const emits = make_emits(order)
    const view = useAddMiReKyouView({
      props: { gkill_api: api, application_config: make_application_config(), kyou: { id: 'target-1' } } as never,
      emits: emits as never,
    })
    await flush()

    await view.save()

    expect(order).toEqual(['add_mirekyou', 'commit_tx', 'registered_kyou'])
    expect(api.discard_tx).not.toHaveBeenCalled()
  })

  test('commit_tx が失敗したら discard_tx して registered_kyou を出さない', async () => {
    const order: string[] = []
    const api = make_api(order)
    api.commit_tx.mockImplementation(() => { order.push('commit_tx'); return Promise.resolve({ committed: [], ...failed }) })
    const emits = make_emits(order)
    const view = useAddMiReKyouView({
      props: { gkill_api: api, application_config: make_application_config(), kyou: { id: 'target-1' } } as never,
      emits: emits as never,
    })
    await flush()

    await view.save()

    expect(api.discard_tx).toHaveBeenCalledTimes(1)
    expect(emitted_events(emits)).toContain('received_errors')
    expect(emitted_events(emits)).not.toContain('registered_kyou')
  })
})

describe('useEditReKyouView（リポスト編集）', () => {
  function build(order: string[]) {
    const api = make_api(order)
    const emits = make_emits(order)
    const kyou = make_editable_kyou('typed_rekyou', {
      id: 'kyou-1',
      target_id: 'target-1',
      related_time: new Date('2025-03-15T09:00:00+09:00'),
    })
    const view = useEditReKyouView({
      props: { gkill_api: api, application_config: make_application_config(), kyou } as never,
      emits: emits as never,
    })
    return { api, emits, view }
  }

  test('関連日時を変えて保存すると update_rekyou → commit_tx → updated_kyou の順', async () => {
    const order: string[] = []
    const { api, emits, view } = build(order)
    await flush()
    view.related_time_string.value = '10:30:00'

    await view.save()

    expect(order).toEqual(['update_rekyou', 'commit_tx', 'updated_kyou'])
    expect(api.discard_tx).not.toHaveBeenCalled()
    expect(emitted_events(emits)).toContain('requested_reload_kyou')
    expect(emitted_events(emits)).toContain('requested_close_dialog')
  })

  test('変更が無ければ何も書かずにエラーを出す', async () => {
    const order: string[] = []
    const { api, emits, view } = build(order)
    await flush()

    await view.save()

    expect(api.update_rekyou).not.toHaveBeenCalled()
    expect(api.commit_tx).not.toHaveBeenCalled()
    expect(emitted_events(emits)).toContain('received_errors')
  })

  test('update_rekyou が失敗したら discard_tx して updated_kyou を出さない', async () => {
    const order: string[] = []
    const { api, emits, view } = build(order)
    api.update_rekyou.mockImplementation(() => Promise.resolve({ ...failed }))
    await flush()
    view.related_time_string.value = '10:30:00'

    await view.save()

    expect(api.commit_tx).not.toHaveBeenCalled()
    expect(api.discard_tx).toHaveBeenCalledTimes(1)
    expect(emitted_events(emits)).not.toContain('updated_kyou')
  })
})

describe('useEditIDFKyouView（ファイル記録の編集）', () => {
  test('関連日時を変えて保存すると update_idf_kyou → commit_tx → updated_kyou の順', async () => {
    const order: string[] = []
    const api = make_api(order)
    const emits = make_emits(order)
    const kyou = make_editable_kyou('typed_idf_kyou', {
      id: 'kyou-1',
      related_time: new Date('2025-03-15T09:00:00+09:00'),
    })
    const view = useEditIDFKyouView({
      props: { gkill_api: api, application_config: make_application_config(), kyou } as never,
      emits: emits as never,
    })
    await flush()
    view.related_time_string.value = '10:30:00'

    await view.save()

    expect(order).toEqual(['update_idf_kyou', 'commit_tx', 'updated_kyou'])
    expect(api.discard_tx).not.toHaveBeenCalled()
  })
})

describe('useEditMiReKyouView（リポストタスク編集）', () => {
  function build(order: string[]) {
    const api = make_api(order)
    const emits = make_emits(order)
    const kyou = make_editable_kyou('typed_mirekyou', {
      id: 'kyou-1',
      target_id: 'target-1',
      board_name: 'Inbox',
      is_checked: false,
      estimate_start_time: null,
      estimate_end_time: null,
      limit_time: null,
    })
    const view = useEditMiReKyouView({
      props: { gkill_api: api, application_config: make_application_config(), kyou } as never,
      emits: emits as never,
    })
    return { api, emits, view }
  }

  test('板名を変えて保存すると update_mirekyou → commit_tx → updated_kyou の順', async () => {
    const order: string[] = []
    const { api, view } = build(order)
    await flush()
    view.mi_board_name.value = 'Work'

    await view.save()

    expect(order).toEqual(['update_mirekyou', 'commit_tx', 'updated_kyou'])
    expect(api.discard_tx).not.toHaveBeenCalled()
  })

  test('commit_tx が失敗したら discard_tx して updated_kyou を出さない', async () => {
    const order: string[] = []
    const { api, emits, view } = build(order)
    api.commit_tx.mockImplementation(() => { order.push('commit_tx'); return Promise.resolve({ committed: [], ...failed }) })
    await flush()
    view.mi_board_name.value = 'Work'

    await view.save()

    expect(api.discard_tx).toHaveBeenCalledTimes(1)
    expect(emitted_events(emits)).toContain('received_errors')
    expect(emitted_events(emits)).not.toContain('updated_kyou')
  })
})
