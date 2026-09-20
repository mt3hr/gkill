# ADR-0626: gkill_get_kyou_history は offset で続きを読む

| | |
|---|---|
| Status | Accepted |
| Date | 2026-09-19 |
| Sources | 2026-09-18 の MCP 実利用報告（3版あるメモに `limit:1` で `has_more:true` だが `next_cursor` も継続用の引数も無い。公開上限 200 版を超えた履歴へ進む経路が定義上無い）。ADR-0605（版履歴は専用ツール） |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/mcp/normalization.go`（`normalizeKyouHistoryArgs`）/ `src/server/gkill/mcp/read_handlers.go`（`gkill_get_kyou_history` の case） |

## Context

型別の `/api/get_xxx` は全版を返し、Node が `slice(0, limit)` で切っていた（ADR-0605）。`has_more` は出るのに
続きを取る引数が無く、`limit` の公開上限 200 を超える履歴には到達できなかった。

## Decision

`offset`（新しい順の先頭から飛ばす版数、既定 0）を足す。応答に `offset` と、`has_more` のときだけ
`next_offset` を載せる。`offset` が版数以上なら 0 件・`has_more:false`（not found にしない）。
古い一覧を握るクライアントからは文字列で届くので `KYOU_HISTORY_STALE_SCHEMA_ARG_KINDS` に載せる（ADR-0609）。

## Rejected alternatives

- **`gkill_get_kyous` と同じ複合カーソル** — 履歴は Node が全版をメモリに持っているので、位置を不透明トークンに
  する理由が無い。offset のほうが呼び出し側が数えられる。
- **Go の型別エンドポイントに limit / offset を降ろす** — 12 種のエンドポイントと Web の呼び出し全部に手が入る。
  履歴は1件ぶんなので Node で切って足りる。
- **`limit` の上限を撤廃する** — 履歴は編集のたびに伸びるので、上限を外すと1回の応答が有界でなくなる。

## Consequences

- `has_more:true` の応答には必ず `next_offset` がある。続きは `offset:next_offset` で取る。
- 版の `data_type` はこの口の語彙（エンティティ名）に揃えて返す（同じ Mi が検索 / 履歴 / 更新応答で
  3通りの射影名を名乗っていた。ADR-0611 の橋の逆向き）。

## Evidence

- 修正後の単体テスト: 3版を `limit:1` で `offset` 0 → `next_offset:1`、`offset:2` → 1件・`has_more:false`、
  `offset:5` → 0件。要約文に「from offset 1」「offset:2」。

## Related tests

- `src/server/gkill/mcp/read_handlers_test.go`（「gkill_get_kyou_history offset and data_type」）
- `src/server/gkill/mcp/normalization_test.go`（`normalizeKyouHistoryArgs` の offset）
