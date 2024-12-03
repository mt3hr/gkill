'use strict'

export abstract class FindQueryBase {

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

    async set_other_value(key: string, value: Object): Promise<void> {
        (this as any)[key] = value
    }

    abstract clone(): FindQueryBase

    constructor() {

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

    }

    parse_words_and_not_words() {
        //TODO
        //TODO timeis
    }

}


