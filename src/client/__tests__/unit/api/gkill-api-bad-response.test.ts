import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest'
import { i18n } from '../../helpers/setup-i18n'

vi.mock('@/i18n', () => ({ i18n }))
vi.mock('@/classes/delete-gkill-cache', () => ({
  default: vi.fn().mockResolvedValue(undefined),
  delete_gkill_config_cache: vi.fn().mockResolvedValue(undefined),
  delete_gkill_all_tag_names_cache: vi.fn().mockResolvedValue(undefined),
  delete_gkill_attached_datas_cache: vi.fn().mockResolvedValue(undefined),
}))

import { GkillAPI } from '@/classes/api/gkill-api'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import { GetKmemoRequest } from '@/classes/api/req_res/get-kmemo-request'

// gkill-api.ts はステータスを見ずに res.json() する設計。プロキシの HTML エラーページや、
// 止まっているサーバの代わりに何かが返す本文は JSON ではないので、以前は SyntaxError が
// unhandledrejection に落ちて画面に何も出なかった。gkill_fetch が JSON 以外を
// bad_response のエラー応答に置き換えることを固定する。
describe('gkill_fetch の JSON 以外の応答', () => {
  const originalFetch = globalThis.fetch

  beforeEach(() => {
    globalThis.fetch = vi.fn()
  })

  afterEach(() => {
    globalThis.fetch = originalFetch
  })

  test('text/html の 502 は bad_response のエラー（errors / messages とも配列）になる', async () => {
    ;(globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue(
      new Response('<html><body>502 Bad Gateway</body></html>', {
        status: 502,
        headers: { 'Content-Type': 'text/html; charset=utf-8' },
      }),
    )
    const res = await GkillAPI.get_instance().get_kmemo(new GetKmemoRequest())

    expect(res.errors).toHaveLength(1)
    expect(res.errors[0].error_code).toBe(GkillErrorCodes.bad_response)
    expect(res.errors[0].error_message).toContain('502')
    expect(res.errors[0].error_kind).toBe('server')
    expect(res.messages).toEqual([])
  })

  test('application/json はそのまま通す（ステータスが 500 でも本文を読む）', async () => {
    ;(globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue(
      new Response(JSON.stringify({ messages: [], errors: [{ error_code: 'ERR000024', error_message: 'x', error_kind: 'server' }] }), {
        status: 500,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
    const res = await GkillAPI.get_instance().get_kmemo(new GetKmemoRequest())
    expect(res.errors[0].error_code).toBe('ERR000024')
  })

  test('Content-Type の無いモック応答は従来どおり素通しする', async () => {
    ;(globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValue({
      json: () => Promise.resolve({ messages: null, errors: null, kmemo: null }),
    })
    const res = await GkillAPI.get_instance().get_kmemo(new GetKmemoRequest())
    expect(res.errors).toBeNull()
  })
})
