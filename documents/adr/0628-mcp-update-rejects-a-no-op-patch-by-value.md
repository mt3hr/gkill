# ADR-0628: 更新は現在値と同じ値だけのパッチも書かずに断る

| | |
|---|---|
| Status | Accepted |
| Date | 2026-09-19 |
| Sources | 2026-09-18 の MCP 実利用報告（「`update_*` は同一内容でも新バージョンを積む（履歴4版で確認）。delete の already deleted ガード相当が欲しい」「`update_mi` で `limit_time` 等を消す手段がない」）。ADR-0616（空のパッチは断る） |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/mcp/lib/write-handlers.mjs`（`runUpdate` / `valuesEquivalent`）/ `src/mcp/lib/write-normalization.mjs`（`mi` の `nullClears`） |

## Context

ADR-0616 は「変わる欄が1つも無い」パッチを断ったが、判定は `normalized[field] !== undefined`（引数が渡されたか）
だけで、現在値と突き合わせていなかった。同じ値を渡した実質 no-op は通り、追記型なので内容の同じ版が積まれる。
`softDeleteOne` だけが現在値（`is_deleted`）を照合していて非対称だった。
あわせて Mi の予定日時3欄には `nullClears` が無く、`null` を渡しても `optionalDatetime` が `undefined` に
畳んで「触らない」と解釈され、一度入れた期限を MCP から二度と外せなかった（エラーも警告も無し）。

## Decision

`runUpdate` は指定された全欄を現在値と比べ（日時はオフセット違いを同じ瞬間として、数値は `Number` で、
`null` は現在値が空なら同じ）、**全欄が同じなら書かずに断る**。1欄でも違えば従来どおり書く。
Mi の `limit_time` / `estimate_start_time` / `estimate_end_time` は `timeis.end_time` と同じ3値パッチ
（未指定 = 触らない、`null` = 消す、値 = その値）にする。

## Rejected alternatives

- **同じ値でも書いて `unchanged:true` を返す** — 版は積まれる。追記型で「何が変わったのか」が区別できない
  という ADR-0616 の理由がそのまま当てはまる。
- **Go の update API で拒否する** — Web の編集画面は「保存」を押した回数だけ版を積む設計で、そこを変えると
  Web の挙動が変わる。MCP の入口で足りる。
- **文字列の完全一致だけで比べる** — 日時は Go が `+09:00` で返し MCP は `Z` で送るので、同じ瞬間を「違う」と
  判定して no-op を通してしまう。

## Consequences

- 同じ内容の再送は「No effective change」で止まる。呼び出し側は値を変えて呼び直す。
- 応答の `updated_xxx.data_type` はエンティティ名に揃える（delete / restore / history と同じ。ADR-0626）。

## Evidence

- 修正後の単体テスト: `title` 同一 + `limit_time` を `Z` 表記で同じ瞬間 → 拒否・更新 API は呼ばれない。
  `limit_time: null` → `mi.limit_time` に `null` が送られる。既に空の欄へ `null` → 拒否。

## Related tests

- `src/mcp/__tests__/write-handlers.test.mjs`（「2026-09-19 additions」の describe）
