# ADR-0313: プラグインのログは gkill_log の別名ファイルへ出し、stderr には WARN / ERROR の接頭辞行だけを残す

| | |
|---|---|
| Status | Accepted |
| Date | 2026-09-20 |
| Sources | `.claude/skills/gkill-plugin/SKILL.md`「プラグインのログは `sdk.LogXxx` を通す」 / `.claude/skills/gkill-cli-ops/SKILL.md`「ログは統合ファイルとレベル別ファイルへ同時に出し」 / [ADR-1001](1001-log-level-by-severity.md) の Consequences「Go 以外はレベル機構がまちまちのまま」 / [ADR-0631](0631-mcp-lives-in-gkill-server.md) の `InitNamed` |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/plugin/sdk/plugin_log.go` / `src/server/gkill/plugin/sdk/log.go` / `src/server/gkill/main/common/gkill_log/child_env.go` |

## Context

プラグイン（`src/plugins/` の7本と、別リポジトリの `gkill_plugin_uguisu`）のログは
**stderr に `WARN: ` / `ERROR: ` を付けた行を書くだけ**だった（`plugin/sdk/log.go` の `LogWarn` / `LogError`）。
gkill_server は子プロセスの stderr を自分の stderr へ素通しし、末尾 4KB をリングに持って
`get_plugin_list` の `last_error` に載せる（`plugin_stderr_ring.go`）。

利用者環境ではプラグインは常駐サーバ（Windows サービス）の子プロセスなので、**素通しした stderr は誰も見ておらず、残るのは最後の 4KB だけ**。
起動・構築完了・1コマンドの所要時間といった節目は、どこにも残らない。
「プロセスが殺され続ける」（ハンドラ期限超過）の調査で、どのコマンドが何ミリ秒かかったかを
知る手段が無かった。ADR-1001 も Consequences で「プラグインは `WARN:` / `ERROR:` の接頭辞だけ」と
現状追認で止めていた。

一方 MCP は 2026-09-20 に Go へ移り、`gkill_log.InitNamed("gkill_mcp_<kind>", ...)` で
`logs/gkill_mcp_<kind>*.log` へ出す形が確立した（ADR-0631）。同じ基盤に載せる条件は揃っていた:
`gkill_log` → `gkill_options` は標準ライブラリだけで閉じた葉で、SDK が既に `api/find_word` で
同じ形の依存を持っている（ADR-0113）。

## Decision

**SDK の `sdk.Run` が起動時に `$GKILL_HOME/logs/gkill_plugin_<name>*.log` を開き、
プラグインのログはすべて `sdk.LogWarn` / `LogError` / `LogInfo` / `LogDebug` を通す。**

| 項目 | 決定 |
|---|---|
| ファイル | `logs/gkill_plugin_<name>.log`（統合）+ `gkill_plugin_<name>_{error,warn,info,access,debug,trace,trace_sql}.log`。`<name>` は `--gkill-plugin-dir` の末尾（manifest の `name`）。既に `gkill_plugin_` で始まる名前は二重にしない |
| 静的フィールド | `app=gkill_plugin` / `plugin=<name>` / `user_id` / `pid` |
| レベル・回転 | 本体の `--log` / `--log_rotate_max_bytes` / `--log_rotate_keep` を**環境変数** `GKILL_LOG_LEVEL` / `GKILL_LOG_ROTATE_MAX_BYTES` / `GKILL_LOG_ROTATE_KEEP` で継ぐ。`gkill_log.Init()` が書き出し（`ExportEnvForChildProcesses`）、子は `os.Environ()` 継承で受ける（`GKILL_HOME` と同じ経路。起動側の `cmd.Env` は nil のまま） |
| stderr | `WARN: ` / `ERROR: ` の行は**これまでどおり stderr にも出す**。`last_error` リングと本体 stderr の素通しを変えない。`LogInfo` / `LogDebug` はファイルだけ |
| SDK 自身の節目 | `plugin start` / `plugin stop`（理由つき）/ `build cache`（結果つき）を Info、**1コマンド1行**（command / id / duration_ms / count / error）を Access |
| 壊れた設定 | 未知のレベル名・非整数は既定（error / 32MiB / 5）へ倒して stderr に1行。ログ dir を作れない・home が分からない（手起動）ときは stderr だけで続行。**panic も exit もしない** |
| source | `runtime.Callers` で Record を組み、プラグイン側の呼び出し行を指す（MCP の `access_log.go` と同じ作法） |

`gkill_log` には panic しない `InitNamedWith(NamedOptions)` と `ParseLevel` を切り出した。
`InitNamed`（本体・MCP）の挙動は変えていない。

## Rejected alternatives

- **stdio プロトコルに `log` コマンドを足して本体のログへ合流させる** — プラグイン→本体の方向は
  応答しか無く、逆向きのメッセージを足すとスロット直列化（容量1のチャネル）の前提が崩れる。
  本体のログに他プロセスの行が混ざるのも読みにくい。
- **本体が子の stderr を読んで gkill_log に取り込む** — レベルも構造も失った文字列を
  Warn か Error のどちらかに丸めるしかない。Info / Access は表現できない。
- **`--gkill-log-level` のようなフラグで渡す** — SDK を使わない第三者のプラグインが標準の
  `flag` で未知のフラグに exit 2 で落ちる（`generate_plugin_cache` が「旧バイナリ」と数える形）。
  環境変数なら知らないプラグインは無視するだけで済む。
- **利用者ごとにファイル名を分ける（`gkill_plugin_<name>_<user>`）** — 探しにくい。
  `user_id` の静的フィールドで足りる。
- **未知のレベル名で起動を止める（MCP と同じ）** — MCP は自分が設定される主体だが、
  プラグインは親から渡される側で、親は検証済みの値しか渡さない。手起動の打ち間違いで
  「rep は候補に出るのに0件」にする理由がない。
- **stderr への出力をやめてファイルだけにする** — `last_error` が空になり、`get_plugin_list` から
  「is_alive=true なのに0件」の理由が読めなくなる（指摘 D2 の再発）。

## Consequences

**同じプラグインを2プロセスが同じファイルへ書くことがある。** 別の利用者の常駐、または
稼働中サーバの常駐と `generate_plugin_cache` の単独起動。行単位の追記（`O_APPEND`）は混ざらないが、
**32MiB を超えたときの回転は、Windows では他方が開いている間 rename に失敗し、閉じるまで
ファイルが育つ**（`sinks.go` は失敗しても書き込みを続ける）。既定の error なら到達しない量。
`pid` の静的フィールドで行の出所は分かる。

**`logs/` のファイル数が増える。** プラグイン1本につき8ファイル（統合 + 7分割）。
本体の `gkill*.log` と MCP の `gkill_mcp_*.log` と同じ並び。

**SDK が gkill 本体のパッケージを2つ余分に import する。** どちらも標準ライブラリだけの葉
（`gkill_log` / `gkill_options`）で、`api/find_word` と同じ形。SDK を使うプラグインの go.mod に
新しい外部依存は増えない。

**`LogDebug` の定義は1行関数のままにする。** `log_level_source_scan_test.go` の
「ブロック末尾の Debug ＝握り潰し」検査が、複数行にすると定義そのものを誤検知する。

## Evidence

実測なし — 構造上の判断（常駐サーバの子プロセスとして動く利用者環境では stderr の読み手が無く、リングは 4KB の固定長。
素通し先が無い以上、ファイルに残す以外に節目と所要時間を後から読む手段が無い）。
依存の大きさは `go list -deps ./gkill/main/common/gkill_log` で確認: gkill 本体のパッケージは `gkill_options` 1つだけで、
その import は `net` / `runtime` / `time` の標準ライブラリだけ。SDK を使うプラグインの go.mod に新しい外部依存は増えない。

## Related tests

- `src/server/gkill/main/common/gkill_log/child_env_test.go`（レベル語彙・panic しない `InitNamedWith`・環境変数の往復・既定値と警告・`Init()` の書き出し）
- `src/server/gkill/plugin/sdk/plugin_log_test.go`（ファイルの置き場と静的フィールド・接頭辞の二重回避・home 不明時は stderr だけ・壊れたレベルの既定化・Access 行と stdout の清浄）
- `src/server/gkill/plugin/sdk/build_cache_test.go`（stderr の接頭辞行が変わっていないこと）
