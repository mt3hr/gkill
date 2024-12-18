'use strict'

import { GkillAPI } from '../gkill-api'
import { FindQueryBase } from './find-query-base'

export class FindKyouQuery extends FindQueryBase {
    query_id: string

    reps: Array<string>
    is_image_only: boolean
    devices: Array<string>
    rep_types: Array<string>
    is_focus_kyou: boolean

    clone(): FindKyouQuery {
        const cloned = new FindKyouQuery()
        cloned.query_id = this.query_id
        cloned.update_cache = this.update_cache
        cloned.use_words = this.use_words
        cloned.keywords = this.keywords
        cloned.words_and = this.words_and
        cloned.words = this.words
        cloned.not_words = this.not_words
        cloned.use_timeis = this.use_timeis
        cloned.timeis_words_and = this.timeis_words_and
        cloned.timeis_keywords = this.timeis_keywords
        cloned.timeis_words = this.timeis_words
        cloned.timeis_not_words = this.timeis_not_words
        cloned.use_timeis_tags = this.use_timeis_tags
        cloned.timeis_tags = this.timeis_tags
        cloned.timeis_tags_and = this.timeis_tags_and
        cloned.tags = this.tags
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
        cloned.reps = this.reps
        cloned.is_image_only = this.is_image_only
        cloned.devices = this.devices
        cloned.rep_types = this.rep_types
        cloned.use_update_time = this.use_update_time
        cloned.update_time = this.update_time
        return cloned
    }

    constructor() {
        super()
        this.query_id = ""
        this.reps = new Array<string>()
        this.is_image_only = false
        this.devices = new Array<string>()
        this.rep_types = new Array<string>()
        this.is_focus_kyou = false
    }
}
