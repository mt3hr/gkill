'use strict'

import type { ApplicationConfig } from "@/classes/datas/config/application-config"
import { MiCheckState } from "./mi-check-state"
import { MiSortType } from "./mi-sort-type"
import type { RepStructElementData } from "@/classes/datas/config/rep-struct-element-data"
import type { FoldableStructModel } from "@/pages/views/foldable-struct-model"
import moment from "moment"
import type { DeviceStructElementData } from "@/classes/datas/config/device-struct-element-data"
import type { RepTypeStructElementData } from "@/classes/datas/config/rep-type-struct-element-data"
import type { TagStructElementData } from "@/classes/datas/config/tag-struct-element-data"
import { collect_inited_tag_names } from "./collect-inited-tag-names"
import { normalize_legacy_find_kyou_query_json } from "./normalize-legacy-find-kyou-query-json"

// 古いビルドが保存した検索条件JSON(フィールド欠落・型不正がありうる)から安全に読むヘルパ。
// 値がセットされていればそれを優先し、無ければ呼び出し元の既定値(コンストラクタ値)を残す。
// かつては欠落フィールドへundefinedを代入していたため、JSON.stringifyがキーごと落とし
// 「一度欠けたフィールドは再保存しても永久に戻らない」固定化が起きていた。
// 既定値へフォールバックすることで、次回保存時にスキーマが自己治癒する
function array_from_json<T>(value: unknown, fallback: Array<T>): Array<T> {
    return Array.isArray(value) ? (value as Array<T>).concat() : fallback
}

// nullable配列用。null は「フィルタ未使用」の正規値なのでフォールバックせずそのまま採る
function nullable_array_from_json<T>(value: unknown, fallback: Array<T> | null): Array<T> | null {
    if (value === null) {
        return null
    }
    return Array.isArray(value) ? (value as Array<T>).concat() : fallback
}

function date_from_json(value: unknown, fallback: Date | null): Date | null {
    if (value === undefined || value === null || value === '') {
        return fallback
    }
    const date = value instanceof Date ? value : new Date(value as string | number)
    return Number.isNaN(date.getTime()) ? fallback : date
}

function nullable_number_from_json(value: unknown, fallback: number | null): number | null {
    if (value === null) {
        return null
    }
    return typeof value === 'number' ? value : fallback
}

// 検索条件。フィルタグループの有効/無効は値の null 判定で表す:
//   - null = フィルタ未使用
//   - 非nullの空配列 [] = フィルタ有効だが空指定
//     （tags/reps/rep_types は0件、timeis_words は「任意のTimeIsに覆われたKyou」）
// undefined は禁止（JSON.stringifyでキーが落ち、localStorage往復とdeep_equalsが壊れる）。
// keywords / timeis_keywords / *_in_sidebar はUI専用フィールド（サーバは読まない）で常に非null。
export class FindKyouQuery {
    query_id: string
    update_cache: boolean

    // キーワード検索（活性の担い手は words / not_words の非null。keywords はUI上の生文字列）
    keywords: string
    words_and: boolean
    words: Array<string> | null
    not_words: Array<string> | null

    // TimeIs検索（活性の担い手は timeis_words / timeis_not_words の非null。
    // timeis_tags は timeis グループ有効時のみ意味を持つ）
    timeis_keywords: string
    timeis_words_and: boolean
    timeis_words: Array<string> | null
    timeis_not_words: Array<string> | null
    timeis_tags: Array<string> | null
    timeis_tags_and: boolean

    // タグ（既定は []（有効・チェック0個=0件）。null にすると既定のwire挙動が変わる）
    tags: Array<string> | null
    hide_tags: Array<string>
    tags_and: boolean

    // 記録保管場所 / タイプ
    reps: Array<string> | null
    rep_types: Array<string> | null

    // 地図（3値そろって有効）
    map_latitude: number | null
    map_longitude: number | null
    map_radius: number | null

    // カレンダー
    calendar_start_date: Date | null
    calendar_end_date: Date | null

    // plaing（非null=その時刻に実行中のTimeIsを検索）
    plaing_time: Date | null

    // 時間帯（week_of_days: null=曜日制限なし / []=0件 / 全7曜日=制限なし）
    period_of_time_start_time_second: number | null
    period_of_time_end_time_second: number | null
    period_of_time_week_of_days: Array<number> | null

    // サイドバーUI状態（wire無関係・常に非null）
    devices_in_sidebar: Array<string>
    rep_types_in_sidebar: Array<string>
    is_enable_map_circle_in_sidebar: boolean
    is_image_only: boolean
    is_focus_kyou_in_list_view: boolean

    // Mi
    mi_board_name: string | null // null=「すべて」。番兵文字列は表示層だけが使う
    mi_sort_type: MiSortType // 常時有効（サーバが値を無条件に読む）
    mi_check_state: MiCheckState // 同上
    for_mi: boolean
    include_create_mi: boolean
    include_check_mi: boolean
    include_limit_mi: boolean
    include_start_mi: boolean
    include_end_mi: boolean
    include_end_timeis: boolean

    // 検索条件JSONからFindKyouQueryを復元する。
    // 保存元は世代がまばら(ryuu/dashboard/dnote等の設定には数世代前のビルドが書いた
    // JSONが残っている)ため、まず旧形式(use_*フラグ入り)を正規化してから、
    // 全フィールドを「値がセットされていればそれを優先、無ければコンストラクタ既定を維持」で読む。
    // 旧キーがインスタンスへ混入すると deep_equals のキー数比較が崩れ、
    // サイドバーの機械的emitガードが永久に効かなくなるため、正規化は省略できない
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    static parse_find_kyou_query(json: any): FindKyouQuery {
        const { json: n } = normalize_legacy_find_kyou_query_json(json)
        const cloned = new FindKyouQuery()
        cloned.query_id = n.query_id as string ?? cloned.query_id
        cloned.update_cache = n.update_cache as boolean ?? cloned.update_cache
        cloned.keywords = n.keywords as string ?? cloned.keywords
        cloned.words_and = n.words_and as boolean ?? cloned.words_and
        cloned.words = nullable_array_from_json(n.words, cloned.words)
        cloned.not_words = nullable_array_from_json(n.not_words, cloned.not_words)
        cloned.timeis_keywords = n.timeis_keywords as string ?? cloned.timeis_keywords
        cloned.timeis_words_and = n.timeis_words_and as boolean ?? cloned.timeis_words_and
        cloned.timeis_words = nullable_array_from_json(n.timeis_words, cloned.timeis_words)
        cloned.timeis_not_words = nullable_array_from_json(n.timeis_not_words, cloned.timeis_not_words)
        cloned.timeis_tags = nullable_array_from_json(n.timeis_tags, cloned.timeis_tags)
        cloned.timeis_tags_and = n.timeis_tags_and as boolean ?? cloned.timeis_tags_and
        cloned.tags = nullable_array_from_json(n.tags, cloned.tags)
        cloned.hide_tags = array_from_json(n.hide_tags, cloned.hide_tags)
        cloned.tags_and = n.tags_and as boolean ?? cloned.tags_and
        cloned.reps = nullable_array_from_json(n.reps, cloned.reps)
        cloned.rep_types = nullable_array_from_json(n.rep_types, cloned.rep_types)
        cloned.map_latitude = nullable_number_from_json(n.map_latitude, cloned.map_latitude)
        cloned.map_longitude = nullable_number_from_json(n.map_longitude, cloned.map_longitude)
        cloned.map_radius = nullable_number_from_json(n.map_radius, cloned.map_radius)
        // 日付フィールドはJSON化でISO文字列になっているのでDateへ復元する。
        // 文字列のまま持つと、サイドバーのgenerate_query(Dateを返す)との
        // deep_equals比較が恒久的に不一致になり、機械的なupdated_queryを
        // 値比較で吸収する安全網が復元列に対して一度も効かなくなる
        cloned.calendar_start_date = date_from_json(n.calendar_start_date, cloned.calendar_start_date)
        cloned.calendar_end_date = date_from_json(n.calendar_end_date, cloned.calendar_end_date)
        cloned.plaing_time = date_from_json(n.plaing_time, cloned.plaing_time)
        cloned.period_of_time_start_time_second = nullable_number_from_json(n.period_of_time_start_time_second, cloned.period_of_time_start_time_second)
        cloned.period_of_time_end_time_second = nullable_number_from_json(n.period_of_time_end_time_second, cloned.period_of_time_end_time_second)
        cloned.period_of_time_week_of_days = nullable_array_from_json(n.period_of_time_week_of_days, cloned.period_of_time_week_of_days)
        cloned.devices_in_sidebar = array_from_json(n.devices_in_sidebar, cloned.devices_in_sidebar)
        cloned.rep_types_in_sidebar = array_from_json(n.rep_types_in_sidebar, cloned.rep_types_in_sidebar)
        cloned.is_enable_map_circle_in_sidebar = n.is_enable_map_circle_in_sidebar as boolean ?? cloned.is_enable_map_circle_in_sidebar
        cloned.is_image_only = n.is_image_only as boolean ?? cloned.is_image_only
        cloned.is_focus_kyou_in_list_view = n.is_focus_kyou_in_list_view as boolean ?? false
        cloned.mi_board_name = typeof n.mi_board_name === 'string' ? n.mi_board_name : null
        cloned.mi_sort_type = n.mi_sort_type as MiSortType ?? cloned.mi_sort_type
        cloned.mi_check_state = n.mi_check_state as MiCheckState ?? cloned.mi_check_state
        cloned.for_mi = n.for_mi as boolean ?? cloned.for_mi
        cloned.include_create_mi = n.include_create_mi as boolean ?? cloned.include_create_mi
        cloned.include_check_mi = n.include_check_mi as boolean ?? cloned.include_check_mi
        cloned.include_limit_mi = n.include_limit_mi as boolean ?? cloned.include_limit_mi
        cloned.include_start_mi = n.include_start_mi as boolean ?? cloned.include_start_mi
        cloned.include_end_mi = n.include_end_mi as boolean ?? cloned.include_end_mi
        cloned.include_end_timeis = n.include_end_timeis as boolean ?? cloned.include_end_timeis
        return cloned
    }

    clone(): FindKyouQuery {
        const cloned = new FindKyouQuery()
        cloned.query_id = this.query_id
        cloned.update_cache = this.update_cache
        cloned.keywords = this.keywords
        cloned.words_and = this.words_and
        cloned.words = this.words === null ? null : this.words.concat()
        cloned.not_words = this.not_words === null ? null : this.not_words.concat()
        cloned.timeis_keywords = this.timeis_keywords
        cloned.timeis_words_and = this.timeis_words_and
        cloned.timeis_words = this.timeis_words === null ? null : this.timeis_words.concat()
        cloned.timeis_not_words = this.timeis_not_words === null ? null : this.timeis_not_words.concat()
        cloned.timeis_tags = this.timeis_tags === null ? null : this.timeis_tags.concat()
        cloned.timeis_tags_and = this.timeis_tags_and
        cloned.tags = this.tags === null ? null : this.tags.concat()
        cloned.hide_tags = this.hide_tags.concat()
        cloned.tags_and = this.tags_and
        cloned.reps = this.reps === null ? null : this.reps.concat()
        cloned.rep_types = this.rep_types === null ? null : this.rep_types.concat()
        cloned.map_latitude = this.map_latitude
        cloned.map_longitude = this.map_longitude
        cloned.map_radius = this.map_radius
        cloned.calendar_start_date = this.calendar_start_date
        cloned.calendar_end_date = this.calendar_end_date
        cloned.plaing_time = this.plaing_time
        cloned.period_of_time_start_time_second = this.period_of_time_start_time_second
        cloned.period_of_time_end_time_second = this.period_of_time_end_time_second
        cloned.period_of_time_week_of_days = this.period_of_time_week_of_days === null ? null : this.period_of_time_week_of_days.concat()
        cloned.devices_in_sidebar = this.devices_in_sidebar.concat()
        cloned.rep_types_in_sidebar = this.rep_types_in_sidebar.concat()
        cloned.is_enable_map_circle_in_sidebar = this.is_enable_map_circle_in_sidebar
        cloned.is_image_only = this.is_image_only
        cloned.is_focus_kyou_in_list_view = this.is_focus_kyou_in_list_view
        cloned.mi_board_name = this.mi_board_name
        cloned.mi_sort_type = this.mi_sort_type
        cloned.mi_check_state = this.mi_check_state
        cloned.for_mi = this.for_mi
        cloned.include_create_mi = this.include_create_mi
        cloned.include_check_mi = this.include_check_mi
        cloned.include_limit_mi = this.include_limit_mi
        cloned.include_start_mi = this.include_start_mi
        cloned.include_end_mi = this.include_end_mi
        cloned.include_end_timeis = this.include_end_timeis
        return cloned
    }

    constructor() {
        this.query_id = ""
        this.update_cache = false
        this.keywords = ""
        this.words_and = false
        this.words = null
        this.not_words = null
        this.timeis_keywords = ""
        this.timeis_words_and = false
        this.timeis_words = null
        this.timeis_not_words = null
        this.timeis_tags = null
        this.timeis_tags_and = false
        // tags/reps の既定は []（有効・チェック0個=0件）。
        // 旧 use_tags=true+tags=[] / use_reps=true+reps=[] の厳密等価
        // （ダッシュボード等がコンストラクタ既定のままwireへ乗せるため、null にすると挙動が変わる）
        this.tags = []
        this.hide_tags = []
        this.tags_and = false
        this.reps = []
        this.rep_types = null
        this.map_latitude = null
        this.map_longitude = null
        this.map_radius = null
        this.calendar_start_date = null
        this.calendar_end_date = null
        this.plaing_time = null
        this.period_of_time_start_time_second = null
        this.period_of_time_end_time_second = null
        this.period_of_time_week_of_days = null
        this.devices_in_sidebar = new Array<string>()
        this.rep_types_in_sidebar = new Array<string>()
        this.is_enable_map_circle_in_sidebar = false
        this.is_image_only = false
        this.is_focus_kyou_in_list_view = false
        this.mi_board_name = null
        this.mi_sort_type = MiSortType.estimate_start_time
        this.mi_check_state = MiCheckState.uncheck
        this.for_mi = false
        this.include_create_mi = true
        this.include_check_mi = false
        this.include_limit_mi = false
        this.include_start_mi = false
        this.include_end_mi = false
        this.include_end_timeis = true
    }

    // keywords / timeis_keywords をパースして words / not_words 系へ反映する。
    // グループが未使用（null）の場合は触らない（null のまま送るのが「未使用」の表現）
    parse_words_and_not_words() {
        if (this.words !== null || this.not_words !== null) {
            const words = new Array<string>()
            const not_words = new Array<string>()
            let next_is_not_word = false
            const words_list = this.keywords.split(" ")
            for (let i = 0; i < words_list.length; i++) {
                const words_list_ = words_list[i].split("　")
                for (let j = 0; j < words_list_.length; j++) {
                    let word = words_list_[j]
                    if (word.startsWith("-")) {
                        next_is_not_word = true
                        word = word.replace("-", "")
                    }
                    if (word === "") {
                        continue
                    } else if (word === "-") {
                        next_is_not_word = true
                        continue
                    } else {
                        if (next_is_not_word) {
                            not_words.push(word)
                        } else {
                            words.push(word)
                        }
                        next_is_not_word = false
                    }
                }
            }
            this.words = words
            this.not_words = not_words
        }

        if (this.timeis_words !== null || this.timeis_not_words !== null) {
            const timeis_words = new Array<string>()
            const timeis_not_words = new Array<string>()
            let next_is_not_word = false
            const timeis_words_list = this.timeis_keywords.split(" ")
            for (let i = 0; i < timeis_words_list.length; i++) {
                const timeis_words_list_ = timeis_words_list[i].split("　")
                for (let j = 0; j < timeis_words_list_.length; j++) {
                    let word = timeis_words_list_[j]
                    if (word.startsWith("-")) {
                        next_is_not_word = true
                        word = word.replace("-", "")
                    }
                    if (word === "") {
                        continue
                    } else if (word === "-") {
                        next_is_not_word = true
                        continue
                    } else {
                        if (next_is_not_word) {
                            timeis_not_words.push(word)
                        } else {
                            timeis_words.push(word)
                        }
                        next_is_not_word = false
                    }
                }
            }
            this.timeis_words = timeis_words
            this.timeis_not_words = timeis_not_words
        }
    }


    // ApplicationConfigから、デフォルトの検索条件を生成する。（rykv用）
    static generate_default_query_for_rykv(application_config: ApplicationConfig): FindKyouQuery {
        const query = new FindKyouQuery()

        // 対象はの3つ。ほかは初期値
        // RepのSummary, Detail
        // Tag
        // Calendar

        // RepのSummary, Detail
        let device_name_walk = (_device: DeviceStructElementData): Array<string> => []
        device_name_walk = (device: DeviceStructElementData): Array<string> => {
            const device_names = new Array<string>()
            const device_children = device.children
            if (device_children) {
                device_children.forEach(child_device => {
                    if (child_device.check_when_inited) {
                        device_names.push(child_device.device_name)
                    }
                    if (child_device) {
                        device_names.push(...device_name_walk(child_device))
                    }
                })
            }
            return device_names
        }
        query.devices_in_sidebar = device_name_walk(application_config.device_struct)

        let rep_type_name_walk = (_rep_type: RepTypeStructElementData): Array<string> => []
        rep_type_name_walk = (rep_type: RepTypeStructElementData): Array<string> => {
            const rep_type_names = new Array<string>()
            const rep_type_children = rep_type.children
            if (rep_type_children) {
                rep_type_children.forEach(child_rep_type => {
                    if (child_rep_type.check_when_inited) {
                        rep_type_names.push(child_rep_type.rep_type_name)
                    }
                    if (child_rep_type) {
                        rep_type_names.push(...rep_type_name_walk(child_rep_type))
                    }
                })
            }
            return rep_type_names
        }
        query.rep_types_in_sidebar = rep_type_name_walk(application_config.rep_type_struct)
        query.apply_rep_summary_sets_to_detaul(application_config, new Set(query.rep_types_in_sidebar), new Set(query.devices_in_sidebar))

        // Tag（TimeIsタグはグループ未使用=nullのまま。サイドバーのTimeIsタグツリーは
        // null のとき collect_inited_tag_names でフォールバック表示する）
        query.tags = collect_inited_tag_names(application_config.tag_struct)

        // Calendar
        if (application_config.rykv_default_period !== -1) {
            query.calendar_start_date = moment(moment().add(-application_config.rykv_default_period, "days").format("YYYY-MM-DD 00:00:00 ZZ")).toDate()
            query.calendar_end_date = moment(moment().format("YYYY-MM-DD 00:00:00 ZZ")).add(1, "days").add(-1, "milliseconds").toDate()
        }

        query.apply_hide_tags(application_config)

        return query
    }

    // ApplicationConfigから、デフォルトの検索条件を生成する。（mi用）
    static generate_default_query_for_mi(application_config: ApplicationConfig): FindKyouQuery {
        const query = new FindKyouQuery()

        // 対象はの3つ。ほかは初期値
        // RepのSummary, Detail
        // Tag
        // Calendar

        // RepはQuery時点では全部入れる。（サーバサイドでMiのRepのみに絞る考慮が入っている）
        // rep_name を持たない混入ノードは検索条件に入れない
        let rep_name_walk = (_rep: RepStructElementData): Array<string> => []
        rep_name_walk = (rep: RepStructElementData): Array<string> => {
            const rep_names = new Array<string>()
            const rep_children = rep.children
            if (rep_children) {
                rep_children.forEach(child_rep => {
                    if (typeof child_rep.rep_name === 'string') {
                        rep_names.push(child_rep.rep_name)
                    }
                    if (child_rep) {
                        rep_names.push(...rep_name_walk(child_rep))
                    }
                })
            }
            return rep_names
        }
        query.reps = rep_name_walk(application_config.rep_struct)

        // Tag
        query.tags = collect_inited_tag_names(application_config.tag_struct)

        // Calendarはない。

        // Mi
        query.for_mi = true

        query.apply_hide_tags(application_config)

        return query
    }

    // ApplicationConfigから、デフォルトの検索条件を生成する。（plaing検索用）
    // カスタム検索条件（plaing_timeis_json_data）が未設定のときの既定動作と、
    // 検索条件エディタの初期表示・クリアの両方がこれを使う。
    // rykv/mi用と違い apply_hide_tags は呼ばない
    // （従来のplaing検索は非表示タグを適用していなかったため、既定動作を変えない）。
    static generate_default_query_for_plaing_timeis(application_config: ApplicationConfig): FindKyouQuery {
        const query = new FindKyouQuery()
        // タグフィルタは未使用（null）。旧 use_tags=false と等価
        query.tags = null

        // RepはQuery時点では全部入れる。（サーバサイドでplaing_timeによりTimeIsのRepのみに絞る考慮が入っている）
        // rep_name を持たない混入ノードは検索条件に入れない
        let rep_name_walk = (_rep: RepStructElementData): Array<string> => []
        rep_name_walk = (rep: RepStructElementData): Array<string> => {
            const rep_names = new Array<string>()
            const rep_children = rep.children
            if (rep_children) {
                rep_children.forEach(child_rep => {
                    if (typeof child_rep.rep_name === 'string') {
                        rep_names.push(child_rep.rep_name)
                    }
                    if (child_rep) {
                        rep_names.push(...rep_name_walk(child_rep))
                    }
                })
            }
            return rep_names
        }
        query.reps = rep_name_walk(application_config.rep_struct)

        return query
    }

    // この検索条件に対して、RepのSummaryをDetailに適用する
    // rep_types, devicesから、repsを算出する
    apply_rep_summary_to_detaul(application_config: ApplicationConfig): void {
        const reps = application_config.rep_struct.children
        const rep_types = application_config.rep_type_struct.children
        const devices = application_config.device_struct.children

        if (!reps || !devices || !rep_types) {
            return
        }

        // チェック済みキーを先に1回の走査で集める。以前はrepノード1つごとに
        // rep_type/deviceツリーを丸ごと再走査する O(reps×(types+devices)) で、
        // repが数百ある環境ではサイドバー同期のフリーズ要因になっていた。
        // indeterminate=false の書き込み先は従来と同じ（書く回数が1回になるだけ）。
        const collect_checked_keys = (structs: Array<FoldableStructModel>): Set<string> => {
            const checked_keys = new Set<string>()
            const walk = (struct: FoldableStructModel): void => {
                struct.indeterminate = false
                if (struct.is_checked) {
                    checked_keys.add(struct.key)
                }
                if (struct.children) {
                    struct.children.forEach(child => walk(child))
                }
            }
            structs.forEach(struct => walk(struct))
            return checked_keys
        }
        this.apply_rep_summary_sets_to_detaul(application_config, collect_checked_keys(rep_types), collect_checked_keys(devices))
    }

    // サマリ(チェック済みのrep_type/deviceキー集合)からrepsを算出して自身へ設定する共通部。
    // サイドバーの対話操作はツリーのis_checked由来のセットを、デフォルト検索条件の生成は
    // 自身のrep_types_in_sidebar/devices_in_sidebar(=check_when_inited由来)のセットを渡す。
    // デフォルト生成でツリーのis_checkedを見ると、永続化された古いチェック状態が
    // check_when_initedと食い違っている実環境でrepsが空になる
    /* private */ apply_rep_summary_sets_to_detaul(application_config: ApplicationConfig, checked_type_keys: Set<string>, checked_device_keys: Set<string>): void {
        const reps = application_config.rep_struct.children
        if (!reps) {
            return
        }

        const check_target_rep_names = new Array<string>()
        let walk_rep = (_rep: RepStructElementData): void => { }
        walk_rep = (rep: RepStructElementData): void => {
            rep.is_checked = false
            const rep_struct = this.rep_to_struct(rep)

            if (checked_type_keys.has(rep_struct.type)
                && checked_device_keys.has(rep_struct.device)
                && !rep.ignore_check_rep_rykv) {
                check_target_rep_names.push(rep.rep_name)
            }

            if (rep.children) {
                rep.children.forEach(child_rep => walk_rep(child_rep))
            }
        }
        reps.forEach(child_rep => walk_rep(child_rep))

        this.reps = check_target_rep_names
    }

    // 引数のrep.nameから{type: "", device: "", time: ""}なオブジェクトを作ります。
    // rep.nameがdvnf形式ではない場合は、{type: rep.name, device: 'なし', time: ''}が作成されます。
    // 実DBには rep_name を持たないノードが混入した実例がある(REP_TYPE_STRUCT の内容が
    // REP_STRUCT キーへ保存されていた)。ここで throw すると既定検索条件の生成と
    // サマリ→記録先詳細の算出が丸ごと死ぬため、dvnf非形式の空名として扱う
    /* private */ rep_to_struct(rep: RepStructElementData): { type: string, device: string, time: string } {
        const rep_name = typeof rep.rep_name === 'string' ? rep.rep_name : ''
        const spl = rep_name.split('_')
        if (spl.length !== 3) {
            return {
                type: rep_name,
                device: 'なし',
                time: ''
            }
        }
        return {
            type: spl[0],
            device: spl[1],
            time: spl[2]
        }
    }
    apply_hide_tags(application_config: ApplicationConfig): void {
        this.hide_tags.splice(0)

        let tag_name_walk = (_tag: TagStructElementData): void => { }
        tag_name_walk = (tag: TagStructElementData): void => {
            const tag_children = tag.children
            if (tag.is_force_hide) {
                this.hide_tags.push(tag.tag_name)
            }
            if (tag_children) {
                tag_children.forEach(child_tag => {
                    if (child_tag) {
                        tag_name_walk(child_tag)
                    }
                })
            }
        }
        tag_name_walk(application_config.tag_struct)
    }
}
