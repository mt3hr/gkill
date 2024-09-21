'use strict'

import type { GkillAPIResponse } from '@/classes/api/gkill-api-response'
import { KFTLRequest } from '../kftl-request'
import type { KFTLStatementLineContext } from '../kftl-statement-line-context'

export class KFTLNlogRequest extends KFTLRequest {

    shop_name: string

    titles: Array<string>

    amounts: Array<Number>

    constructor(request_id: string, context: KFTLStatementLineContext) {
        super(request_id, context)
        this.shop_name = ""
        this.titles = new Array<string>
        this.amounts = new Array<Number>
    }

    async do_request(): Promise<Array<GkillAPIResponse>> {
        throw new Error('Not implemented')
    }

    async set_shop_name(shop_name: string): Promise<void> {
        throw new Error('Not implemented')
    }

    async add_title(title: string): Promise<void> {
        throw new Error('Not implemented')
    }

    async add_amount(amount: Number): Promise<void> {
        throw new Error('Not implemented')
    }

}


