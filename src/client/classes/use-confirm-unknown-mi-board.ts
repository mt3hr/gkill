'use strict'

import { ref, type Ref } from 'vue'
import type { ApplicationConfig } from '@/classes/datas/config/application-config'
import { append_mi_board_to_struct, board_exists_in_mi_board_struct } from '@/classes/mi-board-struct'
import type { ComponentRef } from '@/classes/component-ref'

/**
 * 新しい板名を検出したときの確認ゲート。
 *
 * 板はサーバに独立したエンティティとして存在せず、
 * 「その名前のタスクが1件でもあること」から導出される
 * （`usecase/mi.go` の GetMiBoardList が Mi と MiReKyou の board_name を集めるだけ）。
 * つまりサーバ側では検証しようがなく、板名の打ち間違いは無言で新しい板を生やす。
 * タグの `CONFIRM_UNKNOWN_TAG_*`（use-add-tag-view.ts / use-kftl-view.ts）と同じ形で、
 * クライアント側で保存前に確認を挟む。
 *
 * 判定は ApplicationConfig の板ツリーに対して行う（タグ版が tag_struct を見るのと同じ）。
 * ツリーは起動時の `append_not_found_mi_boards()` で実在する板が流し込まれているので、
 * 追加のリクエストは要らない。
 * 「ツリーに出ている板」＝ユーザが持っている板なので、そこへ入れるときに
 * 「新しい板です」と聞かないのが正しい（タスクが0件になった板もツリーには残る）。
 * 仮想の「すべて」ノードもツリーに居るので既存扱いになるが、
 * これは mi ページに常時出ている列なので「新しい板です」と聞くほうが不自然。
 */
export function useConfirmUnknownMiBoard(options: {
    application_config: () => ApplicationConfig,
}) {
    const { application_config } = options

    // ── Template refs ──
    const confirm_unknown_mi_board_dialog = ref<ComponentRef | null>(null)

    // ── State refs ──
    const unknown_mi_boards: Ref<Array<string>> = ref([])

    // ── Business logic ──
    /** 渡された板名のうち、板ツリーに無いものを重複なく返す */
    function collect_unknown_mi_boards(candidates: Array<string>): Array<string> {
        const struct = application_config().mi_board_struct
        const unknown = new Array<string>()
        for (const candidate of candidates) {
            // 板名を書かなかったときは既定の板にフォールバックするだけなので、
            // ユーザが新しい板名を入力したことにはならない
            if (candidate === "") {
                continue
            }
            if (unknown.includes(candidate)) {
                continue
            }
            if (board_exists_in_mi_board_struct(candidate, struct)) {
                continue
            }
            unknown.push(candidate)
        }
        return unknown
    }

    function open_confirm(unknown: Array<string>): void {
        unknown_mi_boards.value = unknown
        confirm_unknown_mi_board_dialog.value?.show()
    }

    /** 確認を閉じたあとの後片付け。ダイアログ自身は既に閉じている */
    function close_confirm(): void {
        unknown_mi_boards.value = []
    }

    /**
     * 確認を通した板をその場で板ツリーへ足す。
     * 判定の正がツリーなので、足しておかないと同じ板へ2件目を入れるときにまた確認が出る。
     * `close_confirm()` より先に呼ぶこと（`unknown_mi_boards` を読むため）
     */
    function remember_confirmed_mi_boards(): void {
        const struct = application_config().mi_board_struct
        for (const board_name of unknown_mi_boards.value) {
            append_mi_board_to_struct(board_name, struct)
        }
    }

    // ── Return ──
    return {
        // Template refs
        confirm_unknown_mi_board_dialog,

        // State
        unknown_mi_boards,

        // Business logic
        collect_unknown_mi_boards,
        open_confirm,
        close_confirm,
        remember_confirmed_mi_boards,
    }
}
