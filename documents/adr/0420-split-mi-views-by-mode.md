# ADR-0420: mi もモードごとにビューを分け、列の機構を共有の子へ置く

| | |
|---|---|
| Status | Accepted |
| Date | 2026-11-24 |
| Sources | [ADR-0418](0418-split-rykv-views-by-mode.md)（rykv を顔ごとに分けた決定） / `.claude/skills/gkill-client-columns/SKILL.md`「rykv/mi の「列×検索」不変条件」節 / `src/client/pages/views/mi-columns-view-props.ts` の doc コメント / [ADR-0415](0415-exported-view-is-a-separate-flag.md)（出力モードのフラグ） / [ADR-0407](0407-init-on-application-config-loaded.md)（列の初期化トリガ） |
| Supersedes | [ADR-0415](0415-exported-view-is-a-separate-flag.md) のビュー分岐部分 |
| Superseded-by | なし |
| Anchors | `src/client/pages/views/mi-columns-view-props.ts`、`src/client/classes/use-mi-columns.ts` の doc コメント |

## Context

ADR-0418 で rykv を顔ごとに3本のビューへ分けた。そのとき mi は
「共有も出力も無いので分割していない」と書いて据え置いた（0418 の Consequences 第1項）。

その前提が崩れた。**利用者の判断で mi からも HTML 出力できるようにする**ことになり、
出力のラッパが通常の mi と同じ列の子をホストする必要が出た。

据え置いたままで出力を足すと、ADR-0418 が却下した形のどちらかになる。

- **mi のビューにモードのフラグを戻す** — 利用者の「分岐が多すぎて保守しづらい」という
  要望がそのまま再発する。rykv で消したものを mi で復活させることになる
- **出力用に mi の列の機構を複製する** — 列の機構（検索の世代番号・`query_id` の採番・
  localStorage の復元・サイドバー連携）が3箇所になる。この機構は
  「例外もエラーも出さずに静かに壊れる」不変条件の塊で、rykv と mi の2箇所でも
  片方だけ直す事故が何度も起きている

数えると、分ける前の mi は `mi-view.vue` 398行 ＋ `use-mi-view.ts` 1343行で、
列の機構とアプリバーの中身・編集系ダイアログ群10本・FAB・ヘルプが同居していた。
共有版だけは以前から別ビュー（`shared-mi-view.vue` + `use-shared-mi-view.ts`）で、
しかも ADR-0418 が「リテラル作法の先例」として挙げたとおりモードのフラグを持っていなかった。

## Decision

**ADR-0418 と同じ形を mi へ適用する。顔ごとにビューを分け、共有は独立、
通常と出力は列の子を共有する。**

| ファイル | 役割 | モードの分岐 |
|---|---|---|
| `mi-view.vue` + `use-mi-view.ts` | 普段使う mi。アプリバーの中身・編集系ダイアログ群・FAB・ヘルプ | 無し |
| `exported-mi-view.vue` | HTML 出力。題名だけ出し、編集系は1つも置かない | 無し |
| `shared-mi-view.vue` + `use-shared-mi-view.ts` | 共有ページ。1列固定の独立実装（列の機構を使わない） | 無し |
| `mi-columns-view.vue` + `use-mi-columns.ts` | 板の列・検索・列状態の復元・サイドバー連携・詳細ペイン・件数カレンダー・板の D&D | 無し |

**「何ができるか」の props は rykv の写しにしない。** rykv の列は10件受け取るが、
mi は8件で、集合が違う。**mi の2つの顔で同じ値になるものは props にしない**
（ADR-0418 の「常に `true` だった見せかけのフラグを作らない」の続き）。

| mi にある | rykv にある | 理由 |
|---|---|---|
| `enable_open_board_column` | `enable_add_column` | mi は列の「＋」ではなく板ツリーのクリックで列が増える |
| `enable_board_drag_drop` | （無し） | 板をまたいだタスクの移動は mi だけ。**書き込みなので出力では落とす** |
| （無し） | `is_dnote_available` / `is_map_available` / `show_rep_name` | mi に集計・マップが無く、`show_rep_name` は mi では2つの顔で同じ値（列 `false` / 詳細ペイン `true`） |

共通は `enable_context_menu` / `enable_keyboard_shortcuts` / `is_readonly_mi_check` /
`show_column_footer` / `show_timeis_playing_end_button` / `is_sidebar_available` の6件。

**`dialogs` スロットのスロット props は rykv より1つ多い（5件）。**
rykv は `crud_relay` / `all_columns_request_relay` / `querys` / `focused_column_index` の4件だが、
mi は `sub_view_focus_relay` も渡す —— **mi の打刻追加ダイアログだけが、
ダイアログの中でフォーカスが動いたことを上げてくる**（rykv の同じダイアログは上げない。
分割前の rykv も渡していなかった）。rykv の4件をそのまま写すと、
「打刻追加ダイアログの中でフォーカスが動いたとき列が追随する」が
**型もテストも通ったまま消える**。

**板の列を増やせるかを props にする**（`enable_open_board_column`）。
`open_or_focus_board()` は「その板の列が既に開いていればフォーカスを移し、
無ければ列を足して検索する」ので、出力では後半を止める。止めないと
焼いていない `query_id` の列ができ、`get_kyous` が空を返して**空の板が開く**（エラーは出ない）。

**板の D&D の門は2枚にする。** テンプレートの `:draggable` / `@drop` / `@dragover` と、
`onDropBoardTask()` / `move_board_task()` の先頭の早期 return。属性を1つ落とすだけで
書き込みが通ってしまうため。

あわせて**共有 mi に残っていた死にコードを掃除した**。`is_show_kyou_detail_view` /
`is_show_kyou_count_calendar` は `ref(true)` で書き込みもトグルも無く、
テンプレートの5箇所が常に真の条件になっていた。未使用の `share_id` props
（`shared-rykv-view-props.ts` には無い非対称）と、それを渡すためだけにあった
`use-shared-mi-page.ts` の `share_kyou_id` も外した。

## Rejected alternatives

- **mi は据え置き、出力のときだけビューの中でフラグを見る** — ADR-0418 が却下した形そのもの。rykv で26箇所の分岐を消した直後に mi で同じものを作ることになり、利用者の要望（分岐を減らす）に反する。
- **出力用に mi の列の機構を複製する** — 列の機構が3箇所になる。`query_id` の採番・`search_seqs` の世代番号・`querys_backup` の同期・`run_with_sidebar_search_suppressed` の順序はどれも破っても目の前ではエラーにならない種類で、片方だけ直す事故が確実に起きる。
- **rykv の列の子を mi でも使う（1つに統合する）** — 板の見出し・`MI_BOARD_TITLE_HEIGHT` を引いた列の高さ・板の D&D・`MiKyouCountCalendar`（ソート基準つき）・`miBoardQuery` の板ツリー連携は mi だけのもので、逆に Dnote・マップ・Ryuu・列の「＋」は rykv だけのもの。統合すると「どちらの画面で出るのか」を読み解く対象が子の中に戻る（＝分割前と同じ状態が移るだけ）。対称実装として2本並べ、**修正は必ず両方へ**という既存の約束を続ける。
- **rykv の「何ができるか」props 10件をそのまま写す** — `is_dnote_available` / `is_map_available` は mi では常に `false`、`show_rep_name` は常に同じ値になる。読む人が「どこで切り替わるのか」を探す対象を増やすだけ。
- **rykv の `build_rykv_columns_forward_relay` を mi でも使う** — 束を1つに寄せられるが、第1引数の型が緩い（`… | unknown`）ので、将来 rykv 側にイベントを1件足したとき mi が宣言していないイベントを emit しても型で落ちず、ネイティブリスナへ落ちて無言になる。この4段の型機構の存在理由が「1件落ちても型で落ちないのを防ぐ」ことなので、`build_mi_columns_forward_relay` を別に持つ。
- **`update_check_kyous` の `throw new Error('Not implemented')` をこの機会に実装する** — 分割は挙動を変えない作業なので混ぜない。rykv 側も同じ throw で対称。

## Consequences

- **列の機構の正本は `use-rykv-columns.ts` と `use-mi-columns.ts` の対になった。**
  ラッパの対は `use-rykv-view.ts` と `use-mi-view.ts`。**対称実装の修正は必ず両方へ**という
  約束は変わらないが、見比べる相手が2対2になった。ADR-0401 / ADR-0402 / ADR-0404 /
  ADR-0405 / ADR-0406 / ADR-0407 が `use-mi-view.ts` を Anchors / Sources に挙げているのは、
  現在は `use-mi-columns.ts` を指す。
- **ADR-0418 の Consequences 第1項（「mi は共有も出力も無いので分割していない」）は
  もう当てはまらない。** 0418 の decision 自体（rykv を顔ごとに分ける）は有効なので
  Supersede はしていない。
- **E2E が見るセレクタは変えていない**（`.mi_view_wrap` / `.mi_view_table` /
  `.mi_board_column_title` / `.mi_query_editor_sidebar` / `[data-gkill-view-ready]` /
  `.kyou_detail_view`）。準備完了信号を出す要素とルート要素は列の子へ移ったので、
  ソース走査テストの対象ファイル一覧を合わせた
  （`column-view-init-source-scan.test.ts` / `kyou-detail-pane-attached-source-scan.test.ts` /
  `mi-board-column-layout.test.ts` / `tx-bundle-source-scan.test.ts`）。
- **`<style scoped>` は全部列の子へ移した。** scoped CSS は描いたコンポーネントの
  `data-v-*` にしか付かないので、1つでもラッパに残すと無言で効かなくなる
  （板の見出しの 44px が消えると列の高さの引き算とずれる）。
- **FAB の門とキーボードの門が2ファイルに分かれた。** FAB は `mi-view.vue` の
  `v-if="!is_hosted_in_dialog"`、Enter / Ctrl+V は列の `enable_keyboard_shortcuts`。
  ラッパが `:enable_keyboard_shortcuts="!is_hosted_in_dialog"` を書き忘れても型では落ちず、
  ポートで4枚開いて Enter を1回押すとメモ帳が4枚開く形に戻る。
- **`MiViewProps` / `MiViewEmits` は変えていない。** `mi-page.vue` と
  `rudbeckia-page-dialog.vue`（`RykvView` / `DashboardView` と同じ束を共用している）は無改修。
- **`exported-mi-view.vue` はまだどのページからもマウントされていない。** 出力ページが
  焼き込みの印で選ぶ配線は別の決定（HTML 出力の mi 対応）で入れる。
- サイドバーの入れ物は**出力でも mount したままにする**という ADR-0415 の帰結はそのまま。
  mi には今まで無かった `v-if="is_sidebar_available"` を**アプリバーのナビアイコンに**足した
  （`v-navigation-drawer` に付けると `init()` がサイドバーの template ref を取れず、
  列が1本も作られないまま読み込み中オーバーレイが消えない）。

## Evidence

実測なし。分割前の行数（`mi-view.vue` 398行・`use-mi-view.ts` 1343行）と
「何ができるか」props の件数（rykv 10件・mi 8件）はソースを数えたもの（コード由来）。
分割の前後で、lint・型検査・フロントエンドのユニットテスト・資料検査・E2E 全件が通ることを確かめた。

## Related tests

- `src/client/__tests__/unit/classes/column-view-init-source-scan.test.ts` — 列の初期化の不変条件。対象は `use-rykv-columns.ts` と `use-mi-columns.ts`（対称実装）、準備完了信号は `rykv-columns-view.vue` / `mi-columns-view.vue`
- `src/client/__tests__/unit/composables/mi-view-initial-load.test.ts` / `mi-view-search-routing.test.ts` — 列の機構（`useMiColumns` を直接呼ぶ。rykv 側の対は `rykv-view-initial-load.test.ts` / `rykv-view-search-routing.test.ts`）
- `src/client/__tests__/unit/composables/new-tag-column-search.test.ts` / `registered-kyou-local-insert.test.ts` — rykv・mi 両方をパラメタライズする表（`useRykvColumns` / `useMiColumns`）
- `src/client/__tests__/unit/classes/kyou-detail-pane-attached-source-scan.test.ts` — 詳細ペインの付随データ表示（対象ファイルの完全一致リスト）
- `src/client/__tests__/unit/classes/mi-board-column-layout.test.ts` — 板の見出しの高さと `MI_BOARD_TITLE_HEIGHT` の一致
- `src/client/__tests__/unit/classes/tx-bundle-source-scan.test.ts` — tx を通さず Kyou を書いてよいファイルの許可リスト（板の D&D の `update_mi` / `update_mirekyou`）
- `src/client/__tests__/unit/composables/shared-mi-view-dialog.test.ts` — 共有の独立実装
- `src/client/__tests__/e2e/mi-board.spec.ts` / `mi-board-columns.spec.ts` / `column-view-initial-load.spec.ts` / `rudbeckia.spec.ts` — セレクタと準備完了信号の通し
