'use strict'

import type { KyouChangeChannel } from "@/classes/kyou-change-bus"
import type { GkillPropsBase } from "./gkill-props-base"

/**
 * rykv の「列」の中身。検索条件サイドバー・列のテーブル・詳細ペイン・Ryuu・集計・
 * マップ・件数カレンダーをまとめて持つ。
 *
 * **これはモードを知らない。** 通常の rykv（`rykv-view.vue`）と HTML 出力
 * （`exported-rykv-view.vue`）の2つのラッパがこれをホストし、
 * 「何ができるか」を下の props へ**リテラルで**渡す。
 * 共有ページはこれも使わない（列の機構を一切使わないので `shared-rykv-view.vue` が別実装）。
 *
 * ラッパ側が持つのはアプリバーの中身・編集系のダイアログ群・FAB・ヘルプで、
 * それらはスロット（`app_bar_title` / `app_bar_actions` / `dialogs` / `fab`）で差し込む。
 * ダイアログ群は保存した記録を列へ差し込む必要があるので、`dialogs` スロットは
 * 中継束と列の状態をスロット props で渡す。
 */
export interface RykvColumnsViewProps extends GkillPropsBase {
    app_title_bar_height: number
    app_content_height: number
    app_content_width: number
    // ApplicationConfigの取得に失敗した。読み込み中オーバーレイを
    // スピナーからエラー表示＋再試行へ差し替えるために使う
    application_config_load_failed: boolean
    // 列の検索条件とスクロール位置の保存キーの枝番。単独ページと1枚目は空文字
    // （＝従来キーそのまま）。ポートで2枚目以降を開くときだけ '2' / '3' … が入る。
    // 分けないと2枚目が1枚目の列条件を上書きする
    column_state_instance_key: string
    // 画面をまたいだ変更通知の口。null が「単独ページとして動く」＝ publish も購読もしない。
    // ポート(rudbeckia)で開いたときだけ非 null が入る
    kyou_change_channel: KyouChangeChannel | null

    // ── 何ができるか（ラッパがリテラルで渡す） ──
    // 右クリックの編集メニュー。**詳細ダイアログ（ダブルクリック）はこれとは別で常に開ける**
    enable_context_menu: boolean
    // 列を増やす「＋」。出力したファイルでは増やせない（焼いた列だけ）
    enable_add_column: boolean
    // Enter でメモ帳 / Ctrl+V でクリップボード保存。window にキャプチャで張るので、
    // ポート(rudbeckia)で複数枚ホストするときは false（ポート自身のぶんだけ生かす）
    enable_keyboard_shortcuts: boolean
    // Mi のチェックボックスを触れなくするか
    is_readonly_mi_check: boolean
    // 列のフッタ（件数・リロード・画像切替・✕）
    show_column_footer: boolean
    // 記録がどの保管場所から来たか
    show_rep_name: boolean
    // 実行中の打刻を終了するボタン
    show_timeis_playing_end_button: boolean
    // 集計（Dnote）の列とトグル
    is_dnote_available: boolean
    // マップの列とトグル
    is_map_available: boolean
    // 検索条件サイドバーを開く導線。**サイドバー自体は常に mount する** ――
    // init() がその template ref を要るので、消すと列が1本も作られない（ADR-0415）
    is_sidebar_available: boolean
}
