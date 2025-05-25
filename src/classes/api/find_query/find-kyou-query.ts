'use strict'

import { MiCheckState } from "./mi-check-state"
import { MiSortType } from "./mi-sort-type"

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
        cloned.include_create_mi = json.include_create_mi
        cloned.include_check_mi = json.include_check_mi
        cloned.include_limit_mi = json.include_limit_mi
        cloned.include_start_mi = json.include_start_mi
        cloned.include_end_mi = json.include_end_mi
        cloned.include_end_timeis = json.include_end_timeis
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
        this.use_timeis_tags = false
        this.timeis_tags = new Array<string>()
        this.timeis_tags_and = false
        this.tags = new Array<string>()
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
}
