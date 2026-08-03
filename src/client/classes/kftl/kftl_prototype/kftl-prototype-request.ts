'use strict'

import { KFTLRequest } from '../kftl-request'
import type { KFTLStatementLineContext } from '../kftl-statement-line-context'
import { GkillError } from '@/classes/api/gkill-error'

export class KFTLPrototypeRequest extends KFTLRequest {

    constructor(request_id: string, context: KFTLStatementLineContext) {
        super(request_id, context)
    }

    static is_prototype_request(request: KFTLRequest): boolean {
        // クラス名ではなくコンストラクタの同一性で比較する。
        // minify するとクラス名は短縮され、別クラスが同じ名前になりうるため。
        return request.constructor === KFTLPrototypeRequest

    }

    async do_request(): Promise<Array<GkillError>> {
        return new Array<GkillError>()
    }

}


