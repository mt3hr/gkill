import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest'
import { i18n } from '../../helpers/setup-i18n'

vi.mock('@/i18n', () => ({ i18n }))

import {
  push_errors,
  push_messages,
  push_client_exception,
  close_feed_item,
  format_feed_item_for_copy,
  reset_feed_items,
  useGkillMessageFeed,
  info_auto_close_milli_seconds,
} from '@/classes/use-gkill-message-feed'
import { GkillError } from '@/classes/api/gkill-error'
import { GkillMessage } from '@/classes/api/gkill-message'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'

// 画面右上のフィード。2026-09-15 まではサーバ由来のエラーが「閉じられない・2.5秒で消える」で、
// 本物の障害ほど早く消えていた。ここで守るのは:
//   - エラーと warning は閉じるまで残る（自動で消えるのは info だけ）
//   - 同じエラーの連打は1枚にまとまる（×N）
//   - サーバの error_kind / reason からヒント（次の一手）が付く
//   - null / 空 / 中断（canceled）は何も出さない

function make_error(code: string, message: string, kind = '', reason = ''): GkillError {
  const error = new GkillError()
  error.error_code = code
  error.error_message = message
  error.error_kind = kind
  error.reason = reason
  return error
}

function make_message(code: string, message: string, level = 'info'): GkillMessage {
  const gkill_message = new GkillMessage()
  gkill_message.message_code = code
  gkill_message.message = message
  gkill_message.level = level
  return gkill_message
}

describe('useGkillMessageFeed', () => {
  const { feed_items } = useGkillMessageFeed()

  beforeEach(() => {
    vi.useFakeTimers()
    reset_feed_items()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  test('エラーは閉じるまで残り、閉じられる', () => {
    push_errors([make_error('ERR000023', 'メモ追加に失敗しました', 'server')])
    expect(feed_items.value).toHaveLength(1)
    expect(feed_items.value[0].level).toBe('error')
    expect(feed_items.value[0].closable, 'サーバ由来のエラーが閉じられない（旧: show_keep が無く undefined）').toBe(true)

    vi.advanceTimersByTime(60_000)
    expect(feed_items.value, 'エラーが勝手に消えている').toHaveLength(1)

    close_feed_item(feed_items.value[0].id)
    expect(feed_items.value).toHaveLength(0)
  })

  test('info は既定時間で自動的に消える', () => {
    push_messages([make_message('MSG000007', 'メモを追加しました')])
    expect(feed_items.value).toHaveLength(1)
    expect(feed_items.value[0].closable).toBe(false)

    vi.advanceTimersByTime(info_auto_close_milli_seconds + 1)
    expect(feed_items.value).toHaveLength(0)
  })

  test('warning は閉じるまで残る（読み込めなかった rep の警告を「検索完了」と同じ2.5秒で消さない）', () => {
    push_messages([make_message('MSG000090', '記録保管場所を読み込めませんでした (broken_rep)', 'warning')])
    expect(feed_items.value[0].level).toBe('warning')
    expect(feed_items.value[0].closable).toBe(true)

    vi.advanceTimersByTime(60_000)
    expect(feed_items.value).toHaveLength(1)
  })

  test('同じコード+本文の連打は1枚にまとまり ×N になる', () => {
    push_errors([make_error('ERR900088', 'ネットワークエラー', 'network')])
    push_errors([make_error('ERR900088', 'ネットワークエラー', 'network')])
    push_errors([make_error('ERR900088', 'ネットワークエラー', 'network')])
    expect(feed_items.value).toHaveLength(1)
    expect(feed_items.value[0].count).toBe(3)

    // 本文が違えば別のカード
    push_errors([make_error('ERR900088', '別の文面', 'network')])
    expect(feed_items.value).toHaveLength(2)
  })

  test('reason があれば reason のヒント、無ければ kind のヒントが付く', () => {
    push_errors([make_error('ERR000023', 'メモ追加に失敗しました', 'config', 'write_rep_missing')])
    expect(feed_items.value[0].hint).toBe(i18n.global.t('ERROR_HINT_REASON_WRITE_REP_MISSING'))
    expect(feed_items.value[0].reason).toBe('write_rep_missing')

    push_errors([make_error('ERR000070', 'メモが見つかりませんでした', 'not_found')])
    expect(feed_items.value[1].hint).toBe(i18n.global.t('ERROR_HINT_KIND_NOT_FOUND'))
  })

  test('クライアントの入力検証（kind 無し）にはヒントを付けない', () => {
    push_errors([make_error('ERR900013', 'タイトルが空です')])
    expect(feed_items.value[0].hint).toBe('')
  })

  test('null・空配列・本文の無いエラー・中断（canceled）は何も出さない', () => {
    push_errors(null)
    push_errors(undefined)
    push_errors([])
    push_errors([make_error('ERR000001', '')])
    push_errors([make_error('ERR000410', '記録の取得に失敗しました', 'server', 'canceled')])
    push_messages(null)
    push_messages([make_message('MSG000025', '')])
    expect(feed_items.value).toHaveLength(0)
  })

  test('握られなかった例外はコード付きのエラーとして出て、ヒントは再読込の案内', () => {
    push_client_exception(new TypeError('Cannot read properties of null'))
    expect(feed_items.value).toHaveLength(1)
    expect(feed_items.value[0].code).toBe(GkillErrorCodes.unexpected_client_error)
    expect(feed_items.value[0].message).toContain('TypeError: Cannot read properties of null')
    expect(feed_items.value[0].hint).toBe(i18n.global.t('ERROR_HINT_UNEXPECTED_CLIENT_ERROR'))
    expect(feed_items.value[0].closable).toBe(true)
  })

  test('コピー用テキストにコード・reason・本文・ヒント・時刻・パスが載る', () => {
    push_errors([make_error('ERR000023', 'メモ追加に失敗しました', 'config', 'write_rep_missing')])
    const text = format_feed_item_for_copy(feed_items.value[0], '/rykv')
    expect(text).toContain('ERR000023 write_rep_missing')
    expect(text).toContain('kind: config')
    expect(text).toContain('メモ追加に失敗しました')
    expect(text).toContain(i18n.global.t('ERROR_HINT_REASON_WRITE_REP_MISSING'))
    expect(text).toContain('/rykv')
    expect(text).toMatch(/\d{4}-\d{2}-\d{2}T/)
  })
})
