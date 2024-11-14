'use strict'

import type { KFTLRequest } from "./kftl-request"
import { KFTLPrototypeRequest } from "./kftl_prototype/kftl-prototype-request"

export class KFTLRequestMap extends Map<String, KFTLRequest> {
    public override set(request_id: string, request: KFTLRequest) {
        const setted_request = this.get(request_id)
        if (setted_request) {
            if (KFTLPrototypeRequest.is_prototype_request(setted_request)) {
                request.set_tags(setted_request.get_tags())
                request.set_texts(setted_request.get_texts())
                request.set_related_time(setted_request.get_related_time())
            } else {
                throw new Error(`${request_id}のリクエストはすでに設定されています`)
            }
        }
        return super.set(request_id, request)
    }


}


