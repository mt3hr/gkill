'use strict'

import { GkillAPIResponse } from "../gkill-api-response"

export class ExportKyouHTMLResponse extends GkillAPIResponse {

    output_format: string

    file_name: string

    /** single のときだけ入る */
    html: string

    /** sidecar のときだけ入る。HTML と同名フォルダをまとめた ZIP の base64 */
    zip_base64: string

    /**
     * 予算に収まらない・読めないなどで一部を落としたとき真。
     * 何を落としたかは messages に warning（MSG000096）として入る
     */
    partial: boolean

    constructor() {
        super()
        this.output_format = ""
        this.file_name = ""
        this.html = ""
        this.zip_base64 = ""
        this.partial = false
    }

}
