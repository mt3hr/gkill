'use strict'

/**
 * 一覧の行として描かれているかを判定する。
 *
 * KyouListViewの行は高さが固定でoverflow:hiddenなので、はみ出した分は切り落とされる。
 * 参照先Kyouを埋め込むビュー（ReKyou / MiReKyou）は、行に収まる高さしか無いときに
 * 表示を詰めたり、行数ぶん暴発する処理（エラー通知・リクエスト）を抑えたりする必要がある。
 *
 * 行として渡ってくる高さは
 *   91      Mi画面・共有Mi画面・Dashboardのタスクボード行 (56 + 35)
 *   180     rykvの一覧・一覧ダイアログ・Plaing
 * の2通り。行ではない場所は 'unset' / 'auto' を渡す (NaNになるので行ではないと判定される)。
 *
 * 注意: パーセント文字列を渡してはいけない。Number.parseFloat('80%') は 80 になるので
 * 行と誤判定される。実際、Kyouダイアログが '80%' を渡していたせいでMiReKyouの参照先が
 * 丸ごと消えていた (2026-08-09 修正)。行ではない場所は必ず 'unset' か 'auto' にすること。
 * 画像一覧 (kyou-list-view.vue の is_image_only) だけは200pxのセルに詰めるため
 * '100%' を渡しており、行扱いになるのが正しい。
 */
export const KYOU_ROW_MAX_HEIGHT = 120

export function is_row_height(height: number | string): boolean {
    const height_px = typeof height === 'number' ? height : Number.parseFloat(String(height))
    return Number.isFinite(height_px) && height_px < KYOU_ROW_MAX_HEIGHT
}
