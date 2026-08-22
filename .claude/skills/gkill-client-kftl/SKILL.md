---
name: gkill-client-kftl
description: "KFTL（メモ帳）の約束。タブ（kftl-tabs.ts / use-kftl-tabs.ts）、保存マーカーの beforeinput/input 対、複数ウィンドウの二重送信防止と送信タブの排他、メモ帳ダイアログの複数枚化（KFTLDialogHost・slot 採番）、KFTL パーサ（TS/Go の2実装）を扱う。src/client/classes/kftl/・kftl-tabs.ts・use-kftl-tabs.ts・use-kftl-view.ts・kftl-view.vue・mkfl-view.vue・kftl-dialog.vue・use-kftl-dialog-host.ts・src/server/gkill/api/kftl/ を編集するとき必読。「メモ帳が二重登録される」「別のタブへ保存された」「行ラベルが消える」「タブをクリックしただけで保存が走る」の調査でも必読。"
---

# KFTL（メモ帳）の不変条件

対象: `src/client/classes/kftl/**` / `kftl-tabs.ts` / `use-kftl-tabs.ts` / `use-kftl-view.ts` / `kftl-view.vue` / `mkfl-view.vue` / `kftl-dialog.vue` / `use-kftl-dialog-host.ts` / `src/server/gkill/api/kftl/**`

**このファイルは全文が、実際に起きた事故の再発防止である。該当作業では飛ばさずに読むこと。**
多くは「例外もエラーも出さずに静かに壊れる」種類で、破っても目の前ではエラーにならない。

- `gkill/api/kftl/` — KFTL custom text format parser (single package, no sub-packages). Supports both Japanese (。！？、ーー etc.) and ASCII (#!?,-- ~~ /mi /mood /expense /num /url /start /end /timeis /end? /endt /endt?) prefixes
- `classes/kftl/` — KFTL parser (50 statement types; the Go side has 47). Accepts the same Japanese/ASCII prefixes as the Go parser; ASCII constants and match/strip helpers centralized in `kftl-prefixes.ts`

**KFTL（メモ帳）のタブ**（2026-08-16）。`kftl-view.vue` がタブのホストで、`/kftl` ページ・各画面のメモ帳ダイアログ（`kftl-dialog.vue`）・打刻メモ帳（`mkfl-view.vue`）の**3系統すべて**に効く。純関数は `classes/kftl-tabs.ts`、状態は `classes/use-kftl-tabs.ts`。守るべき約束:
- **`v-window` を使わず、アクティブなタブ1枚だけを描画する。** 非表示の textarea は `clientWidth` が0になり、`kftl-statement-line.ts` の `1 + parseInt(text_width / 0)` が **`NaN`**（`Infinity` ではない）を返して行ラベルが丸ごと消える
- **タブの一覧と中身はモジュールシングルトン、いま映しているタブはビューごと。** メモ帳ダイアログは複数枚開けるので、インスタンスごとに配列を持つと片方の古い配列で localStorage を丸ごと上書きしてもう片方のタブが消える（単一文字列だった頃は「上書き」で済んでいた）。逆に `active_tab_id` までストアに置くと、全ウィンドウが同じタブを映して並べて見られない。ストアが持つのは `last_active_tab_id`（次に開くウィンドウの初期表示）だけ。単体テストは `reset_kftl_tabs_for_test()` + `localStorage.clear()` を `beforeEach` で呼ぶこと
- **タブが消えたらビューは追随する。** 別のウィンドウがタブを閉じたり保存で閉じたりすると、そのタブを映していたビューの `active_tab_id` が宙に浮く。`watch(() => tabs.map(t => t.id))` で新旧を比べ、旧インデックスへクランプして隣のタブへ移す（放置すると本文が空文字を返し続ける）
- **送信対象タブは `do_submit(target_tab_id, ...)` の引数で渡す。** `do_submit()` は未知タグ確認・未知板名確認でいったん抜けて応答を待つので、1回の保存で2〜3回呼ばれる。gkill のフローティングダイアログは非モーダル（`App.vue` の `.gkill-float-scrim` が `pointer-events: none`）なので、確認中でも背後のタブバーは押せる。持ち越し用の `submit_target_tab_id` を**引数ではなく直接読ませてはいけない** ―― 確認ダイアログを Escape / ブラウザバックで閉じられると古い値が残り、次の保存が別のタブへ誤配送される。新しい送信（`submit()`）は必ずアクティブなタブを渡す
- **タブ操作のロックは `is_submitting || show_confirm_unknown_tag_dialog`。** `is_requested_submit` は設定の読み込みが終わるまで `true` なので鍵に使えない（起動直後にタブを追加できなくなる）。板名確認をロック条件に入れてもいけない ―― `unknown_mi_boards` はブラウザバックで閉じても空にならないので**永久ロック**になる（タグ確認は `useDialogHistoryStack` がどの閉じ方でも false にするので安全）
- **不正行の判定は送信対象タブから引き直す。** 表示用の `invalid_line_numbers` はアクティブタブのもので、しかも await をまたいで遅れて着地する。これを送信の可否に使うと、タブを切り替えた直後の保存が「おかしな行があります」で止まる
- **保存マーカーの判定は `beforeinput` で控えた本文と `input` 時点の本文を比べ、「確定したマーカー行が増えたか」で行う（`count_save_marker_lines`）。watch（本文の変化）に置いてはいけない。** watch は `flush: 'post'` で中間の値を観測しないうえ、**同じ `input` イベントのリスナー間でマイクロタスクが走る**ので `@input` より先に新しい本文を見てしまう（IMEでは必ず起きる）。「末尾がマーカーか」で見るのも不可。理由の詳細と却下案は [ADR-0037](../../../documents/adr/0037-save-marker-beforeinput-input-pair.md)。「確定した」= その行の後ろに改行がある、なので `！` を打った時点では走らない。守るテストは `kftl-submit-emits.test.ts` の「KFTLの保存マーカー」節と `e2e/kftl-tabs.spec.ts` の「IMEで確定してから改行しても自動で保存される」（**IMEはCDPの `Input.imeSetComposition` でしか再現しない**。`pressSequentially` は打鍵ごとにイベントループが回るので中間の本文を必ず観測してしまい、常に緑になる）
- **保存マーカーによる自動送信の入口は「利用者が選んだ操作」の2つだけ ―― textarea の `@input` と、テンプレート貼り付け。** 判定関数（`maybe_submit_by_save_marker`）は1つのまま、入口だけ2つにすること。`text_area_content` はアクティブタブへの computed なので、判定を watch の内容変化そのものに戻すとタブ切替・localStorage からの復元でも発火し、末尾にマーカーが残ったタブを**クリックしただけで保存が走る**。watch は `@input` が立てた印（`user_input_tab_id`。立てるのは `onTextAreaInput()` **だけ**）が付いているときしか判定しない。**テンプレートはこの印に相乗りさせず `paste_template()` から直接呼ぶ** ―― watch は `new_value === old_value` で早期returnするので、貼る前のタブの本文がテンプレートと同一文字列だと黙って発火しない（タブ化する前も同じ理由で取りこぼしていた）。さらに watch は `flush: 'post'` かつ await を挟むので、判定までにタブを切り替えられると「印のタブ == アクティブタブ」が偽になってこれも黙って落ちる
- **同じタブを2枚のウィンドウが同時に保存しても、登録は1回。** `is_submitting` / `is_requested_submit` は**ビューごと**なので、同じタブを映した別ウィンドウの保存ボタンは止められない（KFTLはtxで束ねて送るので二重送信するとKyouが丸ごと重複する）。送信中のタブidは共有ストアが持ち、`do_submit` の冒頭で `tabs_store.try_begin_submit(target_tab_id)`、`finally` で `end_submit(target_tab_id)`。**claim は `is_requested_submit` ガードの「後」かつ `try` の「外」で取る** ―― 前に置くとガードの return が `finally` を通らず永久ロック、`try` の中で取ると掴めなかった側の `finally` が**勝ったウィンドウの分を解放**する。確認ダイアログで抜けるときは手放し、`confirm_submit` / `confirm_mi_board_submit` からの再入で取り直す（**持ち越すと自己デッドロックする**）。この印はリアクティブにせず localStorage にも出さない（`is_tab_locked` に混ぜると「送信中のタブから切り替えられない」、永続化するとリロードで掴んだままのタブが二度と保存できなくなる）。テンプレート経路は毎回一意な新しいタブを作り、それをアクティブにするのは貼ったウィンドウだけなので、この排他とは無関係に重複しない
- `update_line_labels()` / `refresh_invalid_lines()` は **await をまたぐので世代トークンで最後の1回だけ書き戻す**（前のタブぶんの結果が後から着地して行ラベルが化けるのを防ぐ）
- タブのストアは**独立した `effectScope(true)` の中で作る**。setup の中で素に `watch` を張ると最初に呼んだコンポーネントのスコープに属し、そのコンポーネントが unmount された時点で**永続化ごと止まる**（メモ帳ダイアログを閉じる／画面を移るで再現する）
- DOM は id 引きではなく**テンプレート ref** で掴む（`TextAreaInfo.text_area_element`）。textarea の `id` は複数枚で重複しないよう `useId()` で採番するので、**E2E はクラス `.kftl_text_area` で掴む**
- タブ列の `×` は `v-btn` ではなく `v-icon` にする（`v-tab` は `<button>` を描画するので `<button>` が入れ子になる）。`@click.stop` が無いと閉じると同時にそのタブがアクティブ化される
- **タブ列はタイトル行に同居させる**（別の行にするとテキストエリアの縦が40px削られる）。`resize()` が引くのは `title_height` だけで、`tab_bar_height` は `v-tabs` の `:height` に渡すためだけの定数。`title_height` の実寸は `.kftl_title` の CSS で固定する（`v-card-title` に `height` prop は無いので、渡すだけでは効かない）。測った値をフィードバックすると `kftl-dialog.vue` の ResizeObserver が縮小ループに入る
- 守るテスト: `kftl-tabs.test.ts` / `kftl-tab-store.test.ts` / `kftl-submit-emits.test.ts` の「KFTLのタブ」「KFTLを複数のウィンドウで開く」/ `e2e/kftl-tabs.spec.ts`

**メモ帳ダイアログの複数枚化**（2026-08-16）。5画面（rykv / mi / dashboard / saihate / plaing）は `<KFTLDialogHost>` を1個置くだけで、＋メニューを選ぶたびにウィンドウが増える。呼び出し側は従来どおり `kftl_dialog.value?.show()` のまま（`show()` の意味が「開く or 再フォーカス」から「1枚増やす」に変わった）。配列の持ち方は `rykv-dialog-host` と同じ。守るべき約束:
- **`useFloatingDialog` のキーはウィンドウごとに分ける。** キーは `${key}:pos` / `:size` / `:transparent` の保存先そのものなので、同じキーで複数枚出すと位置とサイズを奪い合う。スロット番号（空いている最小の番号）で `kftl-dialog` / `kftl-dialog-2` … と分け、`centerOffset` に `slot_index * 28` px を渡してずらす（全ダイアログが `centerMode: "always"` なので、ずらさないとピクセル単位で完全に重なる）
- **z-index は「開いているダイアログの並び順」から出す。単調増加のカウンタにしてはいけない** ―― Vuetify の overlay（メニュー / ツールチップ）が 2400 なので、上へ伸ばし続けるとダイアログの中のメニューが下へ潜る。伸びるのは同時に開いている枚数ぶんだけ
- **前面化は自分と子孫をまとめて上げる。** 確認ダイアログは `Teleport to="body"` で親の**兄弟**になるので、素朴に前面化すると親をクリックしただけで確認が後ろへ隠れる。親子は `provide`/`inject` で持つ（コンポーネント木は Teleport をまたいでも保たれる）
- **バックと Escape が閉じるのは「見た目の最前面」。** `use-dialog-history-stack.ts` の `stack` は積んだ順なので、前面化で見た目とずれる。ずれたままだと奥のダイアログが閉じる。前面化のたびに `raise_dialog_history_entries()` で履歴エントリも同じ順へ並べ替える。2つのコンポーザブルは**同じコンポーネントの setup で呼ばれる**ので、`getCurrentInstance()` を鍵にして結んでいる（`stack` の長さは変わらないので履歴の深さ計算は無傷）
- `closed` は `useDialogHistoryStack(is_show_dialog, { onClosed })` から出す。×・Escape・ブラウザバックのどれでも1回だけ上がるので、ホストはこれ1本で一覧から外せる
- ホストは `v-for` でルートが複数になる。呼び出し側が渡してくる Kyou 系の中継束のうちメモ帳が出さないイベントは行き場が無いので、`defineOptions({ inheritAttrs: false })` で黙って捨てる（従来と同じ挙動）
- E2E で `clickFabButton()` を使ってはいけない ―― 先に `dismissFloatingDialogs()` を呼ぶので、開いているメモ帳ウィンドウを閉じてしまい枚数が増えない
- 守るテスト: `floating-dialog-z-order.test.ts` / `kftl-dialog-host.test.ts` / `e2e/kftl-multi-dialog.spec.ts`

## 関連スキル

- [gkill-client-foundation](../gkill-client-foundation/SKILL.md) — 必ず併読（中継束・ダイアログ規約）
- [gkill-client-rudbeckia](../gkill-client-rudbeckia/SKILL.md) — メモ帳ウィンドウの slot 採番はアプリ全体で1つ
- [gkill-client-tags](../gkill-client-tags/SKILL.md) — 未知タグ確認の共有部品と確認の順序

## 詳しい設計と却下案（ADR）

- [ADR-0037 保存マーカーは beforeinput/input の対で](../../../documents/adr/0037-save-marker-beforeinput-input-pair.md)
