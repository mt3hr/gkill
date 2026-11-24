'use strict'

import type { GkillError } from "@/classes/api/gkill-error"
import type { GkillMessage } from "@/classes/api/gkill-message"
import type { Kyou } from "@/classes/datas/kyou"
import type { Tag } from "@/classes/datas/tag"

/**
 * mi の列（`mi-columns-view.vue`）がラッパへ上げるもの。
 *
 * 前半は記録・タグ・テキスト・通知の CRUD の中継で、ラッパがさらにページへ上げる。
 * 後半は「ラッパが持っているものを開いてほしい」という依頼 ——
 * 列は編集系のダイアログも FAB も持たないので、開く相手を知らない。
 *
 * rykv の対になる実装は `rykv-columns-view-emits.ts`（**修正は必ず両方へ**）。
 */
export interface MiColumnsViewEmits {
    (e: 'received_messages', message: Array<GkillMessage>): void
    (e: 'received_errors', errors: Array<GkillError>): void
    (e: 'registered_kyou', registered_kyou: Kyou): void
    (e: 'updated_kyou', updated_kyou: Kyou): void
    (e: 'deleted_kyou', deleted_kyou: Kyou): void
    (e: 'registered_tag', registred_tag: Tag): void
    (e: 'updated_tag', updated_tag: Tag): void
    (e: 'deleted_tag', deleted_tag: Tag): void
    (e: 'registered_text', registered_text: Text): void
    (e: 'updated_text', updated_text: Text): void
    (e: 'deleted_text', deleted_text: Text): void
    (e: 'registered_notification', registered_notification: Notification): void
    (e: 'updated_notification', updated_notification: Notification): void
    (e: 'deleted_notification', deleted_notification: Notification): void
    // 読み込み中オーバーレイの「再読み込み」。設定が取れないと初期化が一度も走らないので、
    // スピナーのままにせず再試行の導線を出している
    (e: 'requested_reload_application_config'): void
    // サイドバーの「HTML で出力」。ダイアログはラッパが持つ
    (e: 'requested_export_kyou_html'): void
    // Enter / Ctrl+V のショートカット。ダイアログはラッパが持つ
    (e: 'requested_show_kftl_dialog'): void
    (e: 'requested_show_save_clipboard_to_file_dialog'): void
}

/**
 * ラッパがページへ**そのまま**中継するイベントと、その引数。
 *
 * 依頼系（`requested_*`）はラッパが自分で処理するので入らない。
 * 1件でも落とすと「編集したのにタグツリーが更新されない」のような形で静かに壊れるため、
 * 束は下の `build_mi_columns_forward_relay` で作り、名前の網羅を型で縛る
 * （`kyou-view-relay.ts` / `rykv-columns-view-emits.ts` と同じ作法）。
 *
 * rykv の束と**共用しない**。共用すると、片方にイベントを足したときに
 * もう片方が宣言していないイベントを emit しても型で落ちず、
 * ネイティブリスナへ落ちて無言になる（この4段の型機構の存在理由そのもの）。
 */
export interface MiColumnsForwardArgs {
    received_messages: [Array<GkillMessage>]
    received_errors: [Array<GkillError>]
    registered_kyou: [Kyou]
    updated_kyou: [Kyou]
    deleted_kyou: [Kyou]
    registered_tag: [Tag]
    updated_tag: [Tag]
    deleted_tag: [Tag]
    registered_text: [Text]
    updated_text: [Text]
    deleted_text: [Text]
    registered_notification: [Notification]
    updated_notification: [Notification]
    deleted_notification: [Notification]
}

export type MiColumnsForwardEventName = keyof MiColumnsForwardArgs

export const mi_columns_forward_event_names = [
    'received_messages',
    'received_errors',
    'registered_kyou',
    'updated_kyou',
    'deleted_kyou',
    'registered_tag',
    'updated_tag',
    'deleted_tag',
    'registered_text',
    'updated_text',
    'deleted_text',
    'registered_notification',
    'updated_notification',
    'deleted_notification',
] as const satisfies ReadonlyArray<MiColumnsForwardEventName>

// 型に足して配列に足し忘れる（＝実行時に中継されない）のを防ぐ。
// 名前を落とすと never にならず、ここでコンパイルエラーになる
type MissingForwardEventName = Exclude<MiColumnsForwardEventName, typeof mi_columns_forward_event_names[number]>
const _assert_all_forward_events_listed: MissingForwardEventName[] = []
void _assert_all_forward_events_listed

export type MiColumnsForwardRelay = {
    [K in MiColumnsForwardEventName]: (...args: MiColumnsForwardArgs[K]) => void
}

/** emits を可変長で呼ぶための形。呼び出しシグネチャの重ね書きはそのままでは spread できない */
type LooseEmits = (event: MiColumnsForwardEventName, ...args: Array<unknown>) => void

/** 列からページへ素通しする束を作る。`v-on="…"` にそのまま渡せる */
export function build_mi_columns_forward_relay(emits: MiColumnsForwardRelay | unknown): MiColumnsForwardRelay {
    const emit = emits as LooseEmits
    const relay: Record<string, (...args: Array<unknown>) => void> = {}
    for (const event_name of mi_columns_forward_event_names) {
        relay[event_name] = (...args: Array<unknown>) => emit(event_name, ...args)
    }
    return relay as unknown as MiColumnsForwardRelay
}
