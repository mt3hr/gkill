/**
 * イベントリレー束のテスト。
 *
 * 以前は各composable / 各ダイアログが `{ 'deleted_kyou': ... }` を手書きで並べており、
 * 63箇所で取りこぼしが起きていた（大半が `requested_open_rykv_dialog`）。
 * 束を1箇所で生成するようにしたので、その生成が全イベントを漏れなく張ることを確認する。
 */
import { describe, expect, it, vi } from 'vitest'

import {
  build_kyou_dialog_relay,
  build_kyou_view_relay,
  kyou_dialog_relay_event_names,
  kyou_view_relay_event_names,
} from '@/classes/kyou-view-relay'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'

function make_emits() {
  return vi.fn() as unknown as KyouViewEmits & ReturnType<typeof vi.fn>
}

describe('build_kyou_view_relay', () => {
  it('ビュー層で中継する18イベントを全部張る', () => {
    const relay = build_kyou_view_relay(make_emits())

    expect(Object.keys(relay).sort()).toEqual([...kyou_view_relay_event_names].sort())
    expect(kyou_view_relay_event_names).toHaveLength(18)
  })

  // ビュー層は自分がフォーカスの発火源。入れ子のKyouViewのぶんまで持ち上げると二重に発火する
  it('フォーカス系は張らない', () => {
    const relay = build_kyou_view_relay(make_emits())

    expect(relay).not.toHaveProperty('focused_kyou')
    expect(relay).not.toHaveProperty('clicked_kyou')
  })

  // 子はクローズ要求を上げてこない。ダイアログが自分で hide() に繋ぐ設計
  it('requested_close_dialog は張らない', () => {
    const relay = build_kyou_view_relay(make_emits())

    expect(relay).not.toHaveProperty('requested_close_dialog')
  })

  it('各ハンドラは同名のイベントへ引数をそのまま素通しする', () => {
    const emits = make_emits()
    const relay = build_kyou_view_relay(emits)

    for (const event_name of kyou_view_relay_event_names) {
      emits.mockClear()
      const handler = relay[event_name] as (...args: Array<unknown>) => void
      handler('first', 'second')
      expect(emits, `${event_name} が素通しされていない`).toHaveBeenCalledWith(event_name, 'first', 'second')
    }
  })

  it('overrides を渡したイベントだけ差し替わる', () => {
    const emits = make_emits()
    const replaced = vi.fn()
    const relay = build_kyou_view_relay(emits, { deleted_kyou: replaced })

    const deleted_kyou = relay.deleted_kyou as (...args: Array<unknown>) => void
    deleted_kyou('kyou')
    expect(replaced).toHaveBeenCalledWith('kyou')
    expect(emits).not.toHaveBeenCalled()

    const updated_kyou = relay.updated_kyou as (...args: Array<unknown>) => void
    updated_kyou('other')
    expect(emits).toHaveBeenCalledWith('updated_kyou', 'other')
  })
})

describe('build_kyou_dialog_relay', () => {
  it('ビュー層の18イベントにフォーカス系2件を足した20イベントを張る', () => {
    const relay = build_kyou_dialog_relay(make_emits())

    expect(Object.keys(relay).sort()).toEqual([...kyou_dialog_relay_event_names].sort())
    expect(kyou_dialog_relay_event_names).toHaveLength(20)
    for (const event_name of kyou_view_relay_event_names) {
      expect(relay, `${event_name} が欠けている`).toHaveProperty(event_name)
    }
    expect(relay).toHaveProperty('focused_kyou')
    expect(relay).toHaveProperty('clicked_kyou')
  })

  it('クリックでフォーカスも動かす override を差し込める', () => {
    const emits = make_emits()
    const relay = build_kyou_dialog_relay(emits, {
      clicked_kyou: (kyou) => { emits('focused_kyou', kyou); emits('clicked_kyou', kyou) },
    })

    const clicked_kyou = relay.clicked_kyou as (...args: Array<unknown>) => void
    clicked_kyou('kyou')
    expect(emits).toHaveBeenCalledWith('focused_kyou', 'kyou')
    expect(emits).toHaveBeenCalledWith('clicked_kyou', 'kyou')
  })
})
