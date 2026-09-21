# ADR-0620: 廃止済み引数は公開スキーマに載せず、受理と古さの検出だけ残す

| | |
|---|---|
| Status | Accepted |
| Date | 2026-09-14 |
| Sources | MCP レビュー P1（`include_id` / `include_rep_name` / `only_latest_data` が deprecated な入力引数として tools/list に残っている指摘）。[gkill-mcp](../../.claude/skills/gkill-mcp/SKILL.md) の節「廃止済み引数は公開スキーマに載せず、受理と古さの検出だけ残す。」 |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/mcp/read_tools.go`（`gkill_get_kyous` の `inputSchema.properties`）/ `src/server/gkill/mcp/find_query_schema.go`（`FIND_QUERY_SCHEMA.properties`）/ `src/server/gkill/mcp/constants.go`（`KYOUS_TOP_LEVEL_FIELDS` / `LEGACY_USE_FLAG_KEYS`）/ `src/server/gkill/mcp/normalization.go`（`DEPRECATED_TOP_LEVEL_ARGS` / `DEPRECATED_QUERY_FIELDS` / `detectStaleSchemaSignals`） |

## Context

`gkill_get_kyous` の公開スキーマには、v2（[ADR-0604](0604-mcp-composite-cursor-strict-limits.md)）で
意味を失った `include_id` / `include_rep_name`（id と rep_name は常時付与）と、MCP 層が常に
true へ強制する `query.only_latest_data`（[ADR-0605](0605-mcp-version-history-is-a-dedicated-tool.md)）が
「Deprecated: accepted for backward compatibility and ignored」として載り続けていた。
`FIND_QUERY_SCHEMA` の説明文も旧 `use_X` フラグの受理規則を説明していた。

受理を残す理由は正しい（古い一覧を握ったセッションは送り続けるし、届いたこと自体が
古さの証拠になる。[ADR-0609](0609-stale-tool-schema-is-warned-only-when-proven.md)）。
しかし公開スキーマに載せる理由は無い。AI はスキーマに載っているものを「使えるもの」と
読むので、`include_id=false` で何が変わるのか、`only_latest_data=false` なら過去版が読めるのか、
と考える余地を作るだけで、そのぶんの説明文とプロパティ定義は全セッションが毎回払う。
2026-09-14 のレビューが「廃止済み入力をスキーマに載せ続けること自体がコンテキスト税」と
指摘したのはここで、**実行時の互換と公開スキーマは分けてよい**。

## Decision

- `gkill_get_kyous` の `inputSchema.properties` から `include_id` / `include_rep_name` を、
  `FIND_QUERY_SCHEMA.properties` から `only_latest_data` を外す。説明文からも旧 `use_X` の
  受理規則を消す
- 受理集合（`KYOUS_TOP_LEVEL_FIELDS` / `KYOUS_QUERY_ALL_FIELDS` / `LEGACY_USE_FLAG_KEYS`）は
  **変えない**。届いたら今までどおり `detectStaleSchemaSignals` が古さの証拠として警告する
- 「公開スキーマのキー集合 = 受理集合 − 廃止済み」を `schema_contract_test.go` が固定する。
  廃止済みの表（`DEPRECATED_TOP_LEVEL_ARGS` / `DEPRECATED_QUERY_FIELDS`）はそのために export した

## Rejected alternatives

- **受理側からも消す（未知キーとして拒否する）** — 古い一覧を握ったセッションの
  `gkill_get_kyous` が全部落ちる。廃止から日が浅いほど、そういうセッションは多い。
  しかも「届いたこと」が古さの証拠で、消すとその証拠も失う
- **公開スキーマに残したまま説明文だけ短くする** — プロパティが載っている限り AI は
  値を決めようとする。「無視される」と書いても「なぜ載っているのか」は消えない
- **`additionalProperties: true` のトップレベルで黙って受ける** — トップレベルは
  `additionalProperties: false` で、JSON Schema を送信前に検証するクライアントが弾く。
  受理の正本は normalizer で、スキーマの `additionalProperties` は緩めない

## Consequences

- 新しく接続したクライアントの一覧には廃止済み引数が無い。古い一覧のクライアントは
  送り続けられ、その呼び出しには古さの警告が付く
- 廃止済み引数を「もう受理しなくてよい」と判断する日が来ても、受理集合から消す前に
  この ADR と `detectStaleSchemaSignals` の証拠表を一緒に見直すこと。片方だけ消すと
  `schema_contract_test.go` が落ちる（それが狙い）
- 今後、引数を廃止するときの手順は「公開スキーマから外す → 受理集合に残す →
  `DEPRECATED_*` へ足す」の3点で、`schema_contract_test.go` が3点の整合を検査する

## Evidence

- tools/list の実測: `gkill_get_kyous` の廃止済み3引数と `use_X` の説明文で約 1.0KB。
  `gkill_status` 追加（+1.2KB × 3サーバ）と相殺して readwrite は 93,740 B → 93,980 B
- 実利用レビュー（ADR-0609 の Sources）では、`include_id` の既定値をめぐる
  指摘が「既に直っているのに見えていなかった」4件のうちの1件だった。載せ続けても
  古い一覧には届かないし、新しい一覧には要らない

## Related tests

- `src/server/gkill/mcp/schema_contract_test.go`（`advertised keys and accepted keys agree`）
  - スキーマのキー集合 = 受理集合 − 廃止済み。廃止済みは受理側にだけある
- `src/server/gkill/mcp/normalization_test.go`（`deprecated arguments are accepted at runtime but absent from the advertised schema`）
  - 公開スキーマに無い・説明文に `use_X` が無い・届いても受理され古さの警告が付く
- `src/server/gkill/mcp/readme_examples_test.go`
  - README の例が実物の正規化器を通る（廃止済み引数を例に残していない）
