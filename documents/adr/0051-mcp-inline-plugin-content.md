# ADR-0051: MCP のプラグイン本文は get_kyous へインライン埋め込みし、同一プラグインへ並列に投げない

| | |
|---|---|
| Status | Accepted |
| Date | 2026-08-05 |
| Sources | `2d10492c` / `d577bba7` / `CLAUDE.md`「Plugin tools」節 / `documents/reverse/plugin-system.md`「13. MCP からのプラグイン内容取得」 |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/mcp/lib/plugin-tools.mjs` |

## Context

プラグインKyou の本文は gkill に保存されておらず、`get_kyous` が返すのはメタデータと `rep_name` / `kyou_id` を持つ `payload.kind="plugin"` だけ。本文はプラグインに聞きに行く必要がある。

当初は1件ずつ取る `gkill_get_plugin_content` ツールを公開していたが、**AI の往復が N+1 回**になり非効率だった。

## Decision

`gkill_get_kyous` に `include_plugin_content:true` を渡すと、**同じレスポンスに本文を埋め込む**。1件ずつ取るツールは廃止する。

インライン化は `plugin-tools.mjs` の `inlinePluginContents` が担当し、`/api/get_plugin_content_html` を **rep ごとに直列・rep 間は並列（既定4）**で叩く。

## Rejected alternatives

- **1件ずつ取るツールを残す（改修前）** — AI の往復が N+1 回。一覧を読むだけで数十回の往復になる。

- **同一プラグインへ並列に投げる** — **速くならない。** プラグインの stdio は1本しかなく呼び出しは直列化される（→ ADR-0020）ので、同時発行しても**順番待ちが伸びるだけ**。待ちきれなかったぶんは `ErrPluginBusy` になる。

  （2026-08-06以前の gkill は30秒デッドラインを排他ロックの**前**に張っていたため、同時発行するとロック待ちで期限を食い潰し、期限切れ時の `Process.Kill()` で**プラグインプロセスが回収されていた**。現在は期限をスロット取得後に張るのでこの誤射はない。）

- **インライン取得の途中で実行中のリクエストを abort する** — abort 自体は現在プロセスに影響しないが、MCPサーバは**古い gkill にも接続しうる**。デッドラインは「新しいリクエストを始めない」だけで実現する。

- **本文を無制限に埋め込む** — レスポンスが膨れて AI のコンテキストを食い潰す。上限は1件4000文字（`plugin_content_max_text_length` で最大200000まで）・合計200000文字・20件・30秒。

## Consequences

各ペイロードに **`content_status`**（`ok` / `truncated` / `skipped` / `error`）が付く。**`ok` 以外なら本文は不完全**なので、AI 側はこれを見て判断する必要がある。

`plugin_content_format` で `text`（既定）/ `html` / `both` を選べる。HTML→テキスト変換は `lib/html-text.mjs`。

**Write 専用サーバには `gkill_get_kyous` が無い**ため、プラグイン本文を読むには ReadWrite サーバを使う。

3サーバ共通のプラグインツールは `gkill_get_plugin_list` の1つだけ（読み取り専用。`post_plugin_config` は公開しない）。

## Evidence

実測なし — 構造からの判断（stdio が1本しかないので並列化に意味が無い）。

症状は実測されている: 2026-08-06以前は同時発行でプラグインプロセスが繰り返し回収されていた。

## Related tests

- `src/mcp/__tests__/plugin-tools.test.mjs`
- `src/mcp/__tests__/html-text.test.mjs`
