/**
 * gkill-tx.ts のテスト。
 *
 * 追加/編集/削除画面は複数の書き込みを tx_id で束ねて commit_tx で確定する（ADR-0410）。
 * 「1本でも失敗したら discard して何も残さない」「commit が失敗しても discard する」を固定する。
 */
import { describe, expect, it, vi } from 'vitest'

vi.mock('@/classes/api/gkill-api', () => ({
  GkillAPI: {
    get_instance: vi.fn(() => ({ get_session_id: vi.fn(() => 'mock-session') })),
    get_gkill_api: vi.fn(() => ({ get_session_id: vi.fn(() => 'mock-session') })),
  },
}))

vi.mock('@/classes/delete-gkill-cache', () => ({
  default: vi.fn().mockResolvedValue(undefined),
}))

import { discard_tx, fetch_committed_kyou, run_in_tx } from '@/classes/gkill-tx'
import delete_gkill_kyou_cache from '@/classes/delete-gkill-cache'

const ok = { messages: [], errors: [] }
const failure = { error_code: 'ERR_TEST', error_message: 'ng', show_keep: true }

function create_api() {
  return {
    generate_uuid: vi.fn(() => 'tx-1'),
    commit_tx: vi.fn(async () => ({ committed: [], ...ok })),
    discard_tx: vi.fn(async () => ok),
    get_kyou: vi.fn(async () => ({ kyou_histories: [{ id: 'kyou-1' }], ...ok })),
  }
}

describe('run_in_tx', () => {
  it('work が通れば commit_tx を1回呼び、discard_tx は呼ばない', async () => {
    const api = create_api()
    const seen = new Array<string>()

    const result = await run_in_tx(api as never, async (tx_id) => {
      seen.push(tx_id)
      return []
    })

    expect(seen).toEqual(['tx-1'])
    expect(result).toEqual({ committed: true, errors: [] })
    expect(api.commit_tx).toHaveBeenCalledTimes(1)
    expect(api.commit_tx.mock.calls[0][0].tx_id).toBe('tx-1')
    expect(api.discard_tx).not.toHaveBeenCalled()
  })

  it('work がエラーを返したら commit せず discard_tx する', async () => {
    const api = create_api()

    const result = await run_in_tx(api as never, async () => [failure as never])

    expect(result.committed).toBe(false)
    expect(result.errors.map(e => e.error_code)).toEqual(['ERR_TEST'])
    expect(api.commit_tx).not.toHaveBeenCalled()
    expect(api.discard_tx).toHaveBeenCalledTimes(1)
    expect(api.discard_tx.mock.calls[0][0].tx_id).toBe('tx-1')
  })

  it('work が throw したら discard_tx してから投げ直す', async () => {
    const api = create_api()

    await expect(run_in_tx(api as never, async () => { throw new Error('boom') })).rejects.toThrow('boom')

    expect(api.commit_tx).not.toHaveBeenCalled()
    expect(api.discard_tx).toHaveBeenCalledTimes(1)
  })

  it('commit_tx が失敗したら discard_tx して committed=false（サーバには何も書かれていない）', async () => {
    const api = create_api()
    api.commit_tx.mockResolvedValue({ committed: [], messages: [], errors: [{ error_code: 'ERR000419', error_message: 'rolled back', show_keep: true }] } as never)

    const result = await run_in_tx(api as never, async () => [])

    expect(result.committed).toBe(false)
    expect(result.errors.map(e => e.error_code)).toEqual(['ERR000419'])
    expect(api.discard_tx).toHaveBeenCalledTimes(1)
  })

  // サーバは成功時 errors を null で返す（omitempty 無し）。null のまま展開すると TypeError
  it('commit_tx の errors が null でも成功として扱う', async () => {
    const api = create_api()
    api.commit_tx.mockResolvedValue({ committed: [], messages: null, errors: null } as never)

    const result = await run_in_tx(api as never, async () => [])

    expect(result.committed).toBe(true)
  })
})

describe('discard_tx', () => {
  it('discard_tx のエラーはそのまま返し、通信断でも投げない', async () => {
    const api = create_api()
    api.discard_tx.mockResolvedValueOnce({ messages: [], errors: [failure] } as never)
    expect((await discard_tx(api as never, 'tx-1')).map(e => e.error_code)).toEqual(['ERR_TEST'])

    api.discard_tx.mockRejectedValueOnce(new Error('offline'))
    await expect(discard_tx(api as never, 'tx-1')).resolves.toEqual([])
  })
})

describe('fetch_committed_kyou', () => {
  it('SW キャッシュを捨ててから get_kyou で引く', async () => {
    const api = create_api()
    vi.mocked(delete_gkill_kyou_cache).mockClear()

    const kyou = await fetch_committed_kyou(api as never, 'kyou-1')

    expect(kyou).toEqual({ id: 'kyou-1' })
    expect(delete_gkill_kyou_cache).toHaveBeenCalledWith('kyou-1')
    expect(api.get_kyou.mock.calls[0][0].id).toBe('kyou-1')
  })

  it('引けなければ null（呼び出し元は一覧全体の引き直しへ落とす）', async () => {
    const api = create_api()
    api.get_kyou.mockResolvedValueOnce({ kyou_histories: [], messages: [], errors: [] })
    expect(await fetch_committed_kyou(api as never, 'kyou-1')).toBeNull()

    api.get_kyou.mockResolvedValueOnce({ kyou_histories: [], messages: [], errors: [failure] } as never)
    expect(await fetch_committed_kyou(api as never, 'kyou-1')).toBeNull()
  })
})
