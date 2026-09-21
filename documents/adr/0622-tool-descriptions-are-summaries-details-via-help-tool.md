# ADR-0622: ツール説明は要約にとどめ、詳細は gkill_get_mcp_help で取り出す

| | |
|---|---|
| Status | Accepted |
| Date | 2026-09-18 |
| Sources | 2026-09-18 の MCP 実利用報告（「最大の構造的課題はツール説明が巨大」。`gkill_get_kyous` と `gkill_submit_kftl` が百科事典化し、read と readwrite で同じ長文が重複する）。`.claude/skills/gkill-mcp/SKILL.md`「ツール一覧の説明文は要約にとどめ、本文は `gkill_get_mcp_help` の topic に置く。」 |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/mcp/help_topics.go`（本文の正本）/ `src/server/gkill/mcp/read_tools.go`（`gkill_get_mcp_help` の定義と要約化した説明文）/ `src/server/gkill/mcp/write_tools.go`（`gkill_submit_kftl` の要約）/ `src/server/gkill/mcp/find_query_schema.go` |

## Context

ツール一覧（tools/list）はクライアントのセッション開始時に丸ごとコンテキストへ載る。事故が起きるたびに
対策を説明文へ書き足してきた結果、2026-09-14 時点で readwrite は約 94KB、`gkill_get_kyous` 単体で 27KB
（説明文 5.1KB + inputSchema 21KB）、`gkill_submit_kftl` の説明文だけで 7.5KB になっていた。
内容そのものは正しい（ADR-0604 / 0605 / 0606 / 0610 / 0619 の事故対策）が、AI はタスクに関係ない説明まで
毎回読むことになり、実クライアントでは巨大応答が安全確認でブロックされる事象も出た。
ADR-0619 の予算ファイルは**増加を止める**が、**減らす**手段が無かった。

## Decision

ツール一覧の説明文は「何をするか・まず使う引数・必ず確認すること・詳細の在処」の要約にとどめる。
本文は `help_topics.go` の topic（search / pagination / mi / data_types / plugin / idf / deleted / rep / kftl。
2026-09-20 に [ADR-0632](0632-config-tree-descriptions-for-mcp.md) が config を足して 10 件）へ
**要約せずに移し**、3サーバ共通の `gkill_get_mcp_help(topic)` で取り出す（引数を省けば topic の一覧。gkill へは往復しない）。
要約化した説明文は「詳細は gkill_get_mcp_help の topic:xxx」と在処を案内する。

## Rejected alternatives

- **説明文をそのまま維持する** — readwrite 94KB。事故対策を足すほど太り、減る方向の力が無い。
- **MCP の resources / prompts で配る** — クライアントの対応がまちまちで、AI 自身からは見えないことが多い。
  ツールなら tools/list に載っているものは必ず呼べる。
- **簡易版と詳細版にツールを分ける（`gkill_get_kyous_simple` 等）** — 同名重複が増えて一覧が太る方向。
  read と readwrite で同じ説明が並ぶ問題も解決しない。
- **説明文を i18n して短い言語に寄せる** — description は静的配列で、`schema_revision` が locale ごとに割れる。
  長さの問題は言語の問題ではない。
- **inputSchema の property description をすべて1文にする** — `num_min` の単位注意や `include_create_mi` の
  「全 false で0件」のように、引数を書く瞬間に読まれないと事故になる文はある。引数の説明は「事故になる1文」まで残し、
  それ以外を topic へ落とした。

## Consequences

- **説明文へ事故対策を足したくなったら、まず help topic へ足す。** 要約側に足すのは「必ず確認すること」に
  昇格する場合だけ。予算テスト（`tool_schema_budget_test.go`）が増加を止め、`help_topics_test.go` が
  移した知識（KFTL の `~~` / `??` / `/endt?`、Mi の射影、warnings の規則 …）の実在を固定する。
- 説明文が名指しするツール名の走査（`tool_handlers_test.go`）は本文には掛からないので、本文が名指しする
  ツール名の実在は `help_topics_test.go` が別に検査する。
- 本文を読まない AI は事故対策を知らないまま呼べる。要約側の「在処」の案内と、`gkill_get_mcp_help` の説明文
  「最初の検索・最初の KFTL・理解できない warnings の前に読め」がその防御線で、`tool_handlers_test.go` が
  要約化した説明文に案内があることを固定する。
- ツール数は read 12 / write 29 / readwrite 33。3サーバの `schema_revision` が変わるので、配布後は各クライアントの
  接続し直しが要る（ADR-0619）。

## Evidence

- tools/list の実測（`gkill_server mcp schema-budget`）: read 47,187 → 36,109 バイト、write 60,256 → 55,381、
  readwrite 94,325 → 77,529（`gkill_get_mcp_help` を足したうえで）
- 残りの大半は `gkill_get_kyous` の inputSchema（`FIND_QUERY_SCHEMA` の 37 プロパティ）。次に削るならそこ

## Related tests

- `src/server/gkill/mcp/help_topics_test.go`
- `src/server/gkill/mcp/tool_handlers_test.go`
- `src/server/gkill/mcp/tool_schema_budget_test.go`
- `src/server/gkill/mcp/schema_contract_test.go`
