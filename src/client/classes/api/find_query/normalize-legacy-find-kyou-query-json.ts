'use strict'

// 旧形式（use_*フラグ入り）の FindKyouQuery JSON を null 判定の新形式へ正規化する。
//
// サーバDB内のJSONは起動時マイグレーションで新形式へ書き換わるが、
// localStorage の保存クエリと ServiceWorker にキャッシュされた古い
// application_config 応答はクライアント側にしか残らないため、
// parse 境界（parse_find_kyou_query / localStorage 読み出し）で必ずこれを通す。
//
// 旧キーが1つでもインスタンスへ混入すると、deep_equals のキー数比較が崩れて
// サイドバーの「機械的な再emitを同値比較で捨てる」ガードが永久に効かなくなる
// （検索中の列をクリックすると飛行中の検索がabortされる不具合が再発する）。
//
// 変換規則:
//   - use_X=false → 対応する値フィールドを null にする（未使用）
//   - use_X=true  → 値を維持する。値が null/undefined の配列系は [] を物質化して
//     旧挙動（空指定=0件、TimeIs覆い判定等）を保存する
//   - use_timeis=false のときは timeis_tags も null にする
//     （旧ゲートは use_timeis && use_timeis_tags の複合だったため）
//   - use_include_id / use_mi_sort_type / use_mi_check_state はキー削除のみ
//   - update_time / use_update_time はクライアント未使用のため両方削除
//   - 最後に use_* キー自体を削除する
// undefined は絶対に残さない（JSON.stringifyでキーが落ち、localStorage往復で
// コンストラクタ既定値が復活してしまうため、未使用は必ず null で表現する）。

const legacy_use_flag_keys = [
    'use_tags',
    'use_reps',
    'use_words',
    'use_timeis',
    'use_timeis_tags',
    'use_map',
    'use_calendar',
    'use_plaing',
    'use_update_time',
    'use_period_of_time',
    'use_rep_types',
    'use_mi_board_name',
    'use_mi_sort_type',
    'use_mi_check_state',
    'use_include_id',
    // クライアントに ids フィールドは無いので値は触らないが、キーを列に入れておかないと
    // 旧JSONが「レガシーでない」と判定されて書き戻しが起きず、use_ids が残り続ける。
    // 列は Go(find_query_legacy_json.go) / MCP(constants.mjs) と同じ16キーで揃える
    'use_ids',
] as const

function flag_value(json: Record<string, unknown>, key: string): { has: boolean, enabled: boolean } {
    if (!Object.prototype.hasOwnProperty.call(json, key)) {
        return { has: false, enabled: false }
    }
    return { has: true, enabled: json[key] === true }
}

function materialize_array(json: Record<string, unknown>, key: string): void {
    if (json[key] === null || json[key] === undefined) {
        json[key] = []
    }
}

function apply_array_group(json: Record<string, unknown>, flag_key: string, value_keys: Array<string>): void {
    const flag = flag_value(json, flag_key)
    if (!flag.has) {
        return
    }
    for (const value_key of value_keys) {
        if (flag.enabled) {
            materialize_array(json, value_key)
        } else {
            json[value_key] = null
        }
    }
}

function apply_nullable_group(json: Record<string, unknown>, flag_key: string, value_keys: Array<string>): void {
    const flag = flag_value(json, flag_key)
    if (!flag.has || flag.enabled) {
        return
    }
    for (const value_key of value_keys) {
        json[value_key] = null
    }
}

export function is_legacy_find_kyou_query_json(json: Record<string, unknown>): boolean {
    for (const key of legacy_use_flag_keys) {
        if (Object.prototype.hasOwnProperty.call(json, key)) {
            return true
        }
    }
    return false
}

 
export function normalize_legacy_find_kyou_query_json(json: Record<string, unknown>): { json: Record<string, unknown>, was_legacy: boolean } {
    // fast path: 旧キーが無ければそのまま返す（冪等）
    if (!is_legacy_find_kyou_query_json(json)) {
        return { json, was_legacy: false }
    }

    const normalized: Record<string, unknown> = { ...json }

    apply_array_group(normalized, 'use_words', ['words', 'not_words'])
    apply_array_group(normalized, 'use_tags', ['tags'])
    apply_array_group(normalized, 'use_reps', ['reps'])
    apply_array_group(normalized, 'use_rep_types', ['rep_types'])

    // TimeIs グループ: 旧ゲートは use_timeis && use_timeis_tags の複合
    const use_timeis = flag_value(normalized, 'use_timeis')
    const use_timeis_tags = flag_value(normalized, 'use_timeis_tags')
    if (use_timeis.has) {
        if (use_timeis.enabled) {
            materialize_array(normalized, 'timeis_words')
            materialize_array(normalized, 'timeis_not_words')
        } else {
            normalized.timeis_words = null
            normalized.timeis_not_words = null
            normalized.timeis_tags = null
        }
    }
    if (use_timeis_tags.has && (use_timeis.enabled || !use_timeis.has)) {
        if (use_timeis_tags.enabled) {
            materialize_array(normalized, 'timeis_tags')
        } else {
            normalized.timeis_tags = null
        }
    }

    apply_nullable_group(normalized, 'use_calendar', ['calendar_start_date', 'calendar_end_date'])
    apply_nullable_group(normalized, 'use_map', ['map_latitude', 'map_longitude', 'map_radius'])
    apply_nullable_group(normalized, 'use_plaing', ['plaing_time'])

    // Mi板名: use=true で値が null/undefined なら ""（旧「空板名比較」の保存）。
    // 番兵文字列や "" の null 化はここではしない（挙動保存優先。UI変換は表示層が担う）
    const use_mi_board_name = flag_value(normalized, 'use_mi_board_name')
    if (use_mi_board_name.has) {
        if (use_mi_board_name.enabled) {
            if (normalized.mi_board_name === null || normalized.mi_board_name === undefined) {
                normalized.mi_board_name = ''
            }
        } else {
            normalized.mi_board_name = null
        }
    }

    // 時間帯: use=true では week_of_days のみ [] を物質化（旧 nil→0件挙動の保存）
    const use_period_of_time = flag_value(normalized, 'use_period_of_time')
    if (use_period_of_time.has) {
        if (use_period_of_time.enabled) {
            materialize_array(normalized, 'period_of_time_week_of_days')
        } else {
            normalized.period_of_time_start_time_second = null
            normalized.period_of_time_end_time_second = null
            normalized.period_of_time_week_of_days = null
        }
    }

    // クライアント未使用フィールドの掃除（hydrate 経由でインスタンスに生えるのを防ぐ）
    delete normalized.update_time

    for (const key of legacy_use_flag_keys) {
        delete normalized[key]
    }

    return { json: normalized, was_legacy: true }
}
