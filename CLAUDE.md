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
| `npm run verify_docs` | Docs CI: checks doc counts against code, cross-links, referenced paths, Mermaid blocks, manual freshness. `--list` prints the measured metrics |
| `npm run test_server` | Go tests (`cd src/server && go test ./...`) |
| `npm run test_client_unit` | Vitest unit tests |
| `npm run test_client_e2e` | Playwright E2E tests (gkill_server + Vite を空きポートで自動起動・停止、`$HOME/gkill_test`使用) |
| `npm run test_mcp` | MCP server tests (Vitest) |
| `npm run release` | Cross-compile release for all platforms |

**Build pipeline** (`install_server` → `prepare_install`): `clean_dist` → clean embed dir → write `version.json` (commit hash + build time + version) → `vue-tsc` type-check → `vite build` → copy `dist/` to `src/server/gkill/api/embed/html/` → copy `src/locales/` to `src/server/gkill/api/embed/i18n/locales/` → `build_manuals` (`src/tools/build_manuals.mjs`: `resources/manual_src/` → `resources/manual/`) → copy `resources/manual/` to `src/server/gkill/api/embed/manual/` (3 copies: html + i18n + manual) → `go install`

**Prerequisites:** Go 1.26.4+ (`src/server/go.mod` declares `go 1.26.4`), Node.js 20.19+ (24.x recommended — `package.json` has no `engines` field, so this is not enforced), `npm i`

**Dev server backend target:** `npm run dev` runs `src/tools/dev.mjs`, a thin wrapper that strips `--api` / `--api-target` (Vite errors on unknown CLI options) and passes the value to Vite as `GKILL_API_PROXY_TARGET`; all other args pass through untouched. Accepts a full URL, a bare port (`--api=19999` → `http://127.0.0.1:19999`), or `host:port`. Precedence: `--api` > `GKILL_API_PROXY_TARGET` > `http://localhost:9999`. `vite.config.ts` proxies `/api`, `/files`, `/zip_cache` and `/resources/manual` to that target — SPA routes (`/rykv`, `/kftl`, `/mi`, …) are deliberately not proxied since vue-router owns them in dev.

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
              # The three shippable plugins (chatgpt / claudeai / claudecode — gkill_example is
              # excluded) take `source_dirs` in config.json (folders/globs), auto-create
              # that config.json next to manifest.json on first start (existing files are never
              # overwritten, via sdk.EnsureConfig + Handler.DefaultConfig), and can print their
              # embedded manifest.json / default config.json via --gkill-print-manifest /
              # --gkill-print-config. Their SQLite caches live under gkill's own cache dir
              # ($GKILL_HOME/caches/plugin_cache/{userID}/{pluginName}/cache.db — resolved in
              # each plugin's cache_path.go from the inherited GKILL_HOME env var, falling back
              # to the plugin folder), so `clear_cache plugin` can wipe them
```

### Two Deployment Modes

- **`gkill_server`** (`src/server/gkill/main/gkill_server/main.go`): Headless HTTP server (browser/PWA)
- **`gkill`** (`src/server/gkill/main/gkill/main.go`): Desktop app wrapping server with go-astilectron

Both use cobra for CLI with shared subcommands: `version`, `dvnf`, `generate_thumb_cache`, `generate_video_cache`, `optimize`, `update_cache`, `clear_cache`, `reset_password`. `idf` is registered by `gkill_server` only (`main/gkill/main.go` does not register `IDFCmd`). Default listen address: `:9999`, TLS disabled, initial user `admin` with no password.

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

Module: `github.com/mt3hr/gkill/src/server` (Go 1.26.4)

Key packages:
- `gkill/api/` — Shared infrastructure: `embed.go` (`//go:embed` serves Vue SPA at `/`), `version.go`, `gkill_version_data.go`, `find_filter.go`, `find_filter_helpers.go`, `find_kyou_context.go`
- `gkill/api/gkill_server_api/` — HTTP API handlers (94 files incl. tests, 1 handler per file). `GkillServerAPI` struct with `serve.go`, `close.go`, route definitions in `gkill_server_api_address.go`. Auth middleware (`auth.go`, `auth_context.go`, `auth_middleware.go`) extracts session→account→device→repositories via `AuthContext`, `authMiddleware`, `authWithReposMiddleware`. Handler registration uses wrapper functions: `wrapNoAuth` (no session), `wrapAuth` (session + account), `wrapAuthRepos` (session + account + device + repositories). Utility files: `filter_local_only.go`, `utils.go`, `web_push.go`. ZIP browsing: `handle_browse_zip_contents.go` (path traversal prevention, Shift_JIS→UTF-8, singleflight dedup).
- `gkill/api/req_res/` — Request/response structs for every endpoint (182 files)
- `gkill/api/kftl/` — KFTL custom text format parser (single package, no sub-packages). Supports both Japanese (。！？、ーー etc.) and ASCII (#!?,-- /mi /mood /expense /num /url /start /end /timeis /end? /endt /endt?) prefixes
- `gkill/api/gkill_plugin/` — Plugin protocol types: `PluginManifest`, `PluginRequest`, `PluginResponse`, `PluginKyou` (stdio newline-delimited JSON)
- `gkill/plugin/sdk/` — Plugin author SDK. `sdk.Run(sdk.Handler{...})` starts the stdio JSON message loop. `Handler` has 7 fields: `FindKyous` (required), `GetKyou`, `GetContentHTML`, `GetConfigHTML`, `PostConfig`, `RepName`, `DefaultConfig`. Plugins are standalone binaries in `src/plugins/`
- `gkill/usecase/` — HTTP-independent business logic (17 files). Extracted from handlers to enable reuse without HTTP context. Functions operate on DAO/repository types directly.
- `gkill/dao/` — Data access layer with `GkillDAOManager` managing SQLite3 databases
- `gkill/dao/reps/` — Repository interfaces and implementations for each data type. `plugin_repository_impl.go` manages plugin subprocess lifecycle (start, mutex-guarded stdio, auto-restart on crash)
- `gkill/main/common/` — Shared CLI commands, server initialization, logging
- `gkill/main/common/gkill_options/` — CLI flag definitions and directory structure

**Key dependencies:** gorilla/mux, modernc.org/sqlite (pure Go), spf13/cobra+viper, asticode/go-astilectron, SherClockHolmes/webpush-go, go-git/go-git, twpayne/go-gpx, nicksnyder/go-i18n/v2, google/uuid

**Doc comments:** `dao/reps/*_repository.go` のインターフェースメソッドは **454/454 documented**、`gkill_server_api` の HandleXxx は **90/90 handlers documented**。どちらも verify_docs が網羅率を機械検査するので、メソッドやハンドラを追加したら doc コメントも書くこと。共通契約の完全文は基底 `dao/reps/repository.go` に1回だけ書き、型別ファイルは「契約は Repository.Xxx を参照。」の1行参照＋差分のみにする（454箇所へ複製するとドリフト源になるため）。ハンドラは「1行説明 / 空行 / パス・HTTPメソッド・認証区分 / req_res 型」の形式。

**Repository pattern:** SQLite-backed data types have 4 implementation layers: `*_repository.go` (interface) → `*_repository_sqlite3_impl.go` → `*_repository_cached_sqlite3_impl.go` → `*_repository_temp_sqlite3_impl.go`. Exceptions: `git_commit_log` (`_local_dir_impl` + `_cached_sqlite3_impl`), `gps_log` (`_gpx_dir_impl`), and `plugin` (`plugin_repository_impl`) read external sources directly and do not have all 4 layers.

**Core entity — "Kyou"** (record). Data types: kmemo (text), timeis (timestamps), lantana (mood 0-10), kc (numeric), nlog (expense), urlog (bookmark), mi (task), idf_kyou (file, with `is_zip` flag for .zip/.cbz), re_kyou (repost), mi_re_kyou (an existing Kyou turned into a task: `target_id` + Mi scheduling fields, no title; `data_type` is `mirekyou_create` / `_check` / `_limit` / `_start` / `_end`, so prefix checks must test `mirekyou` **before** `mi`), tag, text, notification, git_commit_log, gps_log (GPS tracks), plugin_kyou (external plugin data — `typed_plugin` field non-null in TypeScript `Kyou` class; `data_type` is plugin-defined e.g. `claude_conversation`).

**Response structure:** All API responses include `messages []GkillMessage` and `errors []GkillError` (with `error_code` + `error_message`). HTTP 200 for normal responses (check `errors` array), 403 for access denied, 500 for unexpected errors.

### Frontend (Vue 3 + TypeScript) — `src/client/`

Stack: Vue 3 + Vuetify 4 + Vue Router 5 + vue-i18n 11 + Vite 8 + TypeScript 6 + PWA (vite-plugin-pwa + Workbox)

- `router/index.ts` — 13 page routes (login, kftl, mi, rykv, kyou, mkfl, plaing, saihate, dashboard, set_new_password, register_first_account, shared_page, shared_mi) + 1 redirect-only route (`/regist_first_account` → `/register_first_account`, query preserved)
- `pages/views/` — 189 view components, `pages/dialogs/` — 103 dialog components (Escape key closes via `useFloatingDialog`), including ZIP contents browser, plugin HTML views (`plugin-html-view.vue`, `plugin-html-context-menu.vue`, `plugin-config-dialog.vue`), and Dnote trend graph components (`dnote-trend-graph-*` — client-side time-series aggregation via `classes/dnote/dnote-trend-aggregator.ts`, no server API)
- `classes/api/gkill-api.ts` — Singleton `GkillAPI` class (~3,330 lines), client-side API wrapper
- `classes/kftl/` — KFTL parser (41 statement types; the Go side has 39). Accepts the same Japanese/ASCII prefixes as the Go parser; ASCII constants and match/strip helpers centralized in `kftl-prefixes.ts`
- `serviceWorker.ts` — PWA service worker (Workbox precaching, POST caching, push notifications, Web Share Target; `/zip_cache/.*` on NavigationRoute denylist)

**State management:** Props/Emit only. No Pinia/Vuex. `GkillAPI` singleton for backend communication.

**Composable pattern:** Each view/dialog has a corresponding `classes/use-*.ts` composable with the component logic. Props and emits are defined in `-props.ts` / `-emits.ts` files alongside the `.vue` file (many components share a common props/emits type instead of having their own pair). Template refs to child components use the `ComponentRef` type (`classes/component-ref.ts`).

**Naming convention (files):** `{action}-{feature}-{entity}-{component}` (e.g., `add-dnote-item-view.vue`, `confirm-delete-ryuu-item-dialog.vue`), kebab-case. Dnote and Ryuu follow the same pattern.

**Frozen spellings:** `plaing` / `Plaing` は "playing" の誤綴りではなく**製品綴りとして凍結**（ルート `/plaing`、`FindQuery` の `use_plaing` / `plaing_time`、MCP ツールスキーマ、Wear OS データレイヤーパス、マニュアルのページ名、`default_page` の保存値に浸透しているため）。綴り修正の提案はしないこと。詳細は `documents/reverse/glossary.md` の「凍結された綴り」節。

**Naming convention (identifiers):** データクラスのプロパティ/メソッド・ローカル変数・通常関数は snake_case（Go 側 JSON タグとの写像）。コンポーザブルは `useXxx`、イベントコールバックは `onXxx`、CRUD リレーハンドラ束は `xxxHandlers`（いずれも camelCase）。型は PascalCase、enum メンバーは snake_case。`@typescript-eslint/naming-convention` で機械検査される（`eslint.config.js` の `app/naming-convention` ブロック。対象は `src/client` 本体のみで、`__tests__`・`src/mcp`・`src/tools`・`*.d.ts` は別流儀として対象外）。

**i18n:** 7 languages (ja, en, zh, ko, es, fr, de) in `src/locales/`. 856 keys per locale. Flat key-value JSON. Shared between frontend (import) and backend (Go embed).

### MCP Server — `src/mcp/`

3つのMCPサーバーを提供。いずれもstdio (ローカル) / HTTP (OAuth 2.1認証付きリモート) の2モード対応。

| Server | Tools | stdio | HTTP | Port |
|---|---|---|---|---|
| Read | 10 (get_kyous, get_mi_board_list, get_all_tag_names, get_all_rep_names, get_gps_log, get_application_config, get_idf_file, get_idf_file_path + plugin 2) | `npm run mcp:gkill-read` | `npm run mcp:gkill-read-http` | 8808 |
| Write | 25 (9 add + 1 submit_kftl + 1 delete + 9 update + 3 read convenience + plugin 2) | `npm run mcp:gkill-write` | `npm run mcp:gkill-write-http` | 8809 |
| ReadWrite | 30 (read 8 + write 20 + plugin 2) | `npm run mcp:gkill-readwrite` | `npm run mcp:gkill-readwrite-http` | 8810 |

**Plugin tools** (`gkill_get_plugin_list`, `gkill_get_plugin_content`) are shared by all 3 servers via `lib/plugin-tools.mjs` (read-only; `post_plugin_config` is not exposed). Plugin kyou bodies are not stored in gkill — `get_kyous` returns only metadata plus a `payload.kind="plugin"` carrying `rep_name`/`kyou_id`, which `gkill_get_plugin_content` turns into the actual content. It calls `/api/get_plugin_content_html` and by default converts the HTML to plain text via `lib/html-text.mjs` (`format`: `text` (default) / `html` / `both`, `max_text_length` default 20000) because plugin content HTML is mostly presentation CSS/JS.

**Environment variables:** `GKILL_BASE_URL`, `GKILL_USER`, `GKILL_PASSWORD_SHA256`, `MCP_LOG` (default `info`), `GKILL_MCP_MAX_FILE_BYTES` (default 8MB — cap for `get_idf_file` base64), `GKILL_MCP_FILE_LINK_TTL_MS` (default 1h — HTTP-mode file URL token TTL), and for HTTP mode: `MCP_TRANSPORT`, `MCP_PORT`, `MCP_OAUTH_ISSUER`. Also read: `GKILL_PASSWORD` (plaintext alternative to the SHA256), `GKILL_SESSION_ID`, `GKILL_INSECURE`, `GKILL_LOCALE`, `GKILL_FETCH_TIMEOUT_MS`, `GKILL_HOME`.

**IDF file access for AI clients** (3 paths, preferred order): (1) `file_path` — `get_kyous` includes the absolute local path in IDF payloads for **stdio** clients (same machine); read it directly. (2) `file_url` / `file_url_full` — for **HTTP** clients, `get_kyous` includes a per-file, expiring, unguessable public URL served by the MCP server's own `GET /files/{token}` route (`FileLinkStore`, `lib/file-link-store.mjs`); fetch with no auth, any size (images: `file_url` is a thumbnail, `file_url_full` the original). (3) `gkill_get_idf_file` — base64 fallback, size-capped. Local paths never go to remote clients; the transport type (stdio vs HTTP) is the gate. gkill itself stays private — only the MCP server's public surface is used.

### Mobile — `src/android/`, `src/wear_os/`

**Android**: APK wrapper (WebView) bundling the gkill_server binary as `jniLibs/arm64-v8a/libgkill_server.so` and exec'ing it from `nativeLibraryDir` — required because targetSdk 29+ forbids executing files under the app's data dir (W^X). Needs `packaging { jniLibs { useLegacyPackaging = true } }` so the `.so` is extracted as a real file. compileSdk 37 (androidx 1.19.x requires it), targetSdk 36, minSdk 26. **Wear OS**: Gradle multi-module project (phone_companion + watch_app), communicates via Wearable Data Layer. The Gradle wrapper is committed under `src/wear_os/`, so no copying is needed; `npm run setup_wear_os_gradle` re-syncs it from `src/android/` if it ever breaks.

## Lint & Code Quality

ESLint 10 flat config (`eslint.config.js`). Zero violations policy (levels per rule as listed below).

| Rule | Level | Note |
|---|---|---|
| `@typescript-eslint/no-explicit-any` | error | `any` 使用禁止。`unknown` or 具体的型を使うこと |
| `@typescript-eslint/no-empty-object-type` | error | 空の `{}` 型禁止。`type X = ParentType` を使うこと |
| `@typescript-eslint/no-unused-vars` | warn | `_` プレフィックスで無視 (`argsIgnorePattern: '^_'`) |
| `@typescript-eslint/naming-convention` | error | 識別子の命名規約（上記 Naming convention 節）。対象は `src/client` 本体のみ |
| `playwright/no-conditional-in-test` / `no-wait-for-timeout` | error（未移行E2Eのみ warn） | 「静かに成功するテスト」の防止。未移行ファイルは `e2e/playwright-not-migrated` に列挙 |

Go: `slices.SortFunc` (not `sort.Slice`), `for range n` (not `for i := 0; i < n; i++`), `any` (not `interface{}`), `errors.Join` for multi-error collection.

## Language

The codebase (variable names, comments, commit messages) is primarily in Japanese. README and documentation are in Japanese.

**Comment language policy** (現状追認、2026-08 明文化): Go 本体（`src/server/gkill/`、kftl を除く）と `src/plugins/` は日本語。`src/client/` は「構造バナーコメント（`// ── Template refs ──` 等）は英語、処理説明は日本語」の複合。`src/mcp/`・`src/wear_os/`・`src/server/gkill/api/kftl/`（TS 版からの移植、`// Mirrors:` 注記付き）は英語容認。新規コメントは各領域の既存スタイルに合わせること。

## Documentation

- `resources/manual/` — HTML manuals (7 languages, 21 pages per language), embedded via `//go:embed` and served at `/resources/manual/`
- `documents/reverse/` — Reverse-engineered design documents (24 files). See `documents/reverse/README.md` for index. Key files: glossary.md (90 terms), api-endpoints.md (90 endpoints, 88 registered), usecase.md (84 use cases), sequence-diagrams.md (29 diagrams), scenario.md (cross-channel end-to-end usage scenarios with UML), testing-guide.md. `npm run verify_docs` (`src/tools/verify_docs.mjs`) machine-checks the counts, cross-links, referenced paths, Mermaid blocks, and manual freshness — it runs as part of `npm test`, so update the docs when a count changes.
- `src/ABOUT_TEST.md` — Test specification index, links to 23 subdirectory `ABOUT_TEST.md` files
- **What `verify_docs` covers** (extend it rather than hand-maintaining new numbers): file/test counts computed from the tree (`--list` prints them all), count assertions across `documents/reverse/*.md` + `CLAUDE.md` + `src/**/README.md` + `ABOUT_TEST.md`, markdown link resolution for that same set, `src/...` path references (warning), Mermaid block types, manual generation freshness / language page-set parity / a11y invariants / intra-manual links, a **terminology lint** that rejects internal code names (`IDF`, `WAN`, `Kyou`, `MiReKyou`, `Dnote`, …) in `resources/manual_src/` outside `<code>` spans, and a check that every `screen_name` the app passes to `HelpDialog` has a matching manual page. `--parity` (opt-in) reports per-page h2/h3/table drift against the Japanese original.
