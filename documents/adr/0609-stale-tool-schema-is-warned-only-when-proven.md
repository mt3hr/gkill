# ADR-0609: 古いツールスキーマは「証明できるときだけ」警告する

| | |
|---|---|
| Status | Accepted |
| Date | 2026-08-24 |
| Sources | 2026-08-24 の実利用フィードバック（指摘9件のうち4件が「既に直っているのに見えていなかった」）。[ADR-0111](0111-drop-never-implemented-query-fields.md) の Consequences「旧スキーマはセッション寿命で固定される」 |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/mcp/lib/normalization.mjs`（`detectStaleSchemaSignals` / `staleSchemaWarning` / `appendStaleSchemaWarning`）/ `src/mcp/lib/read-handlers.mjs`（`handleReadToolCall`）/ `src/mcp/lib/write-handlers.mjs`（`handleWriteToolCall`）/ `src/mcp/lib/write-normalization.mjs`（`revivesStaleBoolean` / `DELETE_STALE_SCHEMA_ARG_KINDS`） |

## Context

MCP のツール一覧はクライアントがセッション初期化時に1回取るだけで、
**サーバを再起動しても生きているセッションには新しいスキーマが届かない**
（2026-08-24 に live コネクタで実測）。既存の対策は2つあるが、
どちらも**症状が出てから**しか効かない。

- `reviveStaleSchemaArgs` — 新引数が正規JSON文字列で届いたときに型を復元する
- `REMOVED_TOOL_HINTS` — 消したツールを呼んだときに代替を案内する

「自分の持っているスキーマが古い」こと自体を伝える経路が無かった。
その結果、実利用の一周で報告された9件の食い違いのうち**4件**が
「サーバ側では既に直っているが、そのクライアントからは永久に見えない」だった
（GPS の `cursor` 入力、`data_types` フィルタ、`cursor` の説明文、`include_id` の既定値）。
報告者から見ると「未実装」と「見えていないだけ」は区別が付かない。

## Decision

**古さが証明できたときだけ** `warnings[]` へ1行足す。証明になるのは2つだけ。

- 非string型の引数が正規JSON文字列で届いた
  （＝その引数がスキーマに無かった時代のクライアント。`STALE_SCHEMA_ARG_KINDS_BY_TOOL` で照合）
- 廃止済みだが後方互換で受理している引数が届いた
  （`include_id` / `include_rep_name` / `query.only_latest_data` / `LEGACY_USE_FLAG_KEYS` の16本）

文面には証拠（どの引数か）と、**見えていない可能性が高い機能を数個名指し**で入れる。
判定と付与は `handleReadToolCall` の**1箇所**で包む（ツールごとに書くと必ず足し忘れる）。
1行サマリにも印を付ける（本文の `warnings` を読まない経路があるため）。

後付け引数を持つ書き込みツールにも同じ判定を適用する（`handleWriteToolCall` も同じ1箇所方式で包む）。
決定時点では書き込み側に救済表へ載る引数が無く「警告できる面が無い」としていたが、
同日中に `gkill_delete_kyou` / `gkill_restore_kyou` の `targets`（オブジェクト配列）が、
2026-08-30 に `gkill_add_urlog` の `fetch_metadata` / `fetch_favicon` と
`gkill_add_mi` / `gkill_update_mi` の `allow_create_board`（いずれも後付け boolean）が加わり、
適用範囲が読み書き両方へ広がった（決定そのもの —— 証明できるときだけ警告する —— は不変）。
読み取りツールは readwrite サーバからも同じ `handleReadToolCall` を通るので網羅される。

## Rejected alternatives

- **毎応答にサーバ側のスキーマ版を載せ、クライアントに比較させる** —
  クライアントは自分の持っているスキーマの版を知らない（`tools/list` の結果に版は無い）。
  比較する相手が無いので、載せても判定できない
- **`has_more` や0件などの症状から推測して警告する** —
  正当な0件・正当な最終ページと区別が付かない。誤警告が常時出れば、
  本当に効く警告まで読まれなくなる。証拠のあるときだけ出すのが唯一もつ運用
- **古い引数を受理せずエラーにして気づかせる** —
  そのクライアントからは新しい引数を送る手段が無いので、
  「古い」と気づいた瞬間に**何もできなくなる**。救済（`reviveStaleSchemaArgs`）の趣旨と正面から衝突する
- **警告ではなくツール説明文に「古ければ再接続を」と書く** —
  その説明文自体が古いスナップショットの側にあるので、絶対に届かない。
  これはこの問題の核心そのもの
- **`include_id` / `include_rep_name` をスキーマから削除する** —
  `assertKnownKeys` が未知キーを弾くため、旧セッションのクライアントが**今度は動かなくなる**。
  死んだ引数が2本残るコストより、既存クライアントを壊すコストのほうが大きい

## Consequences

- 「もう直っている」を報告される回数が減る。少なくとも報告者が再接続を試せる
- 現行スキーマどおりの呼び出しには何も足さない（誤警告ゼロがこの仕組みの前提条件）
- `gkill_get_kyou_history` の `limit` も他の後付け引数と同じ穴を持っていたので、
  救済表（`KYOU_HISTORY_STALE_SCHEMA_ARG_KINDS`）を足して揃えた
- 救済表は4→6になり、その後 delete/restore・urlog・mi の書き込みツールも加わった。
  **引数を足したら表と `STALE_SCHEMA_ARG_KINDS_BY_TOOL` の両方へ載せる**
  （書き込みの後付け boolean は `ENTITY_FIELD_SPECS` の `revivesStaleBoolean` も立てる。
  この3点セットは `write-tool-handlers.test.mjs` の表駆動メタテストが機械強制する）

## Evidence

- 2026-08-24 の実利用一周: 9件の指摘のうち4件が実装済み機能だった。
  報告者のセッションには `gkill_get_gps_log` の `cursor` / `limit` / `count_only` / `group_by`、
  トップレベル `data_types`、`gkill_get_application_config` の `fields` がいずれも見えていなかった
- 同じセッションが `include_id` を「default: false」と説明されていた。
  現行スキーマは `default: true` + `Deprecated (v2)` で、実装とも一致している

## Related tests

- `src/mcp/__tests__/normalization.test.mjs`
  - `detectStaleSchemaSignals`（証拠2種の検出、現行スキーマでは null、string型は証拠にしない）
  - `normalizeKyouHistoryArgs — stale-schema revival`
- `src/mcp/__tests__/read-handlers.test.mjs`
  - `handleReadToolCall — stale tool schema warning`（gkill 由来の warnings に足すこと、誤警告を出さないこと）
- `src/mcp/__tests__/write-normalization.test.mjs`
  - 後付けフラグの正規化（正規JSON文字列からの復元、trim 込みで検出器と同じ受理範囲）
- `src/mcp/__tests__/write-handlers.test.mjs`
  - urlog / mi の文字列フラグが復元され、成功応答に古さの警告が付くこと（end-to-end）
- `src/mcp/__tests__/write-tool-handlers.test.mjs`
  - 後付け boolean 引数は救済表・型復元の両方に載る（表駆動メタテスト）
