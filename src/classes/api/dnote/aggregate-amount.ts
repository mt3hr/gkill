'use strict'

import { Kyou } from "@/classes/datas/kyou"

export class AggregateAmount {
    amount: number
    title: string
    related_time: Date
    related_kyou: Kyou

    constructor() {
        this.amount = 0
        this.title = ""
        this.related_time = new Date(0)
        this.related_kyou = new Kyou()
    }
}

export async function aggregate_amounts_from_kyous(kyous: Array<Kyou>, abort_controller: AbortController): Promise<Array<AggregateAmount>> {
    const aggregate_amounts = new Array<AggregateAmount>()
    for (let i = 0; i < kyous.length; i++) {
        const kyou = kyous[i]
        kyou.abort_controller = abort_controller
        if (!kyou.typed_nlog) {
            await kyou.load_typed_nlog()
        }
    }
    for (let i = 0; i < kyous.length; i++) {
        const kyou = kyous[i]
        const aggregate_amount = new AggregateAmount()
        if (kyou.typed_nlog) {
            if (isNaN(kyou.typed_nlog.amount.valueOf())) {
                // NaNはスキップ
                continue
            }
            aggregate_amount.amount = kyou.typed_nlog.amount.valueOf()
            aggregate_amount.title = kyou.typed_nlog.title
            aggregate_amount.related_time = kyou.related_time
            aggregate_amount.related_kyou = kyou
            aggregate_amounts.push(aggregate_amount)
        }
    }
    return aggregate_amounts
}