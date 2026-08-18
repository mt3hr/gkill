/**
 * メモ帳ウィンドウの一覧を持つホスト。
 *
 * ＋メニューを押すたびに1枚増える（従来は「開いていれば再フォーカスするだけ」だった）。
 * 位置とサイズの保存キーはスロット番号で分けるので、番号の払い出しをここで固定する。
 */
import { describe, test, expect, vi, beforeEach } from 'vitest'

import {
    KFTL_DIALOG_MAX_COUNT,
    reset_kftl_dialog_host_slots_for_test,
    useKftlDialogHost,
} from '@/classes/use-kftl-dialog-host'

function make_host() {
    const emits = vi.fn()
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    return useKftlDialogHost({ emits: emits as any })
}

// スロット番号はモジュール共有（ホスト単位ではない）。テスト間で持ち越さないよう毎回戻す
beforeEach(() => {
    reset_kftl_dialog_host_slots_for_test()
})

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

    /**
     * slot_index は `kftl-dialog` / `kftl-dialog-2` … という useFloatingDialog の
     * 保存キーそのもの。ホスト単位で採番すると、2つのホストがそれぞれ 0 を配って
     * 位置とサイズを奪い合う。ポート(rudbeckia)ではポート自身とホストした各画面が
     * 同時に KFTLDialogHost を持つので、必ず衝突する
     */
    test('2つのホストが同じスロット番号を配らない', () => {
        const host_a = make_host()
        const host_b = make_host()

        host_a.show()
        host_b.show()
        host_a.show()

        const all_slots = [
            ...host_a.opened_dialogs.value.map(dialog => dialog.slot_index),
            ...host_b.opened_dialogs.value.map(dialog => dialog.slot_index),
        ]
        expect(new Set(all_slots).size, 'ホストをまたいでスロット番号が重複している').toBe(all_slots.length)
    })

    test('上限はホストをまたいで効く', () => {
        const host_a = make_host()
        const host_b = make_host()

        for (let i = 0; i < KFTL_DIALOG_MAX_COUNT; i++) {
            host_a.show()
        }
        host_b.show()

        expect(
            host_a.opened_dialogs.value.length + host_b.opened_dialogs.value.length,
            'ホストごとに上限が効いていて合計が上限を超えている',
        ).toBe(KFTL_DIALOG_MAX_COUNT)
    })

    test('閉じたスロット番号は別のホストが使える', () => {
        const host_a = make_host()
        const host_b = make_host()

        host_a.show()
        host_a.show()
        host_a.close(host_a.opened_dialogs.value[0].id)

        host_b.show()

        expect(host_b.opened_dialogs.value[0].slot_index).toBe(0)
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
