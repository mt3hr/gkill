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

export class FindKyouQuery {
    query_id: string

    use_tags: boolean
    use_reps: boolean
    update_cache: boolean
    use_words: boolean
    keywords: string
    words_and: boolean
    words: Array<string>
    not_words: Array<string>
    use_timeis: boolean
    timeis_words_and: boolean
    timeis_keywords: string
    timeis_words: Array<string>
    timeis_not_words: Array<string>
    use_timeis_tags: boolean
    timeis_tags: Array<string>
    timeis_tags_and: boolean
    tags: Array<string>
    hide_tags: Array<string>
    tags_and: boolean
    use_map: boolean
    map_latitude: Number
    map_longitude: Number
    map_radius: Number
    use_calendar: boolean
    calendar_start_date: Date | null
    calendar_end_date: Date | null
    use_plaing: boolean
    plaing_time: Date | null
    use_update_time: boolean
    update_time: Date | null

    use_period_of_time: boolean
    period_of_time_start_time_second: number | null
    period_of_time_end_time_second: number | null
    period_of_time_week_of_days: Array<number>

    use_rep_types: boolean
    rep_types: Array<string>
    reps: Array<string>

    devices_in_sidebar: Array<string>
    rep_types_in_sidebar: Array<string>
    is_enable_map_circle_in_sidebar: boolean
    is_image_only: boolean
    is_focus_kyou_in_list_view: boolean

    use_mi_board_name: boolean
    mi_board_name: string
    use_mi_sort_type: boolean
    mi_sort_type: MiSortType

    for_mi: boolean
    use_mi_check_state: boolean
    mi_check_state: MiCheckState
    include_create_mi: boolean
    include_check_mi: boolean
    include_limit_mi: boolean
    include_start_mi: boolean
    include_end_mi: boolean
    include_end_timeis: boolean

    use_include_id: boolean

    static parse_find_kyou_query(json: any): FindKyouQuery {
        const cloned = new FindKyouQuery()
        cloned.query_id = json.query_id
        cloned.use_reps = json.use_reps
        cloned.use_tags = json.use_tags
        cloned.update_cache = json.update_cache
        cloned.use_words = json.use_words
        cloned.keywords = json.keywords.concat()
        cloned.words_and = json.words_and
        cloned.words = json.words.concat()
        cloned.not_words = json.not_words.concat()
        cloned.use_timeis = json.use_timeis
        cloned.timeis_words_and = json.timeis_words_and
        cloned.timeis_keywords = json.timeis_keywords.concat()
        cloned.timeis_words = json.timeis_words.concat()
        cloned.timeis_not_words = json.timeis_not_words.concat()
        cloned.use_timeis_tags = json.use_timeis_tags
        cloned.timeis_tags = json.timeis_tags.concat()
        cloned.timeis_tags_and = json.timeis_tags_and
        cloned.tags = json.tags.concat()
        cloned.hide_tags = json.hide_tags.concat()
        cloned.tags_and = json.tags_and
        cloned.use_map = json.use_map
        cloned.map_latitude = json.map_latitude
        cloned.map_longitude = json.map_longitude
        cloned.map_radius = json.map_radius
        cloned.use_calendar = json.use_calendar
        cloned.calendar_start_date = json.calendar_start_date
        cloned.calendar_end_date = json.calendar_end_date
        cloned.use_plaing = json.use_plaing
        cloned.plaing_time = json.plaing_time
        cloned.reps = json.reps.concat()
        cloned.is_image_only = json.is_image_only
        cloned.devices_in_sidebar = json.devices_in_sidebar.concat()
        cloned.rep_types_in_sidebar = json.rep_types_in_sidebar.concat()
        cloned.use_update_time = json.use_update_time
        cloned.update_time = json.update_time
        cloned.is_enable_map_circle_in_sidebar = json.is_enable_map_circle_in_sidebar
        cloned.use_rep_types = json.use_rep_types
        cloned.rep_types = json.rep_types.concat()
        cloned.use_mi_board_name = json.use_mi_board_name
        cloned.mi_board_name = json.mi_board_name
        cloned.use_mi_sort_type = json.use_mi_sort_type
        cloned.mi_sort_type = json.mi_sort_type
        cloned.use_mi_check_state = json.use_mi_check_state
        cloned.mi_check_state = json.mi_check_state
        cloned.for_mi = json.for_mi
        cloned.use_period_of_time = json.use_period_of_time
        cloned.period_of_time_start_time_second = json.period_of_time_start_time_second
        cloned.period_of_time_end_time_second = json.period_of_time_end_time_second
        cloned.period_of_time_week_of_days = json.period_of_time_week_of_days
        cloned.include_create_mi = json.include_create_mi
        cloned.include_check_mi = json.include_check_mi
        cloned.include_limit_mi = json.include_limit_mi
        cloned.include_start_mi = json.include_start_mi
        cloned.include_end_mi = json.include_end_mi
        cloned.include_end_timeis = json.include_end_timeis
        cloned.use_include_id = json.use_include_id
        return cloned
    }

    clone(): FindKyouQuery {
        const cloned = new FindKyouQuery()
        cloned.query_id = this.query_id
        cloned.use_reps = this.use_reps
        cloned.use_tags = this.use_tags
        cloned.update_cache = this.update_cache
        cloned.use_words = this.use_words
        cloned.keywords = this.keywords.concat()
        cloned.words_and = this.words_and
        cloned.words = this.words.concat()
        cloned.not_words = this.not_words.concat()
        cloned.use_timeis = this.use_timeis
        cloned.timeis_words_and = this.timeis_words_and
        cloned.timeis_keywords = this.timeis_keywords.concat()
        cloned.timeis_words = this.timeis_words.concat()
        cloned.timeis_not_words = this.timeis_not_words.concat()
        cloned.use_timeis_tags = this.use_timeis_tags
        cloned.timeis_tags = this.timeis_tags.concat()
        cloned.timeis_tags_and = this.timeis_tags_and
        cloned.tags = this.tags.concat()
        cloned.hide_tags = this.hide_tags.concat()
        cloned.tags_and = this.tags_and
        cloned.use_map = this.use_map
        cloned.map_latitude = this.map_latitude
        cloned.map_longitude = this.map_longitude
        cloned.map_radius = this.map_radius
        cloned.use_calendar = this.use_calendar
        cloned.calendar_start_date = this.calendar_start_date
        cloned.calendar_end_date = this.calendar_end_date
        cloned.use_plaing = this.use_plaing
        cloned.plaing_time = this.plaing_time
        cloned.use_period_of_time = this.use_period_of_time
        cloned.period_of_time_start_time_second = this.period_of_time_start_time_second
        cloned.period_of_time_end_time_second = this.period_of_time_end_time_second
        cloned.period_of_time_week_of_days = this.period_of_time_week_of_days.concat()
        cloned.reps = this.reps.concat()
        cloned.is_image_only = this.is_image_only
        cloned.devices_in_sidebar = this.devices_in_sidebar.concat()
        cloned.rep_types_in_sidebar = this.rep_types_in_sidebar.concat()
        cloned.use_update_time = this.use_update_time
        cloned.update_time = this.update_time
        cloned.is_enable_map_circle_in_sidebar = this.is_enable_map_circle_in_sidebar
        cloned.use_rep_types = this.use_rep_types
        cloned.rep_types = this.rep_types.concat()
        cloned.use_mi_board_name = this.use_mi_board_name
        cloned.mi_board_name = this.mi_board_name
        cloned.use_mi_sort_type = this.use_mi_sort_type
        cloned.mi_sort_type = this.mi_sort_type
        cloned.use_mi_check_state = this.use_mi_check_state
        cloned.mi_check_state = this.mi_check_state
        cloned.for_mi = this.for_mi
        cloned.include_create_mi = this.include_create_mi
        cloned.include_check_mi = this.include_check_mi
        cloned.include_limit_mi = this.include_limit_mi
        cloned.include_start_mi = this.include_start_mi
        cloned.include_end_mi = this.include_end_mi
        cloned.include_end_timeis = this.include_end_timeis
        cloned.use_include_id = this.use_include_id
        return cloned
    }

    constructor() {
        this.query_id = ""
        this.use_tags = true
        this.use_reps = true
        this.update_cache = false
        this.use_words = false
        this.keywords = ""
        this.words_and = false
        this.words = new Array<string>
        this.not_words = new Array<string>()
        this.use_timeis = false
        this.timeis_keywords = ""
        this.timeis_words_and = false
        this.timeis_words = new Array<string>()
        this.timeis_not_words = new Array<string>()
        this.use_timeis_tags = true
        this.timeis_tags = new Array<string>()
        this.timeis_tags_and = false
        this.tags = new Array<string>()
        this.hide_tags = new Array<string>()
        this.tags_and = false
        this.use_map = false
        this.map_latitude = 0
        this.map_longitude = 0
        this.map_radius = 0
        this.use_calendar = false
        this.calendar_start_date = null
        this.calendar_end_date = null
        this.use_plaing = false
        this.plaing_time = null
        this.use_update_time = false
        this.update_time = null
        this.use_period_of_time = false
        this.period_of_time_start_time_second = null
        this.period_of_time_end_time_second = null
        this.period_of_time_week_of_days = [0, 1, 2, 3, 4, 5, 6]
        this.reps = new Array<string>()
        this.is_image_only = false
        this.devices_in_sidebar = new Array<string>()
        this.rep_types_in_sidebar = new Array<string>()
        this.is_focus_kyou_in_list_view = false
        this.is_enable_map_circle_in_sidebar = false
        this.use_rep_types = false
        this.rep_types = new Array<string>()
        this.use_mi_board_name = false
        this.mi_board_name = ""
        this.use_mi_sort_type = false
        this.mi_sort_type = MiSortType.estimate_start_time
        this.use_mi_check_state = false
        this.mi_check_state = MiCheckState.uncheck
        this.for_mi = false
        this.include_create_mi = true
        this.include_check_mi = false
        this.include_limit_mi = false
        this.include_start_mi = false
        this.include_end_mi = false
        this.include_end_timeis = true
        this.use_include_id = true
    }

    parse_words_and_not_words() {
        const words = new Array<string>()
        const not_words = new Array<string>()
        let nextIsNotWord = false
        const words_list = this.keywords.split(" ")
        for (let i = 0; i < words_list.length; i++) {
            const words_list_ = words_list[i].split("　")
            for (let j = 0; j < words_list_.length; j++) {
                let word = words_list_[j]
                if (word.startsWith("-")) {
                    nextIsNotWord = true
                    word = word.replace("-", "")
                }
                if (word === "") {
                    continue
                } else if (word === "-") {
                    nextIsNotWord = true
                    continue
                } else {
                    if (nextIsNotWord) {
                        not_words.push(word)
                    } else {
                        words.push(word)
                    }
                    nextIsNotWord = false
                }
            }
        }
        this.words = words
        this.not_words = not_words

        const timeis_words = new Array<string>()
        const timeis_not_words = new Array<string>()
        nextIsNotWord = false
        const timeis_words_list = this.timeis_keywords.split(" ")
        for (let i = 0; i < timeis_words_list.length; i++) {
            const timeis_words_list_ = timeis_words_list[i].split("　")
            for (let j = 0; j < timeis_words_list_.length; j++) {
                let word = timeis_words_list_[j]
                if (word.startsWith("-")) {
                    nextIsNotWord = true
                    word = word.replace("-", "")
                }
                if (word === "") {
                    continue
                } else if (word === "-") {
                    nextIsNotWord = true
                    continue
                } else {
                    if (nextIsNotWord) {
                        timeis_not_words.push(word)
                    } else {
                        timeis_words.push(word)
                    }
                    nextIsNotWord = false
                }
            }
        }
        this.timeis_words = timeis_words
        this.timeis_not_words = timeis_not_words
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
        query.apply_rep_summary_to_detaul(application_config)

        // Tag
        let tag_name_walk = (_tag: TagStructElementData): Array<string> => []
        tag_name_walk = (tag: TagStructElementData): Array<string> => {
            const tag_names = new Array<string>()
            const tag_children = tag.children
            if (tag_children) {
                tag_children.forEach(child_tag => {
                    if (child_tag.check_when_inited) {
                        tag_names.push(child_tag.tag_name)
                    }
                    if (child_tag) {
                        tag_names.push(...tag_name_walk(child_tag))
                    }
                })
            }
            return tag_names
        }
        query.tags = tag_name_walk(application_config.tag_struct)
        query.timeis_tags = tag_name_walk(application_config.tag_struct)

        // Calendar
        if (application_config.rykv_default_period !== -1) {
            query.use_calendar = true
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
        let rep_name_walk = (_rep: RepStructElementData): Array<string> => []
        rep_name_walk = (rep: RepStructElementData): Array<string> => {
            const rep_names = new Array<string>()
            const rep_children = rep.children
            if (rep_children) {
                rep_children.forEach(child_rep => {
                    rep_names.push(child_rep.rep_name)
                    if (child_rep) {
                        rep_names.push(...rep_name_walk(child_rep))
                    }
                })
            }
            return rep_names
        }
        query.reps = rep_name_walk(application_config.rep_struct)

        // Tag
        let tag_name_walk = (_tag: TagStructElementData): Array<string> => []
        tag_name_walk = (tag: TagStructElementData): Array<string> => {
            const tag_names = new Array<string>()
            const tag_children = tag.children
            if (tag_children) {
                tag_children.forEach(child_tag => {
                    if (child_tag.check_when_inited) {
                        tag_names.push(child_tag.tag_name)
                    }
                    if (child_tag) {
                        tag_names.push(...tag_name_walk(child_tag))
                    }
                })
            }
            return tag_names
        }
        query.tags = tag_name_walk(application_config.tag_struct)

        // Calendarはない。

        // Mi
        query.for_mi = true

        query.apply_hide_tags(application_config)

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

        const check_target_rep_names = new Array<string>()
        let walk_rep = (_rep: RepStructElementData): void => { }
        walk_rep = (rep: RepStructElementData): void => {
            rep.is_checked = false
            const rep_struct = this.rep_to_struct(rep)

            let type_is_match = false
            let device_is_match = false
            let walk = (_struct: FoldableStructModel): void => { }

            walk = (struct: FoldableStructModel): void => {
                struct.indeterminate = false
                if (struct.is_checked && struct.key == rep_struct.type) {
                    if (!type_is_match) {
                        type_is_match = true
                    }
                }
                if (struct.children) {
                    struct.children.forEach(child => walk(child))
                }
            }
            rep_types.forEach(rep_type => walk(rep_type))

            walk = (struct: FoldableStructModel): void => {
                struct.indeterminate = false
                if (struct.is_checked && struct.key == rep_struct.device) {
                    if (!device_is_match) {
                        device_is_match = true
                    }
                }
                if (struct.children) {
                    struct.children.forEach(child => walk(child))
                }
            }
            devices.forEach(device => walk(device))

            if (type_is_match && device_is_match && !rep.ignore_check_rep_rykv) {
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
    /* private */ rep_to_struct(rep: RepStructElementData): { type: string, device: string, time: string } {
        const spl = rep.rep_name.split('_')
        if (spl.length !== 3) {
            return {
                type: rep.rep_name,
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
