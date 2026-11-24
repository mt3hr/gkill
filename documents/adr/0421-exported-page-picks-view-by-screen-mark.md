# ADR-0421: 出力 HTML は焼き込みの印でマウント先を選び、mi のサイドバーは絞れる節だけ残す

| | |
|---|---|
| Status | Accepted |
| Date | 2026-11-24 |
| Sources | [ADR-0420](0420-split-mi-views-by-mode.md)（mi を顔ごとに分けた決定） / [ADR-0415](0415-exported-view-is-a-separate-flag.md)（出力モードと通信の1点止め） / [ADR-0417](0417-exported-sidebar-narrows-only.md)（出力のサイドバーは絞り込み専用） / `src/client/classes/exported-kyou-payload.ts` の `ExportedScreen` / `src/client/pages/shared-page.vue` の `view_type` の出し分け |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/client/classes/exported-kyou-payload.ts` の `screen` の doc コメント、`src/server/gkill/api/req_res/export_kyou_html_request.go` の `Screen` |

## Context

利用者の判断で、タスク画面（mi）からも「HTML で出力」できるようにする。
ADR-0420 で mi を顔ごとのビューへ分け、出力のラッパ（`exported-mi-view.vue`）も用意した。
残っているのは「出力したファイルを開いたとき、どちらのビューとしてマウントするか」。

収集はどちらも列の検索条件で動くので共通で、ガワ（`embed/export/template.html`）も1つ。
違うのは**開くときのビュー**と、**サイドバーで何が絞れるか**だけ。

出力先のサイドバーは「焼いた記録を絞ることしかできない」（ADR-0417）。
判定は `kyou-local-insert.ts` の `does_kyou_match_query` をそのまま使うので、
Kyou 単体では判定できない条件（ワード・TimeIs・地図・playing）は見ない。
rykv のサイドバーはその節を `v-show` で隠している。

mi のサイドバーには rykv に無い節が2つある ——「並び順」（`mi_sort_type`）と
「板」（板ツリー）。並び順は**サーバ側の処理**（`apply_mi_projection`）で、
出力先には無い。板ツリーのクリックは板の列を開く操作で、焼いていない板を開ける。

## Decision

**焼き込みに画面種別の印を1つ足し、出力ページがそれを見てマウント先を選ぶ。**

- 焼き込みの `screen`（`rykv` / `mi`）。要求（`ExportKyouHTMLRequest.Screen`）に載り、
  サーバがそのまま焼き込みへ写す
- **選ぶのは `exported-kyou-page.vue` の1箇所だけ。** ビューの中にモードのフラグを戻さない
  （ADR-0418 / ADR-0420）。先例は共有ページ（`shared-page.vue` の `view_type` の出し分け）
- **知らない値と欠損は `rykv` へ寄せる。** だから印を足す前に出力したファイルもそのまま読める。
  **`schema_version` は上げない**（世代を上げると、上げる前の出力が読めなくなるほうの事故になる）
- ただし**サーバは知らない `screen` を 400 で弾く**（`InvalidExportKyouHTMLOptionError`）。
  出力先が未知の印を rykv へ寄せるので、通すと「mi を出したのに rykv として開く」が静かに起きる

**列の題名は焼き込まない。** 焼いた列の検索条件に `mi_board_name` が入っているので、
板名はそこから出す（null なら `MI_ALL_TITLE`）。足すと板名が条件と題名の2箇所に載り、
片方だけ直すとずれる。

**出力した mi のサイドバーは、絞れる節だけ残す。**

| 節 | 出力先 | 理由 |
|---|---|---|
| ワード | 隠す | Kyou 単体では判定できない。TS でワード照合の4実装目を書かない（ADR-0417） |
| 並び順 | 隠す | 並べ替えはサーバ側の処理で出力先に無い。出したままだと触っても何も起きない |
| 保存済み検索条件の FAB | 隠す | 出力先で開いても意味が無い（rykv と同じ） |
| 完了状態 / 板 / タグ / 期間 | 残す | `does_kyou_match_query` が判定できる |

**隠すのは `v-show`。** `generate_query()` が各節の template ref から値を集めるので、
`v-if` で落とすと null 参照になる。

**板ツリーのクリックは、焼いた板へフォーカスを移すことしかできない**（`enable_open_board_column=false`）。
開いていない板を開けると、焼いていない `query_id` の列ができて `get_kyous` が空を返し、
**空の板が開く**（エラーも警告も出ない）。

出力用 API（`GkillAPIForExportedKyou`）に2本足す。どちらも共有版のままでは使えない ——
`get_saved_mi_find_kyou_querys` は既定クエリ1本を返して板構成が復元されず、
`get_mi_board_list` は throw する（rykv では使わないので共有版に実装が無い）。

**板ツリーは焼き込む前に刈る**（`pruneExportApplicationConfigStructs`）。
刈らないと、出力に含めなかった板名が渡した相手に読まれる。板名は利用者が付けた名前で、
それ自身が知られたくない情報になりうる。許可集合は列の `MiBoardName` から作り、
`null`（「すべて」）の列が1本でもあれば絞らない。

## Rejected alternatives

- **焼き込みに列の題名も足す** — 板名が「焼いた列の条件」と「焼いた題名」の2箇所に載る。片方だけ直したときに題名と中身がずれ、しかもずれても例外は出ない。条件から出せるものを二重に持たない。
- **画面ごとに別の出力ページを作る** — ガワのビルド（`vite.export.config.ts`）がエントリ2本になり、`codeSplitting: false` で1チャンクへ畳む前提が崩れる。ガワは記録0件でも常に乗る下駄なので、2本に増やすと出力の大きさが倍になる。
- **ビューの中で `application_config.for_export_kyou` を見て出し分ける** — ADR-0418 / ADR-0420 の逆行。あのフラグは「深い子へ props が届かない」サイドバーの各節のための別経路で、列の機構には持ち込まない。
- **`schema_version` を上げる** — 印の欠損を既定へ寄せられるので、上げる必要が無い。上げると「上げる前の出力が読めない」ほうの事故になる（出力は利用者の手元と相手の手元に残り続ける）。
- **未知の `screen` をサーバで通し、出力先で rykv へ寄せる** — 「mi を出したのに rykv として開く」が静かに起きる。クライアントの寄せは**古い出力を読むため**のもので、新しい要求の誤りを飲み込むためではない。
- **出力先でも並べ替えられるようにする** — 並べ替えを出力用 API へ注入すれば（絞り込みの `narrow_column_kyous` と同じ作法で）できるが、`apply_mi_projection` の並びの2実装目になる。利用者の判断で「節を隠す」に決めた。
- **`collectKyouBundle` の `ViewType` を mi のとき `"mi"` にする** — あの値は `collectTypedDatas` を丸ごと飛ばし、そこには `MiReKyous` の収集も入っている。mi の板には「すでにある記録をタスクにしたもの」が並ぶので、**その行だけが中身なしで出る**（エラーは出ない）。型別データは列の `IDs` で絞って引くので、`"kyou"` のままでも mi の板に無い型が増えることはない。
- **mi 用の「対象」の文言を rykv と使い回す** — mi の画面では列が板なので「表示中のすべての列」と出る。利用者の判断で2キー足した。

## Consequences

- **出力ダイアログは画面を props で受け取る**（`screen` / `can_choose_dnote` / `can_choose_map`）。
  ホストするラッパがリテラルで渡す。選べない区画は**要求の時点で false 固定**にする ——
  欄を `disabled` にするだけだと、選択済みの値が送信まで残って印と中身が食い違う。
- i18n に「表示中のすべての板」「フォーカスされている板」の2キーが増えた（7言語）。
  `src/server/gkill/api/embed/i18n/locales/` は生成物なので
  `npm run copy_i18n_to_app_embed`（`prepare_install` の一部）で追随する。
  **忘れると `MustLocalizeMessage` が panic する。**
- **mi のサイドバーの共有フッタから「HTML エクスポート」が実際に動くようになった。**
  これまでは `mi-query-editor-sidebar.vue` がそのイベントを素通ししておらず、
  ボタンは出ているのに押すとダイアログが閉じて何も起きなかった（バグ修正でもある）。
- 板ツリーは出力先では `append_not_found_mi_boards()` と `append_all_mi_board()` で
  焼いたデータから組み直す。利用者の設定した木をそのまま焼かないのは、
  含めなかった板名が残るため（タグ・rep と同じ理由。ADR-0417）。
- 出力した mi では、記録の追加・編集・完了のチェック・板の移動ができない。
  チェックは `is_readonly_mi_check`、板の移動は `enable_board_drag_drop` が止める（ADR-0420）。

## Evidence

実測なし —— 「並び順が出力先に無い」ことと「`ViewType: "mi"` が `MiReKyous` の収集も飛ばす」ことは
`kyou_bundle_collector.go` を読んで分かったもの（コード由来）。
出力した HTML を `file://` で開いて、板ごとに読めること・通信が1本も出ないことを確かめた
（`mi-export-html.spec.ts` が同じことを機械で見る）。

## Related tests

- `src/server/gkill/api/gkill_server_api/handle_export_kyou_html_test.go` — 画面の印が焼き込みに乗ること（指定なしは rykv）、未知の画面を弾くこと。**ガワが embed に無いビルドでは skip する**（CI の Go ジョブはフロントをビルドしない）
- `src/server/gkill/api/gkill_server_api/export_config_struct_test.go` — 板ツリーの刈り込み（出力に含めなかった板名が残らないこと・「すべて」の番兵は常に残ること・`null` の列が1本でもあれば刈らないこと）
- `src/client/__tests__/unit/classes/exported-kyou-payload.test.ts` — 印の読み取り（`mi` だけを読み、欠損と未知の値は `rykv` へ寄せる）
- `src/client/__tests__/unit/api/exported-kyou-api.test.ts` — `get_saved_mi_find_kyou_querys` と `get_mi_board_list`（焼いた Mi / MiReKyou の板名を返す。共有版は throw する）
- `src/client/__tests__/unit/composables/exported-kyou-page.test.ts` — 印で組む木を選ぶこと（mi は板ツリーとタグ、印が無ければ rykv で板ツリーを組まない）
- `src/client/__tests__/unit/composables/export-kyou-html-view.test.ts` — 要求に `screen` が乗ること、選べない区画が false 固定で送られること、「対象」の文言が画面で変わること
- `src/client/__tests__/e2e/mi-export-html.spec.ts` — 出力した mi を `file://` で開いて板として読める通し
