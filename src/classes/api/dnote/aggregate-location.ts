'use strict'

import type { Kyou } from "@/classes/datas/kyou"
import moment from "moment"

export class AggregateLocation {
    title: string
    duration_milli_second: number
    constructor() {
        this.title = ""
        this.duration_milli_second = 0
    }
}

export async function aggregate_locations_from_kyous(kyous: Array<Kyou>, abort_controller: AbortController): Promise<Array<AggregateLocation>> {
    const aggregate_locations = new Array<AggregateLocation>()
    const aggregate_locations_map = new Map<string, AggregateLocation>()// map[title]aggregate_location
    for (let i = 0; i < kyous.length; i++) {
        const kyou = kyous[i]
        kyou.abort_controller = abort_controller
        if (kyou.data_type.startsWith("timeis")) {
            if (!kyou.typed_timeis) {
                await kyou.load_typed_timeis()
            }
        } else if (kyou.data_type.startsWith("kmemo")) {
            if (!kyou.typed_kmemo) {
                await kyou.load_typed_kmemo()
            }
        }
    }

    for (let i = 0; i < kyous.length; i++) {
        const kyou = kyous[i]
        let title = ""
        let duration_milli_second = 0

        if (kyou.data_type.startsWith("timeis")) {
            if (kyou.typed_timeis) {
                title = kyou.typed_timeis.title
                const end_time = kyou.typed_timeis.end_time?.getTime()
                if ((kyou.typed_timeis.start_time.getTime() < (end_time ? end_time : 0))) {
                    duration_milli_second = Math.abs(moment.duration(moment(kyou.typed_timeis.start_time).diff(kyou.typed_timeis.end_time)).asMilliseconds())
                } else {
                    duration_milli_second = 0
                }
            }
        } else if (kyou.data_type.startsWith("kmemo")) {
            if (kyou.typed_kmemo) {
                title = kyou.typed_kmemo.content
                duration_milli_second = 0
            }
        }

        if (isNaN(duration_milli_second)) {
            // NaNはスキップ
            continue
        }

        // すでにあればそこに加算する。なければ追加する
        const exist_aggregate_locations = aggregate_locations_map.get(title)
        if (exist_aggregate_locations) {
            exist_aggregate_locations.duration_milli_second += duration_milli_second
            aggregate_locations_map.set(title, exist_aggregate_locations)
        } else {
            const new_aggregate_location = new AggregateLocation()
            new_aggregate_location.title = title
            new_aggregate_location.duration_milli_second = duration_milli_second
            aggregate_locations_map.set(title, new_aggregate_location)
        }
    }
    aggregate_locations_map.forEach(aggregate_location => {
        aggregate_locations.push(aggregate_location)
    })
    return aggregate_locations
}
