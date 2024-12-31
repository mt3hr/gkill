'use strict'

import { Kyou } from '@/classes/datas/kyou'
import { GkillAPIResponse } from '../gkill-api-response'
import type { Text } from '@/classes/datas/text'
import type { Mi } from '@/classes/datas/mi'
import type { Tag } from '@/classes/datas/tag'
import { TimeIs } from '@/classes/datas/time-is'

export class GetSharedMiTasksResponse extends GkillAPIResponse {
    title: string
    mi_kyous: Array<Kyou>
    mis: Array<Mi>
    tags: Array<Tag>
    texts: Array<Text>
    timeiss: Array<TimeIs>

    constructor() {
        super()
        this.title = ""
        this.mi_kyous = new Array<Kyou>()
        this.mis = new Array<Mi>()
        this.tags = new Array<Tag>()
        this.texts = new Array<Text>()
        this.timeiss = new Array<TimeIs>()
    }
}


