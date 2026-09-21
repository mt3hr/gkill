---
name: gkill-cli-ops
description: "gkill の CLI サブコマンドと運用（src/server/gkill/main/）の約束。gkill_server / gkill の2配布形態、update_cache と add_tag が起動中サーバの HTTP クライアントであること、configs 直書きの自己発行セッションとその権限、add_tag のルール JSON（FindKyouQuery）と rep_types_in_sidebar の展開・拒否する指定・消したタグは付け直すこと、clear_cache のユーザー別派生キャッシュ（thumb/video/zip/plugin）、generate_plugin_cache がプラグインを --gkill-build-cache で単独起動して同期構築すること（結果行 built / no_cache の完全一致・旧バイナリの exit 2 と空 stdout は失敗・稼働中サーバの常駐ビルダと同じ cache.db に2つの書き手）、CLI フラグ一覧、RunE + SilenceUsage/SilenceErrors と errors.Join を扱う。src/server/gkill/main/・gkill_options/・local_rep_cache_path.go を編集するとき、サブコマンドを足すとき必読。「削除したタグが復活する」「キャッシュがディスクを埋める」の調査でも必読。"
---

# CLI サブコマンドと運用の約束

対象: `src/server/gkill/main/**` / `src/server/gkill/main/common/gkill_options/**` / `src/server/gkill/dao/reps/local_rep_cache_path.go`

**このファイルは全文が、実際に起きた事故の再発防止である。該当作業では飛ばさずに読むこと。**
多くは「例外もエラーも出さずに静かに壊れる」種類で、破っても目の前ではエラーにならない。

### Two Deployment Modes

- **`gkill_server`** (`src/server/gkill/main/gkill_server/main.go`): Headless HTTP server (browser/PWA)
- **`gkill`** (`src/server/gkill/main/gkill/main.go`): Desktop app wrapping server with go-astilectron

Both use cobra for CLI with shared subcommands: `version`, `dvnf`, `generate_thumb_cache`, `generate_video_cache`, `optimize`, `update_cache`, `clear_cache`, `generate_plugin_cache`, `reset_password`, `add_tag`. `idf` and `mcp` are registered by `gkill_server` only (`main/gkill/main.go` does not register `IDFCmd` / `MCPCmd`). `mcp --kind read|write|readwrite [--transport stdio|http] [--config <path>]` is the MCP server (an HTTP client of the running gkill_server, like `update_cache` / `add_tag`; `main/common/mcp.go` is wiring only, the implementation is package `gkill/mcp` — see [gkill-mcp](../gkill-mcp/SKILL.md)). It replaces the parent's `PersistentPreRun` with its own so that `gkill_log.InitNamed("gkill_mcp_<kind>", "gkill_mcp", ...)` opens `logs/gkill_mcp_<kind>*.log` instead of `gkill*.log`, and never mirrors to stdout (stdout is the JSON-RPC channel in stdio mode). Default listen address: `127.0.0.1:9999` with "local access only" on (loopback-only; the constants live in `dao/server_config/server_config.go`, see [gkill-go-backend](../gkill-go-backend/SKILL.md)), TLS disabled, initial user `admin` with no password. Serving other machines on the LAN is an explicit change in the server settings screen (address + local-only), not a default.

`update_cache` and `add_tag` are **HTTP clients of the running server**, not offline operations — they need gkill_server to be up. Neither can log in with a password (Argon2id), so they mint a short-lived session by writing a row into `configs/account_state.db` directly (`main/common/password_admin.go`: `issueLocalSession` / `issueLocalAdminSession`); the trust basis is being able to write the config dir on the same machine. `update_cache` uses an admin session; `add_tag` uses **the target user's own session**, because `/api/get_kyous` and `/api/add_tag` act as the session's user and an admin session would see admin's repositories. Both resolve the endpoint through `common.ResolveLocalServerEndpoint` (reads `server_config.db`, skips TLS verification for the localhost self-signed cert).

**`add_tag` のルールは検索条件 JSON そのもの。**`add_tag <user_id...> --rules_file <path> | --rule '<json>' [--dry_run]`（2026-09 に `auto_tag` から改名。旧 `--tag_by_rep_prefix` / `--tag_by_rep_name` は撤去）。ルール1件は `{"tag": "<タグ名>", "query": {<FindKyouQuery JSON>}}` で、`--rules_file` はその配列（単一オブジェクトも可）、`--rule` は繰り返し可。`query` はクライアントの `FindKyouQuery` の JSON をそのまま受ける（`main/common/add_tag.go` の `addTagQueryJSON` = `find.FindQuery` の埋め込み + クライアント専用7キー）。**`rep_types_in_sidebar`（画面の「記録種別」= rep 名 `<種別>_<端末>_<日付>` の先頭要素。サーバの `rep_types` とは別物）を書くと、CLI が `/api/get_all_rep_names` の一覧を `rep_to_struct` と同じ規則（`_` で3要素に分かれれば先頭が種別・2つ目が端末、それ以外は名前全体が種別）で分解し、`reps` を上書きしてから送る。**`devices_in_sidebar` 省略＝全端末。実環境の rep は `AutoScreenshot_<端末>_<日付>` の形で増え続けるので、`reps` を列挙する JSON は新しい rep を**静かに取りこぼす**。運用のルールは `rep_types_in_sidebar` で書くこと。端末名に `_` が入る rep は分類されない（UI と同じ）。

復号は厳格で、次の指定は**サーバに触る前に exit 1** にする（理由はどれも「エラーにならず意図と違う範囲に静かに当たる」から）: 未知のキー（`"rep"` の綴り誤りは `reps` 未使用＝全件になる）／絞り込みが1つも無い `query`（`{}`＝全件）／常に0件になる非 null 空配列（`tags: []` `reps: []` `rep_types: []` `ids: []` `period_of_time_week_of_days: []`）／非空の `keywords`・`timeis_keywords`（入力欄の生文字列。`words` / `not_words` を書く）／`update_cache: true`（検索のたびにフルリビルド）／`devices_in_sidebar` 単独／地図3値の一部だけ／`timeis_words` 無しの `timeis_tags`。先頭 BOM は剥がし、旧 `use_*` 形式は `find.MigrateLegacyFindQueryJSON` で受理する。各ルールは展開後の有効な FindQuery を1行 JSON で印字するので、`--dry_run` でそれと `kyous = N` を見てから本番に流す。**UI から貼った JSON は `tags`（チェック済み一覧）・`hide_tags`・`calendar_*` で範囲が狭い**——差分計算は正しいので「target = 0」が静かに出るだけになる。Windows の運用スクリプトは `--rule` を使わない（PowerShell 5.1 はネイティブコマンドへ渡す引数の二重引用符を落とす）。経緯と却下案は [ADR-1002](../../../documents/adr/1002-add-tag-rules-are-find-queries.md)。

「付いているか」の判定は `/api/get_kyous` を2回投げた差分（条件そのまま vs. `tags: [<tag>]` + **`tags_and: true`** を上書き。タグ以外の条件は残るので差分がそのまま未付与分）。`filterTagsKyous` は OR/AND どちらの分岐もタグ名を完全一致（大小無視）で照合する——`gkill` が `gkill_autolog` に当たることはない——ので単一タグの「付いているものだけ」は AND が直接表現。応答の `messages` のうち rep 読み込み失敗・プラグイン失敗の警告は印字する（黙って捨てるとその rep の記録が対象から消えたことに気付けない）。タグ行の ID は画面から付けるときと同じランダムな UUID で、判定はタグ名の差分だけ。だから**画面から消したタグは次回付け直される**（要件は「付いていなければ付ける」。2026-09-11 の判断）。かつては (対象ID, タグ名) の UUIDv5 にして「消したタグは復活しない」を実現していたが、その考慮が不要になったのでやめた——決定的 ID へ戻すと消したタグが二度と付かなくなる（`TestNewAddTagRowUsesFreshUUIDAndAppName` が固定）。裏返しの trade-off として、付与済み判定を取りこぼしたとき（キャッシュが古いまま実行した等）の同名2行を防ぐ保険は無い。運用スクリプトが直前に `update_cache` を回すのはそのため。`CREATE_APP` は `gkill_add_tag`（出所の表示用。判定には使わない。旧 auto_tag 分は `gkill_auto_tag` のまま残る）。

`clear_cache <thumb|video|zip|plugin|all> <all|user_id...>` deletes the on-disk derived caches (`thumb_cache` / `video_cache` / `zip_cache` / `plugin_cache`). The target is required (matching `generate_thumb_cache`/`optimize` etc. which require positional user args): pass the literal `all` to remove the whole cache dirs under `$HOME/gkill/caches/` globally (no user context needed), or one or more user_ids to load each user's repositories (`LoadIDFRepOnly`) and clear only that user's IDF-rep caches via `IDFKyouReps.Clear{Thumb,Video,Zip}Cache(userID)`. `plugin` mode is a plain directory removal (`ClearPluginCache`) and skips the repository load entirely. Missing target or unknown mode prints usage. All three derived caches are keyed per user — `caches/zip_cache/{userID}/{repName}/{sha1(zipPath)}/` and `caches/{thumb,video}_cache/{userID}/{repName}/` — built by `derivedCacheDirForUser` in `dao/reps/local_rep_cache_path.go`. Rep names are NOT unique across users (`filepath.Base(contentDir)`, no UNIQUE constraint), so name matching alone cannot isolate them; that's why all three `Clear*Cache(userID)` take the user id. Thumb/video have no dedicated route — they are only reachable via `/files/{repName}/...?thumb=`. rep名の照合では原理的に守れない理由は [ADR-0705](../../../documents/adr/0705-per-user-derived-cache-dir.md)。

**`generate_plugin_cache <plugin_name|all> <user_id...>` はプラグインを単独モードで直接起動して同期構築する。稼働中サーバには頼らず、stdio プロトコルにもコマンドを足さない。**（`main/common/generate_plugin_cache.go`、2026-09-16）プラグインのキャッシュ（`caches/plugin_cache/{user}/{plugin}/cache.db`）は常駐ビルダが stdio の初回呼び出しの後に非同期で作るので、同期で新しいデータを置いても誰かが画面を開くまで取り込まれず、`update_cache` が作る型別索引も古い cache.db を読む。そこで各プラグインバイナリを `--gkill-plugin-dir <dir> --gkill-user-id <user> --gkill-protocol-version <v> --gkill-build-cache` で起動し（`--gkill-print-manifest` と同じ「stdio ループに入らない起動」の型。SDK 側は `sdk.Run` が `Handler.BuildCache` を同期で1回呼んで終わる）、終わるまで待つ。`plugin_name` は manifest.json の `name`（＝フォルダ名）の完全一致で、無い名前は何も起動せず利用可能な名前を stderr に列挙して exit 1。**利用者の実在確認（account.db）はプラグインの走査より前に置く** —— 走査（`PluginManager.DiscoverPlugins`）は `plugins/{user}/` を MkdirAll するので、打ち間違えた user_id のたびにディレクトリが増える。実行ファイルの解決は `reps.PluginExecutablePath`（Windows の `.exe` 付与）を常駐起動と共有し、CLI 側で再実装しない。**成否は子プロセスの stdout 全体（前後の空白を除く）が `built` か `no_cache` に完全一致するときだけ成功**: exit 0 で stdout が空なのは「フラグを無視して stdio ループに入り stdin EOF で終わった」旧 SDK・古い独自実装のバイナリ、exit 2 は Go の `flag` が未知フラグで落ちた旧 SDK のバイナリで、どちらも失敗として報告する（静かに成功にしない）。SDK を使わないプラグインは自前で結果行を返す必要がある（`gkill_plugin_uguisu` は 2026-09-16 から `no_cache` を返す。配布前の古いバイナリが混ざっている間はその1本だけ失敗行が出る）。子の stdin は繋がない（nil。旧バイナリが EOF で自然に終わりハングしない）、stderr は素通し、stdout だけ捕捉、タイムアウト無し（fitbit の初回構築は実測 155 秒で環境により数倍ぶれる）、プラグインは**逐次**（並列は stderr が混ざり、fitbit の並列パーサが数 GB の ZIP を同時に開く）、1本失敗しても残りを続けて `errors.Join`。`BuildCache` が nil のプラグイン（雛形の `gkill_example`）は `no_cache` で成功扱い（`all` で毎回赤くしない）。**承知の上の制約**: サーバ稼働中に同じプラグインの常駐ビルダが走っていれば同じ WAL の cache.db に2プロセスが書く（`busy_timeout` 6秒で破損はしないが、衝突した側はその周回だけ失敗して次回やり直す。CLI なら exit 1 → 再実行）。サーバ側の型別索引は次の `update_cache`（か TTL 5分）まで新しい cache.db を読まないので、同期スクリプトでは `generate_plugin_cache all <user>` を `update_cache` の**前**に置く。経緯と却下案（HTTP API 追加／stdio に build_cache コマンド／ポーリング／並列／タイムアウト）は [ADR-1003](../../../documents/adr/1003-generate-plugin-cache-runs-plugin-standalone.md)。守るテストは `main/common/generate_plugin_cache_test.go`（偽プラグインをテストバイナリ自身で起動）、`plugin/sdk/build_cache_test.go`（同梱7本の `BuildCache` 配線をソース走査で固定）。

### CLI Flags

| Flag | Default | Description |
|---|---|---|
| `--gkill_home_dir` | `$HOME/gkill` | Home directory for all data/config/logs |
| `--address` | (none) | Override listen address (e.g. `:19999`, `127.0.0.1:19999`). Runtime-only override — the config DB `ADDRESS` value is left untouched, so the settings UI may show a different address than the one actually bound. Resolved via `gkill_options.ResolveServerAddress` / `ServerAddressPortSuffix` |
| `--disable_tls` | `false` | Disable TLS enforcement |
| `--cache_in_memory` | `true` | Cache repository data in memory |
| `--cache_reps_local` | `false` | Cache repositories locally |
| `--goroutine_pool` | `runtime.NumCPU()` | Goroutine pool size |
| `--cache_clear_count_limit` | `3000` | Cache item limit before clearing |
| `--cache_update_duration` | `1m` | Cache refresh interval |
| `--pre_load_users` | (none) | Pre-load specified users' repositories on startup |
| `--log` | (none) | Log level: `none`, `error`, `warn`, `info`, `access`, `debug`, `trace`, `trace_sql`。プラグインの子プロセスへも環境変数 `GKILL_LOG_LEVEL` で継がれる |
| `--log_rotate_max_bytes` | `33554432` (32 MiB) | 1ログファイルの上限。0以下は回転を無効化する |
| `--log_rotate_keep` | `5` | 保持する旧世代数。0以下は旧ファイルを退避せず破棄する |

- `gkill/main/common/` — Shared CLI commands, server initialization, logging
- `gkill/main/common/gkill_options/` — CLI flag definitions and directory structure

**CLIサブコマンドは `RunE` + `SilenceUsage/SilenceErrors`**（指摘 M-8）。失敗で exit 1（main の `log.Fatal` が唯一のエラー出力）。ユーザごとのループは `errors.Join` で集約し、途中失敗でも成功分の結果（reset_password の URL 等）はその場で即出力してから続行する（`os.Exit` を足すと defer のセッション後始末が飛ぶので RunE 経由で返す）。**互換注意**: リポジトリ外の運用スクリプト（同期・配置）が update_cache の失敗を初めて exit code で観測する。`add_tag` は長時間実行向けに `issueLocalSession` の `refresh` でセッション期限を延長する（ルールごとの収集前と、付与の500件ごと。TTL は5分）。ルールの書式誤りは RunE のエラーに加えて stderr にも出す——`log.Fatal` はログファイルにしか書かず、端末には usage と exit 1 しか見えないため。

**ログは統合ファイルとレベル別ファイルへ同時に出し、両方へ同じ回転設定を適用する。**
既定は32 MiB・5世代で、現在のファイルを `.1`、古いものを `.2` 以降へ送る。
Windows は開いたファイルを rename できないため、回転時の Close → rename → reopen の順序を変えないこと。
回転失敗は本体を止めず書き込みを続ける。**プラグインの子プロセスへはレベルと回転設定を環境変数で継ぐ**（`GKILL_LOG_LEVEL` / `GKILL_LOG_ROTATE_MAX_BYTES` / `GKILL_LOG_ROTATE_KEEP`。`gkill_log.Init()` の `ExportEnvForChildProcesses` が書き出し、SDK の `sdk.Run` が `logs/gkill_plugin_<name>*.log` を同じ語彙・同じ回転で開く。`GKILL_HOME` と同じ環境継承なので起動側の `cmd.Env` は触らない。[ADR-0313](../../../documents/adr/0313-plugin-logs-through-gkill-log.md)）。`gkill_log_test.go` が統合・分割の二重出力、静的フィールド、
Windowsでの世代回転、無効化時の非回転を固定する。

**`generate_thumb_cache` / `generate_video_cache` は1件ずつ stat しない。** 生成対象を親ディレクトリごとに `os.ReadDir` して、その列挙結果からキャッシュ名に要るファイルサイズを取る。キャッシュ側もrepごとに1回列挙して名前の集合と突き合わせる（生成済みと `.failed` の印を同じ列挙から2集合で取り、印のある対象は goroutine へ投入しない）。**「キャッシュ名の接頭辞（パスのSHA1）だけ見てサイズを無視する」近似を入れてはいけない** —— 差し替わったファイルのサムネイルが古いまま出続け、しかも画面で見ているぶんには正しいので原因に辿り着けない（[ADR-0212](../../../documents/adr/0212-derived-cache-scan-lists-directories.md)）。ファイル単位の並列化に `threads.Go` を使わないこと（rep単位のファンアウトが既にスロットを持っているので入れ子になる。[ADR-0206](../../../documents/adr/0206-no-nested-threads-go.md)）。

**互換動画へ変換するのは、原本のまま再生できると言い切れないものだけ。** 判定は `videoNeedsCompat`（コンテナ×映像コーデック×画素形式×プロファイル×音声）で、**分からないもの・probe失敗はすべて変換する側に倒す**。クライアントには再生失敗の受け皿が無く、再生できなければエラーも出ずに無音の黒枠になる。緩める向きの変更をするなら受け皿を先に作ること。変換に失敗したら `<キャッシュ名>.failed` を残して次回以降やり直さない（**ctx が切れているときは印を残さない** —— ブラウザが待ちきれずに切っただけのものを恒久的な失敗として焼くと、変換できる動画が二度と変換されなくなる）。印を消すのは `clear_cache video <利用者ID>`（[ADR-0213](../../../documents/adr/0213-transcode-only-what-the-browser-cannot-play.md)）。

**rep 定義のパターン展開は対象ツリーを歩かない。** `GetRepositories` は `**` を含まないパターンを親ディレクトリ1回の列挙で展開する（[ADR-0211](../../../documents/adr/0211-expand-rep-patterns-without-walking.md)）。

**`LoadIDFRepOnly` のときはプラグインを探索しない。** 探索は1プラグインにつき1サブプロセスを起動するので、IDF のrepしか使わない CLI では丸ごと無駄になる。**この分岐は「プラグインのアダプタが `IDFKyouReps` へ入らない」ことに依存している** —— アダプタの登録先を増やすときは必ずここも見ること。破ると、そのプラグインぶんのサムネイルと互換動画がエラーも警告も出ないまま作られなくなる（[ADR-0307](../../../documents/adr/0307-skip-plugin-discovery-when-only-idf-is-needed.md)）。

## 関連スキル

- [gkill-go-backend](../gkill-go-backend/SKILL.md) — 派生キャッシュの構造・rep名で利用者を分離できない理由
- [gkill-build-test](../gkill-build-test/SKILL.md) — ビルドと E2E の実行手順

## 詳しい設計と却下案（ADR）

- [ADR-0705 派生キャッシュはユーザー別ディレクトリ](../../../documents/adr/0705-per-user-derived-cache-dir.md)
- [ADR-0211 パターン展開は木を歩かない](../../../documents/adr/0211-expand-rep-patterns-without-walking.md)
- [ADR-0212 派生キャッシュの走査はディレクトリ列挙](../../../documents/adr/0212-derived-cache-scan-lists-directories.md)
- [ADR-0213 変換するのは再生できないものだけ](../../../documents/adr/0213-transcode-only-what-the-browser-cannot-play.md)
- [ADR-0307 IDFだけ要る経路ではプラグインを探索しない](../../../documents/adr/0307-skip-plugin-discovery-when-only-idf-is-needed.md)
