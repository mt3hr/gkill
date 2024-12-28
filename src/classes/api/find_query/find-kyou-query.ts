'use strict'

import { GkillAPI } from '../gkill-api'
import { FindQueryBase } from './find-query-base'

export class FindKyouQuery extends FindQueryBase {
    query_id: string

    use_rep_types: boolean
    rep_types: Array<string>
    reps: Array<string>

    devices_in_sidebar: Array<string>
    rep_types_in_sidebar: Array<string>
    is_enable_map_circle_in_sidebar: boolean
    is_image_only_in_sidebar: boolean
    is_focus_kyou_in_list_view: boolean

    clone(): FindKyouQuery {
        const cloned = new FindKyouQuery()
        cloned.query_id = this.query_id
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
        cloned.is_image_only_in_sidebar = this.is_image_only_in_sidebar
        cloned.devices_in_sidebar = this.devices_in_sidebar.concat()
        cloned.rep_types_in_sidebar = this.rep_types_in_sidebar.concat()
        cloned.use_update_time = this.use_update_time
        cloned.update_time = this.update_time
        cloned.is_enable_map_circle_in_sidebar = this.is_enable_map_circle_in_sidebar
        cloned.use_rep_types = this.use_rep_types
        cloned.rep_types = this.rep_types.concat()
        return cloned
    }

    constructor() {
        super()
        this.query_id = ""
        this.reps = new Array<string>()
        this.is_image_only_in_sidebar = false
        this.devices_in_sidebar = new Array<string>()
        this.rep_types_in_sidebar = new Array<string>()
        this.is_focus_kyou_in_list_view = false
        this.is_enable_map_circle_in_sidebar = false
        this.use_rep_types = false
        this.rep_types =  new Array<string>()
    }
}
