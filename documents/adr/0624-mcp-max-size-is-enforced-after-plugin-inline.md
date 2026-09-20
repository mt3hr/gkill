# ADR-0624: max_size_mb はプラグイン本文を埋め込んだ後に Node が守り直す

| | |
|---|---|
| Status | Accepted |
| Date | 2026-09-19 |
| Sources | 2026-09-18 の MCP 実利用報告（`max_size_mb:0.002`・`limit:2`・`include_plugin_content:true` で Kyou 配列だけで 3,596 バイトが警告なしに2件返る）。`.claude/skills/gkill-mcp/SKILL.md`「`max_size_mb` はプラグイン本文を埋め込んだ後に Node が守り直す」 |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/mcp/read_handlers.go`（`enforceKyousSizeBudget` / `recountInlinePluginContent`） |

## Context

`max_size_mb` は Go（`handle_get_kyous_mcp.go`）が DTO の JSON を1件ずつ測って厳密に守る（ADR-0604）。
プラグイン Kyou の DTO は本文を持たず `kind` / `plugin_name` だけなので小さく、Go の予算では何件でも入る。
その後 Node の `inlinePluginContents`（ADR-0602）が最大 4000 字 × 20 件の本文を破壊的に足すので、
呼び出し側が受け取る `kyous[]` は上限を超え、しかも警告が無い。ヘルプは「先頭1件だけで超えるときは
1件返して警告」と約束しており、コンテキスト削減を `max_size_mb` に任せられない状態だった。

## Decision

`include_plugin_content` のときは、本文を足した後の `kyous[]` を Node が Go と同じ規則で守り直す
（2件目以降は足す前に判定、先頭1件は超えても返して警告）。押し出した分は次頁へ回す ——
`returned_count` / `remaining_count` / `has_more` を直し、`next_cursor` を最後に残した1件の
`{related_time}::{id}`（Go の `encodeMCPCursor` と同じ形）にし、`warnings[]` に件数を書く。
`plugin_content` の集計は残した分から数え直す。

## Rejected alternatives

- **本文側を切り詰めて件数を守る（`content_text` を短くする）** — `limit` は守れるが `max_size_mb` の意味が
  「本文の長さ」に化け、呼び出し側は `plugin_content_max_text_length` を自分で調整することになる。
  Go の規則（件数を削る）と揃えたほうが説明が1つで済む。
- **Go に本文を取りに行かせて DTO ごと測る** — 本文はプラグインの stdio を直列に往復する（ADR-0602）ので
  Go の検索ハンドラの中で待つと検索全体が本文取得の時間だけ遅くなる。Node 側の後段でよい。
- **警告だけ足して超過はそのまま返す** — 「上限を信頼できる」という要件を満たさない。

## Consequences

- `include_plugin_content` の頁は `limit` より少なく返ることがある（`max_size_mb` は元から厳密上限）。
  押し出した分は `next_cursor` で取れ、`warnings[]` にその旨が出る。
- Node がカーソルを組み立てる唯一の箇所になる。Go の `encodeMCPCursor` の形（ローカル時刻の RFC3339Nano +
  `::` + id）を変えるときはここも直す（DTO の `related_time` は同じ書式で出るので、いまは一致している）。
- Go 側で直列化に失敗して読み飛ばした1件（既存の警告）と同時に起きた場合、その1件は次頁で再出現しうる
  （従来からの挙動。今回の押し出しには含めない）。

## Evidence

- 報告値: 上限約 2,097 バイトに対し Kyou 配列 3,596 バイト・2件・警告なし。
- 修正後の単体テスト: 同じ大きさの3件に2件ぶんの予算 → 2件返り、`remaining_count` +1、`next_cursor` が
  2件目の `{related_time}::{id}`、警告1行。先頭1件だけで超える → 1件＋警告。収まる頁は無変更。

## Related tests

- `src/server/gkill/mcp/read_handlers_test.go`（`enforceKyousSizeBudget` の describe）
