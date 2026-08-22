# ADR-0025: Codex ロールアウトログのスレッド識別子はファイル名の uuid、会話は event_msg レーンだけ

| | |
|---|---|
| Status | Accepted |
| Date | 2026-08-13 |
| Sources | `0caffd06` / `80383e38` / `7b9f0127` / `.claude/skills/gkill-plugin/SKILL.md` の `gkill_plugin_codex` コメント |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/plugins/gkill_plugin_codex/reader.go` |

## Context

OpenAI Codex CLI のロールアウトログ（JSONL）を Kyou 化するにあたり、実データ52ファイル・245MB を調べたところ、**素直な読み方がことごとく外れた**。

- `session_meta.session_id` は**52件中23件に無い**。しかもサブエージェントのファイルでは**親のIDを指す**
- `session_meta` は1ファイルに **1〜13回**繰り返し現れる
- `response_item/message` レーンには**注入された前置き**が混ざる
- ツールの実行結果は**バイトの94.7%**を占めるが、会話の再現には要らない
- 1行が **19.9MB** のものがある

## Decision

- **スレッド識別子は「ファイル名の uuid」**。`session_meta.session_id` は使わない
- **identity は最初の `session_meta` だけ**から取る。ただし `cwd` / git / originator は全出現をマージする
- **会話は `event_msg` レーンのみ**から作る。ツールの*呼び出し*は `response_item` から取り、ツールの*結果*は保存しない
- Kyou ID は **(スレッドid, role, 序数) の UUIDv5**
- `reader.go` は**先頭512バイトで行を分類**し、不要な行はバッファせずに読み捨てる

## Rejected alternatives

- **`session_meta.session_id` をスレッド識別子にする** — 52件中23件に無く、あってもサブエージェントでは親を指す。**別スレッドが同じIDに畳まれる**。

- **`session_meta` の全出現をマージして identity を作る** — サブエージェントのファイルは**2つ目の `session_meta` として親のメタを持つ**。マージすると子が親を名乗る。identity は最初の1つだけ、が正しい。ただし `cwd` / git / originator は追記されていくのでマージする（ここだけ扱いが逆になる）。

- **会話を `response_item/message` から作る** — 注入された前置きが混ざる。利用者が書いていない文が会話として保存される。

- **ツールの実行結果も保存する** — **バイトの94.7%**。保存しても検索の役に立たず、キャッシュだけが肥大する。

- **Kyou ID をレコードの id から作る** — **`event_msg` のレコードには id が無い**。(スレッドid, role, 序数) の UUIDv5 にすると、ログが追記専用なので序数が動かず**IDが安定する**。

- **行を丸ごと読んでから分類する** — 1行19.9MB のものがあるので、分類のためだけに全部バッファするとメモリが跳ねる。先頭512バイトで分類し、不要な行は読み捨てる。

## Consequences

「ファイル名がスレッドの identity」という前提が入るので、**ログファイルを改名すると別スレッドになる**。Codex CLI が改名しないことに依存している。

序数ベースのIDは**追記専用であることに依存する**。過去のレコードが書き換えられたり挿入されたりすると、以降のIDが全部ずれる。

## Evidence

- 実データ52ファイル / 245MB
- `session_meta.session_id`: **52件中23件に欠落**
- `session_meta` の出現回数: 1ファイルあたり **1〜13回**
- ツール実行結果: 全バイトの **94.7%**
- 最大の1行: **19.9MB**
- 構築: 初回 4.5秒 / 増分 1秒未満（52ファイル → 301 Kyou）

## Related tests

- `src/plugins/gkill_plugin_codex/cache_test.go`
