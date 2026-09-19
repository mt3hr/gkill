# ADR-0630: 公開ファイルURLは include_file_urls で頼まれたときだけ発行し、期限を添える

| | |
|---|---|
| Status | Accepted |
| Date | 2026-09-19 |
| Sources | 2026-09-18 の MCP 実利用報告（「頼まれなくても idf ごとに公開URLを2本発行、期限が応答にない」）。ADR-0606（IDF ファイルの3経路）。利用者の判断: オプトイン化＋期限を載せる |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/mcp/lib/payload.mjs`（`MINT_FILE_LINKS` / `applyFileLinks`）/ `src/mcp/lib/mcp-server-base.mjs`（`buildToolResult`）/ `src/mcp/lib/file-link-store.mjs`（`mintLink`） |

## Context

HTTP トランスポートでは `buildToolResult` が全応答に `applyFileLinks` を掛け、idf ペイロードごとに
サムネと原寸の2本のトークンを鋳造していた。AI は URL を取りに行けない（ADR-0606 —— これは人間へ渡すリンク）ので、
頼んでいない呼び出しでは発行された全トークンが無駄で、応答にはいつ切れるかも載っていなかった。

## Decision

`gkill_get_kyous` に `include_file_urls`（既定 false）を足す。true のときだけハンドラが payload に
`MINT_FILE_LINKS`（Symbol。JSON には出ない）の印を立て、`buildToolResult` は HTTP かつ印があるときだけ
`applyFileLinks` を掛ける。それ以外は従来どおり `file_path` を消すだけ。発行した URL には
`file_url_expires_at`（RFC 3339）を添える。

## Rejected alternatives

- **引数を `buildToolResult` まで引数渡しする** — `handleMessage` → `buildToolResult` の全経路に args を通すことになり、
  古い一覧からの文字列 `"true"` の救済も2箇所になる（ADR-0611）。印は payload に1つ立てれば済む。
- **印を通常のプロパティ（`file_urls_requested:true`）にする** — JSON に出て、stdio では意味の無い欄が毎回並ぶ。
- **既定 true のまま期限だけ足す** — 頼んでいないトークンの鋳造が続く。利用者がオプトインを選んだ。

## Consequences

- HTTP クライアントは、人間へリンクを渡すときだけ `include_file_urls:true` を渡す。既定では `file_url` が無い。
- 期限は `GKILL_MCP_FILE_LINK_TTL_MS`（既定1時間）で、応答の `file_url_expires_at` で分かる。
- `FileLinkStore.mint` は残す（`mintLink` の薄い包み）。

## Evidence

- 修正後の単体テスト: 印無しの応答はリモートでも URL が無く `file_path` も消える・ストアは空。印ありは
  従来どおり2本＋`file_url_expires_at` が未来の時刻。`JSON.stringify` に印は出ない。

## Related tests

- `src/mcp/__tests__/file-link.test.mjs`（`buildToolResult file_url injection` の describe）
- `src/mcp/__tests__/read-handlers.test.mjs`（「forwards include_attached_ids and marks file-link minting only when asked」）
