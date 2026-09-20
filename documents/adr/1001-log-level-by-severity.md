# ADR-1001: ログレベルは事象の重さで決める。既定でエラーだけは必ず残す

| | |
|---|---|
| Status | Accepted |
| Date | 2026-08-30 |
| Sources | `7252746a` / `1de20236` / `c7757e98` / `c5f1b208` / `src/server/gkill/main/common/gkill_log/` |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/main/common/gkill_log/log_level_source_scan_test.go` |

## Context

**本番では1行もログが出ていなかった。**

`gkill_log.LogLevelFromCmd` の既定は `none` で、`--log` を付けなければ全レベルが停止する。
本番の NSSM サービス `GkillServer` の起動引数は `--pre_load_users ... --cache_reps_local=true` だけで
`--log` が無く、`$HOME/gkill/logs/` の10ファイルはすべて **0バイト**だった。

同時に、**エラーの記録がほぼ全部 Debug に置かれていた**。非テストの `slog.Log` 1998件のうち
Debug が 1798件（90%）で、その内訳は `"error at defer close"` 941 /
メッセージが `"error"` だけの無情報形 789 / 個別文言 68。つまり **Debug の 99.5% が
エラーの記録**で、Debug が「デバッグ情報」ではなく「エラーの既定の置き場」になっていた。
`usecase/` は210件すべてが Debug で、Warn も Error も1件も無かった。

この2つが重なると「500 が返るのに理由がどこにも出ない」になる。実際 2026-08-30 の障害
（[ADR-0216](0216-detach-a-broken-rep-but-never-silently.md)）では、**たまたま `--log debug` で
動かしていた回のログが残っていたおかげでしか原因に辿り着けなかった**。

逆方向の誤用もあった。`handle_file_serve.go` は Cookie が無いだけの403・404・共有範囲外の403 まで
Error（`gkill_error.log`）へ出していた。未ログインの初回アクセスやボットで、運用者向けのファイルが埋まる。

そして直近3コミット（`ea3a7a60` Debug→Warn、`a6914b6d` Warn→Error）は
「`--log warn` 運用のときに他の警告へ埋もれさせないため」を前提にレベルを上げていたが、
その前提は成立していなかった。

## Decision

**レベルは事象の重さで決める。呼び出し側の成り行きで決めない。**

| レベル | 意味 |
|---|---|
| `Error` | 運用者が**いま**知るべきサーバ側の障害。放置するとデータが壊れる／機能が使えない。利用者の操作では起こらない |
| `Warn` | 動き続けるが結果が痩せる・劣化する。または監査上残す利用者由来の事象（認証失敗・認可拒否・レート制限） |
| `Info` | 起動・終了・rep 構築完了・プラグイン起動といった節目。**1事象1行**で流れ続けない |
| `Access` | HTTP アクセスログ1行 |
| `Debug` | 開発時の詳細。**エラーの置き場ではない** |
| `Trace` / `TraceSQL` | 詳細トレースと SQL |

**判断は次の2つの軸をこの順で見る。**

1. **そのエラーは呼び出し元へ返るか。**
   返る（`return err` / `gkillErrors` に積んで `return`）→ 応答の `errors` に載り、
   `writeErrorStatus` が境界で1行出すので深部は `Debug` でよい。**同じ失敗を2回 Error で書かない。**
   返らない（ログして継続・`defer` 内・goroutine 内）→ **握り潰し。そのログが唯一の記録なので `Debug` 禁止。**
2. **原因が利用者側か、サーバ側か。**
   入力・認証・認可・レート制限 → `Warn`（監査に要る）または `Debug`。
   ディスク・DB・プロセス・設定 → `Error`。

**既定のログレベルは `error`。** 「`gkill_error.log` に出ていなければ起きていない」と
言えることを既定にする。

**失敗したリクエストの1行は `writeErrorStatus` が出す。** ステータスから機械的に決まる
（5xx=Error / 401・403・429=Warn / その他の4xx=Debug）。ステータスの割り当ての正本は
`message.HTTPStatusForErrors`（[ADR-0706](0706-http-status-from-error-code.md)）なので、
レベルもそこから導かれる。1行にはエラーコード・メソッド・パス・ユーザIDを載せる。

**アクセスログのミドルウェアは `Access` のまま触らない。** レベルを可変にすると
`gkill_access.log` の網羅性が崩れる。

**ログには回転を入れる。** 既定は32MiB・5世代（`--log_rotate_max_bytes` / `--log_rotate_keep`）。
既定でログが出るようにする以上、上限のない追記のままにはできない。

**プロセスの生死は `gkill_error.log` に残す。** 標準の `log.Fatal` は標準ロガーの stderr へ
書くだけで、サービスとして動いていると誰も見ない。`gkill_log.Fatal` を通す。

## Rejected alternatives

- **既定を `warn` にする** — Warn には常態化しうるもの（rep パターンの0件マッチ、プラグインの再起動、
  認証の失敗、ローカルキャッシュの更新スキップ）が入る。既定で流れ続けるものを既定に置くと、
  「出ていない＝起きていない」と読めなくなる。`error` なら、1行出ている時点で必ず見る価値がある。

- **既定は `none` のまま、本番サービスの起動引数に `--log warn` を足す** — その環境だけが直る。
  デスクトップ版・Android 同梱サーバ・他の利用者の環境は「何も残らない」ままで、
  次の障害でまた「たまたまデバッグで動かしていた回のログ」に頼ることになる。
  既定を変えれば起動引数の変更は要らない。

- **深部の1798件を触らず、境界の1行だけで済ませる** — 応答へ返らないエラー（`defer` 内の
  ロールバック失敗、write-through キャッシュの書き込み失敗、最新版アドレスの更新失敗）は
  境界に現れない。とくに後者2つは「記録は保存されたのに一覧から消える」型の事故で、
  応答は成功なのでログ以外に痕跡が無い。

- **エラー経路をすべて機械的に Error へ上げる** — 1リクエストの失敗で呼び出し階層ぶんの
  Error が積み、`handle_generate_tls_file.go` に実在した二重ログ（Error の直後に無情報の Debug）が
  全体に広がる。境界の1行があるので、深部は種類で分ければよい。

- **`slog` の標準レベル（Debug/Info/Warn/Error）だけにしてカスタム4レベルをやめる** —
  `Access` は「アクセスログだけ出す」運用に、`TraceSQL` は SQL 全文の出力に使われていて、
  どちらも既存のログファイル分割と `--log` の値に現れている。統合の利得より移行の危険が大きい。

- **レベルの妥当性をレビューだけで守る** — 1998件を人が数えて確かめ続けることはできない。
  実際「ファイル単位で一律 Warn」（`handle_login.go`）「同じファイルの中で Debug と Error が
  無秩序に混在」（`handle_urlog_bookmarklet_address.go`）という形で崩れていた。
  `log_level_source_scan_test.go` の4検査で機械的に固定する。

## Consequences

**既定でログファイルが育つ。** 回転を同時に入れたのはこのため。Error は稀なので既定の
32MiB・5世代で当面足りるが、`--log debug` 以下で長期運用するときは世代数を見直すこと。

**Warn の使い分けが2種類ある。** 「サーバ側の劣化」と「監査に残す利用者由来の事象」で、
どちらも `gkill_warn.log` に混ざる。分けたければファイルを増やすことになるが、
いまは `--log warn` にしたときの視界が汚れない範囲に収まっている。

**`writeErrorStatus` が ctx を取る。** 呼び出し93ファイルは同一形だったので機械置換したが、
新しいハンドラを書く人は `writeErrorStatus(r.Context(), w, response.Errors)` の形で書く必要がある。
`response_status_guard_test.go` がその1行の有無を機械検査している。

**握り潰しの許可リストは運用で腐りうる。** `swallowedDebugAllowlist` に理由なしで足せば検査を
素通りできる。`TestSwallowedDebugAllowlistIsUsed` が実在しない項目を落とすので、
消えたものは気づけるが、「安易に足した」ことは機械では検出できない。

**Go 以外はレベル機構がまちまちのまま。** MCP は `MCP_LOG` の6レベル、プラグインは
`WARN:` / `ERROR:` の接頭辞だけ、クライアントは `console` の3段、Android は `Log.d/i/w/e`。
統一はせず、それぞれの流儀の中で「重さと合っているか」だけを揃えた。
（追記 2026-09-20: MCP は Go へ移って gkill_log の別名ファイルに [ADR-0631](0631-mcp-lives-in-gkill-server.md)、
プラグインも SDK 経由で同じ基盤に載り、stderr の接頭辞行は `last_error` 向けに残した [ADR-0313](0313-plugin-logs-through-gkill-log.md)。
レベルの語彙と既定 `error` はどちらも本体と同じ。）

## Evidence

実測（2026-08-30、`src/server` の非テスト `slog.Log`）:

| | 修正前 | 修正後 |
|---|---:|---:|
| Debug | 1798 | 1518 |
| Warn | 76 | 218 |
| Error | 76 | 203 |
| メッセージが `"error"` だけ | 703 | 0 |

本番サービスのログファイル: `gkill_access.log` / `gkill_debug.log` / `gkill_error.log` /
`gkill_info.log` / `gkill_trace.log` / `gkill_trace_sql.log` / `gkill_warn.log` / `gkill.log` が
すべて0バイト（MCP の2本のみ 1 / 8 バイト）。

`gkill.log`（統合ログ）は `Init()` が `Mode: SplitOnly` を渡していたため、開いた直後から
1バイトも書かれない状態だった。静的フィールド `{"app":"gkill"}` は
`routingHandler.WithAttrs` が `return h` の空実装だったため、資料に「付く」と書いてあるのに
1度も出力されていなかった。

## Related tests

- `src/server/gkill/main/common/gkill_log/log_level_source_scan_test.go`（無情報メッセージの禁止・defer Close の対象別レベル・握り潰しの検出・許可リストの実在）
- `src/server/gkill/main/common/gkill_log/gkill_log_test.go`（静的フィールドの出力・WithGroup・世代回転・回転無効時の非回転）
- `src/server/gkill/api/gkill_server_api/response_status_log_test.go`（ステータスからレベルが決まること・成功時に出さないこと・エラーコードと経路が載ること）
- `src/server/gkill/main/common/gkill_log/no_eager_sql_format_test.go`（TRACE_SQL の引数を先行評価しないこと）
