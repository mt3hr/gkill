'use strict'

import { ref, type Ref } from 'vue'
import type { ApplicationConfig } from '@/classes/datas/config/application-config'
import { tag_exists_in_tag_struct } from '@/classes/tag-struct'
import type { ComponentRef } from '@/classes/component-ref'

/**
 * 新しいタグ名を検出したときの確認ゲート。
 *
 * タグ名は自由入力なので、打ち間違いがそのまま新しいタグとして生えてしまう。
 * サーバ側は名前を検証しない（`usecase/tag.go` の重複チェックはタグIDだけを見る）ので、
 * 保存前にクライアントで確認を挟む。
 *
 * 判定は ApplicationConfig のタグツリーに対して行う。ツリーは起動時の
 * `append_not_found_tags()` で実在するタグ名が流し込まれているので、追加のリクエストは要らない。
 *
 * 板名版の `use-confirm-unknown-mi-board.ts` と対の実装。あちらのコメントが
 * 「タグ版は同じマークアップを各Viewに手書きで複製している」と書いていたものを、
 * 呼び出し元が増えるのに合わせてここへ寄せた。
 */
export function useConfirmUnknownTag(options: {
    application_config: () => ApplicationConfig,
}) {
    const { application_config } = options

    // ── Template refs ──
    const confirm_unknown_tag_dialog = ref<ComponentRef | null>(null)

    // ── State refs ──
    const unknown_tags: Ref<Array<string>> = ref([])

    // ── Business logic ──
    /** 渡されたタグ名のうち、タグツリーに無いものを重複なく返す */
    function collect_unknown_tags(candidates: Array<string>): Array<string> {
        const struct = application_config().tag_struct
        const unknown = new Array<string>()
        for (const candidate of candidates) {
            if (candidate === "") {
                continue
            }
            if (unknown.includes(candidate)) {
                continue
            }
            if (tag_exists_in_tag_struct(candidate, struct)) {
                continue
            }
            unknown.push(candidate)
        }
        return unknown
    }

    function open_confirm(unknown: Array<string>): void {
        unknown_tags.value = unknown
        confirm_unknown_tag_dialog.value?.show()
    }

    /** 確認を閉じたあとの後片付け。ダイアログ自身は既に閉じている */
    function close_confirm(): void {
        unknown_tags.value = []
    }

    // ── Return ──
    return {
        // Template refs
        confirm_unknown_tag_dialog,

        // State
        unknown_tags,

        // Business logic
        collect_unknown_tags,
        open_confirm,
        close_confirm,
    }
}
