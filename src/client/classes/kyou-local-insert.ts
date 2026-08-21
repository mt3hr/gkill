'use strict'

import { toRaw } from 'vue'
import type { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import type { Kyou } from '@/classes/datas/kyou'
import type { Mi } from '@/classes/datas/mi'
import type { MiReKyou } from '@/classes/datas/mi-re-kyou'
import { MiCheckState } from '@/classes/api/find_query/mi-check-state'
import { MiSortType } from '@/classes/api/find_query/mi-sort-type'

/**
 * 追加されたKyouを、列を再検索せずに正しい位置へ差し込むための判定と整列。
 *
 * `/api/get_kyou` はIDでの取得だけでFindQueryを受けないので、
 * 「その列の絞り込みに一致するか」はクライアントで判定するしかない。
 * 判定できるフィルタだけを判定し、判定できないフィルタを使っている列は
 * `undecidable` を返して呼び出し元に従来どおりの再検索をさせる。
 *
 * 並び順・絞り込みの意味論は `src/server/gkill/api/find_filter.go` の写しなので、
 * あちらを変えたらこちらも変えること。対になるテストは
 * `__tests__/unit/classes/kyou-local-insert-mi-parity.test.ts`。
 */

// 「タグ無し」を表す番兵。Go側 find_filter.go の NoTags と同値
const no_tags_sentinel = 'no tags'

// ── Types ──

export type LocalInsertDecision =
    | { kind: 'insert', rows: Array<Kyou> }
    /** 判定できて「この列には入らない」。再検索も不要 */
    | { kind: 'skip' }
    /** 判定しきれない。呼び出し元がこの列だけ再検索すること */
    | { kind: 'undecidable', reason: string }

export type LocalDecidability = { ok: true } | { ok: false, reason: string }

// ── Data type helpers ──

/** mirekyou_* は "mi" で始まるので、Mi判定より必ず先に判定すること */
function is_mi_re_kyou_data_type(data_type: string): boolean {
    return data_type.startsWith('mirekyou')
}

function is_mi_data_type(data_type: string): boolean {
    return is_mi_re_kyou_data_type(data_type) || data_type.startsWith('mi')
}

/** for_mi列の並び替え対象。MiとMiReKyouは同じ規則に従う */
function get_typed_task(kyou: Kyou): Mi | MiReKyou | null {
    if (is_mi_re_kyou_data_type(kyou.data_type)) {
        return kyou.typed_mirekyou
    }
    return kyou.typed_mi
}

// ── Query gate ──

/**
 * rep_typesがクライアントで無視して良い形か。
 * クライアントには rep_name からサーバのrep_typeを引く写像が存在しない
 * (`ApplicationConfig.get_rep_type_from_rep_name()` が返すのはdvnfの接頭辞という別語彙)。
 * ただし for_mi 列はそもそもMi系repしか見ないので ['mi'] はno-opと証明できる。
 */
function is_rep_types_no_op(query: FindKyouQuery): boolean {
    if (query.rep_types === null) {
        return true
    }
    return query.for_mi && query.rep_types.length === 1 && query.rep_types[0] === 'mi'
}

/**
 * その列の検索条件をクライアントだけで判定しきれるか。
 * 判定できないものは、いずれもKyou単体には無い情報
 * (本文・DB全体のTimeIs区間・GPSログ・rep_typeの写像)を要求する。
 */
export function can_decide_query_locally(query: FindKyouQuery): LocalDecidability {
    // 本文検索はrepごとの内容カラムのLIKEと、別系統のTextリポジトリ検索の2経路。
    // Kyouは本文フィールドを持たない
    if (query.words !== null || query.not_words !== null) {
        return { ok: false, reason: 'words' }
    }
    // TimeIs絞り込みはDB全体のTimeIs区間集合が要る。
    // attached_timeis_kyou は別条件(plaing)で引いたものなので代用にならない。
    // timeis_words は空配列でも「任意のTimeIsに覆われたKyou」という有効な指定
    if (query.timeis_words !== null || query.timeis_not_words !== null) {
        return { ok: false, reason: 'timeis' }
    }
    // 地図は3値そろって有効
    if (query.map_latitude !== null && query.map_longitude !== null && query.map_radius !== null) {
        return { ok: false, reason: 'map' }
    }
    if (query.plaing_time !== null) {
        return { ok: false, reason: 'plaing_time' }
    }
    // 画像絞り込みが見る is_image / is_video はGoのKyouにはあるがTSのKyouには無く、
    // clone() で落ちるので判定できない
    if (query.is_image_only) {
        return { ok: false, reason: 'is_image_only' }
    }
    if (!is_rep_types_no_op(query)) {
        return { ok: false, reason: 'rep_types' }
    }
    return { ok: true }
}

/**
 * そのKyouが「1レコード＝1行」で扱える型か。
 *
 * 非mi列では、Mi/MiReKyouは5射影・TimeIsは include_end_timeis 次第で2行になりうる。
 * 1行として差し込むと本来の見え方と食い違うので、その列は再検索に回す。
 * for_mi列に載るのはMi/MiReKyouだけなので、そちらは常に1行。
 */
export function can_decide_kyou_locally(kyou: Kyou, query: FindKyouQuery): LocalDecidability {
    if (query.for_mi) {
        return { ok: true }
    }
    if (is_mi_data_type(kyou.data_type)) {
        return { ok: false, reason: 'mi_multi_projection' }
    }
    if (kyou.data_type.startsWith('timeis')) {
        return { ok: false, reason: 'timeis_multi_projection' }
    }
    return { ok: true }
}

// ── Mi projection ──

/**
 * for_mi列の表示用に related_time / data_type を書き換える。
 * `find_filter.go` の overrideKyous と同一規則で、
 * ソート基準の時刻が未設定なら作成日時へフォールバックして _create を名乗る。
 * 引数のKyouを書き換えるので、呼び出し元はクローンを渡すこと。
 */
export function apply_mi_projection(kyou: Kyou, mi_sort_type: MiSortType): void {
    const task = get_typed_task(kyou)
    if (!task) {
        return
    }
    const prefix = is_mi_re_kyou_data_type(kyou.data_type) ? 'mirekyou' : 'mi'
    let projected_time: Date | null = null
    let suffix: string | null = null
    switch (mi_sort_type) {
        case MiSortType.estimate_start_time:
            projected_time = task.estimate_start_time
            suffix = '_start'
            break
        case MiSortType.estimate_end_time:
            projected_time = task.estimate_end_time
            suffix = '_end'
            break
        case MiSortType.limit_time:
            projected_time = task.limit_time
            suffix = '_limit'
            break
        default:
            break
    }
    if (suffix !== null && projected_time) {
        kyou.related_time = projected_time
        kyou.data_type = prefix + suffix
        return
    }
    kyou.related_time = kyou.create_time
    kyou.data_type = prefix + '_create'
}

// ── Match predicate ──

function matches_reps(kyou: Kyou, query: FindKyouQuery): boolean {
    // null=絞り込み未使用 / 非nullの空配列=0件
    if (query.reps === null) {
        return true
    }
    return query.reps.includes(kyou.rep_name)
}

function equals_ignore_case(left: string, right: string): boolean {
    return left.toLowerCase() === right.toLowerCase()
}

function has_tag_name(kyou: Kyou, tag_name: string): boolean {
    // タグ名の照合は完全一致・大小無視(SQLの LOWER()= と同じ)。部分一致ではない
    return kyou.attached_tags.some(attached_tag => equals_ignore_case(attached_tag.tag, tag_name))
}

/**
 * 非表示タグ。`tags` が未使用(null)のときは適用しない。
 * 「hide_tags に載っている(大小無視)が query.tags には無い(大小区別)」タグを持つと除外される。
 * この大小の非対称は find_filter.go の実装そのままで、
 * hide_tags 側は GetTagsByTagName の照合、query.tags 側は containsString の == による。
 */
function is_hidden_by_hide_tags(kyou: Kyou, query: FindKyouQuery): boolean {
    if (query.tags === null || query.hide_tags.length === 0) {
        return false
    }
    const checked_tag_names = query.tags
    return kyou.attached_tags.some(attached_tag => {
        const is_hide_tag = query.hide_tags.some(hide_tag_name => equals_ignore_case(hide_tag_name, attached_tag.tag))
        if (!is_hide_tag) {
            return false
        }
        return !checked_tag_names.includes(attached_tag.tag)
    })
}

function matches_tags(kyou: Kyou, query: FindKyouQuery): boolean {
    if (query.tags === null) {
        return true
    }
    // タグで絞る指定なのに1つもチェックされていない場合は0件
    if (query.tags.length === 0) {
        return false
    }
    const has_no_tags = kyou.attached_tags.length === 0
    let matched = false
    if (query.tags_and) {
        // 「no tags」も仮想タグとして交差に参加する。
        // 存在しないタグ名が指定されていればANDは成立しない
        matched = query.tags.every(query_tag_name => {
            if (query_tag_name === no_tags_sentinel) {
                return has_no_tags
            }
            return has_tag_name(kyou, query_tag_name)
        })
    } else {
        matched = query.tags.some(query_tag_name => {
            if (query_tag_name === no_tags_sentinel) {
                return has_no_tags
            }
            return has_tag_name(kyou, query_tag_name)
        })
    }
    if (!matched) {
        return false
    }
    return !is_hidden_by_hide_tags(kyou, query)
}

/** カレンダーは両端を含む。タイムゾーン変換はせず瞬間で比べる */
function matches_calendar(time: Date, query: FindKyouQuery): boolean {
    const time_milli_seconds = time.getTime()
    if (query.calendar_start_date !== null && time_milli_seconds < query.calendar_start_date.getTime()) {
        return false
    }
    if (query.calendar_end_date !== null && time_milli_seconds > query.calendar_end_date.getTime()) {
        return false
    }
    return true
}

function to_second_of_day(time: Date): number {
    return time.getHours() * 3600 + time.getMinutes() * 60 + time.getSeconds()
}

/**
 * 時間帯。3値のいずれかが非nullで有効になる。
 * week_of_days は null=曜日制限なし / 非nullの空=0件 / 全7曜日=制限なし。
 * nil を len===0 や len!==7 の分岐へ落とすと全件が消えるので、必ずnullを先に外すこと。
 */
function matches_period_of_time(time: Date, query: FindKyouQuery): boolean {
    const has_period_start = query.period_of_time_start_time_second !== null
    const has_period_end = query.period_of_time_end_time_second !== null
    const has_week_of_days = query.period_of_time_week_of_days !== null
    if (!has_period_start && !has_period_end && !has_week_of_days) {
        return true
    }

    const week_of_days = query.period_of_time_week_of_days
    const filter_weekdays = week_of_days !== null && week_of_days.length !== 7
    if (filter_weekdays && !week_of_days.includes(time.getDay())) {
        return false
    }

    const time_second = to_second_of_day(time)
    // 秒指定はUnix秒で持っているのでローカル時刻の時分秒へ直す
    const start_time_second = query.period_of_time_start_time_second
    const end_time_second = query.period_of_time_end_time_second
    const period_start_second = start_time_second !== null ? to_second_of_day(new Date(start_time_second * 1000)) : 0
    const period_end_second = end_time_second !== null ? to_second_of_day(new Date(end_time_second * 1000)) : 0

    if (has_period_start && has_period_end) {
        if (period_start_second > period_end_second) {
            // 夜跨ぎ
            return time_second >= period_start_second || time_second <= period_end_second
        }
        return time_second >= period_start_second && time_second <= period_end_second
    }
    if (has_period_start) {
        return time_second >= period_start_second
    }
    if (has_period_end) {
        return time_second <= period_end_second
    }
    return true
}

/** FindMi のUNIONを組み立てる「含める」指定のキー */
type MiIncludeKey = 'include_create_mi' | 'include_check_mi' | 'include_limit_mi' | 'include_start_mi' | 'include_end_mi'

/**
 * Miの5射影と、その射影が表示に使う時刻。未設定の射影は null。
 * 時刻の対応は mi_repository_cached_sqlite3_impl.go のUNIONの SELECT に合わせてある
 * (check射影だけは UPDATE_TIME_UNIX を表示時刻に使う)。
 */
function collect_mi_projection_times(task: Mi | MiReKyou): Array<{ included_key: MiIncludeKey, time: Date | null }> {
    return [
        { included_key: 'include_create_mi', time: task.create_time },
        { included_key: 'include_check_mi', time: task.update_time },
        { included_key: 'include_limit_mi', time: task.limit_time },
        { included_key: 'include_start_mi', time: task.estimate_start_time },
        { included_key: 'include_end_mi', time: task.estimate_end_time },
    ]
}

/**
 * for_mi列のMi絞り込み。
 * サーバは2段構えで、どちらも通ったものだけが残る。
 *   (A) 5射影のどれかがカレンダー+時間帯を通る (sortAndTrimKyousMap がKyou行に適用)
 *   (B) 「含める」指定のある射影のどれかが、非nullの時刻でカレンダーを通る (FindMi のUNION)
 */
function matches_mi(kyou: Kyou, query: FindKyouQuery): boolean {
    const task = get_typed_task(kyou)
    if (!task) {
        return false
    }
    // 板名は完全一致・大小区別(SQLに COLLATE NOCASE が無い)。null は「すべて」
    if (query.mi_board_name !== null && task.board_name !== query.mi_board_name) {
        return false
    }
    // checked / uncheck 以外(all・空文字・未知)は全通し
    if (query.mi_check_state === MiCheckState.checked && !task.is_checked) {
        return false
    }
    if (query.mi_check_state === MiCheckState.uncheck && task.is_checked) {
        return false
    }

    const projections = collect_mi_projection_times(task)
    const passes_trim = projections.some(projection =>
        projection.time !== null
        && matches_calendar(projection.time, query)
        && matches_period_of_time(projection.time, query))
    if (!passes_trim) {
        return false
    }
    const passes_find_mi = projections.some(projection =>
        query[projection.included_key] === true
        && projection.time !== null
        && matches_calendar(projection.time, query))
    return passes_find_mi
}

/**
 * そのKyouがその検索条件に一致するか。
 * ゲート(`can_decide_query_locally` / `can_decide_kyou_locally`)を通したあとにだけ呼ぶこと。
 */
export function does_kyou_match_query(kyou: Kyou, query: FindKyouQuery): boolean {
    if (kyou.is_deleted) {
        return false
    }
    if (!matches_reps(kyou, query)) {
        return false
    }
    if (!matches_tags(kyou, query)) {
        return false
    }
    if (query.for_mi) {
        if (!is_mi_data_type(kyou.data_type)) {
            return false
        }
        return matches_mi(kyou, query)
    }
    if (!matches_calendar(kyou.related_time, query)) {
        return false
    }
    return matches_period_of_time(kyou.related_time, query)
}

// ── Comparator ──

function compare_id(a: Kyou, b: Kyou): number {
    if (a.id < b.id) {
        return -1
    }
    if (a.id > b.id) {
        return 1
    }
    return 0
}

/**
 * for_mi列で、その行がソート基準の時刻を持っているか。
 *
 * overrideKyous は「ソート基準が未設定のときに限って」_create を名乗らせるので、
 * data_type の接尾辞だけで「時刻あり/なし」を復元できる。
 * typed_mi を見てはいけない ―― get_kyous が返した既存行は typed_mi が未ロードで、
 * 比較子が既存行に対して動かなくなる。
 *
 * 並び替え(compare_kyou_for_query)と、mi列の時刻スクロール
 * (use-kyou-list-view.ts の scroll_to_time)の両方が「未設定(末尾)セグメントか」の
 * 判定にこれを使う。判定を書き写して二重管理にしないこと。
 */
export function has_mi_sort_key(kyou: Kyou, mi_sort_type: MiSortType): boolean {
    switch (mi_sort_type) {
        case MiSortType.estimate_start_time:
            return kyou.data_type.endsWith('_start')
        case MiSortType.estimate_end_time:
            return kyou.data_type.endsWith('_end')
        case MiSortType.limit_time:
            return kyou.data_type.endsWith('_limit')
        default:
            // create_time ソートでは常に作成日時が入っている
            return true
    }
}

/**
 * `find_filter.go` の sortResultKyous と同じ順序。負ならaが先。
 *
 * 非mi: RelatedTime降順、同着はID昇順。
 *   **秒への切り捨ては本質** ―― サーバは RelatedTime.Unix() で比べるので、
 *   ミリ秒のまま比べると同一秒内の隣接行に対して位置がずれる。
 * mi: ソート基準の時刻を昇順、未設定は末尾へ回して作成日時昇順、同着はID昇順。
 *   miの比較はミリ秒精度(.Unix()ではない)。
 */
export function compare_kyou_for_query(a: Kyou, b: Kyou, query: FindKyouQuery): number {
    if (!query.for_mi) {
        const a_unix = Math.floor(a.related_time.getTime() / 1000)
        const b_unix = Math.floor(b.related_time.getTime() / 1000)
        if (a_unix !== b_unix) {
            return a_unix > b_unix ? -1 : 1
        }
        return compare_id(a, b)
    }

    const a_has_key = has_mi_sort_key(a, query.mi_sort_type)
    const b_has_key = has_mi_sort_key(b, query.mi_sort_type)
    if (a_has_key && !b_has_key) {
        return -1
    }
    if (!a_has_key && b_has_key) {
        return 1
    }
    // 両方あり=射影時刻どうし、両方なし=作成日時どうし。どちらも related_time に入っている
    const a_time = a.related_time.getTime()
    const b_time = b.related_time.getTime()
    if (a_time !== b_time) {
        return a_time < b_time ? -1 : 1
    }
    return compare_id(a, b)
}

// ── Insert ──

/**
 * 整列済みリストの中で、そのKyouが入るべき位置。
 * 列は30万件に達するので二分探索で求める。
 */
export function find_insert_index(list: Array<Kyou>, kyou: Kyou, query: FindKyouQuery): number {
    let low = 0
    let high = list.length
    while (low < high) {
        const middle = (low + high) >>> 1
        if (compare_kyou_for_query(kyou, list[middle], query) < 0) {
            high = middle
        } else {
            low = middle + 1
        }
    }
    return low
}

/**
 * 並び順を保ってリストへ差し込む。差し込んだら true。
 * 同じidが既にあれば何もせず false を返す
 * (同じ registered_kyou の二重発火や、await中に再検索が完了していた場合に効く)。
 *
 * **in-placeで splice すること。** copy-on-write にすると
 * `focused_kyous_list`(= match_kyous_list[focused_column_index] へのエイリアス)と縁が切れ、
 * 件数カレンダーやDnoteが追随しなくなる。30万要素の配列コピーも避けられる。
 */
export function insert_kyou_sorted(list: Array<Kyou>, kyou: Kyou, query: FindKyouQuery): boolean {
    // 重複チェックは**線形のまま**にすること。
    // find_insert_index の近傍だけ見る形に狭めると、再射影されたmi行が
    // リスト内のコピーと違う位置に来るケースを取りこぼし、症状は行の静かな重複になる。
    //
    // 走査は生の配列に対して行う。listはdeepなref配下のリアクティブProxyなので、
    // 素で list[i] を読むと1要素ごとに track と toReactive が走り、
    // 要素ぶんの Proxy と WeakMap エントリを確保する(30万件の列では効く)。
    // 読み取りだけならtoRaw越しでも意味論は同じ。
    const raw_list = toRaw(list)
    for (let i = 0; i < raw_list.length; i++) {
        if (raw_list[i].id === kyou.id) {
            return false
        }
    }
    // ★splice は必ずリアクティブな list に対して行うこと。
    //   toRaw した配列へ差し込むと変更が誰にも通知されない。
    list.splice(find_insert_index(raw_list, kyou, query), 0, kyou)
    return true
}

/**
 * その列へ差し込むべきかの総合判定。
 * for_mi列では related_time / data_type をその列の並び替え規則へ書き換えるので、
 * 呼び出し元はクローンを渡すこと。
 */
export function decide_local_insert(kyou: Kyou, query: FindKyouQuery): LocalInsertDecision {
    const query_gate = can_decide_query_locally(query)
    if (!query_gate.ok) {
        return { kind: 'undecidable', reason: query_gate.reason }
    }
    const kyou_gate = can_decide_kyou_locally(kyou, query)
    if (!kyou_gate.ok) {
        return { kind: 'undecidable', reason: kyou_gate.reason }
    }
    if (kyou.is_deleted) {
        return { kind: 'skip' }
    }
    if (query.for_mi) {
        // for_mi列にkmemo等は原理的に入らない。再検索は不要
        if (!is_mi_data_type(kyou.data_type)) {
            return { kind: 'skip' }
        }
        apply_mi_projection(kyou, query.mi_sort_type)
    }
    if (!does_kyou_match_query(kyou, query)) {
        return { kind: 'skip' }
    }
    return { kind: 'insert', rows: [kyou] }
}

/**
 * リストから id の一致する Kyou を全部取り除く。
 *
 * 走査は生の配列に対して行う。deepなref配下のリアクティブProxy越しに読むと
 * 1要素ごとに track と toReactive が走り、要素ぶんのProxyを確保する(30万件の列では効く)。
 * **splice は必ずリアクティブな `list` に対して行うこと**(でないと誰にも通知されない)。
 * 後ろから走るので、splice しても未走査側のインデックスはずれない。
 *
 * 同じ関数が5つのコンポーザブルに複製されていて、この toRaw の最適化は
 * rykv / mi の2本にしか入っていなかった。追加(insert_kyou_sorted)と対になるので
 * ここに置いて全員が同じものを使う。
 */
export function remove_kyou_from_list_by_id(list: Array<Kyou>, deleted_id: string): void {
    const raw_list = toRaw(list)
    for (let i = raw_list.length - 1; i >= 0; i--) {
        if (raw_list[i].id === deleted_id) {
            list.splice(i, 1)
        }
    }
}
