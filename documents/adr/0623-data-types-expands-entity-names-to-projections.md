# ADR-0623: data_types はエンティティ名を射影へ展開して受理する

| | |
|---|---|
| Status | Accepted |
| Date | 2026-09-18 |
| Sources | 2026-09-18 の MCP 実利用報告（`data_types:["timeis"]` が警告なしで0件、`["timeis","mi","idf"]` が idf 単体と同じ件数）。`.claude/skills/gkill-mcp/SKILL.md`「`data_types` のエンティティ名は射影へ展開して受理する。」 |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/api/gkill_server_api/get_kyous_mcp_helpers.go`（`mcpEntityDataTypeProjections` / `expandMCPDataTypes`）/ `src/server/gkill/api/gkill_server_api/handle_get_kyous_mcp.go`（`applyMCPDataTypesFilter` の呼び出し） |

## Context

`data_type` には語彙が2つある（ADR-0610 / 0611）。検索結果が返すのは射影名（`timeis_start` / `mi_check` …）、
delete / restore / history が受理するのはエンティティ名（`timeis` / `mi`）。ADR-0611 で「応答の射影名を
そのまま次のツールへ渡せる」向きの橋は架けたが、**逆向き**が空いていた。

`data_types` は検索結果の `DataType` との完全一致で、SQL が射影名を焼き込むので素の `timeis` / `mi` / `mirekyou`
に一致する行は存在しない。一方 `knownMCPDataTypes` はその3名を既知として通していたので、未知値警告
（ADR-0604）の対象外だった。結果は「既知の値・警告ゼロ・必ず0件」——ADR-0610 が潰した中で一番たちの悪い形が
素の名前について残っていた。しかも `gkill_get_kyou_history` の enum ではこの3名が正しい値なので、
そこからコピーするのが自然な動線になっている。

## Decision

`data_types` のエンティティ名は、その記録が検索結果に出るときの射影名の全部へ展開して受理する
（`timeis` → `timeis_start` + `timeis_end`、`mi` → 5射影、`mirekyou` → 5射影）。表は Go の
`mcpEntityDataTypeProjections` 1つで、`knownMCPDataTypes` と `miProjectionDataTypes` もそこから引く。
警告側（`collectMCPUnknownValueWarnings`）には展開前の値を渡す——素の `mi` は全射影を含むので、
潰し込み後も「タスク1件 = 1行」になり、`mi_create` だけを指定したときの警告（ADR-0610）は当てはまらない。

## Rejected alternatives

- **既知集合から素の3名を外して未知値警告にする** — 呼び出し側は結局展開を手で書く。「応答の語彙を次のツールへ
  そのまま渡せる」（ADR-0611）と逆向きの穴が残り、history の enum に載る値が検索では未知と言われる矛盾も出る。
- **Node（MCP サーバ）側で展開する** — API を直接叩く経路（Web 以外のクライアント、検証スクリプト）が救われない。
  表が Go と Node の2つになる（ADR-0611）。
- **素の名前に「射影名を使え」という警告だけ足す** — 0件のまま返すことに変わりがなく、AI は結局もう1往復する。
  展開すれば1回で正しい件数が返る。

## Consequences

- `data_types:["mi"]` は `for_mi` 無しでも「タスクの数」を返す（潰し込み後の代表1件ずつ）。`["mi_create"]` の
  ように射影名を明示したときの警告（ADR-0610）は今までどおり出る。
- 表を増やす（新しい射影を持つ型を足す）ときは `mcpEntityDataTypeProjections` だけを直す。
  `knownMCPDataTypes` / `miProjectionDataTypes` へ別に書くと、表が2つになって片方だけ古びる。

## Evidence

- 報告値（2026-09-11〜17 の実データ）: `["timeis"]` 0件・警告なし / `["timeis_start"]` 千数百件 /
  `["mi"]` 0件・警告なし / `["zzz_bogus"]` 0件・警告あり / `["timeis","mi","idf"]` 千数百件 = idf 単体と同じ
- 修正後の回帰テスト: `["timeis"]` = `["timeis_start","timeis_end"]` の件数、`["mi"]` = タスク数、
  `["timeis","mi","kmemo"]` = 3種の合計、`["zzz_bogus"]` の警告は維持

## Related tests

- `src/server/gkill/api/gkill_server_api/handle_get_kyous_mcp_v2_test.go`（`TestExpandMCPDataTypes` / `TestHandleGetKyousMCP_DataTypesAcceptsEntityNames` / `TestHandleGetKyousMCP_DataTypesFilterAndUnknownWarns`）
