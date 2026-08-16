'use strict'

export class TextAreaInfo {

    /**
     * 測る対象の textarea。
     *
     * KFTLView はタブごとに同じ id の textarea を持ちうる（`/mkfl` ではインラインの
     * KFTLView とメモ帳ダイアログが同時にマウントされる）ので、id 引きだと他人の
     * textarea を測ってしまう。呼び出し元はテンプレート ref の実体をここに入れること。
     */
    text_area_element: HTMLElement | null

    /** text_area_element が取れないときのフォールバック用 */
    text_area_element_id: string

    constructor() {
        this.text_area_element = null
        this.text_area_element_id = ""

    }

}
