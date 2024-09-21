'use strict'

import type { KFTLRequest } from "./kftl-request"

export class KFTLRequestMap extends Map<String, KFTLRequest> {

    get(request_id: string): KFTLRequest {
        throw new Error('Not implemented')
    }

    set(request_id: string, request: KFTLRequest): this {
        throw new Error('Not implemented')
    }

}


