'use strict'

import type { KyouViewEmits } from "./kyou-view-emits"

/**
 * 打刻メモ帳（mkfl）のイベント。
 *
 * KyouViewEmits をそのまま写していたが、`requested_reload_kyou` /
 * `requested_update_check_kyous` / `requested_open_rykv_dialog` の3件が抜けており、
 * 中に置いた PlaingTimeIsView が上げてきてもここで行き止まりになっていた。
 * とくに `requested_reload_kyou` はタグ・テキスト・通知の変更の唯一の信号なので、
 * 落とすと付随データの変更が親へ一切届かない。
 * 他のビューと同じく KyouViewEmits を継承して差分だけ足す。
 */
export interface MKFLViewEmits extends KyouViewEmits {
    // KFTLで保存したときの時刻。実行中の一覧を追随させるために使う
    (e: 'saved_kyou_by_kftl', last_added_request_time: Date): void
}
