---
name: gkill-plugin
description: "gkill プラグイン（src/plugins/ の独立バイナリ・plugin/sdk/・本体側 dao/reps/plugin_*.go）の約束。manifest.json の provides と型別/付随データ、PluginTypedIndex から即答してプラグインへ往復しない読み取り、stdio のスロット直列化とプロセス回収条件、常駐ビルダ+WAL、Google Takeout の zip 直読みと CRC32+Size 差分、本文表示 plugin-html-view.vue の iframe 注入手順を扱う。src/plugins/・src/server/gkill/plugin/・api/gkill_plugin/・dao/reps/plugin_*.go・plugin-html-view.vue を触るとき必読。「プラグインの本文がときどき空白」「プロセスが殺され続ける」「歩数が2倍になる」の調査でも必読。"
---

# プラグインの不変条件

対象: `src/plugins/**` / `src/server/gkill/plugin/**` / `src/server/gkill/api/gkill_plugin/**` / `src/server/gkill/dao/reps/plugin_*.go` / `src/client/pages/views/plugin-html-view.vue`

**このファイルは全文が、実際に起きた事故の再発防止である。該当作業では飛ばさずに読むこと。**
多くは「例外もエラーも出さずに静かに壊れる」種類で、破っても目の前ではエラーにならない。

## プラグイン一覧と実測仕様

```
  plugins/    # Standalone plugin binaries (each has its own go.mod, manifest.json, executable)
              #   examples/gkill_example/  — sample plugin (fixed Kyou response)
              #   gkill_plugin_archived_git_commit_log/ — Git repositories archived as zip
              #                              (`name/.git/…`, root `.git/…`, several per zip, nested).
              #                              Reads the zip's central directory only (sdk.OpenSources),
              #                              streams the `.git/**` entries into a go-billy memfs and
              #                              opens it with go-git — never extracts. One Kyou per
              #                              commit with the native shape (ID = hash, rep name =
              #                              repository dir name, times = committer, user = author,
              #                              app = "git"), `provides: ["git_commit_log"]` and
              #                              `data_type: git_commit_log` so the native
              #                              GitCommitLogView draws it and commits that also live
              #                              in a working repo collapse to one row. Declares one
              #                              rep name per repository through `rep_names`.
              #                              Same hash in two zips = one row named after the repo
              #                              whose newest commit is latest (renames follow the new
              #                              name; recomputed every build); fingerprint per repo
              #                              is sha256 of (entry name, CRC32, size). Real data:
              #                              88 zips (97 .git) → 3,447 commits / 78 rep names in ~60s
              #   gkill_plugin_chatgpt/    — ChatGPT conversation history plugin. Reads the export
              #                              ZIP as-is via sdk.OpenSources, never the extracted JSON
              #                              (a loose conversations*.json folder/file is reported as
              #                              `source_problems` on the config screen, not silently 0).
              #                              Per archive `conversations-NNN.json` is preferred, else
              #                              `conversations.json`. Signature = Path:CRC32:Size of the
              #                              entries → whole rebuild on change; the same conversation
              #                              id across zips → the newest update_time wins and the
              #                              other version's messages are dropped (ADR-0310)
              #   gkill_plugin_claudeai/   — Claude.ai conversation history plugin. Same zip rules as
              #                              chatgpt: `conversations-000.zip` holds one
              #                              `conversations.json` (203MB real) whose entry mtime is
              #                              fixed at 1980-01-01, so only CRC32/Size can detect change
              #   gkill_plugin_claudecode/ — Claude Code chat log plugin (one Kyou per human
              #                              message + one per its whole response run,
              #                              source folder configurable, SQLite differential cache)
              #   gkill_plugin_codex/      — OpenAI Codex CLI rollout logs, same Kyou granularity as
              #                              claudecode. Thread id is the FILENAME uuid, never
              #                              `session_meta.session_id` (absent in 23/52 real files and,
              #                              on a sub-agent, holds the PARENT's id). `session_meta`
              #                              repeats 1..13× per file: take identity from the FIRST
              #                              occurrence only (a sub-agent file carries the parent's
              #                              meta as its second) but merge cwd/git/originator across
              #                              all. Conversation comes from the `event_msg` lane only
              #                              (`response_item/message` mixes in injected preambles);
              #                              tool *calls* come from `response_item`, tool *results*
              #                              are never stored — they are 94.7% of the bytes. Kyou IDs
              #                              are UUIDv5 of (thread id, role, ordinal) since event_msg
              #                              records have no id; append-only logs keep them stable.
              #                              One real line is 19.9MB, so `reader.go` classifies from
              #                              the first 512 bytes and drains unwanted lines without
              #                              buffering. Background builder + WAL (4.5s first build /
              #                              <1s incremental on 245MB, 52 files → 301 Kyou)
              #   gkill_plugin_fitbit/     — Google Takeout Fitbit/Google Health, aggregated to
              #                              one KC per (day, metric). 34 metrics, background
              #                              build (~155s first / <1s incremental on a 271MB zip).
              #                              Partials are per (file, day, data source): since
              #                              2025-12 the steps CSV carries the phone's
              #                              `Phone Health Connect` rows beside the watch's on
              #                              the same day, and the Fitbit app's own daily total
              #                              is the watch alone. The fold takes every tracker
              #                              source when one exists, else the first listed
              #                              `secondary_data_sources` (ADR-0312); summing them
              #                              doubled 273 days of steps
              #   gkill_plugin_google_locationhistory/ — Google Takeout location history as
              #                              GPSLog (no Kyou, `emits_kyou: false` so it stays out
              #                              of the rep list). Format detected by content, not path
              # The two Takeout plugins, archived_git_commit_log, chatgpt and claudeai read their
              # zips **as a zip, never extracted** — scan lives in `plugin/sdk/source.go`
              # (`sdk.OpenSources`), shared by all five. Differential test is
              # `(CRC32, Size)`, NOT mtime: Takeout stamps every entry with the same export time, so
              # mtime never moves when content changes. One export = "dir + the takeout timestamp in
              # the zip name", so split `-1-001`/`-1-002` parts merge but a second export dropped in
              # the same folder does not (folder alone would double every summed metric). fitbit
              # keeps an `export` table and folds only the lowest-rank export per (metric, day);
              # locationhistory needs no ranking because its read-time `SELECT DISTINCT (time, lat,
              # lng)` already collapses points across exports.
              # The seven shippable plugins (archived_git_commit_log / chatgpt / claudeai /
              # claudecode / codex / fitbit / locationhistory — gkill_example is
              # excluded) take `source_dirs` in config.json (folders/globs), auto-create
              # that config.json next to manifest.json on first start (existing files are never
              # overwritten, via sdk.EnsureConfig + Handler.DefaultConfig), and can print their
              # embedded manifest.json / default config.json via --gkill-print-manifest /
              # --gkill-print-config, and build their cache synchronously without the stdio loop
              # via --gkill-build-cache (SDK-level; `gkill_server generate_plugin_cache` drives it,
              # see gkill-cli-ops). Their SQLite caches live under gkill's own cache dir
              # ($GKILL_HOME/caches/plugin_cache/{userID}/{pluginName}/cache.db — resolved in
              # plugin/sdk/cache_path.go (sdk.CacheDBPath) from the inherited GKILL_HOME env
              # var, falling back to the plugin folder), so `clear_cache plugin` can wipe them.
              # The 6 older plugins had a byte-identical cache_path.go each; it now lives in the SDK
```

- `gkill/api/gkill_plugin/` — Plugin protocol types: `PluginManifest`, `PluginRequest`, `PluginResponse`, `PluginKyou`, `PluginTypedData`, `PluginGPSLog` (stdio newline-delimited JSON)
- `gkill/plugin/sdk/` — Plugin author SDK. `sdk.Run(sdk.Handler{...})` starts the stdio JSON message loop. Logging goes through `sdk.LogWarn` / `LogError`（stderr + file）and `sdk.LogInfo` / `LogDebug`（file only）— never `os.Stdout`. `Handler` has 10 fields: `FindKyous` (required), `GetKyou`, `GetContentHTML`, `GetConfigHTML`, `PostConfig`, `GetGPSLogs`, `RepName`, `RepNames`, `DefaultConfig`, `BuildCache` (called once, synchronously, outside the stdio loop when the binary is started with `--gkill-build-cache`; nil = "no cache", reported as `no_cache`). Plugins are standalone binaries in `src/plugins/`

**1本のプラグインが複数の rep 名を名乗れる（`rep_names`）。** `get_rep_name` の応答に `rep_names[]` を載せると（SDK は `Handler.RepNames`）、gkill はその名前を `get_all_rep_names`・rep 名の絞り込み（`find_filter.go` Step4）・本文取得の引き当て（`PluginManager.GetPluginByRepName`）・MCP の `get_rep_infos.plugins[]` / `get_plugin_list.rep_names` に使う。**null（欄なし）と `[]` は別**: null は「未対応」で manifest の `rep_name` 1つ、`[]` は「いまは0個」。名前を列挙する側は `GetRepName` を直接見ず **`reps.RepNamesOf`** を通すこと（`GetRepName` は代表名で、MatchReps のキーとログ用）。本体側は `pluginRepositoryImpl.GetRepNames` が **TTL 60秒**でキャッシュし、失敗しても前回値か manifest 名にフォールバックしてエラーにしない（Step4 は fan-out の外で逐次に走るので、失敗を返すと全検索が落ち、毎回取りに行くと全検索が期限ぶん止まる）。申告済みの名前は `warnPluginRepNameMismatchOnce` の対象外。zip の Git リポジトリを束ねる `gkill_plugin_archived_git_commit_log` が最初の利用者（[ADR-0308](../../../documents/adr/0308-plugin-multiple-rep-names.md)）。

**プラグインの型別/付随データ:** `manifest.json` の `provides`（既定は空＝従来どおり）に種別を書くと、そのプラグインの記録が **native と同じ型別リポジトリに載る**。`kc` を宣言して `data_type: "kc"` を返せば `typed_kc` が埋まり Dnote の推移グラフで集計できる。`git_commit_log` を宣言して `data_type: "git_commit_log"` で `typed.git_commit_log{commit_message, addition, deletion}` を返せば `GitCommitLogReps` に載り、クライアントは native の `GitCommitLogView` で描き、MCP は `git_commit_log` payload を組み、**稼働中リポジトリと同じハッシュのコミットは `(ID, data_type, related_time)` の重複除去で1件に畳まれる**（別 data_type にすると2件並ぶ。[ADR-0309](../../../documents/adr/0309-plugin-provides-git-commit-log.md)）。型別アダプタは全部読み取り専用で、`AddXxxInfo` はエラー、`WriteXxxRep` には決して入らない。`tag` を宣言すればタグ一覧（`get_all_tag_names`）に載るので、rykv の既定の絞り込み「タグ無し」から漏れる問題が起きない。`gpslog` は Kyou ではないので専用コマンド `get_gps_logs`（ページング必須）で受け渡す。**`provides` が `gpslog` だけのプラグインには型別索引を作らない**（`PluginManifest.NeedsTypedIndex()`）—— 索引の材料が1件も無いので、作ると `state:"never_built"` / `record_count:0` で永久に固定され「索引が壊れている」と誤読される。GPSの取り込み状況は `get_plugin_list` の別枠 `gps_index`。また `emits_kyou:false` のプラグインの `rep_name` / `data_type` は**どの検索値でもない**ので、「渡せる値」を並べる一覧（`get_all_rep_names` / `get_rep_infos` の `plugins[]` / 既知 data_type 集合）には載せないこと（[ADR-0608](../../../documents/adr/0608-plugin-role-is-emits-kyou-and-provides.md)）。

アダプタ（`dao/reps/plugin_typed_adapters.go` / `plugin_attached_adapters.go` / `gps_log_repository_plugin_impl.go`）の**読み取りは決してプラグインへ往復しない**。`PluginTypedIndex`（`plugin_typed_index.go`）が `find_kyous` 1回ぶんの不変スナップショットを持ち、そこから即答する。プラグイン呼び出しは容量1のスロットで直列化されるので、1件ずつ聞きに行くと一覧の行数ぶんの直列 stdio 呼び出しになりプロセスが殺され続ける。`GetLatestDataRepositoryAddress` は**型別は空・付随は実データ**（型別が返すと `UpdateTime` の揺れでレコードごと消え、付随が返さないと `--cache_in_memory=false` でタグが全部落ちる）。アダプタの `Close` は no-op（プロセスを閉じるのは本体と `PluginManager.CloseAll` だけ）。登録は `gkill_dao_manager.go` の `KCReps`→`Reps` コピーループより**後**（先だと二重検索になる）。詳細は `documents/reverse/plugin-system.md` の14章、却下案は [ADR-0302](../../../documents/adr/0302-plugin-provides-typed-index.md)。

本体側の受け口（`gkill/dao/reps/`）のプロセス管理:

`plugin_repository_impl.go` manages plugin subprocess lifecycle (start, slot-guarded stdio, one persistent reader goroutine per process, response-ID matching, auto-restart on crash). **呼び出し元のキャンセル（HTTPクライアントの切断）ではプロセスを回収しない**（フロントは全リクエストに `AbortController` を張っているため）。回収するのは gkill 自身のデッドライン超過時だけ。直列化は mutex ではなく容量1のチャネル（`callSlot`）で行い、**期限はスロットを取ってから張る**。順番待ちの上限は別枠（`maxPluginQueueWait` 既定10秒）で、待ちきれなければ `ErrPluginBusy` を返すだけでプロセスには手を出さない。プラグイン rep は `Repositories` の fan-out でスレッドプールのスロットを取らない（`goForRep`）ので、プラグインのロック待ちで検索全体が止まることはない

**プラグインの重い構築は常駐ビルダ + WAL + バッチcommit**（2026-08-21、監査 M-6）。claudecode も codex/fitbit と同じく `builder.go`（`EnsureStarted`/`Kick`/`loop`、mu(DB初期化)/buildMu(構築)分離で読み取り無待機、WAL 自前DSN、`cache_meta` に進捗）へ移行済み。`GetMessages`/`GetMessage`/`GetStats` は refresh を呼ばず現キャッシュ即答+Kick。同期・単一tx構築（デッドラインkill→進捗ゼロループ）を新規に書かないこと。 却下案（同期構築／ロック共有／デッドライン延長）と実測は [ADR-0305](../../../documents/adr/0305-plugin-background-builder-wal.md)。**同期構築を書いてよい唯一の場所は `Handler.BuildCache`**（`--gkill-build-cache` の単独モードで、stdio ループの外。ハンドラ期限が無い）。同梱7本は常駐ビルダが呼ぶのと同じ構築関数（`buildOnce` / `build` / `refresh`）をそこから呼ぶだけで、`EnsureStarted` / `Kick` は起こさない。配線の欠落は `plugin/sdk/build_cache_test.go` のソース走査が落とす（欠けると `generate_plugin_cache all` でそのプラグインだけ `no_cache` になり、エラーも出ない）。

**プラグインのログは `sdk.LogXxx` を通す。** `sdk.Run` が起動時（flag 解析の直後）に `$GKILL_HOME/logs/gkill_plugin_<name>.log`（統合）と `gkill_plugin_<name>_{error,warn,info,access,debug,trace,trace_sql}.log` を開く（`<name>` は `--gkill-plugin-dir` の末尾＝manifest の `name`。既に `gkill_plugin_` で始まる名前は二重にしない。静的フィールドは `app=gkill_plugin` / `plugin` / `user_id` / `pid`。`plugin/sdk/plugin_log.go`）。レベルと回転は本体の `--log` / `--log_rotate_*` を **環境変数 `GKILL_LOG_LEVEL` / `GKILL_LOG_ROTATE_MAX_BYTES` / `GKILL_LOG_ROTATE_KEEP`** で継ぐ（`gkill_log.Init()` が書き出し、子は環境継承で受ける。`GKILL_HOME` と同じ経路なので起動側の `cmd.Env` は nil のまま。フラグにしないのは、SDK を使わない第三者バイナリが未知フラグで exit 2 になるため）。`LogWarn` / `LogError` は **stderr の `WARN: ` / `ERROR: ` 行をこれまでどおり出した上で**ファイルにも書き、`LogInfo` / `LogDebug` はファイルにだけ書く（stderr のリングは 4KB しか無いので節目で肝心のエラーを押し出さない）。SDK 自身は `plugin start` / `plugin stop` / `build cache` を Info、**1コマンド1行**（command / id / duration_ms / count / error）を Access で残す——「プロセスが殺され続ける」の調査で、どのコマンドが何ミリ秒かかったかはここにしか出ない。壊れた環境変数は既定（error / 32MiB / 5）へ倒して stderr に1行、ログ dir が作れない・home が分からない手起動は stderr だけで続行し、**ログの都合でプラグインを止めない**（panic も exit もしない）。同じプラグインを2プロセス（別利用者、常駐 + `generate_plugin_cache`）が同じファイルへ書きうるので、32MiB の回転は Windows では他方が閉じるまで失敗してファイルが育つ（承知の上。`pid` で出所は分かる）。`LogDebug` の定義は1行関数のままにする（`log_level_source_scan_test` の握り潰し検査が複数行の定義を誤検知する）。設計と却下案は [ADR-0313](../../../documents/adr/0313-plugin-logs-through-gkill-log.md)。

**stderr の診断は人間が読む前提で書く。** gkill はプラグインプロセスの stderr 末尾4KBをリングに保持し、`get_plugin_list` の `last_error` に載せる（`plugin_stderr_ring.go`。「is_alive=true なのに0件」の最初の手がかり。全文はログファイル側）。**プラグイン側に「実パスを書くな」の制約は課さない** —— 読み取り元やホームは診断に要る（ホームが `systemprofile` へ化ける事故はそれでしか分からない）。代わりに出口で伏せる: リングには生のまま入れ、`handle_get_plugin_list.go` が `message.RedactEnvironmentSpecific` を通してユーザー名を伏せ、MCP は中身自体を返さず `has_last_error` だけを返す（[ADR-0707](../../../documents/adr/0707-redact-environment-specific-strings.md)）。**stdout はプロトコルのチャネルなので、診断を書くのは必ず stderr かログファイル側**（`os.Stdout` に1バイトでも書くと NDJSON が壊れる）。

**プラグイン本文（`plugin-html-view.vue`）の iframe 越しの受け渡し**（2026-08-15 に「Ryuu の PluginKyou がときどき空白」「本文をダブルクリックしても KyouDialog が開かない」を潰して確立）。本文の入れ方は `is_list_view = typeof props.height === 'number'` で分岐し、一覧は `srcdoc` 直書き、Ryuu(`'fit-content'`)・rykv 詳細ペイン(`'auto'`)・KyouDialog(`'unset'`) は定数ローダーへの postMessage 注入（ダイアログの `pushState` より後に iframe がナビゲートするとブラウザバックが2回要るため、ローダー方式は外せない）。注入は**ローダーが `gkill_plugin_loader_ready` を名乗ってから**行う ―― `iframe.contentWindow` は about:blank の時点から真なので、それを見て先に送るとリスナー未登録の iframe に届いて黙って消え、`sent_html` のせいで送り直さないぶん本文が二度と入らない（ServiceWorker が `get_plugin_content_html` をキャッシュ優先で返すので、2回目以降の表示ほど負けやすい＝「ときどき」）。ready のたびに `sent_html` を落として送り直し、Kyou が変わったら iframe ごと作り直す（`:key`。`document.open()` はローダーが `window` に張ったリスナーごと捨てるので使い回すと2件目が入らない）。**`@load` で `sent_html` を落としてはいけない**（`document.close()` でも `load` は発火しうるので注入ループになる）。iframe 内のダブルクリックは親のDOMへ伝播しないので、本文HTMLの末尾に転送スクリプトを足して `gkill_iframe_dblclick` を受け、**親が本物の `dblclick` を撃ち直す**（新しい emit 経路を作らないので `kyou-view.vue` と `ryuu-item-view.vue` の既存の `@dblclick` がそのまま拾う）。テーマ通知を最初のサイズ通知だけに限るのは、本文側がテーマ受信で測り直して送り返すため（毎回返すと10ms周期のピンポンになる）。`plugin-html-view.test.ts` が守る。

**ワード検索の判定はプラグインが唯一の判定者。SDK の `Query.MatchText` を使い、自前のループを書かない。** gkill 本体はプラグインが返した Kyou の本文を持たないので、`words` / `not_words` を再判定しない（`plugin_repository_impl.go` の `pluginKyouMatchesQuery` は Calendar と IDs だけ）。判定規則は本体の各 rep と同じで、正本は `api/find_word`（大小無視の部分一致、肯定語は「text に含む OR ID が語で始まる」、除外語は text だけ、空語は無視）。2026-09-12 まで SDK に判定が無く、4プラグインが同じループをコピペし、fitbit は空文字の語で結果が変わり、雛形の `gkill_example` はワードを完全に無視していた（雛形を写した第三者プラグインも無視する）。**照合に掛けるテキストはプラグインが決める**（chatgpt / claudeai は本文＋会話タイトル、claudecode / codex は `search_text`＋セッション名、fitbit は指標名・キー・単位・数値・デバイス・日付）。一覧をループで判定するときは `q.Matcher()` を1回作って回す。**単語で絞るときは SQL の LIMIT を押し込まない**（`matcher.HasWords()` で切り替える。絞る前に切ると後段で落ちたぶん取りこぼす。codex / fitbit の `QueryKyous` / `FindKyous`）。**型別アダプタ（`plugin_typed_adapters.go`）は索引の型別データを native と同じ列で判定する**（KC=TITLE+NUM_VALUE、Mi=TITLE+BOARD_NAME、Lantana=MOOD 等）ので、`rep_types` 指定や Mi 画面ではプラグインが独自に照合対象へ足した語（fitbit の "Fitbit"）は当たらない。索引にプラグインの照合テキストを持たせる案は却下（[ADR-0113](../../../documents/adr/0113-word-filter-columns-and-id-prefix.md)）。守るテストは `plugin/sdk/match_words_test.go`、各プラグインの `find_kyous_test.go`、`dao/reps/plugin_typed_adapters_test.go` の `TestPluginTypedAdapter_FindKyousFiltersByWord`。

**`get_plugin_list` の `rep_names` は Kyou を出すプラグインでは常に載せ、値は「`query.reps` に渡せる rep 名」にする。** 申告があればその値（索引未構築なら `[]`）、申告しない・取得に失敗したプラグインは `[manifest の rep_name]`、`emits_kyou:false` は `null`。2026-09-19 まで申告があるときだけ載せる `omitempty` で、zip の Git リポジトリを束ねるプラグインの索引が未構築だとキーごと消え、応答には検索に使えない名札 `"ArchivedGit"` しか見えなかった（2026-09-18 の実利用報告）。manifest の `rep_name` を `query.reps` に渡すと Go の `pluginManifestRepNameHint` が「名札であって検索値ではない。`rep_names[]` を使え」と名指しする（[ADR-0311](../../../documents/adr/0311-plugin-list-rep-names-are-always-the-query-values.md)）。**manifest の `description` は実装の粒度と合わせること** —— `gkill_plugin_claudecode` は「1ターン=1Kyou」と書いたまま実装は「1発言=1Kyou（人間の発言と応答で2件）」になっていた（2026-09-19 に文言を直した。`data_type` の `claude_code_turn` は改名しない）。

## 関連スキル

- [gkill-go-backend](../gkill-go-backend/SKILL.md) — `len(XxxReps) == 1` 判定禁止（provides プラグイン1つで長さが2になる）と検索フィルタ
- [gkill-find-query](../gkill-find-query/SKILL.md) — ワード検索の照合規則（SQL / Go / SDK の3実装を揃える）
- [gkill-mcp](../gkill-mcp/SKILL.md) — MCP からのプラグイン本文取得（`include_plugin_content`、同一プラグインへ並列に投げない）
- [gkill-client-foundation](../gkill-client-foundation/SKILL.md) — クライアント全域の規約（plugin-html-view の周辺）

## 詳しい設計と却下案（ADR）

- [ADR-0301 キャンセルとプロセス回収の切り分け](../../../documents/adr/0301-plugin-cancel-vs-kill.md)
- [ADR-0302 provides と PluginTypedIndex](../../../documents/adr/0302-plugin-provides-typed-index.md)
- [ADR-0303 差分判定は CRC32 と Size](../../../documents/adr/0303-plugin-cache-use-crc32-and-size.md)
- [ADR-0304 GPSLog 専用プラグインは Rep に出さない](../../../documents/adr/0304-plugin-emits-kyou-false.md)
- [ADR-0305 常駐ビルダと WAL](../../../documents/adr/0305-plugin-background-builder-wal.md)
- [ADR-0306 Codex のスレッドIDはファイル名から](../../../documents/adr/0306-codex-thread-id-from-filename.md)
- [ADR-0308 プラグインが複数の rep 名を名乗る](../../../documents/adr/0308-plugin-multiple-rep-names.md)
- [ADR-0309 provides に git_commit_log](../../../documents/adr/0309-plugin-provides-git-commit-log.md)
- [ADR-0310 ChatGPT / Claude.ai はエクスポート ZIP のまま読み、展開済み JSON は読まない](../../../documents/adr/0310-chat-export-plugins-read-zip.md)
- [ADR-0312 fitbit は同じ日に並ぶデータソースを合算せず、時計を優先して1系統だけ採る](../../../documents/adr/0312-fitbit-one-data-source-per-day.md)
- [ADR-0313 プラグインのログは gkill_log の別名ファイルへ出し、stderr には WARN / ERROR の接頭辞行だけを残す](../../../documents/adr/0313-plugin-logs-through-gkill-log.md)
- [ADR-0113 ワード検索の型別の対象列と ID の前方一致（プラグインは SDK の判定を使う）](../../../documents/adr/0113-word-filter-columns-and-id-prefix.md)
- [ADR-0707 端末固有の文字列は出口で伏せる](../../../documents/adr/0707-redact-environment-specific-strings.md)
