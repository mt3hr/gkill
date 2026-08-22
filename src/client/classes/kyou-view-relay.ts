'use strict'

// Pinia/Vuex を入れない理由（中継の網羅性を型で保証している）:
// documents/adr/0038-props-emit-only-no-pinia.md

import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { Kyou } from '@/classes/datas/kyou'
import type { Tag } from '@/classes/datas/tag'
import type { Text } from '@/classes/datas/text'
import type { Notification } from '@/classes/datas/notification'
import type { RykvDialogKind, RykvDialogPayload } from '@/pages/views/rykv-dialog-kind'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'

/**
 * 子から親へそのまま中継するイベントと、その引数。
 *
 * `KyouViewEmits` の全21イベントのうち、次の3つを除いた18件。
 * - `requested_close_dialog` … 子は上げてこない。ダイアログが自分で `hide()` に繋ぐ設計
 *   （`*-dialog.vue` の `@requested_close_dialog="hide()"`）
 * - `focused_kyou` / `clicked_kyou` … ビュー層は**自分が発火源**なので中継しない。
 *   入れ子のKyouViewのフォーカスまで持ち上げると、外側と内側で二重に発火する
 *   （ダイアログ層だけが中継する。`KyouFocusRelayArgs` 参照）
 */
export interface KyouViewRelayArgs {
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
    requested_update_check_kyous: [Array<Kyou>, boolean]
    requested_reload_kyou: [Kyou]
    requested_reload_list: []
    requested_open_rykv_dialog: [RykvDialogKind, Kyou, RykvDialogPayload?]
}

/**
 * フォーカス/クリック。ダイアログ層だけが中継する。
 * ダイアログの中身は「別の場所に置かれたKyou」なので、そこでのフォーカス移動は
 * 呼び出し元まで伝える（`use-rykv-dialog-host-item.ts` 参照）。
 */
export interface KyouFocusRelayArgs {
    focused_kyou: [Kyou]
    clicked_kyou: [Kyou]
}

export type KyouDialogRelayArgs = KyouViewRelayArgs & KyouFocusRelayArgs

export type KyouViewRelayEventName = keyof KyouViewRelayArgs
export type KyouDialogRelayEventName = keyof KyouDialogRelayArgs

/**
 * 中継するイベント名の実体。
 *
 * これがリレー束の網羅性の唯一の情報源。`KyouViewRelay` が
 * `KyouViewRelayArgs` の全キーを持つマップ型なので、1つでも実装を落とすと
 * コンパイルエラーになる。手書きの束を並べていた頃は63箇所で取りこぼしが発生していた
 * （大半が `requested_open_rykv_dialog`）ので、増やすときは
 * `KyouViewRelayArgs` とこの配列の両方に足す。
 */
export const kyou_view_relay_event_names = [
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
    'requested_update_check_kyous',
    'requested_reload_kyou',
    'requested_reload_list',
    'requested_open_rykv_dialog',
] as const satisfies ReadonlyArray<KyouViewRelayEventName>

export const kyou_focus_relay_event_names = [
    'focused_kyou',
    'clicked_kyou',
] as const satisfies ReadonlyArray<keyof KyouFocusRelayArgs>

export const kyou_dialog_relay_event_names = [
    ...kyou_view_relay_event_names,
    ...kyou_focus_relay_event_names,
] as const satisfies ReadonlyArray<KyouDialogRelayEventName>

// 型に足して配列に足し忘れる（＝実行時に中継されない）のを防ぐ。
// 名前を落とすと never にならず、ここでコンパイルエラーになる
type MissingViewEventName = Exclude<KyouViewRelayEventName, typeof kyou_view_relay_event_names[number]>
type MissingDialogEventName = Exclude<KyouDialogRelayEventName, typeof kyou_dialog_relay_event_names[number]>
const _assert_all_view_events_listed: MissingViewEventName[] = []
const _assert_all_dialog_events_listed: MissingDialogEventName[] = []
void _assert_all_view_events_listed
void _assert_all_dialog_events_listed

/** `v-on="…"` にそのまま渡せるハンドラ束。キーは必ず全件揃う */
export type KyouViewRelay = { [K in KyouViewRelayEventName]: (...args: KyouViewRelayArgs[K]) => void }
export type KyouDialogRelay = { [K in KyouDialogRelayEventName]: (...args: KyouDialogRelayArgs[K]) => void }

/** emits を可変長で呼ぶための形。KyouViewEmits は呼び出しシグネチャの重ね書きなのでそのままでは spread できない */
type LooseEmits = (event: KyouDialogRelayEventName, ...args: Array<unknown>) => void
type LooseRelay = Record<string, (...args: Array<unknown>) => void>

function build_relay(
    emits: KyouViewEmits,
    event_names: ReadonlyArray<KyouDialogRelayEventName>,
    overrides?: Record<string, unknown>,
): LooseRelay {
    const emit = emits as LooseEmits
    const relay: LooseRelay = {}
    for (const event_name of event_names) {
        relay[event_name] = (...args: Array<unknown>) => emit(event_name, ...args)
    }
    return Object.assign(relay, overrides)
}

/**
 * ビュー層のリレー束を作る。
 *
 * ```ts
 * const crudRelayHandlers = build_kyou_view_relay(emits)
 * ```
 *
 * 挙動を変えたいイベントだけ `overrides` で差し替える。
 */
export function build_kyou_view_relay(
    emits: KyouViewEmits,
    overrides?: Partial<KyouViewRelay>,
): KyouViewRelay {
    return build_relay(emits, kyou_view_relay_event_names, overrides) as unknown as KyouViewRelay
}

/**
 * ダイアログ層のリレー束を作る。ビュー層の18件にフォーカス系2件を足したもの。
 *
 * どちらを使うかの基準は「ダイアログかどうか」ではなく
 * **「自分がフォーカスの発火源かどうか」**。
 * 子が上げてきた `focused_kyou` / `clicked_kyou` を素通しするだけの中間層
 * （`dnote-item-list-view` 等）は、自分では発火しないのでこちらを使う。
 * 関数名に `dialog` と付いているせいで誤読されやすいので注意。
 */
export function build_kyou_dialog_relay(
    emits: KyouViewEmits,
    overrides?: Partial<KyouDialogRelay>,
): KyouDialogRelay {
    return build_relay(emits, kyou_dialog_relay_event_names, overrides) as unknown as KyouDialogRelay
}

/**
 * ページ最上位の `RykvDialogHost` に渡すハンドラ束。
 *
 * ページには emit 先の親がいないので、`build_kyou_*_relay` と違って
 * **未指定のイベントは no-op で埋める**。
 *
 * ただし次の5つだけは省略できない。
 * 「Kyou を抱えているページが自分で決めなければならないこと」であり、
 * `dashboard-page.vue` はここを全部落としていて
 * 「ダッシュボードでは何を編集しても画面が更新されない」状態になっていた。
 * 束を n 個スプレッドする書き方だと1つ欠けても型エラーにならないので、
 * 型で必須にして取りこぼしをコンパイルエラーにする。
 */
export type KyouDialogHostRequiredHandlers =
    Pick<KyouDialogRelay, 'updated_kyou' | 'deleted_kyou' | 'requested_reload_kyou' | 'requested_open_rykv_dialog'>
    & { closed: (dialog_id: string) => void }

/** 必須の5件を除いた残り。重複して書くと型エラーになる */
export type KyouDialogHostOptionalHandlers = Omit<Partial<KyouDialogRelay>, keyof KyouDialogHostRequiredHandlers>

export type KyouDialogHostHandlers = KyouDialogRelay & { closed: (dialog_id: string) => void }

export function build_kyou_dialog_host_handlers(
    required: KyouDialogHostRequiredHandlers,
    overrides?: KyouDialogHostOptionalHandlers,
): KyouDialogHostHandlers {
    const handlers: LooseRelay = {}
    for (const event_name of kyou_dialog_relay_event_names) {
        handlers[event_name] = () => { /* ページ側で処理しないイベントは握りつぶす */ }
    }
    return Object.assign(handlers, overrides, required) as unknown as KyouDialogHostHandlers
}

/**
 * `received_errors` / `received_messages` だけを上げてくる子のための最小の emits。
 *
 * 設定の struct 編集ビューや Add*Dialog のように、CRUD を一切上げない子は多い。
 * そこへ18件/20件の束を渡しても意味が無いので、この2件だけの束を使う。
 */
export interface ErrorMessageEmits {
    (e: 'received_errors', errors: Array<GkillError>): void
    (e: 'received_messages', messages: Array<GkillMessage>): void
}

export interface ErrorMessageRelay {
    received_errors: (errors: Array<GkillError>) => void
    received_messages: (messages: Array<GkillMessage>) => void
}

/**
 * エラー/メッセージだけを素通しする束を作る。
 *
 * ```ts
 * const errorMessageRelayHandlers = build_error_message_relay(emits)
 * ```
 *
 * 同じ中身を9箇所が手書きしていて、しかも名前が
 * `errorMessageRelayHandlers` / `errorsMessagesRelayHandlers` / `errorMessageHandlers` の
 * 3種類に割れていた。名前は `errorMessageRelayHandlers` に揃える。
 */
export function build_error_message_relay(emits: ErrorMessageEmits): ErrorMessageRelay {
    return {
        received_errors: (errors: Array<GkillError>) => emits('received_errors', errors),
        received_messages: (messages: Array<GkillMessage>) => emits('received_messages', messages),
    }
}
