# ADR-0610: Mi の射影が for_mi に依存することは、潰し込みを外さずに警告と説明で見せる

| | |
|---|---|
| Status | Accepted |
| Date | 2026-08-24 |
| Sources | 2026-08-24 の実利用レビュー3本（読み取り一巡・読み書き一巡・書き込み破壊試験）。本番 read サーバでの実測 |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/api/find_filter.go`（`isMiData` の gate）/ `src/server/gkill/api/gkill_server_api/get_kyous_mcp_helpers.go`（`miProjectionWarning` / `PROJECTION_TO_ENTITY_DATA_TYPE` 相当は `src/mcp/lib/constants.mjs`） |

## Context

実利用の AI が `data_types:["mi_create"]` で「今週作ったタスク」を数え、**0件・警告なし**を
「今週タスクを作っていない」と読んだ。実際には12件あった。

原因は「Mi rep が検索されない」ことではない。Mi / MiReKyou の rep は常に検索対象に入る。
潰しているのは `find_filter.go` の

```go
isMiData := strings.HasPrefix(currentKyou[0].DataType, "mi") && findCtx.ParsedFindQuery.ForMi
```

で、**`for_mi` が無いと Mi の5射影が代表1件へ潰される**。代表を選ぶ
`compareKyouEntryPriority`（`find_filter_helpers.go`）は `_start` 接尾辞を優先し、
次に `DataType` の辞書昇順なので `mi_check` < `mi_create`。しかも `MI.IS_CHECKED` は
`NOT NULL` で `mi_check` 射影の行は全 Mi に必ず存在する。
つまり **`mi_create` はめったに生き残らない**。ただし「絶対に出ない」ではない ——
`CREATE_TIME` が窓の中で `UPDATE_TIME` が窓の外にある Mi（作ったあと窓の外で更新したもの）は
`mi_create` が代表として残る。**この違いを「出ない」と書いたのは誤りだった**（後述の Evidence）。

逆に `for_mi:true` のときは `overrideKyous` が `mi_sort_type` に従って `DataType` を
**書き換える**（未指定なら `mi_create` 固定）。**見える `data_type` はレコードの属性ではなく
`for_mi` の関数**だった。TimeIs の gate には `ForMi` 条件が無く、Mi だけが非対称。

`data_types` は検索後の後段フィルタで、`mi_create` は既知の値なので警告の対象外だった。
「既知の値・警告ゼロ・件数が実際より激減」という一番たちの悪い形になっていた。

本番実測: 素の日付検索の `group_by:data_type` は `mi_check` 11 / `mi_start` 9 /
`mirekyou_check` 8 を返し **`mi_create` は0**。`for_mi`+`include_create_mi` を足すと
`mi_create` 12 / `mirekyou_create` 3。

同じレビューで、`gkill_add_mi` の応答 `data_type:"mi_create"` を
`gkill_delete_kyou` へそのまま渡すと enum 違反で落ちることも報告された。
`ENTITY_TARGETS` のキー（`mi` / `timeis`）と DTO の `data_type`（`mi_create` / `timeis_start`）は
**別語彙**で、その区別はどこにも書かれていなかった。KFTL の `created[]` だけは
エンティティ語彙なので通る、という三者三様だった。

## Decision

- **`&& ForMi` は外さない。** 代わりに `data_types` に Mi/MiReKyou の射影名があって
  `for_mi` が立っていないとき、`collectMCPUnknownValueWarnings` から
  「射影は代表1件へ潰れるので `mi_create` は実際の作成数よりずっと少なく出る。`for_mi` と `include_*_mi` を立てろ」と警告する（**「0件になる」とは言わない**）
- **`for_mi` を自動で立てない**（MCP 側でも Go 側でも）
- 説明文を実測へ揃える: `for_mi` は Mi **と MiReKyou** を返すこと、射影が `mi_sort_type` に
  従うのは `for_mi` を立てたときだけであること、素の検索では `mi_check` / `mi_start` が出ること
- **射影名を delete / restore / history が受理する。** `src/mcp/lib/constants.mjs` に
  射影名→エンティティ種別の対応を1箇所置き、3ツールの入口で通す。既存のエンティティ種別は
  そのまま有効で、受理値が増えるだけ

## Rejected alternatives

- **`find_filter.go` の `&& ForMi` を外して根治する** — `data_types:["mi_create"]` は自然に
  通るようになるが、**素の日付検索で1つの Mi が最大5件に増える**。件数・ページング・カーソルが
  全クライアント（rykv / mi 画面 / Web / Wear / 既存の全 MCP セッション）で変わる。
  この gate はまさにそれを防ぐために存在する
- **MCP 側（`normalizeKyouArgs`）で `for_mi` を自動で立てる** — `ForMi=true` は
  `selectMatchRepsFromQuery` で検索対象を Mi/MiReKyou へ**限定**するので、
  `data_types:["mi_create","kmemo"]` のような指定が黙って壊れる。
  しかも `include_*_mi` も同時に立てないと `filterMiForMi` で全滅する
- **Go 側（`handle_get_kyous_mcp`）で自動で立てる** — 同上。加えてカーソルの押し下げより前に
  クエリを書き換えることになり、ページ間で条件が変わる
- **射影名を受理せず、エラー文で「`mi` を渡せ」と案内するだけ** — 往復が1回増える。
  しかも現在の説明文が「`gkill_get_kyous` の結果から `data_type` を取れ」と案内しており、
  その結果が射影名である以上、**案内どおりにやると必ず1回落ちる**構造は残る
- **DTO の `data_type` をエンティティ種別に統一する** — 射影名は「どの時刻で並んでいるか」を
  表しており、`mi_limit` と `mi_start` が同じ `mi` になると Mi 画面の並び替えが表現できない

## Consequences

- 既存の検索結果・件数・ページングは1件も変わらない（警告と説明文だけが増える）
- `gkill_add_mi` → `gkill_delete_kyou` のように、応答をそのまま次のツールへ渡せるようになった
- 「2つの語彙がある」ことが3ツールの説明文に入った。対応表は1箇所（`constants.mjs`）
- `for_mi` を立てずに Mi を数えたい場合の答えは依然として無い。
  そこは `for_mi` を立てた別のクエリで数えるしかない（警告がそう案内する）

## Evidence

- 本番 read サーバ実測（2026-08-24、期間 2026-08-18〜24）:
  素の `group_by:data_type` は `mi_check` 11 / `mi_start` 9 / `mirekyou_check` 8、`mi_create` は0。
  `data_types:["mi_create"]` 単独は total_count 0・warnings なし。
  `for_mi`+`include_create_mi` で `mi_create` 12 / `mirekyou_create` 3
- **同じ検索を別の期間（2026-08-11〜17）でやると `mi_create` は1件出る**
  （`mirekyou_create` は5件）。`data_types:["mi_create"]` 単独でも 1件。
  最初の実測だけを見て説明文へ「matches nothing」「essentially never」と書いたのは誤りで、
  2026-08-25 の実利用レビューがこの記述の嘘を指摘した。**片方の期間の実測から
  「常にこうなる」と書かない**というのが、この ADR がいちばん高くついた教訓
- `MI` テーブルの `IS_CHECKED` は `NOT NULL`（`mi_repository_sqlite3_impl.go`）で、
  `mi_check` 射影の条件は `IS_CHECKED IS NOT NULL`。全 Mi に必ず1行ある

## Related tests

- `src/server/gkill/api/gkill_server_api/handle_get_kyous_mcp_v2_test.go`
  - `TestHandleGetKyousMCP_MiProjectionWithoutForMiWarns`
  - `TestHandleGetKyousMCP_MiProjectionWithForMiDoesNotWarn`（誤警告を出さない）
  - `TestHandleGetKyousMCP_NonMiDataTypeDoesNotWarnAboutForMi`
- `src/mcp/__tests__/write-normalization.test.mjs`（`projection data_type is accepted by delete / restore`）
- `src/mcp/__tests__/normalization.test.mjs`（`normalizeKyouHistoryArgs — projection data_type`）
