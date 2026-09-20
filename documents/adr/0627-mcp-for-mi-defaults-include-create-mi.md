# ADR-0627: for_mi だけの検索は MCP の入口で include_create_mi を補う

| | |
|---|---|
| Status | Accepted |
| Date | 2026-09-19 |
| Sources | 2026-09-18 の MCP 実利用報告（「`for_mi` で `include_*_mi` 未指定 → 0件。警告は親切だが MCP 層で `include_create_mi` を既定 ON にする方が安全」）。ADR-0610（射影は `for_mi` に依存する） |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/mcp/normalization.go`（`normalizeKyouArgs` の for_mi 補完・`FOR_MI_DEFAULT_PROJECTION_NOTE`） |

## Context

`for_mi:true` で5つの `include_*_mi` が1つも立っていない検索は、射影の供給源が無いので必ず0件になる
（`mi_repository_sqlite3_impl.go` の `sqlSegments` が空）。Go は警告を返す（ADR-0610 の派生）が、実利用の AI は
0件を「タスクが無かった」と読んだ（2026-08-25）。ADR-0610 は「`for_mi` を勝手に立てない」と決めたが、
それは `for_mi` が検索対象を Mi / MiReKyou に**限定**するからで、`for_mi` を呼び出し側が明示したときの
射影の既定とは別の話。

## Decision

`for_mi:true` かつ5フラグが全て未指定 / false のとき、MCP の入口で `include_create_mi:true` を補い、
補ったことを応答の `warnings[]` の先頭に1行載せる（Go の警告と同じ列）。`for_mi` 自体は補わない。

## Rejected alternatives

- **Go 側で補う** — API を直接叩く経路の意味論（「全 false は0件」）を変えることになり、Web の Mi 画面が
  依存する挙動に触る。MCP の入口だけで済む。
- **黙って補う（警告なし）** — 「見ているのは作成時刻の射影だ」と知らずに期限や見積の話をされる。
  補ったことは言う。
- **何もせず警告だけ（従来）** — 0件と警告を同時に返しても、0件のほうが先に読まれる。

## Consequences

- `for_mi:true` 単独の検索は `mi_create` 射影で返る。他の時刻軸を見るには `include_*_mi` を明示する。
- 明示したフラグが1つでもあれば触らない（`include_limit_mi:true` だけの検索は期限のある Mi だけ）。

## Evidence

- 修正後の単体テスト: `{for_mi:true}` → 送信クエリに `include_create_mi:true`、`warnings[0]` に
  「include_create_mi:true was assumed」。`{for_mi:true, include_limit_mi:true}` は無変更。

## Related tests

- `src/server/gkill/mcp/normalization_test.go`（「normalizeKyouArgs assumes include_create_mi」）
- `src/server/gkill/mcp/read_handlers_test.go`（「for_mi without a projection flag」）
