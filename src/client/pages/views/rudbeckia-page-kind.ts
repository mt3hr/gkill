'use strict'

/**
 * ポート（開発コード rudbeckia）がダイアログとして開ける画面の種類。
 *
 * メモ帳 / 打刻メモ帳は専用の追加ダイアログ（kftl-dialog / mkfl-dialog）が既にあるので
 * ここには含めない。記録(kyou)とさいはては対象外。
 */
export type RudbeckiaPageKind = 'rykv' | 'mi' | 'plaing' | 'dashboard'

export interface OpenedRudbeckiaPageDialog {
    id: string

    kind: RudbeckiaPageKind

    /**
     * 位置・サイズの保存キーを決める番号。**種類ごと**に空いている最小の番号を取る
     * （メモ帳ウィンドウ = kftl-dialog-host と同じ方式）。
     * 同じキーで2枚出すと `${key}:pos` / `:size` を奪い合うので必ず分ける
     */
    slot_index: number

    /**
     * 中央からずらす量を決める番号。**種類をまたいで**空いている最小の番号を取る。
     *
     * slot_index で代用してはいけない ―― 種類ごとの採番なので4種類とも 0 になり、
     * 4枚が完全に重なって1枚にしか見えなくなる
     */
    cascade_index: number
}

/**
 * 同時に開ける画面ウィンドウの上限（種類ごと）。
 *
 * 列の検索条件とスクロール位置は `gkill-api.ts` の保存キーを
 * `slot_index` 由来の枝番で分けてあるので、複数枚でも互いを壊さない
 * （slot 0 は従来キーそのまま＝単独ページと同じ列を引き継ぐ）。
 *
 * ライフログビュー1枚で数十万件の配列を持ちうるので、無制限にはしない。
 * 上限に達したら新しく開かず、その種類の最前面へフォーカスを移す。
 */
export const RUDBECKIA_PAGE_DIALOG_MAX_COUNT_PER_KIND = 3

/** ずらす段数の上限。これを超えたら先頭へ戻して、画面外まで流れないようにする */
export const RUDBECKIA_PAGE_DIALOG_CASCADE_SLOT_COUNT = 6
