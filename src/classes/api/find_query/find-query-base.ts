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
    use_plaing: boolean
    plaing_time: Date | null

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
        this.use_plaing = false
        this.plaing_time = null
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
