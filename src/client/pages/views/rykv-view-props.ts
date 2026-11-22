'use strict'

import type { KyouChangeChannel } from "@/classes/kyou-change-bus"
import type { GkillPropsBase } from "./gkill-props-base"

export interface RykvViewProps extends GkillPropsBase {
    app_title_bar_height: number
    app_content_height: number
    app_content_width: number
    is_shared_rykv_view: boolean
    // HTML へ出力したファイルの中で描かれている。共有ページ（is_shared_rykv_view）と
    // 同じく読み取り専用だが、**共有が落としている集計（Dnote）は残す**ので別フラグにする。
    // 出力時に選ばれなかった区画は下の is_exported_with_* で落とす
    is_exported_rykv_view: boolean
    // 出力時に「含めるもの」で選ばれたか。is_exported_rykv_view が false のときは見ない
    is_exported_with_dnote: boolean
    is_exported_with_map: boolean
    // 検索条件サイドバー。出力先では絞り込み専用（焼いた記録の外は出ない）
    is_exported_with_sidebar: boolean
    share_title: string
    // ポート(rudbeckia)のフローティングダイアログの中で描かれている。
    // このビューは単独ページとしても使われるので、既定は false。
    // true のとき: 自前のFABを出さない(ポートのFABが唯一)、Enter/Ctrl+Vの
    // ショートカットを登録しない(windowレベルなので枚数ぶん多重登録される)、
    // 画面切替メニューはページ遷移せず requested_navigate_page を上げる
    is_hosted_in_dialog: boolean
    // 列の検索条件とスクロール位置の保存キーの枝番。単独ページと1枚目は空文字
    // （＝従来キーそのまま）。ポートで2枚目以降を開くときだけ '2' / '3' … が入る。
    // 分けないと2枚目が1枚目の列条件を上書きする
    column_state_instance_key: string
    // 画面をまたいだ変更通知の口。null が「単独ページとして動く」＝ publish も購読もしない。
    // ポート(rudbeckia)で開いたときだけ非 null が入る
    kyou_change_channel: KyouChangeChannel | null
    // ApplicationConfigの取得に失敗した。読み込み中オーバーレイを
    // スピナーからエラー表示＋再試行へ差し替えるために使う
    application_config_load_failed: boolean
}
