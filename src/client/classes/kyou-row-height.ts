'use strict'

/**
 * 一覧の行として描かれているかを判定する。
 *
 * KyouListViewの行は高さが固定でoverflow:hiddenなので、はみ出した分は切り落とされる。
 * 参照先Kyouを埋め込むビュー（ReKyou / MiReKyou）は、行に収まる高さしか無いときに
 * 表示を詰めたり、行数ぶん暴発する処理（エラー通知・リクエスト）を抑えたりする必要がある。
 *
 * 実際に渡ってくる高さは
 *   91      Mi画面・共有Mi画面・Dashboardのタスクボード行 (56 + 35)
 *   180     rykvの一覧・一覧ダイアログ・Plaing
 *   'unset' 詳細ペイン・追加/編集ダイアログ
 * の3通り。'unset' はNaNになるので行ではないと判定される。
 */
export const KYOU_ROW_MAX_HEIGHT = 120

export function is_row_height(height: number | string): boolean {
    const height_px = typeof height === 'number' ? height : Number.parseFloat(String(height))
    return Number.isFinite(height_px) && height_px < KYOU_ROW_MAX_HEIGHT
}
