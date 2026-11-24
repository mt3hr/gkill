'use strict'

import type { KyouChangeChannel } from "@/classes/kyou-change-bus"
import type { GkillPropsBase } from "./gkill-props-base"

/**
 * mi（タスク画面）の「列」の中身。板の列・検索条件サイドバー・詳細ペイン・件数カレンダーを
 * まとめて持つ。
 *
 * **これはモードを知らない。** 普段使う mi（`mi-view.vue`）と HTML 出力
 * （`exported-mi-view.vue`）の2つのラッパがこれをホストし、「何ができるか」を
 * 下の props へ**リテラルで**渡す。
 * 共有ページはこれも使わない（列の機構を一切使わないので `shared-mi-view.vue` が別実装）。
 *
 * ラッパ側が持つのはアプリバーの中身・編集系のダイアログ群・FAB・ヘルプで、
 * それらはスロット（`app_bar_title` / `app_bar_actions` / `dialogs` / `fab`）で差し込む。
 * ダイアログ群は保存した記録を列へ差し込む必要があるので、`dialogs` スロットは
 * 中継束と列の状態をスロット props で渡す。
 *
 * rykv の対になる実装は `rykv-columns-view-props.ts`（コピー由来の対称実装。
 * **修正は必ず両方へ**）。「何ができるか」の並びが rykv と同じでないのは、
 * **mi で2つの顔が同じ値になるものは props にしていない**ため
 * （常に同じ値の見せかけのフラグを作らない。ADR-0418）。
 * rykv にある `is_dnote_available` / `is_map_available` / `enable_add_column` も
 * mi には集計・マップ・列の「＋」が無いので作っていない。
 */
export interface MiColumnsViewProps extends GkillPropsBase {
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
    // 板ツリーのクリックで、まだ開いていない板の列を増やせるか。
    // 出力したファイルでは増やせない（焼いた板だけ）―― 増やせると、焼いていない
    // query_id の列ができて get_kyous が空を返し、**空の板が開く**（エラーは出ない）
    enable_open_board_column: boolean
    // 板をまたいだタスクの移動（ドラッグ&ドロップ）。書き込みなので出力では落とす
    enable_board_drag_drop: boolean
    // Enter でメモ帳 / Ctrl+V でクリップボード保存。window にキャプチャで張るので、
    // ポート(rudbeckia)で複数枚ホストするときは false（ポート自身のぶんだけ生かす）
    enable_keyboard_shortcuts: boolean
    // Mi のチェックボックスを触れなくするか
    is_readonly_mi_check: boolean
    // 列のフッタ（件数・リロード・画像切替・✕）
    show_column_footer: boolean
    // 実行中の打刻を終了するボタン（詳細ペイン）
    show_timeis_playing_end_button: boolean
    // 検索条件サイドバーを開く導線。**サイドバー自体は常に mount する** ――
    // init() がその template ref を要るので、消すと列が1本も作られない（ADR-0415）
    is_sidebar_available: boolean
}
