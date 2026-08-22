---
name: gkill-client-rudbeckia
description: "ポート（開発コード rudbeckia）とフローティングウィンドウの約束。ビューを v-layout で包む理由、scoped CSS に 100vh を書かない、ホストされたビューの FAB と Enter/Ctrl+V 登録の禁止、ポートの FAB の z-index、複数枚の保存キー instance_key と slot 採番・cascade 段差の区別を扱う。rudbeckia-page.vue・use-rudbeckia-page.ts・rykv-view.vue・mi-view.vue・dashboard-view.vue・plaing-time-is-view.vue を編集するとき、ビューをウィンドウとしてホストするとき必読。「バーが画面最上部へ飛ぶ」「ウィンドウが重なって1枚に見える」「2枚目が1枚目の条件を上書きする」「集計リストだけ消える」の調査でも必読。"
---

# ポート（rudbeckia）とフローティングウィンドウの不変条件

対象: `src/client/pages/rudbeckia-page.vue` / `use-rudbeckia-page.ts` / ホストされる `rykv-view.vue` / `mi-view.vue` / `dashboard-view.vue` / `plaing-time-is-view.vue`

**このファイルは全文が、実際に起きた事故の再発防止である。該当作業では飛ばさずに読むこと。**
多くは「例外もエラーも出さずに静かに壊れる」種類で、破っても目の前ではエラーにならない。

**ポート（開発コード rudbeckia）に画面をウィンドウとして載せる**（2026-08-17）。`/rudbeckia` は背景と FAB だけの1画面で、ライフログビュー / タスク / 実行中 / ダッシュボードをフローティングウィンドウとして開く。**開発コード `rudbeckia` は URL・ファイル名・識別子・保存キー・マニュアルのファイル名に使い、「ポート」は i18n の値とマニュアル本文にだけ出す。** 守るべき約束:
- **ホストするのはページではなくビュー**（`rykv-view` / `mi-view` / `plaing-time-is-view` / `dashboard-view`）。ページは ApplicationConfig の取得・テーマ・`useConfigStructSync`・メッセージ表示・`resize_content`・`reset_dialog_history` を持つので、ダイアログの中に N 個置くわけにいかない。ポートがその1つぶんを担う
- **`v-app-bar` / `v-navigation-drawer` / `v-main` を持つビューは `<v-layout :height :width>` で包む。** Vuetify は親レイアウトがあると `rootZIndex` を 100 下げ、レイアウト項目を `fixed` から `absolute` へ切り替える（`vuetify/lib/composables/layout.js:94,211,262`）。包まないと**画面最上部へ飛んでポート自身の上に重なる**。レイアウト部品を持たない実行中(plaing)は包まない
- **包んだらビューのルートを箱いっぱいに重ねる。** `absolute` になったバー類の基準は `<v-layout>` ではなく最も近い位置指定済み祖先＝`.rykv_view_wrap` 等。`.dashboard_view_wrap` / `.plaing_timeis_view_wrap` は `position: relative` を持たないので埋め込み用の非スコープ CSS で付ける
- **ビューの scoped CSS に `100vh` / `100vw` とメディアクエリを書かない。** ダイアログの中では基準が画面ではなく箱。`props.app_content_*` から `v-bind` する
- **`drawer_mode_is_mobile` は `props.app_content_width` の computed。** 初期化時の1回代入だとダイアログをリサイズしてもドロワーの一時表示モードが切り替わらない。rykv と mi は対称なので必ず両方へ
- **ホストしたビューは自前の FAB を出さない**（`is_hosted_in_dialog`）。`.position-fixed` は `position: fixed` なのでダイアログを抜けて画面右下へ居座り、ポートの FAB と重なる。ついでに打刻メモ帳ダイアログの二重 FAB もこれで消えた
- **ポートの FAB は `.position-fixed-rudbeckia`（`z-index: 2000`）。** 素の `.position-fixed` だとウィンドウ（`z-index: 1100 + 並び順`）に覆われ、**唯一の操作導線が押せなくなる**。Vuetify の overlay(2400) は追い越さないこと（追い越すと FAB 自身のメニューが下へ潜る）
- **ホストしたビューで Enter / Ctrl+V を登録しない。** `useScopedEnterForKFTL` / `useScopedCtrlVForClipboard` は `window` にキャプチャで張るので、4画面ぶん登録すると1回の Enter でメモ帳が4枚開く
- **ホストしたビューの画面切替メニューは `router.replace` しない。** `navigate_to_page` は `reset_dialog_history()`（モジュール共有）を呼ぶので、ポートで開いている他のウィンドウまで一斉に閉じる。`requested_navigate_page` を上げてホストに決めさせる
- **位置とサイズの保存キー（`slot_index`）と、中央からずらす段数（`cascade_index`）は別物。** 前者は種類ごと、後者は種類をまたいだ採番。ずらす量を `slot_index` で決めると4種類とも 0 になり、**4枚が完全に重なって1枚にしか見えない**
- **ウィンドウの未リサイズ時サイズは非スコープ CSS で確定させる**（Teleport 先には `data-v-` が付かない）。確定させたぶん ResizeObserver の実測をそのまま子へ渡す。`kftl-dialog.vue:72-82` の「`userSize` が無いときは既定値」ガードと**併用しない**（併用すると固定した高さが無視される）
- **ウィンドウの中の `v-card` を一括リセットする規則は「自前でレイアウトを組むカード」を壊す。** ポートは `.rudbeckia-page-dialog .gkill-floating-dialog__body .v-card { display: block; … }` で App.vue の「中身はカード1枚」前提の規則を打ち消しているが、これは詳細度3で **Dnote のルート（`v-card.dnote_view`）にも当たる**。Dnote は `fill_height` のとき自分が flex column の器になって残り高さを集計リストへ配る作り（`.dnote_list_view` = `height:0` + `flex-grow:1`）なので、`display:block` へ戻されると `.dnote-scroll-wrap` までの `height:100%` の鎖が auto に落ち、`flex-basis:0` の `.dnote_list_table_root` が**高さちょうど0**になって**集計リストだけが消える**（集計項目とグラフは自然高さのまま残るので気付きにくい）。例外は `.v-card.fill_height_mode` の1つだけ足してある。守るテストは `e2e/rudbeckia.spec.ts` の「集計ビューの集計リスト区画が高さ0に潰れない」

**複数枚（列状態の分離）**
- **列の検索条件とスクロール位置の保存キーはインスタンスごとに分ける。** `gkill-api.ts` の `set/get_saved_{rykv,mi}_{find_kyou_querys,scroll_indexs}` は `instance_key` を**必須引数**で取る（省略可能にすると渡し忘れが黙って通り、2枚目が1枚目を上書きする）。空文字＝従来キーで、単独ページと1枚目が今までの保存内容を引き継ぐ
- **枝番に uuid を使わない。** `${base}_${N+1}` にすること。uuid だと復元時に同じキーを引けず、列が毎回まっさらに戻ったうえで localStorage のキーが増え続ける
- **slot N を slot 0 の保存値から種付けしない。** `query_id` が重複し、`query_id → 列` の逆引きが別インスタンスへ誤配送する。2枚目は既定クエリ1本で開くのが正しい
- **メモ帳ウィンドウの slot 採番はホスト単位ではなくアプリ全体で1つ**（`use-kftl-dialog-host.ts` のモジュール共有 Set）。ポートではポート自身とホストした各画面が同時に `KFTLDialogHost` を持つので、ホスト内だけの採番では必ず `kftl-dialog` の保存キーが衝突する。ホストの unmount で掴んだ番号を返すこと

## 関連スキル

- [gkill-client-foundation](../gkill-client-foundation/SKILL.md) — 必ず併読（フローティングダイアログ・自動フォーカス）
- [gkill-client-columns](../gkill-client-columns/SKILL.md) — 画面間の変更伝播（`kyou-change-bus.ts`）と列状態
- [gkill-client-kftl](../gkill-client-kftl/SKILL.md) — メモ帳ダイアログの複数枚化

## 詳しい設計と却下案（ADR）

- [ADR-0034 列の同一性は query_id（関連）](../../../documents/adr/0034-column-identity-query-id.md)
- [ADR-0038 Props/Emit のみ（関連）](../../../documents/adr/0038-props-emit-only-no-pinia.md)
