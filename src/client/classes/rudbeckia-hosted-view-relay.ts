'use strict'

import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { Kyou } from '@/classes/datas/kyou'
import type { Tag } from '@/classes/datas/tag'
import type { Text } from '@/classes/datas/text'
import type { Notification } from '@/classes/datas/notification'
import type { RudbeckiaPageDialogEmits } from '@/pages/dialogs/rudbeckia-page-dialog-emits'
import type { RudbeckiaPageDialogHostEmits } from '@/pages/views/rudbeckia-page-dialog-host-emits'
import type { RudbeckiaPageKind } from '@/pages/views/rudbeckia-page-kind'

/** ポートがダイアログとして開ける画面かどうか */
export function is_rudbeckia_page_kind(page_name: string): page_name is RudbeckiaPageKind {
    return page_name === 'rykv' || page_name === 'mi' || page_name === 'plaing' || page_name === 'dashboard'
}

/** ホストしたビューが出す17件のイベント名。手書きで羅列すると取りこぼす */
const hosted_view_relay_event_names = [
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
    'requested_show_application_config_dialog',
    'requested_reload_application_config',
    'saved_kyou_by_kftl',
] as const

type HostedViewRelayEventName = typeof hosted_view_relay_event_names[number]

export interface RudbeckiaHostedViewRelayArgs {
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
    requested_show_application_config_dialog: []
    requested_reload_application_config: []
    saved_kyou_by_kftl: [Date]
}

// 名前の配列とインターフェースがずれたらコンパイルエラーになる
// （kyou-view-relay.ts の Exclude 網羅チェックと同じ仕掛け）
type MissingHostedViewEventName = Exclude<keyof RudbeckiaHostedViewRelayArgs, HostedViewRelayEventName>
const _assert_all_hosted_view_events_listed: MissingHostedViewEventName[] = []
void _assert_all_hosted_view_events_listed

export type RudbeckiaHostedViewRelay =
    { [K in keyof RudbeckiaHostedViewRelayArgs]: (...args: RudbeckiaHostedViewRelayArgs[K]) => void }
    & { requested_navigate_page: (page_name: string) => void }

// emits を可変長で呼ぶための形。kyou-view-relay.ts の同名の型と同じ意味
type LooseEmits = (event: string, ...args: Array<unknown>) => void
// 値の型は never[]。パラメータは反変なので、具体的な型を取るハンドラも代入できる。
// 最後に as unknown as で本来の型へ戻すのは kyou-view-relay.ts と同じ
type LooseRelay = Record<string, (...args: never[]) => void>

/**
 * ポートの画面ウィンドウが、中のビューのイベントを親へ中継する束。
 *
 * 17件はそのまま素通し。画面切替（`requested_navigate_page`）だけは読み替える:
 * ポートがダイアログとして開ける画面なら `requested_open_page`（＝ウィンドウを増やす）、
 * それ以外（メモ帳・打刻メモ帳・さいはて・ポート自身）は `requested_navigate_page` のまま
 * 上げてページ遷移させる。
 */
export function build_rudbeckia_hosted_view_relay(emits: RudbeckiaPageDialogEmits): RudbeckiaHostedViewRelay {
    const emit = emits as unknown as LooseEmits
    const relay: LooseRelay = {}
    for (const event_name of hosted_view_relay_event_names) {
        relay[event_name] = ((...args: Array<unknown>) => emit(event_name, ...args)) as (...args: never[]) => void
    }
    relay['requested_navigate_page'] = (page_name: string) => {
        if (is_rudbeckia_page_kind(page_name)) {
            emit('requested_open_page', page_name)
            return
        }
        emit('requested_navigate_page', page_name)
    }
    return relay as unknown as RudbeckiaHostedViewRelay
}

/** ホスト（画面ウィンドウの一覧）がポートのページへ中継する束。名前の一覧は上と共有する */
export type RudbeckiaPageDialogHostRelay =
    RudbeckiaHostedViewRelay
    & { requested_open_page: (kind: RudbeckiaPageKind) => void }

export function build_rudbeckia_page_dialog_host_relay(
    emits: RudbeckiaPageDialogHostEmits,
): RudbeckiaPageDialogHostRelay {
    const emit = emits as unknown as LooseEmits
    const relay: LooseRelay = {}
    for (const event_name of hosted_view_relay_event_names) {
        relay[event_name] = ((...args: Array<unknown>) => emit(event_name, ...args)) as (...args: never[]) => void
    }
    // ホストは読み替えない。ダイアログが振り分けたものをそのまま上げる
    relay['requested_open_page'] = (kind: RudbeckiaPageKind) => emit('requested_open_page', kind)
    relay['requested_navigate_page'] = (page_name: string) => emit('requested_navigate_page', page_name)
    return relay as unknown as RudbeckiaPageDialogHostRelay
}
