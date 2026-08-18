'use strict'

import type { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'

/**
 * 検索を期間の窓へ分割するための純関数。
 *
 * 目的は総処理量の削減ではなく、**最初の行が出るまでの時間とサーバのピークメモリ**。
 * 全期間を1回で引くと、サーバは条件に合う全件を作ってから返すので、
 * 30年ぶんを指定した瞬間に数十秒とGB級のメモリを使う。
 * 新しい側から窓を切って順に引けば、最初の窓ぶんはすぐ表示できる。
 *
 * ## 分割してよい条件（`build_search_windows` が null を返さない条件）
 *
 * 窓に切ってよいのは「レコード単体で合否が決まる」検索だけ。
 * 次のどれかが有効なときは**絶対に分割してはいけない**。分割すると
 * エラーも出ないまま結果が変わる:
 *
 * - `for_mi` … 並び替えのキーが related_time ではなく、期間で絞る列も射影ごとに違う
 * - TimeIs絞り込み（`timeis_words` / `timeis_not_words` が非null）
 *   … サーバは同じ期間でTimeIsも引く。窓をまたぐTimeIsに覆われたKyouが落ちる
 * - 地図絞り込み（`map_latitude` 等）… GPSログも同じ期間で引かれる
 * - `plaing_time` … 期間ではなく1時点の指定で、そもそも件数が小さい
 * - `is_image_only` … 画像グリッドは行単位で組み直すので、追記のたびに全体を作り直す
 *
 * 上限（`calendar_end_date`）が無いときも分割しない。始点が決まらないため。
 */
export interface SearchWindow {
    /** 窓の下限（この時刻を含む）。null は「下限なし」＝最後の窓 */
    start: Date | null
    /** 窓の上限（この時刻を含む） */
    end: Date
}

/**
 * 窓の境界は**1秒**ずらす。
 *
 * サーバは期間を `RELATED_TIME_UNIX >= ?` / `<= ?` で比べ、
 * バインド値は `time.Unix()`（秒へ切り捨て）になる。
 * ミリ秒だけずらすと隣り合う窓が同じ秒を指し、その秒のレコードが**両方の窓に出る**。
 * 1秒ずらせば重複も隙間も出ない（DBの時刻は秒精度なので、間に落ちる値が無い）。
 */
const window_boundary_gap_millis = 1000

/** 上限が無いときに遡る下限。これより古い記録は1つの窓にまとめて取る */
const oldest_window_floor = new Date(0)

/**
 * 窓の幅（新しい側から古い側へ）。指数的に広げる。
 *
 * 新しいほど見る頻度が高く件数も多いので細かく、古くなるほど粗くする。
 * 全期間（30年）でも20窓程度に収まる。
 */
const window_span_days = [7, 14, 30, 60, 120, 240, 480, 960, 1920]

function add_days(base: Date, days: number): Date {
    return new Date(base.getTime() - days * 24 * 60 * 60 * 1000)
}

/** その検索条件を窓へ分割してよいか */
export function can_split_search_into_windows(query: FindKyouQuery): boolean {
    if (query.for_mi) {
        return false
    }
    if (query.is_image_only) {
        return false
    }
    if (query.plaing_time !== null) {
        return false
    }
    if (query.timeis_words !== null || query.timeis_not_words !== null) {
        return false
    }
    if (query.map_latitude !== null || query.map_longitude !== null || query.map_radius !== null) {
        return false
    }
    if (query.calendar_end_date === null) {
        return false
    }
    return true
}

/**
 * 検索条件を新しい順の窓へ分割する。分割してはいけない条件なら null。
 *
 * 窓は隣どうしで重ならず隙間も空かない。最後の窓は元の下限（無ければ十分過去）まで伸ばす。
 * 1窓に収まるなら分割しない（null を返す）。
 */
export function build_search_windows(query: FindKyouQuery): Array<SearchWindow> | null {
    if (!can_split_search_into_windows(query)) {
        return null
    }
    const end = query.calendar_end_date as Date
    const floor = query.calendar_start_date ?? oldest_window_floor
    if (floor.getTime() > end.getTime()) {
        return null
    }

    const windows = new Array<SearchWindow>()
    let current_end = end
    for (let i = 0; i < window_span_days.length; i++) {
        const candidate_start = add_days(current_end, window_span_days[i])
        if (candidate_start.getTime() <= floor.getTime()) {
            break
        }
        windows.push({ start: candidate_start, end: current_end })
        current_end = new Date(candidate_start.getTime() - window_boundary_gap_millis)
    }
    // 残り（元の下限まで）。下限が無い検索でもここで必ず閉じる
    windows.push({
        start: query.calendar_start_date === null ? null : floor,
        end: current_end,
    })

    if (windows.length <= 1) {
        return null
    }
    return windows
}

/** 窓を当てはめた検索条件を作る。元のクエリは変更しない */
export function apply_search_window(query: FindKyouQuery, window: SearchWindow): FindKyouQuery {
    const windowed = query.clone()
    windowed.calendar_start_date = window.start
    windowed.calendar_end_date = window.end
    return windowed
}
