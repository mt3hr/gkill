'use strict'

import { KFTLRequest } from '../kftl-request'
import type { KFTLStatementLineContext } from '../kftl-statement-line-context'
import { GkillError } from '@/classes/api/gkill-error'
import type { GkillAPI } from '@/classes/api/gkill-api'
import type { ApplicationConfig } from '@/classes/datas/config/application-config'
import type { RepeatSpec } from '../kftl_repeat/kftl-repeat-spec'
import { i18n } from '@/i18n'

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


    /**
     * 「繰り返しの対象になるレコードが無い」ことを表す。
     *
     * プロトタイプはタグ・テキスト・関連時刻の置き場所でしかない。「？？」を本文より前や
     * タグ行だけの位置に書くと付け先がこれしか無く、受け入れると何も作らずに黙って終わる。
     */
    override set_repeat_spec(_spec: RepeatSpec): void {
        throw new Error(i18n.global.t("KFTL_REPEAT_NO_TARGET_MESSAGE_TITLE"))
    }

    override clone_for_repeat(new_request_id: string, day_shift: number): KFTLRequest {
        const cloned = new KFTLPrototypeRequest(new_request_id, this.get_context())
        this.copy_base_state_for_repeat(cloned, day_shift)
        return cloned
    }

    override async find_existing_for_repeat(_gkill_api: GkillAPI, _application_config: ApplicationConfig | null, _from: Date, _to: Date): Promise<Set<number>> {
        // set_repeat_spec が断るのでここへは来ない
        return new Set<number>()
    }

}


