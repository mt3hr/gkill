/**
 * メモ帳ウィンドウの一覧を持つホスト。
 *
 * ＋メニューを押すたびに1枚増える（従来は「開いていれば再フォーカスするだけ」だった）。
 * 位置とサイズの保存キーはスロット番号で分けるので、番号の払い出しをここで固定する。
 */
import { describe, test, expect, vi } from 'vitest'

import { KFTL_DIALOG_MAX_COUNT, useKftlDialogHost } from '@/classes/use-kftl-dialog-host'

function make_host() {
    const emits = vi.fn()
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    return useKftlDialogHost({ emits: emits as any })
}

describe('useKftlDialogHost', () => {
    test('最初は1枚も開いていない', () => {
        const host = make_host()
        expect(host.opened_dialogs.value.length).toBe(0)
    })

    test('show() を呼ぶたびにウィンドウが増える', () => {
        const host = make_host()

        host.show()
        host.show()
        host.show()

        expect(host.opened_dialogs.value.length).toBe(3)
        expect(host.opened_dialogs.value.map(dialog => dialog.slot_index)).toEqual([0, 1, 2])
    })

    test('ウィンドウごとに別のidを持つ', () => {
        const host = make_host()
        host.show()
        host.show()

        const ids = host.opened_dialogs.value.map(dialog => dialog.id)
        expect(new Set(ids).size).toBe(2)
    })

    test('closed で一覧から外れる', () => {
        const host = make_host()
        host.show()
        host.show()
        const first_id = host.opened_dialogs.value[0].id

        host.close(first_id)

        expect(host.opened_dialogs.value.map(dialog => dialog.id)).not.toContain(first_id)
        expect(host.opened_dialogs.value.length).toBe(1)
    })

    test('知らないidを閉じようとしても何も起きない', () => {
        const host = make_host()
        host.show()

        host.close('no-such-dialog')

        expect(host.opened_dialogs.value.length).toBe(1)
    })

    // スロット番号は位置・サイズの保存キーになる。空いた番号を使い回さないと
    // 番号が青天井に増えて、保存済みの位置と結び付かなくなる
    test('スロット番号は空いている最小のものを取る', () => {
        const host = make_host()
        host.show()
        host.show()
        host.show()

        host.close(host.opened_dialogs.value[1].id)
        expect(host.opened_dialogs.value.map(dialog => dialog.slot_index)).toEqual([0, 2])

        host.show()
        expect(host.opened_dialogs.value.map(dialog => dialog.slot_index)).toEqual([0, 2, 1])
    })

    test('上限を超えて増えない', () => {
        const host = make_host()
        for (let i = 0; i < KFTL_DIALOG_MAX_COUNT + 3; i++) {
            host.show()
        }

        expect(host.opened_dialogs.value.length).toBe(KFTL_DIALOG_MAX_COUNT)
    })

    test('上限に達したら最後のウィンドウへフォーカスを移す', () => {
        const host = make_host()
        for (let i = 0; i < KFTL_DIALOG_MAX_COUNT; i++) {
            host.show()
        }
        const focus_newest = vi.fn()
        host.dialog_refs.value = host.opened_dialogs.value.map(() => ({ show: vi.fn() }))
        host.dialog_refs.value[host.dialog_refs.value.length - 1] = { show: focus_newest }

        host.show()

        expect(focus_newest).toHaveBeenCalledTimes(1)
        expect(host.opened_dialogs.value.length).toBe(KFTL_DIALOG_MAX_COUNT)
    })

    // どのウィンドウから来たかに依らないので、同じハンドラ束を全枚数へ配れる
    test('ウィンドウのイベントはそのまま上へ流す', () => {
        const emits = vi.fn()
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        const host = useKftlDialogHost({ emits: emits as any })

        host.dialogRelayHandlers.requested_reload_list()
        const saved_at = new Date(0)
        host.dialogRelayHandlers.saved_kyou_by_kftl(saved_at)

        expect(emits).toHaveBeenCalledWith('requested_reload_list')
        expect(emits).toHaveBeenCalledWith('saved_kyou_by_kftl', saved_at)
    })
})
