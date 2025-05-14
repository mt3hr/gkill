'use strict'

import { Kyou } from '@/classes/datas/kyou'
import { GkillAPIResponse } from '../gkill-api-response'
import type { Text } from '@/classes/datas/text'
import type { Mi } from '@/classes/datas/mi'
import type { Tag } from '@/classes/datas/tag'
import { TimeIs } from '@/classes/datas/time-is'
import type { Kmemo } from '@/classes/datas/kmemo'
import type { KC } from '@/classes/datas/kc'
import type { Nlog } from '@/classes/datas/nlog'
import type { Lantana } from '@/classes/datas/lantana'
import type { URLog } from '@/classes/datas/ur-log'
import type { IDFKyou } from '@/classes/datas/idf-kyou'
import type { ReKyou } from '@/classes/datas/re-kyou'
import type { GitCommitLog } from '@/classes/datas/git-commit-log'
import type { GPSLog } from '@/classes/datas/gps-log'

export class GetSharedKyousResponse extends GkillAPIResponse {
    title: string
    view_type: string
    kyous: Array<Kyou>
    kmemos: Array<Kmemo>
    kcs: Array<KC>
    mis: Array<Mi>
    nlogs: Array<Nlog>
    lantanas: Array<Lantana>
    timeiss: Array<TimeIs>
    urlogs: Array<URLog>
    idf_kyous: Array<IDFKyou>
    rekyous: Array<ReKyou>
    git_commit_logs: Array<GitCommitLog>
    gps_logs: Array<GPSLog>
    attached_tags: Array<Tag>
    attached_texts: Array<Text>
    attached_timeiss: Array<TimeIs>

    constructor() {
        super()
        this.title = ""
        this.view_type = "rykv"
        this.kyous = new Array<Kyou>()
        this.kmemos = new Array<Kmemo>()
        this.kcs = new Array<KC>()
        this.mis = new Array<Mi>()
        this.nlogs = new Array<Nlog>()
        this.lantanas = new Array<Lantana>()
        this.timeiss = new Array<TimeIs>()
        this.urlogs = new Array<URLog>()
        this.idf_kyous = new Array<IDFKyou>()
        this.rekyous = new Array<ReKyou>()
        this.git_commit_logs = new Array<GitCommitLog>()
        this.gps_logs = new Array<GPSLog>()
        this.attached_tags = new Array<Tag>()
        this.attached_texts = new Array<Text>()
        this.attached_timeiss = new Array<TimeIs>()
    }
}


