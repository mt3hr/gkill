/**
 * Cascade delete tests.
 *
 * Kyou削除時に、付随するTag/Text/Notificationと、それを参照しているReKyou/MiReKyouも
 * 連鎖して論理削除されることを確認する。
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'

vi.mock('@/i18n', () => ({
  default: { global: { t: (key: string) => key, locale: 'ja' } },
  i18n: { global: { t: (key: string) => key, locale: 'ja' } },
}))

vi.mock('@/classes/api/gkill-api', () => ({
  GkillAPI: {
    get_instance: vi.fn(() => ({
      get_session_id: vi.fn(() => 'mock-session'),
      generate_uuid: vi.fn(() => 'mock-uuid'),
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

import { build_deleted_kyou_stub, cascade_delete_kyou } from '@/classes/cascade-delete-kyou'
import delete_gkill_kyou_cache from '@/classes/delete-gkill-cache'

const base_time = new Date('2026-01-01T00:00:00Z')

interface MetaStub {
  id: string
  target_id: string
  is_deleted: boolean
  update_time: Date
  clone: () => MetaStub
}

interface RefStub extends MetaStub {
  data_type: string
}

interface TargetIDRequestStub {
  target_id: string
  force_reget: boolean
}

interface ErrorStub {
  error_code: string
  error_message: string
  show_keep: boolean
}

function make_meta_stub(id: string, target_id: string, overrides: Partial<MetaStub> = {}): MetaStub {
  const stub: MetaStub = {
    id,
    target_id,
    is_deleted: false,
    update_time: base_time,
    clone: () => ({ ...stub, clone: stub.clone }),
    ...overrides,
  }
  return stub
}

function make_ref_stub(id: string, target_id: string, overrides: Partial<RefStub> = {}): RefStub {
  const stub: RefStub = {
    id,
    target_id,
    data_type: 're_kyou',
    is_deleted: false,
    update_time: base_time,
    clone: () => ({ ...stub, clone: stub.clone }),
    ...overrides,
  }
  return stub
}

function make_kmemo_kyou(id: string) {
  const typed_kmemo = make_meta_stub(id, '')
  return {
    id,
    data_type: 'kmemo',
    typed_kmemo,
    load_typed_datas: vi.fn().mockResolvedValue([]),
  }
}

/**
 * 隣接リストを引くAPIモック。graphのキーがtarget_id、値がそれを参照しているReKyou。
 * BFSと循環参照のテストがそのまま書ける。
 */
function create_cascade_api(graph: {
  rekyous?: Record<string, Array<RefStub>>
  mirekyous?: Record<string, Array<RefStub>>
  tags?: Record<string, Array<MetaStub>>
  texts?: Record<string, Array<MetaStub>>
  notifications?: Record<string, Array<MetaStub>>
} = {}) {
  const call_order = new Array<string>()
  const ok = { messages: [] as Array<never>, errors: new Array<ErrorStub>() }

  return {
    call_order,
    api: {
      get_tags_by_target_id: vi.fn(async (req: TargetIDRequestStub) => {
        call_order.push(`get_tags:${req.target_id}`)
        return { tags: graph.tags?.[req.target_id] ?? [], ...ok }
      }),
      get_texts_by_target_id: vi.fn(async (req: TargetIDRequestStub) => {
        call_order.push(`get_texts:${req.target_id}`)
        return { texts: graph.texts?.[req.target_id] ?? [], ...ok }
      }),
      get_notifications_by_target_id: vi.fn(async (req: TargetIDRequestStub) => {
        call_order.push(`get_notifications:${req.target_id}`)
        return { notifications: graph.notifications?.[req.target_id] ?? [], ...ok }
      }),
      get_rekyous_by_target_id: vi.fn(async (req: TargetIDRequestStub) => {
        call_order.push(`get_rekyous:${req.target_id}`)
        return { rekyous: graph.rekyous?.[req.target_id] ?? [], ...ok }
      }),
      get_mirekyous_by_target_id: vi.fn(async (req: TargetIDRequestStub) => {
        call_order.push(`get_mirekyous:${req.target_id}`)
        return { mirekyous: graph.mirekyous?.[req.target_id] ?? [], ...ok }
      }),
      update_tag: vi.fn(async (req: { tag: MetaStub }) => { call_order.push(`update_tag:${req.tag.id}`); return ok }),
      update_text: vi.fn(async (req: { text: MetaStub }) => { call_order.push(`update_text:${req.text.id}`); return ok }),
      update_notification: vi.fn(async (req: { notification: MetaStub }) => { call_order.push(`update_notification:${req.notification.id}`); return ok }),
      update_rekyou: vi.fn(async (req: { rekyou: RefStub }) => { call_order.push(`update_rekyou:${req.rekyou.id}`); return ok }),
      update_mirekyou: vi.fn(async (req: { mirekyou: RefStub }) => { call_order.push(`update_mirekyou:${req.mirekyou.id}`); return ok }),
      update_kmemo: vi.fn(async (req: { kmemo: MetaStub }) => { call_order.push(`update_kmemo:${req.kmemo.id}`); return ok }),
    },
  }
}

type CascadeAPIStub = ReturnType<typeof create_cascade_api>['api']

const application_config = { device: 'test-device', user_id: 'admin', for_share_kyou: false }

function run(kyou: ReturnType<typeof make_kmemo_kyou>, api: CascadeAPIStub) {
    return cascade_delete_kyou({
        kyou: kyou as never,
        gkill_api: api as never,
        application_config: application_config as never,
    })
}

describe('cascade_delete_kyou', () => {
  beforeEach(() => {
    vi.mocked(delete_gkill_kyou_cache).mockClear()
  })

  it('付随するTag/Text/Notificationも論理削除する', async () => {
    const { api } = create_cascade_api({
      tags: { 'kmemo-1': [make_meta_stub('tag-1', 'kmemo-1')] },
      texts: { 'kmemo-1': [make_meta_stub('text-1', 'kmemo-1')] },
      notifications: { 'kmemo-1': [make_meta_stub('notification-1', 'kmemo-1')] },
    })

    const result = await run(make_kmemo_kyou('kmemo-1'), api)

    expect(result.errors).toHaveLength(0)
    expect(api.update_tag).toHaveBeenCalledTimes(1)
    expect(api.update_text).toHaveBeenCalledTimes(1)
    expect(api.update_notification).toHaveBeenCalledTimes(1)

    const tag_req = api.update_tag.mock.calls[0][0]
    expect(tag_req.tag.is_deleted).toBe(true)
    expect(tag_req.tag.update_device).toBe('test-device')
    expect(tag_req.tag.update_user).toBe('admin')
    expect(tag_req.tag.update_app).toBe('gkill')
  })

  it('参照しているReKyouとMiReKyouを論理削除する', async () => {
    const { api } = create_cascade_api({
      rekyous: { 'kmemo-1': [make_ref_stub('rekyou-1', 'kmemo-1')] },
      mirekyous: { 'kmemo-1': [make_ref_stub('mirekyou-1', 'kmemo-1', { data_type: 'mirekyou_create' })] },
    })

    const result = await run(make_kmemo_kyou('kmemo-1'), api)

    expect(api.update_rekyou).toHaveBeenCalledTimes(1)
    expect(api.update_mirekyou).toHaveBeenCalledTimes(1)
    expect(api.update_kmemo).toHaveBeenCalledTimes(1)
    expect(result.deleted_ids.sort()).toEqual(['kmemo-1', 'mirekyou-1', 'rekyou-1'])
  })

  it('多段の参照を再帰的に辿る', async () => {
    const { api } = create_cascade_api({
      rekyous: {
        'kmemo-1': [make_ref_stub('rekyou-B', 'kmemo-1')],
        'rekyou-B': [make_ref_stub('rekyou-C', 'rekyou-B')],
        'rekyou-C': [make_ref_stub('rekyou-D', 'rekyou-C')],
      },
    })

    const result = await run(make_kmemo_kyou('kmemo-1'), api)

    expect(result.deleted_ids.sort()).toEqual(['kmemo-1', 'rekyou-B', 'rekyou-C', 'rekyou-D'])
    expect(api.update_rekyou).toHaveBeenCalledTimes(3)
  })

  it('循環参照でも止まる', async () => {
    // A(kmemo-1) ← B、B ← A相当のReKyou。visited集合が無いと無限に回る
    const { api } = create_cascade_api({
      rekyous: {
        'kmemo-1': [make_ref_stub('rekyou-B', 'kmemo-1')],
        'rekyou-B': [make_ref_stub('rekyou-A2', 'rekyou-B')],
        'rekyou-A2': [make_ref_stub('rekyou-B', 'rekyou-A2')],
      },
    })

    const result = await run(make_kmemo_kyou('kmemo-1'), api)

    expect(result.deleted_ids.sort()).toEqual(['kmemo-1', 'rekyou-A2', 'rekyou-B'])
    expect(api.update_rekyou).toHaveBeenCalledTimes(2)
  })

  it('自己参照でも止まる', async () => {
    const { api } = create_cascade_api({
      rekyous: {
        'kmemo-1': [make_ref_stub('rekyou-self', 'kmemo-1')],
        'rekyou-self': [make_ref_stub('rekyou-self', 'rekyou-self')],
      },
    })

    const result = await run(make_kmemo_kyou('kmemo-1'), api)

    expect(result.deleted_ids.sort()).toEqual(['kmemo-1', 'rekyou-self'])
  })

  it('逆引きを全部終えてから削除し、Kyou自身は最後に消す', async () => {
    const { api, call_order } = create_cascade_api({
      rekyous: {
        'kmemo-1': [make_ref_stub('rekyou-B', 'kmemo-1')],
        'rekyou-B': [make_ref_stub('rekyou-C', 'rekyou-B')],
      },
    })

    await run(make_kmemo_kyou('kmemo-1'), api)

    const first_update_index = call_order.findIndex(c => c.startsWith('update_'))
    const last_discovery_index = call_order.map((c, i) => c.startsWith('get_') ? i : -1)
      .reduce((max, i) => Math.max(max, i), -1)
    // 参照先を消すと逆引きできなくなりうるので、discoveryは1本もupdateを投げる前に終わっている
    expect(last_discovery_index).toBeLessThan(first_update_index)

    // Kyou自身(update_kmemo)は参照元(update_rekyou)より後
    expect(call_order.indexOf('update_kmemo:kmemo-1')).toBeGreaterThan(call_order.indexOf('update_rekyou:rekyou-B'))
    expect(call_order.indexOf('update_kmemo:kmemo-1')).toBeGreaterThan(call_order.indexOf('update_rekyou:rekyou-C'))
  })

  it('ReKyou自身に付いているTagも消す', async () => {
    const { api } = create_cascade_api({
      rekyous: { 'kmemo-1': [make_ref_stub('rekyou-B', 'kmemo-1')] },
      tags: { 'rekyou-B': [make_meta_stub('tag-on-rekyou', 'rekyou-B')] },
    })

    await run(make_kmemo_kyou('kmemo-1'), api)

    expect(api.update_tag).toHaveBeenCalledTimes(1)
    expect(api.update_tag.mock.calls[0][0].tag.id).toBe('tag-on-rekyou')
  })

  it('1本失敗しても他は全部投げてエラーを集約する', async () => {
    const { api } = create_cascade_api({
      tags: {
        'kmemo-1': [make_meta_stub('tag-1', 'kmemo-1'), make_meta_stub('tag-2', 'kmemo-1')],
      },
      texts: { 'kmemo-1': [make_meta_stub('text-1', 'kmemo-1')] },
    })
    const failure = { error_code: 'ERR_TEST', error_message: 'failed', show_keep: true }
    api.update_tag.mockImplementationOnce(async () => ({ messages: [], errors: [failure] }))

    const result = await run(make_kmemo_kyou('kmemo-1'), api)

    expect(result.errors).toHaveLength(1)
    expect(result.errors[0].error_code).toBe('ERR_TEST')
    expect(api.update_tag).toHaveBeenCalledTimes(2)
    expect(api.update_text).toHaveBeenCalledTimes(1)
    expect(api.update_kmemo).toHaveBeenCalledTimes(1)
  })

  it('付随データの逆引きはforce_regetを立てる', async () => {
    // Service Workerがtarget_id単位でキャッシュしているので、立て忘れると古い一覧を読む
    const { api } = create_cascade_api()

    await run(make_kmemo_kyou('kmemo-1'), api)

    expect(api.get_tags_by_target_id.mock.calls[0][0].force_reget).toBe(true)
    expect(api.get_texts_by_target_id.mock.calls[0][0].force_reget).toBe(true)
    expect(api.get_notifications_by_target_id.mock.calls[0][0].force_reget).toBe(true)
  })

  it('消した全idのキャッシュを落とす', async () => {
    const { api } = create_cascade_api({
      rekyous: { 'kmemo-1': [make_ref_stub('rekyou-B', 'kmemo-1')] },
    })

    await run(make_kmemo_kyou('kmemo-1'), api)

    const cleared = vi.mocked(delete_gkill_kyou_cache).mock.calls.map(call => call[0]).sort()
    expect(cleared).toEqual(['kmemo-1', 'rekyou-B'])
  })

  it('削除済みのReKyouは辿らない', async () => {
    const { api } = create_cascade_api({
      rekyous: {
        'kmemo-1': [make_ref_stub('rekyou-deleted', 'kmemo-1', { is_deleted: true })],
      },
    })

    const result = await run(make_kmemo_kyou('kmemo-1'), api)

    expect(result.deleted_ids).toEqual(['kmemo-1'])
    expect(api.update_rekyou).not.toHaveBeenCalled()
  })

  it('同じidの履歴が返ってもupdate_timeが最新の1件だけを消す', async () => {
    const older = make_ref_stub('rekyou-B', 'kmemo-1', { update_time: base_time })
    const newer = make_ref_stub('rekyou-B', 'kmemo-1', { update_time: new Date(base_time.getTime() + 3600_000) })
    const { api } = create_cascade_api({ rekyous: { 'kmemo-1': [older, newer] } })

    await run(make_kmemo_kyou('kmemo-1'), api)

    expect(api.update_rekyou).toHaveBeenCalledTimes(1)
    expect(api.update_rekyou.mock.calls[0][0].rekyou.update_time.getTime()).not.toBe(base_time.getTime())
  })

  it('参照が深すぎるときは打ち切ってエラーを返す', async () => {
    // 上限32を超える長さの鎖を作る
    const rekyous: Record<string, Array<RefStub>> = {}
    let previous_id = 'kmemo-1'
    for (let i = 0; i < 40; i++) {
      const next_id = `rekyou-${i}`
      rekyous[previous_id] = [make_ref_stub(next_id, previous_id)]
      previous_id = next_id
    }
    const { api } = create_cascade_api({ rekyous })

    const result = await run(make_kmemo_kyou('kmemo-1'), api)

    expect(result.errors).toHaveLength(1)
    expect(result.errors[0].error_code).toBe('ERR900093')
    // 打ち切っても、そこまでに集めた分は消す
    expect(api.update_rekyou).toHaveBeenCalled()
  })

  it('共有画面では何もしない', async () => {
    const { api } = create_cascade_api({
      rekyous: { 'kmemo-1': [make_ref_stub('rekyou-B', 'kmemo-1')] },
    })

    const result = await cascade_delete_kyou({
      kyou: make_kmemo_kyou('kmemo-1') as never,
      gkill_api: api as never,
      application_config: { device: '', user_id: '', for_share_kyou: true } as never,
    })

    expect(result.deleted_ids).toEqual([])
    expect(api.update_kmemo).not.toHaveBeenCalled()
  })
})

describe('build_deleted_kyou_stub', () => {
  it('idだけを持つKyouを作る', () => {
    const stub = build_deleted_kyou_stub('kmemo-1')
    expect(stub.id).toBe('kmemo-1')
    expect(stub.is_deleted).toBe(true)
  })
})
