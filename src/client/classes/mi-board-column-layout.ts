'use strict'

/**
 * Mi板の列（mi-view / shared-mi-view）のレイアウト定数。
 *
 * 列は「板名の見出し + KyouListView」で app_content_height を分け合う。
 * KyouListView の実高さは渡した list_height ちょうど（本体 list_height - 48 + フッター48px）
 * なので、見出しのぶんを引いて渡さないと列がコンテンツ領域からはみ出す。
 * 逆に引きすぎると列の下に空白が残る。
 *
 * ここの値は Vuetify の v-card-title の既定サイズ
 * （font-size 1.375rem = 22px × line-height 1.2727… = 28px、padding .5rem 上下で16px）
 * と一致しているが、Vuetify のバージョンアップで黙ってずれないよう、
 * 見出し側にも `.mi_board_column_title { height: … }` を書いて固定してある。
 * 変更するときは両方を同時に直すこと。
 */
export const MI_BOARD_TITLE_HEIGHT = 44
