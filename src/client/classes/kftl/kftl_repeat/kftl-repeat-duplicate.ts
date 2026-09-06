'use strict'

import type { GkillAPI } from '../../api/gkill-api'
import { MiCheckState } from '../../api/find_query/mi-check-state'
import { GetKyousRequest } from '../../api/req_res/get-kyous-request'
import type { Kyou } from '../../datas/kyou'

/**
 * 繰り返しブロック「？？」の3行目が no（既定）のときの既存判定。
 *
 * 返すのは「既にある記録のアンカー時刻」の集合（epoch ミリ秒）で、展開はその時刻の回を飛ばす。
 * 型ごとの「同じ記録」の定義は各リクエストクラスの find_existing_for_repeat が持つ。
 *
 * **絞るのは候補全体を覆う期間だけ。** 型別の中身（タイトル・本文）はサーバの検索が返さないので、
 * data_type で絞ってから残ったぶんだけ load_typed_datas して突き合わせる。
 * 期間で1回引いたあと**ヒット件数ぶんの往復**が要る（Go 側は型別リポジトリを直接引けるので1回で済む）。
 *
 * Mirrors: src/server/gkill/api/kftl/kftl_repeat_duplicate.go
 */
export async function find_existing_anchors(
    gkill_api: GkillAPI,
    from: Date,
    to: Date,
    for_schedule: boolean,
    matches_data_type: (data_type: string) => boolean,
    anchor_of: (kyou: Kyou) => Date | null,
): Promise<Set<number>> {
    const request = new GetKyousRequest()
    // **コンストラクタの既定値をそのまま送ってはいけない。** FindKyouQuery は
    // tags / reps を `[]` で初期化する。null が「フィルタ未使用」で、非nullの空配列は
    // 「0件指定」なので、既定のまま送るとサーバは tags で無条件に0件にし
    // （find_filter.go の filterTagsKyous）、rep 名の許可集合も空になる。
    // 既存判定が常に空を返し、3行目が no でも全回が作られる
    // ＝ 同じテキストを送るたびに増える。Go 側（kftl_repeat_duplicate.go の
    // newRepeatDuplicateQuery）が Tags / Reps を nil のままにしているのと揃える
    request.query.tags = null
    request.query.reps = null
    request.query.calendar_start_date = from
    request.query.calendar_end_date = to
    if (for_schedule) {
        // タスク / リポストタスクは予定日時のどれかが期間に入るものを拾う。
        // **射影を1つでも落とすと、その日時軸を持つ記録を取りこぼして重複を作る**
        request.query.for_mi = true
        request.query.include_create_mi = true
        request.query.include_check_mi = true
        request.query.include_limit_mi = true
        request.query.include_start_mi = true
        request.query.include_end_mi = true
        // **チェック状態でも絞らない。** 既定は uncheck なので、そのままだと
        // 完了済みのタスクが既存判定に引っかからず、再送で重複する。
        // Go 側は MiCheckState を設定せず、空文字が filterMiForMi の default 節で
        // 全件対象になる。TS は enum に空が無いので all を明示する
        request.query.mi_check_state = MiCheckState.all
    }

    const response = await gkill_api.get_kyous(request)
    if (response.errors && response.errors.length !== 0) {
        throw new Error(response.errors[0].error_message)
    }

    const anchors = new Set<number>()
    for (const kyou of response.kyous ?? []) {
        // data_type は型別データを読まなくても手元にある。ここで絞ってから往復する
        if (kyou.is_deleted || !matches_data_type(kyou.data_type)) {
            continue
        }
        const errors = await kyou.load_typed_datas()
        if (errors.length !== 0) {
            continue
        }
        const anchor = anchor_of(kyou)
        if (anchor !== null) {
            anchors.add(anchor.getTime())
        }
    }
    return anchors
}

/**
 * タスク / リポストタスクのアンカー（最初に埋まっている予定日時）。
 * **anchor_time_for_repeat と同じ順序で見ること。** ずれると既存判定が噛み合わなくなる。
 */
export function schedule_anchor_of(estimate_start: Date | null, estimate_end: Date | null, limit: Date | null): Date | null {
    for (const time of [estimate_start, estimate_end, limit]) {
        if (time !== null && time !== undefined) {
            return time
        }
    }
    return null
}

/**
 * data_type の判定。
 *
 * **`mirekyou` を `mi` より先に見ること。** `mi_` で始まる判定を先に書くと
 * `mirekyou_create` がタスク側に食われる（AGENTS.md の「prefix checks must test mirekyou before mi」）。
 */
export function is_mi_data_type(data_type: string): boolean {
    return !data_type.startsWith("mirekyou") && data_type.startsWith("mi")
}

export function is_mirekyou_data_type(data_type: string): boolean {
    return data_type.startsWith("mirekyou")
}

export function is_timeis_data_type(data_type: string): boolean {
    return data_type.startsWith("timeis")
}
