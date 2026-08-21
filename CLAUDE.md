# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

gkill is a lifelogging application that records and reviews life events (memos, bookmarks, tasks, mood values, expenses, timestamps, files, git commits, etc.). Standalone client-server app: Go backend + Vue 3 frontend, compiled into a single binary with the frontend embedded via `//go:embed`. Licensed under MIT.

## Build & Development Commands

All commands are npm scripts defined in `package.json`. No CGO required (pure Go SQLite driver).

| Command | Purpose |
|---|---|
| `npm run dev` | Start Vite dev server (frontend only). `-- --api=<url>` selects the backend |
| `npm run build` | Build frontend (`vue-tsc --build` + `vite build` in parallel) |
| `npm run lint` | ESLint with auto-fix (flat config, `eslint.config.js`) |
| `npm run install_server` | Full build: frontend → embed → `go install` (headless HTTP server) |
| `npm run install_app` | Full build: frontend → embed → `go install` (desktop app with go-astilectron window) |
| `npm run go_install` | Go install only (skip frontend rebuild) |
| `npm run go_mod` | Regenerate `go.mod` and `go.sum` from scratch |
| `npm test` | Run all tests (build + docs verification + server + client + MCP + plugins + Android + Wear OS) |
| `npm run test_plugins` | Go tests for each standalone plugin module under `src/plugins/` |
| `npm run vet_plugins` | `go vet` for the same plugin modules (CI runs it before `test_plugins`; `npm test` does not) |
| `npm run verify_docs` | Docs CI: checks doc counts against code, cross-links, referenced paths, Mermaid blocks, manual freshness. `--list` prints the measured metrics |
| `npm run test_server` | Go tests (`cd src/server && go test ./...`) |
| `npm run test_client_unit` | Vitest unit tests |
| `npm run test_client_e2e` | Playwright E2E tests (gkill_server + Vite を空きポートで自動起動・停止、`$HOME/gkill_test`使用) |
| `npm run test_mcp` | MCP server tests (Vitest) |
| `npm run release` | Cross-compile release for all platforms |

**Build pipeline** (`install_server` → `prepare_install`): `clean_dist` → clean embed dir → write `version.json` (commit hash + build time + version) → `vue-tsc` type-check → `vite build` → copy `dist/` to `src/server/gkill/api/embed/html/` → copy `src/locales/` to `src/server/gkill/api/embed/i18n/locales/` → `build_manuals` (`src/tools/build_manuals.mjs`: `resources/manual_src/` → `resources/manual/`) → copy `resources/manual/` to `src/server/gkill/api/embed/manual/` (3 copies: html + i18n + manual) → `go install`

**Bare `go build` caveat:** any gkill binary panics at startup if `embed/i18n/locales` is empty (`gkill/api`'s `init()` reads it), so a `go build` without a preceding `npm run copy_i18n_to_app_embed` produces a binary that compiles but dies immediately.

**Prerequisites:** Go 1.26.6+ (`src/server/go.mod` declares `go 1.26.6`), Node.js 20.19+ (24.x recommended — `package.json` has no `engines` field, so this is not enforced), `npm i`

**Dev server backend target:** `npm run dev` runs `src/tools/dev.mjs`, a thin wrapper that strips `--api` / `--api-target` (Vite errors on unknown CLI options) and passes the value to Vite as `GKILL_API_PROXY_TARGET`; all other args pass through untouched. Accepts a full URL, a bare port (`--api=19999` → `http://127.0.0.1:19999`), or `host:port`. Precedence: `--api` > `GKILL_API_PROXY_TARGET` > `http://localhost:9999`. `vite.config.ts` proxies `/api`, `/files`, `/zip_cache` and `/resources/manual` to that target — SPA routes (`/rykv`, `/kftl`, `/mi`, `/rudbeckia`, …) are deliberately not proxied since vue-router owns them in dev.

**E2E test environment:** `$HOME/gkill_test` as dedicated gkill home. `npm run test_client_e2e` automatically cleans, starts gkill_server (fresh admin, no password) and a Vite dev server, runs Playwright, and stops both. See `src/client/__tests__/e2e/run-e2e.mjs`.

**Ports are never fixed in tests.** `run-e2e.mjs` allocates free ports from the OS (`src/client/__tests__/e2e/free-port.mjs`) and passes them as `--address 127.0.0.1:<port>` (gkill_server) and `--port <port> --strictPort` (Vite), so a production gkill_server occupying `:9999` never collides. The ports reach the tests via `GKILL_E2E_BASE_URL` / `GKILL_E2E_VITE_URL` (read by `playwright.config.ts`, `check-server.ts`, `auth.setup.ts`), and Vite's `/api` proxy target via `GKILL_API_PROXY_TARGET` (`vite.config.ts`) — which also prevents E2E writes from ever reaching a production server. Leftover-process cleanup only kills `gkill_server` processes whose command line contains `gkill_test`. Nothing else in `npm test` binds a fixed port: Go tests use `httptest` (ephemeral), and MCP/Android/Wear OS tests bind none.

**Test details:** See `documents/reverse/testing-guide.md`, `src/ABOUT_TEST.md` for per-directory test specs.

## Architecture

### Source Directory Structure

```
src/
  client/     # Vue 3 + TypeScript frontend (App.vue, main.ts, classes/, pages/)
  server/     # Go backend (go.mod at src/server/, packages under gkill/)
  mcp/        # MCP server (read/write/readwrite, stdio + HTTP OAuth 2.1)
  android/    # Android APK wrapper (WebView + bundled gkill_server binary)
  wear_os/    # Wear OS project (phone_companion/ + watch_app/)
  locales/    # i18n JSON files (ja, en, zh, ko, es, fr, de) — shared by frontend & backend
  tools/      # Utility scripts
  plugins/    # Standalone plugin binaries (each has its own go.mod, manifest.json, executable)
              #   examples/gkill_example/  — sample plugin (fixed Kyou response)
              #   gkill_plugin_chatgpt/    — ChatGPT conversation history plugin
              #   gkill_plugin_claudeai/   — Claude.ai conversation history plugin
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
              #                              build (~155s first / <1s incremental on a 271MB zip)
              #   gkill_plugin_google_locationhistory/ — Google Takeout location history as
              #                              GPSLog (no Kyou, `emits_kyou: false` so it stays out
              #                              of the rep list). Format detected by content, not path
              # The two Takeout plugins read the export **as a zip, never extracted** — scan lives
              # in `plugin/sdk/source.go` (`sdk.OpenSources`), shared by both. Differential test is
              # `(CRC32, Size)`, NOT mtime: Takeout stamps every entry with the same export time, so
              # mtime never moves when content changes. One export = "dir + the takeout timestamp in
              # the zip name", so split `-1-001`/`-1-002` parts merge but a second export dropped in
              # the same folder does not (folder alone would double every summed metric). fitbit
              # keeps an `export` table and folds only the lowest-rank export per (metric, day);
              # locationhistory needs no ranking because its read-time `SELECT DISTINCT (time, lat,
              # lng)` already collapses points across exports.
              # The six shippable plugins (chatgpt / claudeai / claudecode / codex / fitbit /
              # locationhistory — gkill_example is
              # excluded) take `source_dirs` in config.json (folders/globs), auto-create
              # that config.json next to manifest.json on first start (existing files are never
              # overwritten, via sdk.EnsureConfig + Handler.DefaultConfig), and can print their
              # embedded manifest.json / default config.json via --gkill-print-manifest /
              # --gkill-print-config. Their SQLite caches live under gkill's own cache dir
              # ($GKILL_HOME/caches/plugin_cache/{userID}/{pluginName}/cache.db — resolved in
              # plugin/sdk/cache_path.go (sdk.CacheDBPath) from the inherited GKILL_HOME env
              # var, falling back to the plugin folder), so `clear_cache plugin` can wipe them.
              # The 6 plugins had a byte-identical cache_path.go each; it now lives in the SDK
```

### Two Deployment Modes

- **`gkill_server`** (`src/server/gkill/main/gkill_server/main.go`): Headless HTTP server (browser/PWA)
- **`gkill`** (`src/server/gkill/main/gkill/main.go`): Desktop app wrapping server with go-astilectron

Both use cobra for CLI with shared subcommands: `version`, `dvnf`, `generate_thumb_cache`, `generate_video_cache`, `optimize`, `update_cache`, `clear_cache`, `reset_password`, `auto_tag`. `idf` is registered by `gkill_server` only (`main/gkill/main.go` does not register `IDFCmd`). Default listen address: `:9999`, TLS disabled, initial user `admin` with no password.

`update_cache` and `auto_tag` are **HTTP clients of the running server**, not offline operations — they need gkill_server to be up. Neither can log in with a password (Argon2id), so they mint a short-lived session by writing a row into `configs/account_state.db` directly (`main/common/password_admin.go`: `issueLocalSession` / `issueLocalAdminSession`); the trust basis is being able to write the config dir on the same machine. `update_cache` uses an admin session; `auto_tag` uses **the target user's own session**, because `/api/get_kyous` and `/api/add_tag` act as the session's user and an admin session would see admin's repositories. Both resolve the endpoint through `common.ResolveLocalServerEndpoint` (reads `server_config.db`, skips TLS verification for the localhost self-signed cert).

`auto_tag <user_id...>` adds tags by repository rule — `--tag_by_rep_prefix '<rep name prefix>=<tag>'` and `--tag_by_rep_name '<rep type>'` (tag = the rep's own name), both repeatable, plus `--dry_run`. It decides "already tagged?" by diffing two `/api/get_kyous` calls (unfiltered vs. `tags` non-null + **`tags_and: true`**). Both the OR and AND branches of `filterTagsKyous` match tag names exactly (case-insensitive) — `gkill` never matches `gkill_autolog` — so either would work; `tags_and` is kept because a single-tag "has this tag" query is what AND expresses directly. Tag IDs are a UUIDv5 of (target id, tag name), so reruns never duplicate: the server rejects an existing ID with `ERR000056` `AlreadyExistTagError`, which the command counts as a skip. That also means a **manually deleted tag is never resurrected**, since the deleted row keeps the same ID.

`clear_cache <thumb|video|zip|plugin|all> <all|user_id...>` deletes the on-disk derived caches (`thumb_cache` / `video_cache` / `zip_cache` / `plugin_cache`). The target is required (matching `generate_thumb_cache`/`optimize` etc. which require positional user args): pass the literal `all` to remove the whole cache dirs under `$HOME/gkill/caches/` globally (no user context needed), or one or more user_ids to load each user's repositories (`LoadIDFRepOnly`) and clear only that user's IDF-rep caches via `IDFKyouReps.Clear{Thumb,Video,Zip}Cache(userID)`. `plugin` mode is a plain directory removal (`ClearPluginCache`) and skips the repository load entirely. Missing target or unknown mode prints usage. All three derived caches are keyed per user — `caches/zip_cache/{userID}/{repName}/{sha1(zipPath)}/` and `caches/{thumb,video}_cache/{userID}/{repName}/` — built by `derivedCacheDirForUser` in `dao/reps/local_rep_cache_path.go`. Rep names are NOT unique across users (`filepath.Base(contentDir)`, no UNIQUE constraint), so name matching alone cannot isolate them; that's why all three `Clear*Cache(userID)` take the user id. Thumb/video have no dedicated route — they are only reachable via `/files/{repName}/...?thumb=`.

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
| `--log` | (none) | Log level: `none`, `error`, `warn`, `info`, `access`, `debug`, `trace`, `trace_sql` |

### Backend (Go) — `src/server/`

Module: `github.com/mt3hr/gkill/src/server` (Go 1.26.6)

Key packages:
- `gkill/api/` — Shared infrastructure: `embed.go` (`//go:embed` serves Vue SPA at `/`), `version.go`, `gkill_version_data.go`, `find_filter.go`, `find_filter_helpers.go`, `find_kyou_context.go`
- `gkill/api/gkill_server_api/` — HTTP API handlers (101 files incl. tests, 1 handler per file). `GkillServerAPI` struct with `serve.go`, `close.go`, route definitions in `gkill_server_api_address.go`. Auth middleware (`auth.go`, `auth_context.go`, `auth_middleware.go`) extracts session→account→device→repositories via `AuthContext`, `authMiddleware`, `authWithReposMiddleware`. Handler registration uses wrapper functions: `wrapNoAuth` (no session), `wrapAuth` (session + account), `wrapAuthRepos` (session + account + device + repositories). Utility files: `filter_local_only.go`, `utils.go`, `web_push.go`. ZIP browsing: `handle_browse_zip_contents.go` (path traversal prevention, Shift_JIS→UTF-8, singleflight dedup).
- `gkill/api/req_res/` — Request/response structs for every endpoint (186 files)
- `gkill/api/kftl/` — KFTL custom text format parser (single package, no sub-packages). Supports both Japanese (。！？、ーー etc.) and ASCII (#!?,-- ~~ /mi /mood /expense /num /url /start /end /timeis /end? /endt /endt?) prefixes
- `gkill/api/gkill_plugin/` — Plugin protocol types: `PluginManifest`, `PluginRequest`, `PluginResponse`, `PluginKyou`, `PluginTypedData`, `PluginGPSLog` (stdio newline-delimited JSON)

**プラグインの型別/付随データ:** `manifest.json` の `provides`（既定は空＝従来どおり）に種別を書くと、そのプラグインの記録が **native と同じ型別リポジトリに載る**。`kc` を宣言して `data_type: "kc"` を返せば `typed_kc` が埋まり Dnote の推移グラフで集計できる。`tag` を宣言すればタグ一覧（`get_all_tag_names`）に載るので、rykv の既定の絞り込み「タグ無し」から漏れる問題が起きない。`gpslog` は Kyou ではないので専用コマンド `get_gps_logs`（ページング必須）で受け渡す。

アダプタ（`dao/reps/plugin_typed_adapters.go` / `plugin_attached_adapters.go` / `gps_log_repository_plugin_impl.go`）の**読み取りは決してプラグインへ往復しない**。`PluginTypedIndex`（`plugin_typed_index.go`）が `find_kyous` 1回ぶんの不変スナップショットを持ち、そこから即答する。プラグイン呼び出しは容量1のスロットで直列化されるので、1件ずつ聞きに行くと一覧の行数ぶんの直列 stdio 呼び出しになりプロセスが殺され続ける。`GetLatestDataRepositoryAddress` は**型別は空・付随は実データ**（型別が返すと `UpdateTime` の揺れでレコードごと消え、付随が返さないと `--cache_in_memory=false` でタグが全部落ちる）。アダプタの `Close` は no-op（プロセスを閉じるのは本体と `PluginManager.CloseAll` だけ）。登録は `gkill_dao_manager.go` の `KCReps`→`Reps` コピーループより**後**（先だと二重検索になる）。詳細は `documents/reverse/plugin-system.md` の14章。

**`len(XxxReps) == 1` でキャッシュrepを判定してはいけない。** アダプタの append は「キャッシュrepで `XxxReps` を1個に差し替える」処理より後なので、`provides` を持つプラグインが1つ入るだけで長さが2になる。書き込み後のキャッシュ反映は構築時に控えた `GkillRepositories.CachedReps` を見る `repositories.WriteThroughXxxCache(ctx, ...)` を使うこと（54箇所）。読み取りはキャッシュrepしか見ず下層repへフォールバックしないので、反映を飛ばすと追加したタグが次の `UpdateCache` まで最大1分見えず、その間にPWAが古い応答をキャッシュし直すと**更新IDの再通知が来ないぶん恒久的に古いまま焼き付く**。再発は `usecase/write_through_cache_test.go` の `TestNoRepsCountCacheGuard` がソース走査で落とす。
- `gkill/plugin/sdk/` — Plugin author SDK. `sdk.Run(sdk.Handler{...})` starts the stdio JSON message loop. `Handler` has 8 fields: `FindKyous` (required), `GetKyou`, `GetContentHTML`, `GetConfigHTML`, `PostConfig`, `GetGPSLogs`, `RepName`, `DefaultConfig`. Plugins are standalone binaries in `src/plugins/`
- `gkill/usecase/` — HTTP-independent business logic (17 files). Extracted from handlers to enable reuse without HTTP context. Functions operate on DAO/repository types directly.
- `gkill/dao/` — Data access layer with `GkillDAOManager` managing SQLite3 databases
- `gkill/dao/reps/` — Repository interfaces and implementations for each data type. `plugin_repository_impl.go` manages plugin subprocess lifecycle (start, slot-guarded stdio, one persistent reader goroutine per process, response-ID matching, auto-restart on crash). **呼び出し元のキャンセル（HTTPクライアントの切断）ではプロセスを回収しない**（フロントは全リクエストに `AbortController` を張っているため）。回収するのは gkill 自身のデッドライン超過時だけ。直列化は mutex ではなく容量1のチャネル（`callSlot`）で行い、**期限はスロットを取ってから張る**。順番待ちの上限は別枠（`maxPluginQueueWait` 既定10秒）で、待ちきれなければ `ErrPluginBusy` を返すだけでプロセスには手を出さない。プラグイン rep は `Repositories` の fan-out でスレッドプールのスロットを取らない（`goForRep`）ので、プラグインのロック待ちで検索全体が止まることはない
- `gkill/main/common/` — Shared CLI commands, server initialization, logging
- `gkill/main/common/gkill_options/` — CLI flag definitions and directory structure

**Key dependencies:** gorilla/mux, modernc.org/sqlite (pure Go), spf13/cobra+viper, asticode/go-astilectron, SherClockHolmes/webpush-go, go-git/go-git, twpayne/go-gpx, nicksnyder/go-i18n/v2, google/uuid

**Doc comments:** `dao/reps/*_repository.go` のインターフェースメソッドは **457/457 documented**、`gkill_server_api` の HandleXxx は **92/92 handlers documented**。どちらも verify_docs が網羅率を機械検査するので、メソッドやハンドラを追加したら doc コメントも書くこと。共通契約の完全文は基底 `dao/reps/repository.go` に1回だけ書き、型別ファイルは「契約は Repository.Xxx を参照。」の1行参照＋差分のみにする（457箇所へ複製するとドリフト源になるため）。ハンドラは「1行説明 / 空行 / パス・HTTPメソッド・認証区分 / req_res 型」の形式。

**Repository pattern:** SQLite-backed data types have 4 implementation layers: `*_repository.go` (interface) → `*_repository_sqlite3_impl.go` → `*_repository_cached_sqlite3_impl.go` → `*_repository_temp_sqlite3_impl.go`. Exceptions: `git_commit_log` (`_local_dir_impl` + `_cached_sqlite3_impl`), `gps_log` (`_gpx_dir_impl`), and `plugin` (`plugin_repository_impl`) read external sources directly and do not have all 4 layers.

**`FindQuery.IDs` は分割して渡す（SQLのバインド変数には上限がある）:** IDリストは各repのSQLへ `ID IN (?, ?, ...)` として展開される。このとき **Mi / MiReKyou は5射影のUNIONで、5本それぞれに同じリストを丸ごと展開する**ので、バインド変数は `5N+5` になり、SQLite の上限（`SQLITE_MAX_VARIABLE_NUMBER` = 32766）を **N=6553 で超える**（実測: 6552は成功、6553で `too many SQL variables`）。分割は `reps` 側の入口に置いてあるので、**新しくIDリストを渡す検索を書くときは `findChunkedByIDs`（`repositories.go`）を通すこと**。今かかっているのは `Repositories.findKyous` / `MiRepositories.FindMi` / `MiReKyouRepositories.FindMiReKyou` と、`GkillRepositories.FindTags` / `FindTexts` の中で最新版アドレスからIDリストを作っている箇所。IDリストはORの羅列なので、分割しても結果は変わらない（塊どうしでIDが重ならないので重複も出ない）。上限は `maxIDsPerFindQuery`（4000）。**この壊れ方は静かなので気付きにくい** ―― 失敗しても `GkillError` が立たず、`/api/get_kyous` も `/api/get_kyous_mcp` も HTTP 200 + `errors: null` + 0件で返っていた（内部の `err` は Debug ログにしか出ない）。受け皿として `message.EnsureNotEmpty` で `ERR000410` を必ず立てるようにしてある。**「失敗したのに GkillError が空」で return する分岐を新しく作らないこと。** 回帰は `repositories_id_chunk_test.go` / `handle_get_kyous_mcp_test.go` の `TestHandleGetKyousMCP_ManyIDs` / `message/gkill_error_test.go` が守る。

**rep名の絞り込みは「検索するrep」ではなく「検索結果」でやる:** `find_filter.go` の `selectMatchRepsFromQuery` で **`UnWrap()` の戻り値を `MatchReps` に入れてはいけない**。`UnWrap()` はインメモリキャッシュrep（と `--cache_reps_local` のローカルコピー層）を剥がして生のディスクrepに戻すので、**キャッシュを丸ごとバイパス**して端末別重複repぶんディスクを舐めることになる。実データでは11個のキャッシュrepが約940個の生repに化け、gitだけでプロファイル1窓あたり20.7秒を使っていた（2026-08-19 実測。GUIは `reps` を常に送るので**通常の検索が毎回この経路だった**）。`UnWrap()` を使ってよいのは「そのラッパに選ばれた実repが1つでもあるか」の**枝刈り判定だけ**（これは省けない ―― 省くとキャッシュOFF・「1種別だけチェック」・「一致0件」がむしろ悪化する）。実際の絞り込みは `findKyous` の `filterKyousByRepName` が `Kyou.RepName` で行う。落とし穴が4つある: **(1)** 本文ヒット由来の2本目の検索（`matchTextFindByIDQuery`）にも同じ絞り込みが要る、**(2)** 全部落ちたIDは**キーごと消す**（空スライスを残すと `kyous[0]` を見る `filterLocationKyous` / `filterMiForMi` / `overrideKyous` が panic する）、**(3)** `Reps == nil` は「未指定」で `len()` で判定すると全件消える、**(4)** **`RepName` が空の行は残す** ―― キャッシュrepへの write-through は呼び出し側の値をそのまま INSERT するので追加直後の行は `REP_NAME` が空で、落とすと**いま追加した記録が最大1分間一覧から消える**。`dao/reps` 側に置いてもいけない（ReKyou/MiReKyou のワード委譲が利用者のクエリをそのまま `FindKyousSequential` へ渡すので、チェックしていないrepに参照先があるリポストが黙って語句検索に当たらなくなる）。**(5)** 書き込み側に**実在しないrep名を入れさせない** ―― 非空のrep名は「実在するが選ばれていないrep」として落とされるので、合成した名前を渡すと記録が黙って消える。実例: `commit_tx`（KFTLの送信経路）は一時リポジトリから読み直した記録をそのまま write-through していたが、`GetXxxByTXID` は `? AS REP_NAME` に temp rep の名前（`KmemoTemp` / `KC_TEMP` / `lantana_temp` …）を差し込んで返すため、**KFTLで書いた記録だけが一覧から丸ごと消えていた**（2026-08-19 修正。実rep名を入れてから write-through する）。フィルタ側に `*Temp` の例外を足してはいけない（temp rep 名は流儀がばらばらで、一覧を2箇所で維持することになる）。守るテストは `select_match_reps_cache_test.go` / `find_kyou_rep_name_filter_test.go` / `get_kyous_rep_filter_test.go` / `get_kyous_tx_rep_filter_test.go`（**すべてキャッシュON/OFFの両方**）。

**タグ絞り込みの取得は2経路。切り替えは「タグ名の個数」で決める:** `find_filter.go` の `collectTagsForFilter` が `MatchTags`（クエリのタグ名に一致するタグ）・`RelatedTagIDs`（タグが1つでも付いているIDの集合）・`AllHideTagsWhenUnchecked`（強制非表示タグ）の3つを作る。**タグ名の照合をSQLへ降ろすと `LOWER(TAG) = LOWER(?)` になり、列に関数がかかるので索引が効かず、全行に `LOWER()` を適用したうえでそれを名前の数だけ繰り返す**（O(行数 × 名前の数)）。一方「全部取ってGoで `strings.EqualFold`」は名前の数によらず O(行数) だが、`reps.Tag`（240B + 文字列10本）を行数ぶん実体化する確保を毎回払う。2万タグでの交差点は**およそ30**（`dao/reps/tag_find_bench_test.go` に実測表）。どちらも行数に比例するので交差する名前の個数は行数によらずほぼ一定で、閾値は `maxTagNamesForSQLFilter`（32。**クエリのタグ名と非表示タグ名の合計**で見る）。**「タグ無し」仮想タグ（`NoTags`）を使う検索では、名前の個数によらずGo側で照合する** —— `RelatedTagIDs` のために結局は全タグを取るので、そこから名前を拾うぶんはタダになる（以前は「全タグの取得」「クエリのタグ名で絞る検索」「非表示タグ名で絞る検索」を別々に投げていて、本番プロファイルでタグ名の絞り込みが実質CPUの44%＝40.4秒、非表示タグの取得がさらに11.2秒を占めていた。統合後は実質CPUが90.9秒→52.3秒）。照合は**完全一致・大小無視**で、`filterTagsKyous` のAND分岐と同じ意味論（SQLは TAG 列だけでなく ID 列とも突き合わせていたので、そこも写してある）。**2経路が同じ結果を出すことをテストで固定してある**（`find_filter_test.go`）—— ずれるとタグの個数によって検索結果が変わるという静かな壊れ方になる。`RelatedTagIDs` の読み手は `NoTags` 分岐しか無いので、`NoTags` を使わない検索では作らない（走らせ忘れると**全件が「タグなし」扱い**になりタグの付いた記録まで返る。回帰は `get_kyous_tag_filter_test.go` と `get_kyous_regressions_test.go` の `TestHandleGetKyous_TimeIsTagsFilterWorksWithoutKyouTagFilter`）。強制非表示タグも2経路の一致を `TestHandleGetKyous_HideTagsBothPaths` が固定する。

**キャッシュのフルリビルドは「実DBが変わったときだけ」:** cached rep の `UpdateCache` は `DELETE FROM` + 全行再INSERT なので、変わっていない rep まで作り直すと共有の書き込みロックを握ったまま全種類の検索が止まる。抑止は `dbFileChangeDetector`（mtime + サイズ）で、基準を進めるのは**再構築成功後の `CommitCacheRebuild` だけ**（失敗した回を取りこぼさないため）。`--cache_reps_local=true` のときに挟まる `*_local_cached.go` も同じ約束を守ること ―― **「コピーが要るか」の判定は必ず `os.Remove` より前に置く**。消してから `os.Stat` すると常に「要コピー」になり、毎回1.3GBコピーし直したうえに `LastUpdateCacheChanged()` が常に true を返して抑止が丸ごと無効化される（実データで phase1 が 0.2秒→1〜2分。2026-08-12修正）。共通ヘルパは `local_rep_cache_copy.go`、回帰検出は `local_rep_cache_granular_test.go`。**ReKyou / MiReKyou は変更検知に載せてはいけない**（コピー省略のみ可）: この2つはターゲット解決のためアドレス確定後にもう一度更新される仕様で、mtime判定を入れると2回目が飛んでターゲット未解決の中身が残る。

**Core entity — "Kyou"** (record). Data types: kmemo (text), timeis (timestamps), lantana (mood 0-10), kc (numeric), nlog (expense), urlog (bookmark), mi (task), idf_kyou (file, with `is_zip` flag for .zip/.cbz), re_kyou (repost), mi_re_kyou (an existing Kyou turned into a task: `target_id` + Mi scheduling fields, no title; `data_type` is `mirekyou_create` / `_check` / `_limit` / `_start` / `_end`, so prefix checks must test `mirekyou` **before** `mi`; the client's `load_typed_datas` guarantees this structurally by sorting its prefix table longest-first rather than relying on the order the branches are written in — `kyou-typed-data-dispatch.test.ts`), tag, text, notification, git_commit_log, gps_log (GPS tracks), plugin_kyou (external plugin data — `typed_plugin` field non-null in TypeScript `Kyou` class; `data_type` is plugin-defined e.g. `claude_conversation`).

**Response structure:** All API responses include `messages []GkillMessage` and `errors []GkillError` (with `error_code` + `error_message`). HTTP 200 for normal responses (check `errors` array), 403 for access denied, 500 for unexpected errors. The Go struct tags carry no `omitempty`, so **on success `messages` / `errors` come back as `null`, not as an empty array**. On the client always go through `res.errors ?? []` **before spreading or `concat`** (a bare `[...res.errors]` throws `TypeError`; `[].concat(null)` silently pushes `null` as an element). `if (res.errors && res.errors.length !== 0) { ... }` のガードで囲うのも同じく正しく、そちらが約180箇所と多数派。**ガード方式をわざわざ `?? []` へ書き換える必要はない** —— 守るべきなのは「null のまま展開しない」の1点だけ。

**検索条件（FindQuery）の null 判定セマンティクス:** かつて存在した `use_*` 有効化フラグ（14個）は**全廃**され、いまは **値フィールドが非null（Go では非nil）ならそのフィルタが有効**。`FindQuery` は 55→41 フィールド。間違えると例外もエラーも出ずに静かに 0 件になるので、以下は規約として守ること。
- `null` / `nil` = フィルタ未使用、**非nullの空配列 `[]` = フィルタ有効かつ0件指定**。唯一の例外は `timeis_words: []` で「任意の TimeIs に覆われた Kyou」を意味する
- 3値そろって初めて有効になるグループがある（地図の `map_latitude` / `map_longitude` / `map_radius`）。Go 側は `HasWordFilter()` / `HasTimeIsFilter()` / `HasCalendarFilter()` / `HasMapFilter()` / `HasPeriodOfTimeFilter()` の**ゲートヘルパ経由で判定する**（生の nil 比較を書き散らさない）
- `PeriodOfTimeWeekOfDays` は **nil を先行ガードで弾く**こと。`len==0` / `len!=7` の分岐へ落とすと全件が消える（`find_filter.go` の `sortAndTrimKyousMap` と `sqlite3impl_util.go` の両方に同じ罠がある）
- TypeScript 側で `undefined` は禁止。`JSON.stringify` でキーが落ち、localStorage 往復でコンストラクタ既定値が復活し、`deep_equals` のキー数比較が壊れてサイドバーの機械的 re-emit ガードが死ぬ。未使用は必ず `null` で表現する。**禁止の対象は「永続化・比較される値」**（`FindQuery` のフィールド、localStorage へ入るオブジェクト、`deep_equals` に掛ける値）。関数の省略可能引数（`show(query?: FindKyouQuery)`）や Vuetify の `:color="… ? 'error' : undefined"`（prop 既定値を効かせる用法）は対象外
- ただし `FindKyouQuery` のコンストラクタ既定は `tags` / `reps` だけ **`null` ではなく `[]`**（旧 `use_tags=true` + 空配列と厳密等価にするため）
- Mi の板名は `mi_board_name: null` が「すべて」。番兵は `classes/mi-board-names.ts` の **`MI_ALL_BOARD_KEY`（= ハードコードの `"すべて"`。ロケール非依存）** でサイドバー専用、null への変換は `use-mi-query-editor-sidebar.ts` の1点に集約されている。**i18n の訳語（`MI_ALL_BOARD_NAME_TITLE`）と比較してはいけない** ―― ツリーが emit するのはノードの `key` で、それは `append_all_mi_board()` が入れた `"すべて"` 固定なので、訳語と比べると日本語以外のロケールで「すべて」が全件に戻らず 0 件になる（表示名だけが `ALL_MI_BOARD_NAME` / `MI_ALL_TITLE`）
- 旧形式JSONの移行は3実装が**同じ16キー**を扱う: Go `api/find/find_query_legacy_json.go`、client `classes/api/find_query/normalize-legacy-find-kyou-query-json.ts`、MCP `mcp/lib/constants.mjs` の `LEGACY_USE_FLAG_KEYS`。どれかが欠けると、そのフラグを送る古いクライアントの保存クエリが移行されない（MCP では未知キー扱いで throw する）。共有URL用の `share_kyou_info.db` は起動時にスキーマ 1.0.0→1.1.0 で**保存済みJSONそのものを書き換える**（共有URLは配布済みで再発行できないため）

### Frontend (Vue 3 + TypeScript) — `src/client/`

Stack: Vue 3 + Vuetify 4 + Vue Router 5 + vue-i18n 11 + Vite 8 + TypeScript 6 + PWA (vite-plugin-pwa + Workbox)

- `router/index.ts` — 13 page routes (login, kftl, mi, rykv, kyou, mkfl, plaing, saihate, dashboard, rudbeckia, set_new_password, register_first_account, shared_page) + 2 redirect-only routes（`/regist_first_account` → `/register_first_account`、`/shared_mi` → `/shared_page`。どちらも query を引き継ぐ）。**旧パスの吸収は redirect でやること** ―― コンポーネントの setup から `router.replace` すると、`<script setup>` に top-level await があるページでは初回ナビゲーションが完了しなくなる（`/shared_mi` が実際にそうなっていた。share_id 無しで throw して setup ごと落ちていたため、redirect が一度も走らず露見していなかった）
- `pages/views/` — 202 view components, `pages/dialogs/` — 116 dialog components (Escape key closes via `useFloatingDialog`), including ZIP contents browser, plugin HTML views (`plugin-html-view.vue`, `plugin-html-context-menu.vue`, `plugin-config-dialog.vue`), and Dnote trend/correlation graph components (client-side aggregation, no server API)
- `classes/api/gkill-api.ts` — Singleton `GkillAPI` class (~3,400 lines), client-side API wrapper
- `classes/kftl/` — KFTL parser (50 statement types; the Go side has 47). Accepts the same Japanese/ASCII prefixes as the Go parser; ASCII constants and match/strip helpers centralized in `kftl-prefixes.ts`
- `classes/cascade-delete-kyou.ts` — cascade delete for Kyou. The attached Tag / Text / Notification and the ReKyou / MiReKyou that reference the Kyou are looked up in reverse via `GetReKyousByTargetID` / `GetMiReKyousByTargetID` and logically deleted together with it. Depth cap 32 (`max_cascade_depth`), 16 lookups in flight per level (`request_chunk_size`). **The Kyou itself is deleted last** (deleting it first makes the server's `FindKyous` drop the referencing records from its results, so the reverse lookup can no longer find them). No TXID / `commit_tx` is used, so a partial commit is possible. On failure: ERR900093 `cascade_delete_depth_exceeded` / ERR900094 `cascade_delete_failed`, i18n key `FAILED_CASCADE_DELETE_KYOU_MESSAGE`
- `serviceWorker.ts` — PWA service worker (Workbox precaching, POST caching, push notifications, Web Share Target; `/zip_cache/.*` on NavigationRoute denylist)

**State management:** Props/Emit only. No Pinia/Vuex. `GkillAPI` singleton for backend communication.

**Android共有（Web Share Target）の二重保存対策**（2026-08-16）。`POST /share-target` は `serviceWorker.ts` が
`respondWith` で丸ごと処理し、その場で `add_urlog` / `add_kmemo` を叩いてから `/saihate` へ 303 で送る。
**Android はタスク（アプリ履歴）から復帰すると同じ共有インテントを再配送する**ので、初回とビット単位で同じ
multipart POST がもう一度届き、素直に保存すると2件目ができる。台帳は `classes/share-target-dedup.ts`。守るべき約束:
- **再配送と意図的な再共有は内容から区別できない。** 見分ける手は「保存済みの内容を覚えておく」以外に無いので、
  台帳（`gkill-share-dedup-cache` の `/__gkill_share_dedup/ledger`、直近100件・24時間）と内容の完全一致で判定する
- **台帳へ載せるのは保存が成功したときだけ。** 応答を見ずに載せると、保存できていないのに次の共有が24時間弾かれる
  （`is_successful_gkill_response` で HTTP ok と `errors` 空を確認する）
- **重複は黙って捨てず最果てで確認を出す**（`confirm-save-duplicated-shared-data-dialog.vue`）。
  黙って捨てると意図的な再共有が理由なく消える
- **「それでも保存する」も `/share-target` を通す**（フォームに `gkill_force` を立て、応答は 303 ではなく JSON）。
  ページ側で `add_urlog` / `add_kmemo` を組み立て直すと保存が2実装に割れる
- **共有由来のクエリ（`is_saved` / `share_result`）は最果てに入った瞬間に落とす。** 残したままアプリ履歴から
  開き直されると「保存しました」がもう一度出て、保存していないのに二重保存に見える
- 台帳は Kyou キャッシュとは**別のキャッシュ**に置く（`activate` が `KYOU_CACHE_NAME` を丸ごと消すので、
  同居させると版が上がるたびに台帳が飛ぶ）
- 守るテスト: `share-target-dedup.test.ts`

**Composable pattern:** ロジックを持つ view/dialog は必ず対応する `classes/use-*.ts` にコンポーネントのロジックを置き、`.vue` の `<script setup>` は「import・`defineProps`・`defineEmits`・コンポーザブル呼び出しの分割代入・`defineExpose`」だけにする（dialogs は **116/116** がこの形）。例外は**ロジックが1行も無いコンポーネント** ―― props と emits を宣言してテンプレートから直接 emit するだけの`sidebar-header.vue` / `manage-share-task-list-view.vue` / `confirm-delete-share-task-list-view.vue` は、空の `return {}` を返すだけのコンポーザブルを置かない（実際に3本そういう死んだファイルが残っていた）。 Props and emits are defined in `-props.ts` / `-emits.ts` files alongside the `.vue` file (many components share a common props/emits type instead of having their own pair). Template refs to child components use the `ComponentRef` type (`classes/component-ref.ts`). Never hand-write the CRUD relay handler bundle that only forwards child events to the parent: call `build_kyou_view_relay(emits, overrides?)` (view layer, 18 events) or `build_kyou_dialog_relay(emits, overrides?)` (dialog layer, those 18 plus `focused_kyou` / `clicked_kyou` = 20 events) from `classes/kyou-view-relay.ts`, assign it as `const crudRelayHandlers = build_kyou_view_relay(emits)`, and pass it to the template as `v-on="crudRelayHandlers"` (one line replaces the whole `@…` list). Swap out only the events whose behavior differs via `overrides`. Of the 21 events in `KyouViewEmits`, `requested_close_dialog` is not relayed (the dialog wires it to itself with `@requested_close_dialog="hide()"`), and `focused_kyou` / `clicked_kyou` are not relayed by the view layer (a nested view would fire them twice — only the dialog layer relays them). When adding an event, add it to **both** `KyouViewRelayArgs` and `kyou_view_relay_event_names`; adding it to only one breaks the build through the `Exclude` exhaustiveness check. どちらの束を使うかの基準は「ダイアログかどうか」ではなく **「自分がフォーカスの発火源かどうか」**。子が上げてきた `focused_kyou` / `clicked_kyou` を素通しするだけの中間層（`dnote-item-list-view` 等）は自分では発火しないので `build_kyou_dialog_relay` が正しい（名前に `dialog` と付いているせいで誤読されやすい）。ページ最上位の `RykvDialogHost` には `build_kyou_dialog_host_handlers(required, overrides?)` を使う ―― ページには emit 先の親がいないので未指定イベントは no-op で埋まるが、`updated_kyou` / `deleted_kyou` / `requested_reload_kyou` / `requested_open_rykv_dialog` / `closed` の5件は型で必須にしてあり、書き忘れるとコンパイルエラーになる。`__tests__/unit/classes/relay-bundle-source-scan.test.ts` が「`v-on` で渡した束と `@中継イベント` を同じ要素に併記していないか（両方登録されて二重に発火する）」をソース走査で検査する。

**Kyou の再読込:** `classes/kyou-reload.ts` の `refresh_kyou(kyou, query?)` / `refresh_kyou_in_list(list, kyou, options?)` を使い、**手順を手書きしない**。正しい手順は「SWキャッシュ削除 → `reload(true, query)` → `is_typed_data_loaded = false` → `load_all(query, true)`」の4つで、1つでも欠けると引き直しに失敗する。とくに `load_all` の第2引数(`force_attached`)を落とすと、`Kyou.clone()` が `is_attached_tags_loaded` を引き継ぐせいで `InfoBase.load_attached_tags(false)` が早期returnし、**添付タグを一度も引き直さない**（「タグを足しても表示が変わらない」の正体）。書き戻し方は呼び出し元が選ぶ。**既定の in-place splice を使うこと** ―― `model_value` が親の配列そのものになるダイアログで必須なだけでなく、rykv/mi の列でも必須。列の配列を作り直すと `focused_kyous_list`（= `match_kyous_list[focused_column_index]` へのエイリアス）が黙って切れ、件数カレンダーと Dnote がフォーカス列に追随しなくなる（2026-08-18 に rykv/mi を `replace` から in-place へ戻した。守るテストは `rykv-view-search-routing.test.ts` の「reload_kyouは列の配列を作り直さない」）。`replace` による copy-on-write が正しいのは、配列そのものを `Ref` に持ち他からエイリアスされない dashboard / plaing / shared-mi だけ。書き戻す位置は `refresh_kyou_in_list` が **await のあとに id で取り直す**（待っている間に局所挿入や削除でリストが動くため、待つ前のインデックスで splice すると別の行を潰す）。

引き直しの**合流には `new_reload_batch()` の値を渡す**。1回の更新でリスト・focused・開いているダイアログが独立に引き直すので、`reload_kyou` の先頭で1回だけ取って全部に同じ値を渡すと往復が1回で済む（`refresh_kyou` の第3引数 / `refresh_kyou_in_list` の `options.requested_at`）。渡さないと呼び出し時刻が使われ、飛行中の引き直しには相乗りしない。**無条件に合流させてはいけない**: `open_rykv_dialog` はダイアログを開くたびに引き直しを投げるので、それがまだ飛行中のうちに保存すると、更新前の Kyou を掴んで列・focused・ダイアログを一斉に古い内容へ戻していた（「ダイアログを閉じたら一覧が更新されない」の正体）。

引き直しは**呼び出し元のダイアログや行より長生きさせる**。`fetch_refreshed_kyou` は作業用クローンに専用の `AbortController` を入れ直しており、`KyouView` の `onUnmounted` abort で保存直後の引き直しが道連れになることはない。引き直し中は `is_kyou_reloading(id)` が真になり、`KyouView` が**中身を残したまま**右上にスピナーを重ねる（`.kyou_reloading`。中身を差し替える `show_loading_indicator` とは別物で、消すと行がちらつき詳細ビューでは高さが跳ねる）。状態は id キーのモジュールレベルに置く ―― `KyouView` は id が同じなら再マウントされず props が差し替わるだけなので、コンポーネントローカルに持つと倒せない。

**タグ/テキスト/通知の変更は `updated_kyou` を出さない。** 唯一の信号が `requested_reload_kyou` なので、Kyou を抱えて表示するコンポーネントは必ずこれを処理すること。

**プラグイン本文（`plugin-html-view.vue`）の iframe 越しの受け渡し**（2026-08-15 に「Ryuu の PluginKyou がときどき空白」「本文をダブルクリックしても KyouDialog が開かない」を潰して確立）。本文の入れ方は `is_list_view = typeof props.height === 'number'` で分岐し、一覧は `srcdoc` 直書き、Ryuu(`'fit-content'`)・rykv 詳細ペイン(`'auto'`)・KyouDialog(`'unset'`) は定数ローダーへの postMessage 注入（ダイアログの `pushState` より後に iframe がナビゲートするとブラウザバックが2回要るため、ローダー方式は外せない）。注入は**ローダーが `gkill_plugin_loader_ready` を名乗ってから**行う ―― `iframe.contentWindow` は about:blank の時点から真なので、それを見て先に送るとリスナー未登録の iframe に届いて黙って消え、`sent_html` のせいで送り直さないぶん本文が二度と入らない（ServiceWorker が `get_plugin_content_html` をキャッシュ優先で返すので、2回目以降の表示ほど負けやすい＝「ときどき」）。ready のたびに `sent_html` を落として送り直し、Kyou が変わったら iframe ごと作り直す（`:key`。`document.open()` はローダーが `window` に張ったリスナーごと捨てるので使い回すと2件目が入らない）。**`@load` で `sent_html` を落としてはいけない**（`document.close()` でも `load` は発火しうるので注入ループになる）。iframe 内のダブルクリックは親のDOMへ伝播しないので、本文HTMLの末尾に転送スクリプトを足して `gkill_iframe_dblclick` を受け、**親が本物の `dblclick` を撃ち直す**（新しい emit 経路を作らないので `kyou-view.vue` と `ryuu-item-view.vue` の既存の `@dblclick` がそのまま拾う）。テーマ通知を最初のサイズ通知だけに限るのは、本文側がテーマ受信で測り直して送り返すため（毎回返すと10ms周期のピンポンになる）。`plugin-html-view.test.ts` が守る。

**rykv/mi の「列×検索」不変条件**（2026-08-09 の誤配送根絶で確立。崩すと「検索結果が別の列に出る」が再発する）: 列の同一性は `query_id`（**列の誕生時のみ採番、以後不変**）。`:key`・DOM id・テンプレート ref 逆引き（`get_kyou_list_view`）・`abort_controllers` / `search_seqs`（Map）のキーは全部これ。列リロード・画像トグル・サイドバー clear 系で**再採番してはいけない**（列が remount され、検索結果の帰属も切れる）。検索は列ごとの世代番号 `search_seqs` で「最後の検索だけが書き戻せる」。`focused_query` を更新してよいのはフォーカス列の検索だけ（無条件更新するとサイドバーが別列条件に乗っ取られ query_id 重複→誤配送になる）。サイドバー編集の宛先は `focused_column_index` ではなく `querys.findIndex(query_id)` で解決する。検索ボタンはサイドバーの `generate_query(列のquery_id)` で「今見えている条件」から検索する（`rykv_hot_reload` OFF 時は編集が列に保存されないため。なお既定は **ON**、サーバ既定値 true）。サイドバーの `emits_current_query` は**再生成結果が同期済みクエリと同値なら emit しない**（フォーカス切替で子ビューのprops同期の残響が機械的に届くのを検索にしないための値比較ガード。これが破れると「検索中の列をクリック→飛行中の検索がabortされ最初からやり直し」が再発する。`generate_query` は同期済みクエリに対して恒等であること —— とくに `include_*_mi` をtrue固定でドリフトさせない。`rykv-sidebar-mechanical-emission.test.ts` が守る）。`use-rykv-view.ts` と `use-mi-view.ts` はコピー由来の対称実装なので**修正は必ず両方へ**（`rykv-view-search-routing.test.ts` / `mi-view-search-routing.test.ts` が対で守る）。フォーカス切替時の検索抑止は**コールバック式 `run_with_sidebar_search_suppressed(fn)` だけ**を使う（`skip=true → fn()でリアクティブ書き込み → nextTick(解除)` の順が本質。書き込みより先に nextTick を登録すると、Vue の nextTick が resolvedPromise へ直結して解除がウォッチャ flush より先に走り、抑止が一度も効かない — 2026-08-10 のタブフリーズ回帰の正体）。サイドバーの子クエリビュー（Rep/Tag/TimeIs/Map/Calendar）は **props 同期では emit しない**：TimeIs は同期経路に `disable_emits=true` + `pre_uncheck_all=true`（累積させない）、Map は同期時 emit なし + radius ウォッチャの値ガード、Calendar は `clicked_date` の同値エコーガード（`sidebar-child-query-sync-emission.test.ts` が守る）。`use-kyou-list-view.ts` の `scroll_to` は世代カウンタ + 上限（2秒）つきで、打ち切り時は scrollTop を直接代入する（無限リトライに戻すと 0 件列への 50ms 周期の強制レイアウトが増殖する）。

**rykv/mi/dashboard の初期化順序**（2026-08-17 に「初期取得の完了まで全画面を隠す」段階を外して確立）。順序は **列の骨組みを確定 → 可視化 → 検索**で、`inited` / `is_loading` は初期検索の完了に依存しない。守るべき約束:
- **`init()` の起動条件は `props.application_config.is_loaded` の watch**。サイドバーの `@inited` へ戻してはいけない ―― あれは子クエリビューの「その節が描けた」の集約でしかなく、設定の到着を表していたのは「`immediate` の付いていない `application_config` watch から emit する子がいる」という偶然だった（mi では実質 CalendarQuery 1つが律速し、しかもその節は `application_config` のフィールドを1つも読まない）。節を1つ画面から外すだけで画面ごとスピナーで固まる
- **サイドバーの `inited` 集約は無い**が、`inited_*_for_query_sidebar` の各フラグは**残す**。子へ `:inited` prop として降り、子が「初回同期か再同期か」を判定している（消すと props 同期のたびにチェックが列をまたいで累積する）
- **`init()` は hot reload の ON/OFF で分岐せず、列の骨組み（`querys` / `querys_backup` / `match_kyous_list`）を検索より前に確定させる**。1本ずつ足すと「列が確定した瞬間」が定義できず、復元中にユーザが列を足したとき `search(i, ...)` の固定 index と衝突する。`querys_backup` を先に埋めるのは、機械的な残響が `search()` の `deep_equals` 早期returnで確実に落ちるようにするため
- **`init()` で `skip_search_this_tick` を立てっぱなしにしない。** あれは「1tick分の残響を捨てる」短命フラグで、初期化全体の門番に流用すると機械的な emit が1つ届いただけで `onSidebarUpdatedQuery` が消費し、複数列のとき1列目の完了で抑止が途中で解ける。抑止は `run_with_sidebar_search_suppressed` だけを使う
- **復元の検索は `preserve_scroll=true`** で呼ぶ。`inited` が早期に立つので、落とすと `search()` が `scroll_to(0)` を撃って保存済みの復元先を潰す
- **`onSidebarUpdatedQuery` に「初期化が終わるまで捨てる」早期returnを置かない。** 初期検索の飛行中でもユーザの編集は通す。同じ `query_id` を共有するので `abort_controllers` が復元を中断し、`search_seqs` の世代照合が遅れて届いた復元結果を捨てる（＝ユーザが勝つ）
- **`onColumnScrollList` は検索中の列の通知を捨てる。** リストを空にした副作用のスクロール通知を取り込むと `preserve_scroll` の復元先が0で潰れ、保存位置にも焼き付く
- **E2E の準備完了信号はルート要素の `data-gkill-view-ready`**（`is_view_ready` = `inited && !is_restoring_columns && running_search_count === 0`）。全画面オーバーレイが初期検索を待たなくなったので、`.v-overlay .v-progress-circular` の `.first()` は列スピナーを掴み「出る前に `toBeHidden` が通る」窓ができる。`crud-helpers.ts` の `waitForColumnViewReady` がこの属性を待つ。**真偽値をそのまま bind してはいけない**（Vue は false のとき属性ごと消すので「属性の有無」で判定するセレクタが壊れる）
- **①（ApplicationConfig 待ち）は残す。** サイドバーは `v-show` なので未ロードでも描画済みで例外も API 呼び出しも出ないが、`generate_default_query_for_rykv` が `device_struct` / `rep_type_struct` / `tag_struct` / `rykv_default_period` / `hide_tags` を読むので、未ロードで既定クエリを作ると既定期間と強制非表示タグが黙って落ちる（保存済みクエリがある通常ケースは localStorage 由来なので影響しない＝初回起動のユーザだけが踏む）
- **設定取得の失敗は永久スピナーにしない。** `load_application_config()` は `res.errors` で早期returnし ref を差し替えないうえ `.catch()` も無かったので、失敗すると `is_loaded` が永久に false で画面が固まっていた。`application_config_load_failed` を立ててオーバーレイの中身をエラー＋再試行ボタンへ差し替える（文言は既存の `FAILED_GET_APPLICATION_CONFIG_MESSAGE` / `RELOAD_TITLE`）
- dashboard は列を持たないが同じ方針。初回ロードは日付変更時と同じ `fetch_for_date()` に寄せ、パネル単位のローディングで進行を見せる
- 守るテスト: `rykv-view-initial-load.test.ts` / `mi-view-initial-load.test.ts`（対）/ `column-view-init-source-scan.test.ts`（ソース走査）/ `dashboard-page-reload.test.ts` の「ApplicationConfig 取得の失敗」節 / `e2e/column-view-initial-load.spec.ts`

**検索を期間の窓へ刻んで複数回 `get_kyous` を投げてはいけない**（2026-08-18 に入れて 08-19 に撤去）。
狙いは「最初の行が出るまでの時間」とサーバのピークメモリだったが、3つとも外した。
- **総時間が伸びる。** 1リクエストぶんの固定費が窓の数だけ掛かる ―― `getAllTags`（全repの全タグ走査）は
  rykv の既定クエリでは `tags` が非nullなので `find_filter.go:106` の条件が必ず真になり**毎回**走る。
  repのファンアウトと最新版アドレスのスナップショットも同じ。窓数は既定31日で3、1年で6、下限なしで10
- **「検索が終わってもスピナーが回り続ける」ように見える。** 1窓目の結果を列へ入れて表示するのに、
  スピナーを消すのは全窓が終わったあとだから
- **静かに取りこぼす。** サーバの期間判定は2段階で精度が違う ―― SQL は `.Unix()`（秒切り捨て）、
  `find_filter.go` の `passesPeriodFilter` は `time.Before`/`After`（ナノ秒）。境界を**秒**でずらすと、
  境界 `S` に対する `S-0.5秒` のレコードが新しい側では `Before(S)` に弾かれ、古い側では
  `After(S-1秒)` に弾かれて**どちらの窓にも入らない**（秒未満を保持できるプラグイン・gitの記録が対象）

ピークメモリのために分割するなら、固定費を1回で済ませられる**サーバの中**でやること。
**列に部分的な結果を出さないこと** ―― 件数カレンダー・Dnote・Ryuu・フッタの件数はどれも
「列が全件を持っている」前提で書かれていて、部分状態ではエラーも出さずに間違える
（Ryuu は列の配列をサーバ検索の代わりに使うので、間違った「近くの記録」を返す）。
フッタの件数は `has_loaded` が立つまで出さない。守るテストは
`rykv-view-search-routing.test.ts` の「検索は期間が広くても1回で引く」「検索中の列に部分的な結果を出さない」。

**記録の追加は再検索せず、その1件を列へ差し込む**（2026-08-15）。追加系ビューは `registered_kyou` だけを出し、`requested_reload_list` は「サーバが Kyou を返さなかった」ときのフォールバックに退いた。受け口は `use-registered-kyou-local-insert.ts` 1つで、rykv / mi / dashboard がこれを使う。判定と整列は純関数 `classes/kyou-local-insert.ts` にあり、意味論は `server/gkill/api/find_filter.go` の写し。守るべき約束:
- **`/api/get_kyou` は FindQuery を受けない**ので、「その列の条件に一致するか」はクライアントで判定する。判定できるのは rep / タグ（完全一致・大小無視、`"no tags"` 番兵つき）/ カレンダー（両端含む）/ 時間帯 / mi の板名（大小**区別**）・完了状態・`include_*_mi`。判定できないのは本文検索・TimeIs・地図・plaing・画像のみ・`rep_types`（rep_name→rep_type の写像がクライアントに無い）で、**これらを使う列だけ従来どおり再検索する**。判定を足すときは「判定できないものを判定できると誤って宣言しない」ことがすべて —— 誤ると例外もエラーも出ずに黙って一致しない行が出る
- 並び順は非mi=`RelatedTime` 降順（**`.Unix()` 相当に秒へ切り捨ててから**比較。ミリ秒のままだと同一秒の隣接行で位置がずれる）、mi=ソート基準の時刻の昇順で**未設定は末尾**。mi の「未設定」は `typed_mi` ではなく `data_type` の接尾辞で判定する（一覧の既存行は `typed_mi` が未ロードなので、`typed_mi` を要求する比較子は既存行に対して動かない）
- 差し込みは **in-place `splice`**。`focused_kyous_list` は `match_kyous_list[focused_column_index]` へのエイリアスなので、参照ごと差し替えると件数カレンダーや Dnote と縁が切れる（30万件のコピーも避けられる）。ただし Dnote は命令的 reload なので配列を触るだけでは追随せず、明示的に呼び直すこと
- `add_*` の応答は **hydrate を通っていない生 JSON**（`related_time` が文字列、`clone()` も無い）。受け口で必ず実体化する
- KFTL は送信全体を tx で包むが、**tx 中の `add_*` は `added_kyou` を返せない**。リクエストクラスが id だけ積み（`get_result_kyou_ids()`）、`commit_tx` の**あと**に `get_kyou` で引いてから emit する。「終了」系は既存 TimeIs の更新なので `updated_kyou`
- 守るテスト: `kyou-local-insert.test.ts` / `kyou-local-insert-mi-parity.test.ts`（Go の `find_filter_mi_test.go` と対）/ `registered-kyou-local-insert.test.ts`（rykv・mi 両方でパラメタライズ）/ `kftl-submit-emits.test.ts`

**Kyou の追加/編集画面のタグ欄**（2026-08-17）。タグが付く追加/編集画面 **18本すべて**（追加7 = kc/lantana/nlog/time-is/ur-log/mi/mi-re-kyou、編集10 = 前記＋kmemo/idf-kyou/re-kyou、ReKyou作成の `confirm-re-kyou-view`）に共通の子ビュー `pages/views/edit-kyou-tags-view.vue` を1行置くだけで、保存の一度の操作で Kyou とタグをまとめて登録できる。子は値を集めるだけで、実際の登録は親の `save()` が `classes/kyou-tags.ts` 経由で行う（`add-notification-for-add-mi-view.vue` と同じ形）。守るべき約束:
- **`add_tag` が完了してから `registered_kyou` を emit する。** 局所挿入（`use-registered-kyou-local-insert.ts`）は渡された Kyou をそのまま使わず `refresh_kyou` で引き直すので、その時点でサーバにタグが入っていれば `attached_tags` 込みで差し込まれる。逆に先に emit すると `kyou-local-insert.ts` の `matches_tags()` が空のタグ列を見て「一致しない」と判定し、**エラーも警告も出ないまま行が現れない**。順序が唯一の防御線
- **編集画面の「更新がなかったらエラー」ガードはタグの変更でも通す**（10本すべて、エラーコードは `*_is_no_update`）。ただし**本体が無変更なら `update_*` を呼ばない** —— 呼ぶと中身の同じ新しい版が1つ増える。判定は各コンポーザブルの `is_body_changed()` に切り出してある
- **タグの変更は `updated_kyou` を出さない。** 反映信号は `requested_reload_kyou` だけなので、タグだけ変えたときも必ず出す
- **`tx_id` は使わない。** TXID指定時のタグ／Kyou は一時リポジトリにしか無いので `add_tag` は `added_tag` を返せず（`handle_add_tag.go`）、`registered_tag` を上げられない。しかも `commit_tx` はDBトランザクションではなく部分確定しうる（`handle_commit_tx.go`）ので束ねても原子性は買えない
- **同じ名前の重複はクライアントで落とす。** サーバの重複チェックはタグIDだけを見る（`usecase/tag.go`）ので、入力欄の中の重複も、削除マークの付いていない既存タグと同名のものも `get_tag_names()` が落とす
- **既存タグは `get_tags_by_target_id` で子ビューが自分で引く。** 編集ビューの `load()` が呼ぶのは `load_typed_datas()` だけで `props.kyou.attached_tags` は空のまま
- **⊗ を押した既存タグは保存を押すまで消さない**（押し間違えを戻せるように）。実削除は `is_deleted=true` の版を足す `update_tag`
- **確認はタグ → 板名の順に1つずつ**（`use-kftl-view.ts` の `do_submit` と同じ）。mi / mi-re-kyou の4画面は `do_save(skip_unknown_tag_check, skip_unknown_mi_board_check)` の再入フラグで表現する。確認ダイアログは非モーダルなので、再入のたびに子ビューから値を取り直すこと
- **タグ欄は既存フィールドより後ろ（アクション行の直前）に置く。** E2E ヘルパ `fillDialogField(dialog, N, ...)` は入力欄の位置インデックスで掴むので、前に挿すと既存 spec が総崩れになる
- 未知タグ確認は共有部品 `pages/dialogs/confirm-unknown-tag-dialog.vue` + `classes/use-confirm-unknown-tag.ts`（板名版と対）。`add-tag-view` / KFTL に手書き複製されていたマークアップと、`add_tag` の手順を12本のコンテキストメニュー・削除確認から寄せた
- **「確認が開いているか」を呼び出し元が持つときは `closed` イベントで倒す。** `unknown_tags` の空判定で代用すると、ブラウザバックで閉じたときに空にならないので開きっぱなし扱いになる（KFTLのタブ操作が永久ロックされる）。ただし **`closed` は `requested_confirm` より先に来る**（ダイアログが `hide()` してから emit するため）ので、確認の続行で読む値（KFTLの `submit_target_tab_id` 等）を `closed` で消してはいけない
- 守るテスト: `kyou-tags.test.ts` / `edit-kyou-tags-view.test.ts` / `add-views.test.ts` の「registered_kyou は add_tag が終わってから emit される」/ `edit-views.test.ts` の「タグ欄」節 / `e2e/add-dialog-crud.spec.ts` の「URLogを本文とタグ入りで一度に追加できる」

**利用者がその場で作ったタグは、開いている列の検索条件へ足す**（2026-08-19）。
直している不具合は **「タグを付けて追加した記録が、追加した直後に一覧から消える」**。
既定クエリは「絞らない」を `tags = null` ではなく **「そのときの `check_when_inited` タグ名の列挙」**として
物質化する（`find-kyou-query.ts` の `generate_default_query_for_rykv`）。それが localStorage の列状態へ
丸ごと落ちる一方で、タグ宇宙（`tag_struct`）は毎回サーバから引き直して育つので、**列の条件だけが保存時点で凍る**。
タグが1つも無い時期に作られた列は `tags = ["no tags"]` の1件だけになり、`tags` は非nullなのでフィルタは有効
＝ **タグの付いた記録は1件も通らない**（サーバ検索 `find_filter.go` と局所挿入 `kyou-local-insert.ts` の
`matches_tags` の両方で落ち、**エラーも警告も出ない**）。実装は `classes/use-registered-tag-column-filter.ts` の1つで、
rykv / mi が同じものを使う。守るべき約束:
- **使ってよいのは「そのタグがタグツリーに無かった」という決定可能な事実だけ。** 未知だった＝利用者がついさっき
  作った＝「意図的にチェックを外した」ことは原理的にありえない。逆に**既知のタグは触らない**（「保存後に増えたタグ」と
  「利用者が外したタグ」は現状の保存データでは区別できない）。推測しないので新しい永続フィールドは要らない
- **判定は `emits('registered_tag', …)` より前に、同期で行う。** emit 先（`use-rykv-page.ts` / `use-mi-page.ts`）の
  `check_tag_update` がタグツリーへ足したあとでは、「ついさっき作った」ことを二度と知れない。
  順序は `column-view-init-source-scan.test.ts` がソース走査で固定する
- **`tags_and === true` の列には足してはいけない。** AND は `query.tags.every(...)` なので `["no tags", "新タグ"]` の積は
  必ず空になる（新タグ付きの記録は `has_no_tags === false` で落ち、タグ無しの記録は `has_tag_name(新タグ)` で落ちる）。
  足しても目当ての記録は救えず、ほかの記録を巻き込んで**列を丸ごと消す**。`tags === null` の列も触らない（元から通る）
- **`querys` と `querys_backup` を同じ tick で揃え、`run_with_sidebar_search_suppressed` で包む。** 控えがずれると
  サイドバーの機械的な残響が `search()` の `deep_equals` 早期returnで落ちなくなる
- **localStorage へは自分で書かない。** `search()` が必ず `set_saved_*` を通るので、引き直しを通せば揃う。
  自前で書くと「条件だけ変わって引き直さない」経路が生まれ、**次回起動時だけ列が変わる**という最悪の非対称になる
- **1tick ぶん溜めてから1回だけ流す。** `add_tags_to_target` はタグを1件ずつ登録して1件ずつ emit するので
  （`kyou-tags.ts`）、まとめないと新タグ3つで列あたり3本 `search()` が走り2本が abort されるだけになる
- **ポート（rudbeckia）へも配る。** タグが検索条件に効くようになったので `kyou-change-bus.ts` の
  「タグは配る必要が無い」という前提が崩れた。publish するのは**未知と判定した発生元だけ**で、
  受け手は既知判定を**やり直さない**（届く頃にはツリーに載っているので必ず取りこぼす）。`apply_registered_tag` は
  optional（dashboard / plaing は列のタグ絞り込みを持たない）。受け手では `reload_list` の畳み込みより**先**に適用する
  （逆だと旧条件のまま全件取り直す）
- **今回の修正では直らないもの**（区別できる情報が保存データに無い）: 他端末で作られたタグ、過去に作ったタグ、
  プラグインKyouのタグ。本命は**既定クエリの物質化をやめて `tags = null` にする**ことだが、強制非表示タグの意味論が
  `query.tags` の中身に依存している（`kyou-local-insert.ts` の `!checked_tag_names.includes(...)` ＝
  「チェックされていなければ隠す」）ため、Go とクライアントの対称な再定義が要る別件。
  我々の書き換えから `check_tag_update` 着地までの間にサイドバーを触られると `generate_query` が
  ツリーの選択から `tags` を作り直すので新タグが落ちる、も同様に別件
- 守るテスト: `registered-tag-column-filter.test.ts` / `new-tag-column-search.test.ts`（rykv・mi 両方）/
  `column-view-init-source-scan.test.ts` / `kyou-change-bus.test.ts` /
  `e2e/regression-fixes.spec.ts` の「新規タグを付けて追加した記録が、画面遷移せずに一覧へ残る」
  （**画面遷移しないことが本質** ―― 遷移すると既定クエリを作り直すのでこの不具合をすり抜ける）

**ログイン画面ではセッション無効の飛ばしを止める**（`is_on_login_page`、`gkill-api.ts` の `check_auth`）。
`check_auth` はセッション無効系のエラーコード（`ERR000013` / `ERR000002` / `ERR000238` / `ERR000373`）を見つけると
`clear_browser_datas()` してから `location.replace("/")` する。ところが**ログイン失敗も同じコード帯を通る**
（存在しないユーザIDは `ERR000002` `AccountNotFoundError`、無効化されたアカウントは `ERR000238`）。
素直に飛ばすとページごと作り直され、`login-page.vue` がいま出したばかりのエラー表示が消える。
利用者からは「画面が一瞬光って、理由も出ないまま元のまま」に見える。
- **行き先は同じ `/` なので、飛ばさないことで失うものは無い。** セッションIDのクリアだけは行う
- 判定は `pathname === '/'`（と空文字）だけ。**共有ページを足してはいけない** ―― 共有ページは
  セッションではなく `share_id` で認証するので、そもそもこのコード帯を通らない
- ガードは `location.replace` の**手前**に置く。順序は `check-auth-login-page.test.ts` がソース走査で固定する
  （`check_auth` の本体を切り出し、ガードの位置が replace より前であることを添字で比べる）
- 守るテスト: `check-auth-login-page.test.ts` / `e2e/login.spec.ts` の
  「login with invalid credentials shows error」


**KFTL（メモ帳）のタブ**（2026-08-16）。`kftl-view.vue` がタブのホストで、`/kftl` ページ・各画面のメモ帳ダイアログ（`kftl-dialog.vue`）・打刻メモ帳（`mkfl-view.vue`）の**3系統すべて**に効く。純関数は `classes/kftl-tabs.ts`、状態は `classes/use-kftl-tabs.ts`。守るべき約束:
- **`v-window` を使わず、アクティブなタブ1枚だけを描画する。** 非表示の textarea は `clientWidth` が0になり、`kftl-statement-line.ts` の `1 + parseInt(text_width / 0)` が **`NaN`**（`Infinity` ではない）を返して行ラベルが丸ごと消える
- **タブの一覧と中身はモジュールシングルトン、いま映しているタブはビューごと。** メモ帳ダイアログは複数枚開けるので、インスタンスごとに配列を持つと片方の古い配列で localStorage を丸ごと上書きしてもう片方のタブが消える（単一文字列だった頃は「上書き」で済んでいた）。逆に `active_tab_id` までストアに置くと、全ウィンドウが同じタブを映して並べて見られない。ストアが持つのは `last_active_tab_id`（次に開くウィンドウの初期表示）だけ。単体テストは `reset_kftl_tabs_for_test()` + `localStorage.clear()` を `beforeEach` で呼ぶこと
- **タブが消えたらビューは追随する。** 別のウィンドウがタブを閉じたり保存で閉じたりすると、そのタブを映していたビューの `active_tab_id` が宙に浮く。`watch(() => tabs.map(t => t.id))` で新旧を比べ、旧インデックスへクランプして隣のタブへ移す（放置すると本文が空文字を返し続ける）
- **送信対象タブは `do_submit(target_tab_id, ...)` の引数で渡す。** `do_submit()` は未知タグ確認・未知板名確認でいったん抜けて応答を待つので、1回の保存で2〜3回呼ばれる。gkill のフローティングダイアログは非モーダル（`App.vue` の `.gkill-float-scrim` が `pointer-events: none`）なので、確認中でも背後のタブバーは押せる。持ち越し用の `submit_target_tab_id` を**引数ではなく直接読ませてはいけない** ―― 確認ダイアログを Escape / ブラウザバックで閉じられると古い値が残り、次の保存が別のタブへ誤配送される。新しい送信（`submit()`）は必ずアクティブなタブを渡す
- **タブ操作のロックは `is_submitting || show_confirm_unknown_tag_dialog`。** `is_requested_submit` は設定の読み込みが終わるまで `true` なので鍵に使えない（起動直後にタブを追加できなくなる）。板名確認をロック条件に入れてもいけない ―― `unknown_mi_boards` はブラウザバックで閉じても空にならないので**永久ロック**になる（タグ確認は `useDialogHistoryStack` がどの閉じ方でも false にするので安全）
- **不正行の判定は送信対象タブから引き直す。** 表示用の `invalid_line_numbers` はアクティブタブのもので、しかも await をまたいで遅れて着地する。これを送信の可否に使うと、タブを切り替えた直後の保存が「おかしな行があります」で止まる
- **保存マーカーの判定は `beforeinput` で控えた本文と `input` 時点の本文を比べ、「確定したマーカー行が増えたか」で行う（`count_save_marker_lines`）。watch（本文の変化）に置いてはいけない。** 理由が2つある。(1) watch は `flush: 'post'` なので、1回のフラッシュ窓の中で本文が2回変わると**1回しか呼ばれず、中間の値（マーカーで終わっている本文）は一度も観測されない**。(2) DOMの仕様上、**同じ `input` イベントのリスナーとリスナーの間でマイクロタスクが走る**ので、Vue の post flush（＝この watch）が `@input` ハンドラより**先**に新しい本文を観測しうる。IMEで「変換の確定」と「改行」を続けて打つと (2) が必ず起き、watch 側に判定や基準の更新を置くと**判定が走らない／増分0に見える**（2026-08-19 修正。「IMEから順当に入力すると効かないのに、バックスペースを押すと効く」の正体で、`beforeinput`→`input` の対に移して解消した）。「末尾がマーカーか」で見るのも不可 —— (1) で黙って落ちるうえ、1行目のマーカーを拾えず、マーカーの後ろに空行が1本あるだけで効かない。増分で見ると、1行目のマーカーも拾え、既にマーカーが残っている本文を編集しただけでは再送信しない、も同時に成り立つ。「確定した」= その行の後ろに改行がある、なので `！` を打った時点では走らない。守るテストは `kftl-submit-emits.test.ts` の「KFTLの保存マーカー」節と `e2e/kftl-tabs.spec.ts` の「IMEで確定してから改行しても自動で保存される」（**IMEはCDPの `Input.imeSetComposition` でしか再現しない**。`pressSequentially` は打鍵ごとにイベントループが回るので中間の本文を必ず観測してしまい、常に緑になる）
- **保存マーカーによる自動送信の入口は「利用者が選んだ操作」の2つだけ ―― textarea の `@input` と、テンプレート貼り付け。** 判定関数（`maybe_submit_by_save_marker`）は1つのまま、入口だけ2つにすること。`text_area_content` はアクティブタブへの computed なので、判定を watch の内容変化そのものに戻すとタブ切替・localStorage からの復元でも発火し、末尾にマーカーが残ったタブを**クリックしただけで保存が走る**。watch は `@input` が立てた印（`user_input_tab_id`。立てるのは `onTextAreaInput()` **だけ**）が付いているときしか判定しない。**テンプレートはこの印に相乗りさせず `paste_template()` から直接呼ぶ** ―― watch は `new_value === old_value` で早期returnするので、貼る前のタブの本文がテンプレートと同一文字列だと黙って発火しない（タブ化する前も同じ理由で取りこぼしていた）。さらに watch は `flush: 'post'` かつ await を挟むので、判定までにタブを切り替えられると「印のタブ == アクティブタブ」が偽になってこれも黙って落ちる
- **同じタブを2枚のウィンドウが同時に保存しても、登録は1回。** `is_submitting` / `is_requested_submit` は**ビューごと**なので、同じタブを映した別ウィンドウの保存ボタンは止められない（KFTLはtxで束ねて送るので二重送信するとKyouが丸ごと重複する）。送信中のタブidは共有ストアが持ち、`do_submit` の冒頭で `tabs_store.try_begin_submit(target_tab_id)`、`finally` で `end_submit(target_tab_id)`。**claim は `is_requested_submit` ガードの「後」かつ `try` の「外」で取る** ―― 前に置くとガードの return が `finally` を通らず永久ロック、`try` の中で取ると掴めなかった側の `finally` が**勝ったウィンドウの分を解放**する。確認ダイアログで抜けるときは手放し、`confirm_submit` / `confirm_mi_board_submit` からの再入で取り直す（**持ち越すと自己デッドロックする**）。この印はリアクティブにせず localStorage にも出さない（`is_tab_locked` に混ぜると「送信中のタブから切り替えられない」、永続化するとリロードで掴んだままのタブが二度と保存できなくなる）。テンプレート経路は毎回一意な新しいタブを作り、それをアクティブにするのは貼ったウィンドウだけなので、この排他とは無関係に重複しない
- `update_line_labels()` / `refresh_invalid_lines()` は **await をまたぐので世代トークンで最後の1回だけ書き戻す**（前のタブぶんの結果が後から着地して行ラベルが化けるのを防ぐ）
- タブのストアは**独立した `effectScope(true)` の中で作る**。setup の中で素に `watch` を張ると最初に呼んだコンポーネントのスコープに属し、そのコンポーネントが unmount された時点で**永続化ごと止まる**（メモ帳ダイアログを閉じる／画面を移るで再現する）
- DOM は id 引きではなく**テンプレート ref** で掴む（`TextAreaInfo.text_area_element`）。textarea の `id` は複数枚で重複しないよう `useId()` で採番するので、**E2E はクラス `.kftl_text_area` で掴む**
- タブ列の `×` は `v-btn` ではなく `v-icon` にする（`v-tab` は `<button>` を描画するので `<button>` が入れ子になる）。`@click.stop` が無いと閉じると同時にそのタブがアクティブ化される
- **タブ列はタイトル行に同居させる**（別の行にするとテキストエリアの縦が40px削られる）。`resize()` が引くのは `title_height` だけで、`tab_bar_height` は `v-tabs` の `:height` に渡すためだけの定数。`title_height` の実寸は `.kftl_title` の CSS で固定する（`v-card-title` に `height` prop は無いので、渡すだけでは効かない）。測った値をフィードバックすると `kftl-dialog.vue` の ResizeObserver が縮小ループに入る
- 守るテスト: `kftl-tabs.test.ts` / `kftl-tab-store.test.ts` / `kftl-submit-emits.test.ts` の「KFTLのタブ」「KFTLを複数のウィンドウで開く」/ `e2e/kftl-tabs.spec.ts`

**メモ帳ダイアログの複数枚化**（2026-08-16）。5画面（rykv / mi / dashboard / saihate / plaing）は `<KFTLDialogHost>` を1個置くだけで、＋メニューを選ぶたびにウィンドウが増える。呼び出し側は従来どおり `kftl_dialog.value?.show()` のまま（`show()` の意味が「開く or 再フォーカス」から「1枚増やす」に変わった）。配列の持ち方は `rykv-dialog-host` と同じ。守るべき約束:
- **`useFloatingDialog` のキーはウィンドウごとに分ける。** キーは `${key}:pos` / `:size` / `:transparent` の保存先そのものなので、同じキーで複数枚出すと位置とサイズを奪い合う。スロット番号（空いている最小の番号）で `kftl-dialog` / `kftl-dialog-2` … と分け、`centerOffset` に `slot_index * 28` px を渡してずらす（全ダイアログが `centerMode: "always"` なので、ずらさないとピクセル単位で完全に重なる）
- **z-index は「開いているダイアログの並び順」から出す。単調増加のカウンタにしてはいけない** ―― Vuetify の overlay（メニュー / ツールチップ）が 2400 なので、上へ伸ばし続けるとダイアログの中のメニューが下へ潜る。伸びるのは同時に開いている枚数ぶんだけ
- **前面化は自分と子孫をまとめて上げる。** 確認ダイアログは `Teleport to="body"` で親の**兄弟**になるので、素朴に前面化すると親をクリックしただけで確認が後ろへ隠れる。親子は `provide`/`inject` で持つ（コンポーネント木は Teleport をまたいでも保たれる）
- **バックと Escape が閉じるのは「見た目の最前面」。** `use-dialog-history-stack.ts` の `stack` は積んだ順なので、前面化で見た目とずれる。ずれたままだと奥のダイアログが閉じる。前面化のたびに `raise_dialog_history_entries()` で履歴エントリも同じ順へ並べ替える。2つのコンポーザブルは**同じコンポーネントの setup で呼ばれる**ので、`getCurrentInstance()` を鍵にして結んでいる（`stack` の長さは変わらないので履歴の深さ計算は無傷）
- `closed` は `useDialogHistoryStack(is_show_dialog, { onClosed })` から出す。×・Escape・ブラウザバックのどれでも1回だけ上がるので、ホストはこれ1本で一覧から外せる
- ホストは `v-for` でルートが複数になる。呼び出し側が渡してくる Kyou 系の中継束のうちメモ帳が出さないイベントは行き場が無いので、`defineOptions({ inheritAttrs: false })` で黙って捨てる（従来と同じ挙動）
- E2E で `clickFabButton()` を使ってはいけない ―― 先に `dismissFloatingDialogs()` を呼ぶので、開いているメモ帳ウィンドウを閉じてしまい枚数が増えない
- 守るテスト: `floating-dialog-z-order.test.ts` / `kftl-dialog-host.test.ts` / `e2e/kftl-multi-dialog.spec.ts`

**ポート（開発コード rudbeckia）に画面をウィンドウとして載せる**（2026-08-17）。`/rudbeckia` は背景と FAB だけの1画面で、ライフログビュー / タスク / 実行中 / ダッシュボードをフローティングウィンドウとして開く。**開発コード `rudbeckia` は URL・ファイル名・識別子・保存キー・マニュアルのファイル名に使い、「ポート」は i18n の値とマニュアル本文にだけ出す。** 守るべき約束:
- **ホストするのはページではなくビュー**（`rykv-view` / `mi-view` / `plaing-time-is-view` / `dashboard-view`）。ページは ApplicationConfig の取得・テーマ・`useConfigStructSync`・メッセージ表示・`resize_content`・`reset_dialog_history` を持つので、ダイアログの中に N 個置くわけにいかない。ポートがその1つぶんを担う
- **`v-app-bar` / `v-navigation-drawer` / `v-main` を持つビューは `<v-layout :height :width>` で包む。** Vuetify は親レイアウトがあると `rootZIndex` を 100 下げ、レイアウト項目を `fixed` から `absolute` へ切り替える（`vuetify/lib/composables/layout.js:94,211,262`）。包まないと**画面最上部へ飛んでポート自身の上に重なる**。レイアウト部品を持たない実行中(plaing)は包まない
- **包んだらビューのルートを箱いっぱいに重ねる。** `absolute` になったバー類の基準は `<v-layout>` ではなく最も近い位置指定済み祖先＝`.rykv_view_wrap` 等。`.dashboard_view_wrap` / `.plaing_timeis_view_wrap` は `position: relative` を持たないので埋め込み用の非スコープ CSS で付ける
- **ビューの scoped CSS に `100vh` / `100vw` とメディアクエリを書かない。** ダイアログの中では基準が画面ではなく箱。`props.app_content_*` から `v-bind` する
- **`drawer_mode_is_mobile` は `props.app_content_width` の computed。** 初期化時の1回代入だとダイアログをリサイズしてもドロワーの一時表示モードが切り替わらない。rykv と mi は対称なので必ず両方へ
- **ホストしたビューは自前の FAB を出さない**（`is_hosted_in_dialog`）。`.position-fixed` は `position: fixed` なのでダイアログを抜けて画面右下へ居座り、ポートの FAB と重なる。ついでに打刻メモ帳ダイアログの二重 FAB もこれで消えた
- **ポートの FAB は `.position-fixed-rudbeckia`（`z-index: 2000`）。** 素の `.position-fixed` だとウィンドウ（`z-index: 1100 + 並び順`）に覆われ、**唯一の操作導線が押せなくなる**。Vuetify の overlay(2400) は追い越さないこと（追い越すと FAB 自身のメニューが下へ潜る）
- **ホストしたビューで Enter / Ctrl+V を登録しない。** `useScopedEnterForKFTL` / `useScopedCtrlVForClipboard` は `window` にキャプチャで張るので、4画面ぶん登録すると1回の Enter でメモ帳が4枚開く
- **ホストしたビューの画面切替メニューは `router.replace` しない。** `navigate_to_page` は `reset_dialog_history()`（モジュール共有）を呼ぶので、ポートで開いている他のウィンドウまで一斉に閉じる。`requested_navigate_page` を上げてホストに決めさせる
- **位置とサイズの保存キー（`slot_index`）と、中央からずらす段数（`cascade_index`）は別物。** 前者は種類ごと、後者は種類をまたいだ採番。ずらす量を `slot_index` で決めると4種類とも 0 になり、**4枚が完全に重なって1枚にしか見えない**
- **ウィンドウの未リサイズ時サイズは非スコープ CSS で確定させる**（Teleport 先には `data-v-` が付かない）。確定させたぶん ResizeObserver の実測をそのまま子へ渡す。`kftl-dialog.vue:72-82` の「`userSize` が無いときは既定値」ガードと**併用しない**（併用すると固定した高さが無視される）
- **ウィンドウの中の `v-card` を一括リセットする規則は「自前でレイアウトを組むカード」を壊す。** ポートは `.rudbeckia-page-dialog .gkill-floating-dialog__body .v-card { display: block; … }` で App.vue の「中身はカード1枚」前提の規則を打ち消しているが、これは詳細度3で **Dnote のルート（`v-card.dnote_view`）にも当たる**。Dnote は `fill_height` のとき自分が flex column の器になって残り高さを集計リストへ配る作り（`.dnote_list_view` = `height:0` + `flex-grow:1`）なので、`display:block` へ戻されると `.dnote-scroll-wrap` までの `height:100%` の鎖が auto に落ち、`flex-basis:0` の `.dnote_list_table_root` が**高さちょうど0**になって**集計リストだけが消える**（集計項目とグラフは自然高さのまま残るので気付きにくい）。例外は `.v-card.fill_height_mode` の1つだけ足してある。守るテストは `e2e/rudbeckia.spec.ts` の「集計ビューの集計リスト区画が高さ0に潰れない」

**複数枚（列状態の分離）**
- **列の検索条件とスクロール位置の保存キーはインスタンスごとに分ける。** `gkill-api.ts` の `set/get_saved_{rykv,mi}_{find_kyou_querys,scroll_indexs}` は `instance_key` を**必須引数**で取る（省略可能にすると渡し忘れが黙って通り、2枚目が1枚目を上書きする）。空文字＝従来キーで、単独ページと1枚目が今までの保存内容を引き継ぐ
- **枝番に uuid を使わない。** `${base}_${N+1}` にすること。uuid だと復元時に同じキーを引けず、列が毎回まっさらに戻ったうえで localStorage のキーが増え続ける
- **slot N を slot 0 の保存値から種付けしない。** `query_id` が重複し、`query_id → 列` の逆引きが別インスタンスへ誤配送する。2枚目は既定クエリ1本で開くのが正しい
- **メモ帳ウィンドウの slot 採番はホスト単位ではなくアプリ全体で1つ**（`use-kftl-dialog-host.ts` のモジュール共有 Set）。ポートではポート自身とホストした各画面が同時に `KFTLDialogHost` を持つので、ホスト内だけの採番では必ず `kftl-dialog` の保存キーが衝突する。ホストの unmount で掴んだ番号を返すこと

**画面間の変更伝播（`classes/kyou-change-bus.ts`）**
- **購読側へ渡してよいのは emit を含まない適用関数だけ。** 中継束（`crudRelayHandlers`）を渡すと適用のたびに `emits(...)` が走ってホストが再 publish し、通知が無限に往復する。そのために `onDeletedKyou` は `apply_deleted_kyou`（適用のみ）と `onDeletedKyou`（適用＋emit）に割ってある
- **通知は seq 付きの追記ログ。** スカラー（最新の1件）だと同じ tick に複数件起きたとき最後の1件しか見えず、残りが黙って落ちる（KFTLの複数行保存が典型）
- **`KyouChangeBus.last_seq` は Ref ではなくメソッド。** Ref をそのまま公開すると、チャネルのオブジェクトが `reactive()` に包まれたとき Vue が自動アンラップして `.value` が `undefined` になり、**伝播が黙って効かなくなる**（テストのハーネスが実際に踏んだ）
- **バスは props で配る。`provide`/`inject` にしない。** 既存のテストは `useRykvView({props, emits})` をコンポーネントインスタンスの外から素で呼ぶので、`inject()` は警告を出して既定値へ落ちる ＝ テストでは伝播が効かないのに緑になる
- **`new_reload_batch()` は発生元のビューが採番**し、自分のローカル適用と通知の両方に同じ値を使う。ホスト側で採番すると `kyou-reload.ts:226` の合流条件（`started_at >= requested_at`）に間に合わず、同じ Kyou を画面の枚数ぶん取りに行く
- **自分が出した通知は受けない**（`origin_id`）。受けると発生元が二重適用する。追加は `insert_kyou_sorted` の id 重複判定で救われるが、削除と引き直しは救われない
- **`reload_list` は1ドレイン1回に畳む。** 畳まないと1回の KFTL 保存で開いている画面ぶんの全件検索が走る
- **後から開いたウィンドウは購読開始時点の最大 seq から始める。** 0 から始めると過去の変更を全部再生する
- **`requested_reload_kyou` を握り潰さない。** タグ・テキスト・通知の変更はこれしか出さないので、配らないと付随データの変更が他の画面に一切届かない
- **`requested_update_check_kyous` は配らない。** 列ごとの選択状態であり、rykv/mi では未実装（throw する）
- ポートのFABから追加した記録も配る（発生元 `rudbeckia-page`）。配らないと「＋から足したのに並べている一覧に出ない」になる
- 守るテスト: `e2e/rudbeckia.spec.ts` / `kyou-change-bus.test.ts` / `kyou-change-propagation.test.ts` / `dashboard-view-reload.test.ts` / `dashboard-page-reload.test.ts` / `column-view-init-source-scan.test.ts` / `kftl-dialog-host.test.ts` / `rykv・mi-view-initial-load.test.ts`（対）

**Mi の板名の並び順と板ツリーのクリック**（2026-08-16）。純関数は `classes/mi-board-names.ts`、守るテストは `mi-board-names.test.ts` / `mi-board-query.test.ts`。
- **板名プルダウンの並び順は ApplicationConfig の板ツリーが正。** `get_mi_board_list` は Go の map を回して集めているので**順序を保証しない**（`dao/reps/mi_repositories.go` / `mi_re_kyou_repositories.go`。interface の doc コメントにも明記）。素で `:items` に渡すと読み込むたびに並びが入れ替わるので、5箇所の `v-select`（add/edit Mi・add/edit MiReKyou・設定の既定の板）はどれも `sort_mi_board_names_by_config_order()` を通す
- 並べ替えは **`computed`** で表現する。`use-edit-mi-view.ts` / `use-mi-re-kyou-schedule-fields.ts` には `application_config` の watch が無く、ある側の watch も**参照同一性**しか見ないので（設定ダイアログでの並べ替えのような deep な変更では発火しない）、代入時に1回ソートすると板を並べ替えても追随しない。API の生の一覧は `mi_board_names_source` に持ち、`update_board_name()` の push 先もそちら
- 設定画面だけは並べ替えの基準が **`cloned_application_config`**。子の板構成ダイアログの「適用」は clone にだけ書くので、props を見ると並べ替え直後に既定板のプルダウンだけ古い順で取り残される
- **設定にしか無い名前を候補へ足さない。** とくに「すべて」は `append_all_mi_board()` が入れる仮想ノードで実在の板ではないので、Mi/MiReKyou の板名候補に混ぜてはいけない
- **サイドバーの板ツリーのルート行では何も開かない**（`resolve_clicked_mi_board_names()`）。ルートは `folder_name=''` で描いていて見た目は空白だが `.tree_item { min-width: 200px }` のクリック領域が残っており、踏むと `click_group_by_user()` が**自分自身の key（`__root__`）を含めて**サブツリー全部の key を上げてくる。素通しすると `__root__` という名前の列 + 板の数だけの列が一度に開く。判定は「フォルダ扱いのノード（`is_dir` か `board_name` が空）の key が混ざっていたらグループ行のクリック → 何も開かない」。**ツリーに無い key は開く** ―― 作った直後で `append_not_found_mi_boards()` がまだ拾えていないだけかもしれず、落とすと「板をクリックしても何も起きない」（エラーも出ない）になる
- 共有の `use-foldable-struct.ts` の `click_group_by_user` は**触らない**。自分の key を含める挙動は tag/rep/timeis のチェックボックス経路も使っており、そちらは「フォルダ行クリックで配下を一括チェック」が意図された機能（`foldable-struct-check.test.ts` が等価性を固定している）。直すのは読み取り側（下記）

**ツリーの「入れ物」を検索条件へ混ぜない**（2026-08-16）。判定は `pages/views/foldable-struct-model.ts` の **`is_struct_container_node(struct)`**（`is_dir` またはキーが `__root__`）の1つだけ。守るテストは `foldable-struct-selected-items.test.ts` / `collect-inited-tag-names.test.ts`。
- ルートとフォルダは並べ替えのための器でしかないのに、`key` にはフォルダ名が（ルートは `__root__` が）そのまま入る。チェックの入ったノードの key は**そのまま検索条件（`tags` / `reps` / `devices_in_sidebar` / `rep_types_in_sidebar` / `timeis_tags`）として流れる**ので、入れ物を混ぜると実在しない名前の条件が紛れ込む。**OR検索では無害だが AND検索（`tags_and` 等）では必ず0件になる**ため、症状が出るときだけ出る
- ルート行は `folder_name=''` の空白帯（`.tree_item { min-width: 200px }`）としてクリックでき、踏むと `click_group_by_user` が `__root__` を載せてくる。**誤クリックだけで条件に入る**
- 除外する場所は4つ: `use-foldable-struct.ts` の `get_selected_items()`（対話経路。tag/rep/timeis の3コンポーザブルが**唯一この関数から**条件を読む）、`collect-inited-tag-names.ts`、`find-kyou-query.ts` の `device_name_walk` / `rep_type_name_walk`（既定クエリ経路）、同 `apply_rep_summary_to_detaul` の `collect_checked_keys`。**最後のものは `indeterminate=false` のクリアが入れ物にも要るので、walk は打ち切らず集合へ入れないだけにする**
- **フォルダ名と同名のタグが実在しても条件は落ちない。** `apply_check_state_to_struct` が key 一致でツリー全体を走査して葉のほうにもチェックを入れるため。実運用の `TAG_STRUCT` ではフォルダの大半が同名タグを持たない純粋な入れ物で、しかもその一部は `check_when_inited=true` で保存されている ―― つまり誤クリックしなくても既定の検索条件に幽霊タグが入りうる
- 入れ物かどうかを `is_dir` だけで見ないこと。保存済みJSONのルートに `is_dir` が無い実例があり（`gkill-api.ts` は `children` が falsy のときしか立てない）、そのときルートは**葉として描かれて `__root__` がそのまま条件に入る**

**Context menus:** never compute the popup position by hand. Call `useContextMenuPosition()` (`classes/use-context-menu-position.ts`) for `is_show` / `menu_target` / `open_at(e)`, and bind the template as `<v-menu v-model="is_show" :target="menu_target" location="bottom start">`. Vuetify's connected location strategy measures the rendered menu and flips/shifts it to stay inside the viewport. The previous approach copy-pasted `left: min(innerWidth - 130, x); top: min(max(50, innerHeight - (8 + 48 * N)), y)` into all 25 context-menu composables — the 130px width was unrelated to the real list width, and `N` had to be hand-synced with the template's item count (the struct-family menus sat at `N=2` while actually having 5 items). `.gkill_context_menu_list { max-height: 70vh; overflow-y: scroll }` in `App.vue` still caps very long menus.

**チェックツリーへの適用は単一パスで:** サイドバーのチェック状態をツリーへ書き戻すときは `classes/foldable-struct-check.ts` の `apply_check_state_to_struct(root, items, state, pre_uncheck_all)` を使い、**「項目1つごとにツリー全体を再帰走査」を書かない**。旧実装は O(項目数×ノード数) で、全ノード訪問が Vue の deep reactive proxy 越しになるため、rep 376個の実環境では列フォーカス切替の click が同期6.5秒（トレース実測）に達していた（2026-08-10 修正。rep/tag/timeis の3コンポーザブルと `apply_rep_summary_to_detaul` が対象。旧実装との等価性は `foldable-struct-check.test.ts` が担保）。なお `InfoBase` 系データクラスに TS `private` フィールドを足してはいけない —— `ref()` の UnwrapRef が private を落とし `Ref<Array<Kyou>>` への代入が全所で型エラーになる（ES `#` も reactive Proxy 越しの `this` で壊れる）。内部フィールドは underscore 公開 + getter/setter にする。

**端末種別の判定:** `'ontouchstart' in window || navigator.maxTouchPoints > 0` を手書きしない。`useDeviceKind()`（`classes/use-device-kind.ts`）の `device_kind` / `is_pc` / `is_tablet` / `is_smart_phone` / `has_touch` を使う。**ドラッグ&ドロップの可否は `is_pc`、タッチ端末向けの代替導線（ロングプレスでの contextmenu 補完など）は `has_touch` と、見る値が違う**。以前は「タッチできるか」の2値を両用途に兼用していたため、タッチパネル搭載 Windows ノートで D&D が無効になっていた。判定は「UAでスマホ確定 → `any-pointer: fine` かつ `any-hover: hover` なら PC → UAでタブレット確定 → 画面短辺600px」の順で、この順序自体が仕様（iPad + トラックパッドは PC 扱い、スタイラス対応スマホはスマホ扱い）。状態はモジュールレベルのシングルトンで、`foldable-struct.vue` のようにノード数ぶん再帰インスタンス化されるコンポーネントから呼んでも購読は2本のまま。純関数 `classify_device_kind(env)` が判定の実体で、テストはここを突く。D&D を無効化してもコンテキストメニューの「上へ / 下へ / フォルダへ移動」（`classes/foldable-struct-move.ts`）で操作できる。

**ダイアログの自動フォーカス:** ダイアログを開いたら本文の最初のテキスト入力欄にカーソルが載る。実装は `useFloatingDialog` の `autofocus` オプション（既定 true）1箇所で、候補選びは `classes/dialog-autofocus.ts` の純関数 `find_autofocus_target()`。**view 側に `autofocus` を撒いてはいけない** ―― 入力欄を持つ view はサイドバーやページ直下でも使われるので、ページ読込時にフォーカスを奪う。探索は `.gkill-floating-dialog__body` の中だけ（ヘッダには透過トグルの `v-checkbox` と×が必ず先頭にあり、ルートから探すと必ずそれを掴む）。既に `autofocus` を書いた要素があるダイアログでは何もせず Vuetify に任せる。`readonly`（日付ピッカーの見せかけ入力）・`disabled`・非表示・`.v-selection-control` 配下・`inputmode="none"` の `v-select` は候補から外す（`v-autocomplete` / `v-combobox` は打ち込めるので対象）。入力欄が内側の `v-if` で遅れて生えるダイアログのために `MutationObserver` で2秒だけ見張り、一度当てたら切る。自前でフォーカス先を決めているダイアログ（`save-clipboard-to-file-dialog` は保存ボタン）は `autofocus: false` で切る。

**行判定の高さにパーセントを渡さない:** `classes/kyou-row-height.ts` の `is_row_height()` は高さを `Number.parseFloat` して 120 未満なら一覧の行とみなす。`parseFloat('80%')` は **80** なので、詳細ペインやダイアログでパーセントを渡すと行扱いになり、`mi-re-kyou-view.vue` の参照先ブロック（`v-if="!is_compact"`）が丸ごと消える。行ではない場所は `'unset'` か `'auto'` を渡すこと。例外は画像一覧（`kyou-list-view.vue` の `is_image_only`）だけで、200pxのセルに詰めるため意図的に `'100%'` を渡している。`__tests__/unit/classes/kyou-view-height-source-scan.test.ts` がソース走査で守る。

**設定は「適用」を押すまでサーバへ送らない:** ApplicationConfig の子ダイアログ（tag/rep/rep_type/device/kftl_template/mi_board の struct 系6つ + Dnote / Ryuu / Dashboard / PlaingTimeIs）の「適用」は**組み立てだけ**で、API を1つも呼ばない。送信は `use-application-config-view.ts` の `update_application_config()` の1箇所に閉じている。子の適用先は `cloned_application_config` **だけ**で、`props.application_config` を直接書いてはいけない（書くと設定画面のキャンセルが効かなくなる）。props の差し替え（板ツリー/タグツリーの追随）で未適用の編集が消えないようにするのは `has_pending_child_edits` の役目。ロケールとダークテーマは選ばせるために即時プレビューするので、`cancel_pending_changes()` で開いた時点の値へ戻す（×・Escape・キャンセルのどれでも通るよう `application-config-dialog.vue` の `hide()` から呼ぶ）。**例外は ServerConfigDialog** ―― 別エンティティ（`update_server_config`）なので自分の「適用」で送る。

**Naming convention (files):** `{action}-{feature}-{entity}-{component}` (e.g., `add-dnote-item-view.vue`, `confirm-delete-ryuu-item-dialog.vue`), kebab-case. Dnote and Ryuu follow the same pattern.

**Frozen spellings:** `plaing` / `Plaing` は "playing" の誤綴りではなく**製品綴りとして凍結**（ルート `/plaing`、`FindQuery` の `plaing_time`、MCP ツールスキーマ、Wear OS データレイヤーパス、マニュアルのページ名、`default_page` の保存値に浸透しているため）。綴り修正の提案はしないこと。詳細は `documents/reverse/glossary.md` の「凍結された綴り」節。

**Naming convention (identifiers):** データクラスのプロパティ/メソッド・ローカル変数・通常関数は snake_case（Go 側 JSON タグとの写像）。コンポーザブルは `useXxx`、イベントコールバックは `onXxx`、CRUD リレーハンドラ束は `xxxHandlers`（束の生成は `kyou-view-relay.ts` に一元化。いずれも camelCase）。型は PascalCase、enum メンバーは snake_case。`@typescript-eslint/naming-convention` で機械検査される（`eslint.config.js` の `app/naming-convention` ブロック。対象は `src/client` 本体のみで、`__tests__`・`src/mcp`・`src/tools`・`*.d.ts` は別流儀として対象外）。

**i18n:** 7 languages (ja, en, zh, ko, es, fr, de) in `src/locales/`. 919 keys per locale. Flat key-value JSON. Shared between frontend (import) and backend (Go embed).

### MCP Server — `src/mcp/`

3つのMCPサーバーを提供。いずれもstdio (ローカル) / HTTP (OAuth 2.1認証付きリモート) の2モード対応。

| Server | Tools | stdio | HTTP | Port |
|---|---|---|---|---|
| Read | 9 (get_kyous, get_mi_board_list, get_all_tag_names, get_all_rep_names, get_gps_log, get_application_config, get_idf_file, get_idf_file_path + plugin 1) | `npm run mcp:gkill-read` | `npm run mcp:gkill-read-http` | 8808 |
| Write | 24 (9 add + 1 submit_kftl + 1 delete + 9 update + 3 read convenience + plugin 1) | `npm run mcp:gkill-write` | `npm run mcp:gkill-write-http` | 8809 |
| ReadWrite | 29 (read 8 + write 20 + plugin 1) | `npm run mcp:gkill-readwrite` | `npm run mcp:gkill-readwrite-http` | 8810 |

**Plugin tools** — 3サーバ共通は `lib/plugin-tools.mjs` の `gkill_get_plugin_list` 1つだけ（読み取り専用。`post_plugin_config` は公開しない）。プラグインKyouの本文は gkill に保存されておらず、`get_kyous` が返すのはメタデータと `rep_name`/`kyou_id` を持つ `payload.kind="plugin"` だけ。本文は **`gkill_get_kyous` に `include_plugin_content:true` を渡して同じレスポンスに埋め込ませる**（1件ずつ取る `gkill_get_plugin_content` ツールは廃止。AIの往復が N+1 回になり非効率だったため）。インライン化は `plugin-tools.mjs` の `inlinePluginContents` が担当し、`/api/get_plugin_content_html` を rep ごとに直列・rep 間は並列（既定4）で叩き、`lib/html-text.mjs` でHTML→テキスト変換して `payload.content_text` に入れる。各ペイロードには `content_status`（`ok`/`truncated`/`skipped`/`error`）が付き、`ok` 以外なら本文は不完全。上限は1件4000文字（`plugin_content_max_text_length` で最大200000まで）・合計200000文字・20件・30秒。`plugin_content_format` で `text`（既定）/`html`/`both` を選べる。

> **同一プラグインへ並列に投げてはいけない。** プラグインの stdio は1本しかなく呼び出しは直列化されるので、同時発行しても速くならず順番待ちが伸びるだけ。待ちきれなかったぶんは `ErrPluginBusy` になる。インライン取得が実行中のリクエストを abort しないのも同じ配慮（abort 自体は現在プロセスに影響しないが、MCPサーバは古い gkill にも接続しうる）。デッドラインは「新しいリクエストを始めない」だけで実現している。Write専用サーバには `gkill_get_kyous` が無いため、プラグイン本文を読むには ReadWrite サーバを使う。
>
> （2026-08-06以前の gkill は30秒デッドラインを排他ロックの**前**に張っていたため、同時発行するとロック待ちで期限を食い潰し、期限切れ時の `Process.Kill()` でプラグインプロセスが回収されていた。現在は期限をスロット取得後に張るのでこの誤射はない。）

**Environment variables:** `GKILL_BASE_URL`, `GKILL_USER`, `GKILL_PASSWORD_SHA256`, `MCP_LOG` (default `info`), `GKILL_MCP_MAX_FILE_BYTES` (default 8MB — cap for `get_idf_file` base64), `GKILL_MCP_FILE_LINK_TTL_MS` (default 1h — HTTP-mode file URL token TTL), and for HTTP mode: `MCP_TRANSPORT`, `MCP_PORT`, `MCP_OAUTH_ISSUER`. Also read: `GKILL_PASSWORD` (plaintext alternative to the SHA256), `GKILL_SESSION_ID`, `GKILL_INSECURE`, `GKILL_LOCALE`, `GKILL_FETCH_TIMEOUT_MS`, `GKILL_HOME`.

**IDF file access for AI clients** (3 paths, preferred order): (1) `file_path` — `get_kyous` includes the absolute local path in IDF payloads for **stdio** clients (same machine); read it directly. (2) `file_url` / `file_url_full` — for **HTTP** clients, `get_kyous` includes a per-file, expiring, unguessable public URL served by the MCP server's own `GET /files/{token}` route (`FileLinkStore`, `lib/file-link-store.mjs`); fetch with no auth, any size (images: `file_url` is a thumbnail, `file_url_full` the original). (3) `gkill_get_idf_file` — base64 fallback, size-capped. Local paths never go to remote clients; the transport type (stdio vs HTTP) is the gate. gkill itself stays private — only the MCP server's public surface is used.

**HTTPモードの1リクエスト文脈は `server.current*` 共有フィールドに書かず、不変の `requestContext={sessionId,userId,remoteAddr}` を `handlePayload→handleMessage→handleToolCall/buildToolResult` へ引数で流す**（2026-08-21、監査 C-02）。以前は `_lastTokenUserId` と `server.currentSessionId/currentUserId/currentRemoteAddr` に書いて await をまたいで読んでいたため、並行リクエストで別要求の user/session が混線し、他人のセッションに紐づく file-link URL 発行や書き込みレコードの作成者誤記が起きた。`mcp-server-base.mjs` の入口で `requestContext ?? Object.freeze({...this.current*})` にフォールバックするので stdio と既存の直接呼び出しテストは無改修。**http-transport 側から `server.current*` への書き込みを復活させないこと**（渡し忘れは sessionId=null→gkill側401で顕在化し、静かな混線には戻らない）。OAuth は S256 必須・未登録 client_id は認可拒否（`oauth-server.mjs` の `_validateAuthorizeParams`）、公開ファイル配信は nosniff + CSP sandbox（Go 側 `withUserContentSecurityHeaders` のミラー）、`oauth-store.mjs` の保存は temp+rename の 0600。守るテストは `src/mcp/__tests__/http-transport.test.mjs`（Bearer 401 = C-01 回帰・並行分離 = C-02 回帰・M-06）。

**共有ページのファイル配信は共有クエリの結果に含まれるファイルだけ**（2026-08-21、監査 C-03）。`handle_file_serve.go` の `sharedID != ""` 経路は、rep名一致だけで IDF rep へ委譲せず、`shared_file_authz.go` の `collectSharedIDFFilePaths` で共有 `FindQuery` を再評価して許可パス集合を作り、要求パスが集合に無ければ403にする（`handle_get_shared_kyous.go` と同一手順を共有＝一覧を2箇所で維持しない）。パス正規化は `idf_file_url.go` の `cleanRelativeURLPath` と同一。セッション経路はフルアクセスのまま（`sharedKyouInfo != nil` のときだけゲート）。守るテストは `shared_file_authz_test.go`。

**利用者入力URL・そのページが指す og:image / #landingImage の取得は必ず `api/safefetch` を通す**（2026-08-21、監査 H-04）。`safefetch.GetCapped`（scheme検査・`Dialer.Control` での接続先IP検証・サイズ上限）と `safefetch.CheckImageDimensions`（`image.DecodeConfig` で復号前に総ピクセル検査）で SSRF・無制限read・画像爆弾を防ぐ。`dao/reps/ur_log.go` の getBody/getFavicon/getImageOG/getAmazonImage と `gkill_server_api` の `httpGetBase64Data`（ブックマークレット）が利用。既定は private 拒否（loopback/RFC1918/link-local(メタデータ)/multicast/unspecified）。`http.Get` を新しく直に書かないこと。守るテストは `api/safefetch/safefetch_test.go`。

**型別 `GetXxx(id, nil)` は最新版を返す**（2026-08-21、監査 H-07）。`dao/reps` の各 `*_repository_sqlite3_impl.go` / `*_repository_cached_sqlite3_impl.go` の単体取得は `onlyLatestData := query.OnlyLatestData`（`false` 固定にしない。`GenerateFindSQLCommon` はこの引数しか見ない）と `slices.MaxFunc(UpdateTime)`（`&xxx[0]` を返さない）で最新版を選ぶ。`GkillRepositories.GetKyou` はアドレス表 nil でも panic しない（プラグインKyou・追加直後の記録）。守るテストは `get_typed_latest_version_test.go` / `gkill_repositories_get_kyou_test.go` / `rows_err_check_test.go`。

**外部URLを取る新規コードや子goroutine内の index 参照に注意**: `strings.SplitN(x, ",", 2)` は末尾要素 `parts[len(parts)-1]` を取る（`[1]` を長さ未確認で参照しない。子goroutine内 panic は `recoverMiddleware` で回収できずプロセスが落ちる＝監査 H-01）。ファイル書き込みは tmp+rename で原子的に（Override で原本を壊さない＝M-03）。ZIP展開は件数・総展開量・圧縮比の上限つき（`handle_browse_zip_contents.go` の `maxZip*`＝M-04）。`http.Server` は `ReadHeaderTimeout`/`IdleTimeout`/`MaxHeaderBytes` を張り、認証前ボディは `maxAuthBodyBytes` でキャップ（H-03）。全レスポンスに `securityHeadersMiddleware`（nosniff / X-Frame-Options / Referrer-Policy）。CLIサブコマンドは `InitGkillServerAPI` 失敗で `return`（nil panic 回避＝M-8）、CLIの自己発行セッションは `IsLocalAppUser=false`（最小権限）。ログインは非存在ユーザとパスワード誤りを同じ error_code + 文言に統一し、非存在時もダミー Argon2id を実行（ユーザ列挙対策＝S3-login）。

**IDF走査の一時停止は参照カウント**（2026-08-21、監査 M-02）。`SetSkipIDF(true/false)` は共有 `*bool` ではなく `*atomic.Int64` を増減し「カウント>0 で skip」。重なるアップロード（と UpdateCache）が互いのフラグを倒し合って watcher が走る/変更を取りこぼすのを防ぐ。カウントが0へ戻ったときだけ catch-up 走査を1回キックする（`UpdateCache` の Add(-1) では catch-up しない＝1分周期のリビルド無限ループになるため。`SetSkipIDF` はアップロードハンドラからしか呼ばれず watcher から再入しない）。

**CLIサブコマンドは `RunE` + `SilenceUsage/SilenceErrors`**（2026-08-21、監査 M-8）。失敗で exit 1（main の `log.Fatal` が唯一のエラー出力）。ユーザごとのループは `errors.Join` で集約し、途中失敗でも成功分の結果（reset_password の URL 等）はその場で即出力してから続行する（`os.Exit` を足すと defer のセッション後始末が飛ぶので RunE 経由で返す）。**互換注意**: `SyncDatas` 等が update_cache の失敗を初めて exit code で観測する。`auto_tag` は長時間実行向けに `issueLocalSession` の `refresh` でセッション期限を延長する。

**プラグインの重い構築は常駐ビルダ + WAL + バッチcommit**（2026-08-21、監査 M-6）。claudecode も codex/fitbit と同じく `builder.go`（`EnsureStarted`/`Kick`/`loop`、mu(DB初期化)/buildMu(構築)分離で読み取り無待機、WAL 自前DSN、`cache_meta` に進捗）へ移行済み。`GetMessages`/`GetMessage`/`GetStats` は refresh を呼ばず現キャッシュ即答+Kick。同期・単一tx構築（デッドラインkill→進捗ゼロループ）を新規に書かないこと。

### Mobile — `src/android/`, `src/wear_os/`

**Android同梱サーバはループバック限定**（2026-08-21、監査 S3-android-main）。`MainActivity` の ProcessBuilder は `--address 127.0.0.1:9999` を渡す（無指定だと全インターフェース待受＝LANの第三者が無認証で全記録を読み書きできる）。activity に `configChanges`（回転で Activity を再生成させず、SQLite書き込み中の `kill -9` を防ぐ。起動はポート先行プローブで既存サーバを再利用）と `onBackPressedDispatcher`（WebView goBack）。cleartext は `network_security_config.xml` で localhost 限定、外部ストレージ権限は `maxSdkVersion` 付き・起動ゲートは非ブロッキング（M-15）。**Wear companion の TLS は TOFU/ピン留め**（H-05）: `GkillServerTrust.kt` がプラットフォーム既定で検証し、失敗時のみ保存済み SHA-256 フィンガープリント一致で許可（trust-all は全廃）。ピン学習は companion の「保存&接続テスト」で利用者承認時のみ。ウォッチ→電話の送信は `WearRequestWorker`（WorkManager）で Service 破棄を跨ぎ、`WearSubmitLedger` で重複再配送を確認へ回す（S3-wear）。KFTL送信にはサーバ側冪等キーも付く: `GkillWearableListenerService` がメッセージ1件ごとに UUID を採番して WorkRequest の不変入力に載せるので、同じ要求のワーカー再送では同じキーになり `handle_submit_kftl_text.go` の `kftlIdempotencyStore`（TTL 10分・成功時のみ記録）が二重登録を畳む。意図的な再送は別メッセージ＝別キーなので畳まれない。**冪等キーを内容ハッシュにしないこと**（意図的な同一内容の再送が畳まれて記録できなくなる）。**ハンドラでの `markDone` 配線を落とさないこと** ―― ストア単体テストは `markDone` を直接呼ぶので配線漏れを見逃す（ビルドも vet も素通しする＝`idempotencyKey` は `alreadyDone` 分岐で使われ未使用にならない）。守るテストは `kftl_idempotency_test.go`（ストア単体）/ `handle_submit_kftl_text_test.go`（2回叩いて2回目が畳まれる end-to-end。同一キー=1件・別キー=2件・キー無し=2件）/ `WearRequestHandlerTest.kt`。

**Android**: APK wrapper (WebView) bundling the gkill_server binary as `jniLibs/arm64-v8a/libgkill_server.so` and exec'ing it from `nativeLibraryDir` — required because targetSdk 29+ forbids executing files under the app's data dir (W^X). Needs `packaging { jniLibs { useLegacyPackaging = true } }` so the `.so` is extracted as a real file. compileSdk 37 (androidx 1.19.x requires it), targetSdk 36, minSdk 26. **Wear OS**: Gradle multi-module project (phone_companion + watch_app), communicates via Wearable Data Layer. The Gradle wrapper is committed under `src/wear_os/`, so no copying is needed; `npm run setup_wear_os_gradle` re-syncs it from `src/android/` if it ever breaks.

## Lint & Code Quality

ESLint 10 flat config (`eslint.config.js`). Zero violations policy (levels per rule as listed below).

| Rule | Level | Note |
|---|---|---|
| `@typescript-eslint/no-explicit-any` | error | `any` 使用禁止。`unknown` or 具体的型を使うこと |
| `@typescript-eslint/no-empty-object-type` | error | 空の `{}` 型禁止。`type X = ParentType` を使うこと |
| `@typescript-eslint/no-unused-vars` | warn | `_` プレフィックスで無視 (`argsIgnorePattern: '^_'`) |
| `@typescript-eslint/naming-convention` | error | 識別子の命名規約（上記 Naming convention 節）。対象は `src/client` 本体のみ |
| `playwright/no-conditional-in-test` / `no-wait-for-timeout` | error（全 spec） | 「静かに成功するテスト」の防止。免除リストは撤去済み。CI は `npx eslint --max-warnings 0` |

Go: `slices.SortFunc` (not `sort.Slice`), `for range n` (not `for i := 0; i < n; i++`), `any` (not `interface{}`), `errors.Join` for multi-error collection.

## Language

The codebase (variable names, comments, commit messages) is primarily in Japanese. README and documentation are in Japanese.

**Comment language policy** (現状追認、2026-08 明文化): Go 本体（`src/server/gkill/`、kftl を除く）と `src/plugins/` は日本語。`src/client/` は「構造バナーコメント（`// ── Template refs ──` 等）は英語、処理説明は日本語」の複合。`src/mcp/`・`src/wear_os/`・`src/server/gkill/api/kftl/`（TS 版からの移植、`// Mirrors:` 注記付き）は英語容認。新規コメントは各領域の既存スタイルに合わせること。

## Documentation

- `resources/manual/` — HTML manuals (7 languages, 22 pages per language), embedded via `//go:embed` and served at `/resources/manual/`
- `documents/reverse/` — Reverse-engineered design documents (24 files). See `documents/reverse/README.md` for index. Key files: glossary.md (96 terms), api-endpoints.md (92 endpoints, 90 registered), usecase.md (87 use cases), sequence-diagrams.md (29 diagrams), scenario.md (cross-channel end-to-end usage scenarios with UML), testing-guide.md. `npm run verify_docs` (`src/tools/verify_docs.mjs`) machine-checks the counts, cross-links, referenced paths, Mermaid blocks, and manual freshness — it runs as part of `npm test`, so update the docs when a count changes.
- `src/ABOUT_TEST.md` — Test specification index, links to 23 subdirectory `ABOUT_TEST.md` files
- **What `verify_docs` covers** (extend it rather than hand-maintaining new numbers): file/test counts computed from the tree (`--list` prints them all), count assertions across `documents/reverse/*.md` + `CLAUDE.md` + `src/**/README.md` + `ABOUT_TEST.md`, markdown link resolution for that same set, `src/...` path references (warning; the backtick scan strips fenced blocks first — pairing across a ``` fence used to hide 65% of them), **a check that every filename mentioned in a doc actually exists** (this one alone caught 17 ghost entries left behind by deletions; `_`-prefixed suffix patterns and `xxx` placeholders are exempt), Mermaid block types, manual generation freshness / language page-set parity / a11y invariants / intra-manual links, a **terminology lint** that rejects internal code names (`IDF`, `WAN`, `Kyou`, `MiReKyou`, `Dnote`, `rudbeckia`, …) in `resources/manual_src/` outside `<code>` spans and `href`/`src` values, and a check that every `screen_name` the app passes to `HelpDialog` has a matching manual page. `--parity` (opt-in) reports per-page h2/h3/table drift against the Japanese original.
