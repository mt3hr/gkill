'use strict'

import type { Kyou } from "@/classes/datas/kyou"
import moment from "moment"

export class AggregatePeople {
    title: string
    duration_milli_second: number
    type: Set<'通話' | '対面'>

    constructor() {
        this.title = ""
        this.duration_milli_second = 0
        this.type = new Set<'通話' | '対面'>()
    }
}

export async function aggregate_peoples_from_kyous(kyous: Array<Kyou>): Promise<Array<AggregatePeople>> {
    const aggregate_peoples = new Array<AggregatePeople>()
    const aggregate_peoples_map = new Map<string, AggregatePeople>()// map[title]aggregate_people
    for (let i = 0; i < kyous.length; i++) {
        const kyou = kyous[i]
        let title = ""
        let duration_milli_second = 0
        let type: '通話' | '対面' = '対面'
        if (kyou.data_type.startsWith("timeis")) {
            if (!kyou.typed_timeis) {
                await kyou.load_typed_timeis()
            }
            if (kyou.typed_timeis) {
                title = kyou.typed_timeis.title
                duration_milli_second = Math.abs(moment.duration(moment(kyou.typed_timeis.start_time).diff(kyou.typed_timeis.end_time)).asMilliseconds())
            }
        } else if (kyou.data_type.startsWith("kmemo")) {
            if (!kyou.typed_kmemo) {
                await kyou.load_typed_kmemo()
            }
            if (kyou.typed_kmemo) {
                title = kyou.typed_kmemo.content
                duration_milli_second = 0
            }
        }
        await kyou.load_attached_tags()
        for (let j = 0; j < kyou.attached_tags.length; j++) {
            const tag = kyou.attached_tags[i]
            if (tag.tag === "あ") {
                type = "対面"
            } else if (tag.tag === "通話") {
                type = "通話"
            }
        }
        if (type === '通話' || type === '対面') {
            // すでにあればそこに加算する。なければ追加する
            const exist_aggregate_peoples = aggregate_peoples_map.get(title)
            if (exist_aggregate_peoples) {
                exist_aggregate_peoples.duration_milli_second += duration_milli_second
                exist_aggregate_peoples.type.add(type)
                aggregate_peoples_map.set(title, exist_aggregate_peoples)
            } else {
                const new_aggregate_people = new AggregatePeople()
                new_aggregate_people.title = title
                new_aggregate_people.duration_milli_second = duration_milli_second
                new_aggregate_people.type.add(type)
                aggregate_peoples_map.set(title, new_aggregate_people)
            }
        }
    }
    aggregate_peoples_map.forEach(aggregate_people => {
        aggregate_peoples.push(aggregate_people)
    })
    return aggregate_peoples
}

