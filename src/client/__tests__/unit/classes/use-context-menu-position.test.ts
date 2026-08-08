/**
 * コンテキストメニュー共通の位置composableのテスト。
 *
 * 以前は各composableが innerWidth / innerHeight から位置を見積もっていたが、
 * 実寸を測っていないので項目数を足すたびに静かにずれていた。
 * 今はクリック座標をそのまま Vuetify の :target に渡し、はみ出しの回避は
 * Vuetify の connected location strategy（実測 + flip/shift）に任せる。
 * ここで検証するのは「クリック座標が素通しで target になること」だけでよい。
 */
import { describe, expect, it } from 'vitest'

import { useContextMenuPosition } from '@/classes/use-context-menu-position'

function make_event(x: number, y: number): MouseEvent {
    return new MouseEvent('contextmenu', { clientX: x, clientY: y })
}

describe('useContextMenuPosition', () => {
    it('初期状態は閉じていて原点', () => {
        const { is_show, menu_target } = useContextMenuPosition()
        expect(is_show.value).toBe(false)
        expect(menu_target.value).toEqual([0, 0])
    })

    it('open_at でクリック座標が target になり、開く', () => {
        const { is_show, menu_target, open_at } = useContextMenuPosition()
        open_at(make_event(731, 402))
        expect(menu_target.value).toEqual([731, 402])
        expect(is_show.value).toBe(true)
    })

    it('座標はクランプされない（ビューポート外の値もそのまま渡す）', () => {
        // クランプは Vuetify 側が実寸を測ってから行う。
        // ここで先に丸めると、測る前の値で二重に補正することになる
        const { menu_target, open_at } = useContextMenuPosition()
        open_at(make_event(99999, -50))
        expect(menu_target.value).toEqual([99999, -50])
    })

    it('開き直すと最新のクリック座標に追従する', () => {
        const { menu_target, open_at } = useContextMenuPosition()
        open_at(make_event(10, 20))
        open_at(make_event(300, 400))
        expect(menu_target.value).toEqual([300, 400])
    })

    it('position_x / position_y を直接動かしても target に反映される', () => {
        const { position_x, position_y, menu_target } = useContextMenuPosition()
        position_x.value = 5
        position_y.value = 7
        expect(menu_target.value).toEqual([5, 7])
    })

    it('インスタンスごとに状態が独立している', () => {
        const a = useContextMenuPosition()
        const b = useContextMenuPosition()
        a.open_at(make_event(1, 2))
        expect(b.is_show.value).toBe(false)
        expect(b.menu_target.value).toEqual([0, 0])
    })
})
