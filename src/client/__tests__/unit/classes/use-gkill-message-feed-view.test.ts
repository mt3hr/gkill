import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest'
import { i18n } from '../../helpers/setup-i18n'

vi.mock('@/i18n', () => ({ i18n }))

import { push_errors, reset_feed_items, useGkillMessageFeed } from '@/classes/use-gkill-message-feed'
import { useGkillMessageFeedView } from '@/classes/use-gkill-message-feed-view'
import { GkillError } from '@/classes/api/gkill-error'
import { GkillMessageCodes } from '@/classes/api/message/gkill_message'

// 画面右上のエラー一覧（gkill-message-feed-view.vue）の表示ロジック。
// 一覧の中身は use-gkill-message-feed.test.ts が守る。ここで守るのは
//   - role="alert" はエラーだけ（E2E と支援技術がこの形で掴む）。warning / info に付けると
//     「検索完了」のたびに読み上げが割り込む
//   - コピーはクリップボードが使えない環境（http の LAN アクセス等）で prompt へ落ち、
//     「コピーしました」を出さない

function make_error(message: string): GkillError {
  const error = new GkillError()
  error.error_code = 'ERR000001'
  error.error_message = message
  error.error_kind = 'server'
  return error
}

describe('useGkillMessageFeedView', () => {
  const { feed_items } = useGkillMessageFeed()

  beforeEach(() => {
    vi.useFakeTimers()
    reset_feed_items()
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.unstubAllGlobals()
  })

  test('role="alert" と色はエラーだけ、warning は色だけ、info はどちらも無し', () => {
    const view = useGkillMessageFeedView()
    expect(view.alert_role('error')).toBe('alert')
    expect(view.alert_role('warning')).toBeUndefined()
    expect(view.alert_role('info')).toBeUndefined()
    expect(view.alert_color('error')).toBe('error')
    expect(view.alert_color('warning')).toBe('warning')
    expect(view.alert_color('info')).toBeUndefined()
  })

  test('閉じるとフィードから消え、has_items が追随する', () => {
    const view = useGkillMessageFeedView()
    expect(view.has_items.value).toBe(false)
    push_errors([make_error('落ちた')])
    expect(view.has_items.value).toBe(true)
    expect(view.show_footer(feed_items.value[0])).toBe(true)

    view.onClickClose(feed_items.value[0].id)
    expect(feed_items.value).toHaveLength(0)
    expect(view.has_items.value).toBe(false)
  })

  test('コピーできたら「コピーしました」の info が1枚増える', async () => {
    const write_text = vi.fn().mockResolvedValue(undefined)
    vi.stubGlobal('navigator', { clipboard: { writeText: write_text } })
    const view = useGkillMessageFeedView()
    push_errors([make_error('落ちた')])

    await view.onClickCopy(feed_items.value[0])

    expect(write_text).toHaveBeenCalledTimes(1)
    expect(write_text.mock.calls[0][0]).toContain('落ちた')
    const copied = feed_items.value.find((item) => item.code === GkillMessageCodes.copied_error_detail)
    expect(copied, '「コピーしました」が出ていない').toBeDefined()
    expect(copied!.level).toBe('info')
  })

  test('クリップボードが使えなければ prompt に本文を出し、「コピーしました」は出さない', async () => {
    vi.stubGlobal('navigator', { clipboard: { writeText: vi.fn().mockRejectedValue(new Error('NotAllowedError')) } })
    const prompt = vi.fn()
    vi.stubGlobal('prompt', prompt)
    window.prompt = prompt
    const view = useGkillMessageFeedView()
    push_errors([make_error('落ちた')])

    await view.onClickCopy(feed_items.value[0])

    expect(prompt).toHaveBeenCalledTimes(1)
    expect(prompt.mock.calls[0][1]).toContain('落ちた')
    expect(feed_items.value.some((item) => item.code === GkillMessageCodes.copied_error_detail)).toBe(false)
  })
})
