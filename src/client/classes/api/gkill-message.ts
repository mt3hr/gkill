'use strict'

export class GkillMessage {

    message_code: string

    message: string

    // "info"（既定。数秒で消える）か "warning"（成功はしたが対処が要る。閉じるまで残す）
    level: string

    show_keep: boolean

    constructor() {
        this.message_code = ""
        this.message = ""
        this.level = "info"
        this.show_keep = false
    }

}
