'use strict'

import { InfoBase } from './info-base'
import type { GkillError } from '../api/gkill-error'
import type { GitCommitLog } from './git-commit-log'
import type { IDFKyou } from './idf-kyou'
import type { Kmemo } from './kmemo'
import type { Lantana } from './lantana'
import type { Mi } from './mi'
import type { Nlog } from './nlog'
import type { ReKyou } from './re-kyou'
import type { TimeIs } from './time-is'
import type { URLog } from './ur-log'
import { InfoIdentifier } from './info-identifier'

export class Kyou extends InfoBase {

    image_source: string

    attached_histories: Array<Kyou>

    typed_kmemo: Kmemo | null

    typed_urlog: URLog | null

    typed_nlog: Nlog | null

    typed_timeis: TimeIs | null

    typed_mi: Mi | null

    typed_lantana: Lantana | null

    typed_idf_kyou: IDFKyou | null

    typed_git_commit_log: GitCommitLog | null

    typed_rekyou: ReKyou | null

    async load_attached_histories(): Promise<Array<GkillError>> {
        throw new Error('Not implemented')
    }

    async load_attached_datas(): Promise<Array<GkillError>> {
        throw new Error('Not implemented')
    }

    async clear_attached_histories(): Promise<Array<GkillError>> {
        throw new Error('Not implemented')
    }

    async clear_attached_datas(): Promise<Array<GkillError>> {
        throw new Error('Not implemented')
    }

    async load_typed_kmemo(): Promise<Array<GkillError>> {
        throw new Error('Not implemented')
    }

    async load_typed_urlog(): Promise<Array<GkillError>> {
        throw new Error('Not implemented')
    }

    async load_typed_nlog(): Promise<Array<GkillError>> {
        throw new Error('Not implemented')
    }

    async load_typed_timeis(): Promise<Array<GkillError>> {
        throw new Error('Not implemented')
    }

    async load_typed_mi(): Promise<Array<GkillError>> {
        throw new Error('Not implemented')
    }

    async load_typed_lantana(): Promise<Array<GkillError>> {
        throw new Error('Not implemented')
    }

    async load_typed_idf_kyou(): Promise<Array<GkillError>> {
        throw new Error('Not implemented')
    }

    async load_typed_git_commit_log(): Promise<Array<GkillError>> {
        throw new Error('Not implemented')
    }

    async load_typed_rekyou(): Promise<Array<GkillError>> {
        throw new Error('Not implemented')
    }

    async clear_typed_datas(): Promise<Array<GkillError>> {
        throw new Error('Not implemented')
    }

    async reload(): Promise<Array<GkillError>> {
        throw new Error('Not implemented')
    }

    async clone(): Promise<Kyou> {
        throw new Error('Not implemented')
    }

    generate_info_identifer(): InfoIdentifier {
        const info_identifer = new InfoIdentifier()
        info_identifer.id = this.id
        info_identifer.create_time = this.create_time
        info_identifer.update_time = this.update_time
        return info_identifer
    }

    constructor() {
        super()
        this.image_source = ""

        this.attached_histories = new Array<Kyou>()

        this.typed_kmemo = null

        this.typed_urlog = null

        this.typed_nlog = null

        this.typed_timeis = null

        this.typed_mi = null

        this.typed_lantana = null

        this.typed_idf_kyou = null

        this.typed_git_commit_log = null

        this.typed_rekyou = null
    }

}


