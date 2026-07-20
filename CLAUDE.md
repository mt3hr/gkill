# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

gkill is a lifelogging application that records and reviews life events (memos, bookmarks, tasks, mood values, expenses, timestamps, files, git commits, etc.). Standalone client-server app: Go backend + Vue 3 frontend, compiled into a single binary with the frontend embedded via `//go:embed`. Licensed under MIT.

## Build & Development Commands

All commands are npm scripts defined in `package.json`. No CGO required (pure Go SQLite driver).

| Command | Purpose |
|---|---|
| `npm run dev` | Start Vite dev server (frontend only) |
| `npm run build` | Build frontend (`vue-tsc --build` + `vite build` in parallel) |
| `npm run lint` | ESLint with auto-fix (flat config, `eslint.config.js`) |
| `npm run install_server` | Full build: frontend → embed → `go install` (headless HTTP server) |
| `npm run install_app` | Full build: frontend → embed → `go install` (desktop app with go-astilectron window) |
| `npm run go_install` | Go install only (skip frontend rebuild) |
| `npm run go_mod` | Regenerate `go.mod` and `go.sum` from scratch |
| `npm test` | Run all tests (server + client + MCP + Android + Wear OS) |
| `npm run test_server` | Go tests (`cd src/server && go test ./...`) |
| `npm run test_client_unit` | Vitest unit tests |
| `npm run test_client_e2e` | Playwright E2E tests (gkill_server自動起動・停止、`$HOME/gkill_test`使用) |
| `npm run test_mcp` | MCP server tests (Vitest) |
| `npm run release` | Cross-compile release for all platforms |

**Build pipeline** (`install_server`): clean embed dir → write `version.json` (commit hash + build time + version) → `vue-tsc` type-check → `vite build` → copy `dist/` to `src/server/gkill/api/embed/html/` → copy `src/locales/` to `src/server/gkill/api/embed/i18n/locales/` → `go install`

**Prerequisites:** Go 1.26.0+, Node.js 20.19+ (24.x recommended), `npm i`

**E2E test environment:** `$HOME/gkill_test` as dedicated gkill home. `npm run test_client_e2e` automatically cleans, starts gkill_server (fresh admin, no password), runs Playwright, and stops the server. See `src/client/__tests__/e2e/run-e2e.mjs`.

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
```

### Two Deployment Modes

- **`gkill_server`** (`src/server/gkill/main/gkill_server/main.go`): Headless HTTP server (browser/PWA)
- **`gkill`** (`src/server/gkill/main/gkill/main.go`): Desktop app wrapping server with go-astilectron

Both use cobra for CLI with shared subcommands: `version`, `idf`, `dvnf`, `generate_thumb_cache`, `generate_video_cache`, `optimize`, `update_cache`. Default listen address: `:9999`, TLS disabled, initial user `admin` with no password.

### CLI Flags

| Flag | Default | Description |
|---|---|---|
| `--gkill_home_dir` | `$HOME/gkill` | Home directory for all data/config/logs |
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
- `gkill/api/gkill_server_api/` — HTTP API handlers (86 files, 1 handler per file). `GkillServerAPI` struct with `serve.go`, `close.go`, route definitions in `gkill_server_api_address.go`. Auth middleware (`auth.go`, `auth_context.go`, `auth_middleware.go`) extracts session→account→device→repositories via `AuthContext`, `authMiddleware`, `authWithReposMiddleware`. Handler registration uses wrapper functions: `wrapNoAuth` (no session), `wrapAuth` (session + account), `wrapAuthRepos` (session + account + device + repositories). Utility files: `filter_local_only.go`, `utils.go`, `web_push.go`. ZIP browsing: `handle_browse_zip_contents.go` (path traversal prevention, Shift_JIS→UTF-8, singleflight dedup).
- `gkill/api/req_res/` — Request/response structs for every endpoint (176 files)
- `gkill/api/kftl/` — KFTL custom text format parser (single package, no sub-packages). Supports both Japanese (。！？、ーー etc.) and ASCII (#!?,-- /mi /mood /expense /num /url /start /end /timeis /end? /endt /endt?) prefixes
- `gkill/api/gkill_plugin/` — Plugin protocol types: `PluginManifest`, `PluginRequest`, `PluginResponse`, `PluginKyou` (stdio newline-delimited JSON)
- `gkill/plugin/sdk/` — Plugin author SDK. `sdk.Run(sdk.Handler{FindKyous, GetContentHTML, GetConfigHTML})` starts the stdio JSON message loop. Plugins are standalone binaries in `src/plugins/`
- `gkill/usecase/` — HTTP-independent business logic (16 files). Extracted from handlers to enable reuse without HTTP context. Functions operate on DAO/repository types directly.
- `gkill/dao/` — Data access layer with `GkillDAOManager` managing SQLite3 databases
- `gkill/dao/reps/` — Repository interfaces and implementations for each data type. `plugin_repository_impl.go` manages plugin subprocess lifecycle (start, mutex-guarded stdio, auto-restart on crash)
- `gkill/main/common/` — Shared CLI commands, server initialization, logging
- `gkill/main/common/gkill_options/` — CLI flag definitions and directory structure

**Key dependencies:** gorilla/mux, modernc.org/sqlite (pure Go), spf13/cobra+viper, asticode/go-astilectron, SherClockHolmes/webpush-go, go-git/go-git, twpayne/go-gpx, nicksnyder/go-i18n/v2, google/uuid

**Repository pattern:** Each data type has 4 implementation layers: `*_repository.go` (interface) → `*_repository_sqlite3_impl.go` → `*_repository_cached_sqlite3_impl.go` → `*_repository_temp_sqlite3_impl.go`

**Core entity — "Kyou"** (record). Data types: kmemo (text), timeis (timestamps), lantana (mood 0-10), kc (numeric), nlog (expense), urlog (bookmark), mi (task), idf_kyou (file, with `is_zip` flag for .zip/.cbz), re_kyou (repost), tag, text, notification, git_commit_log, gps_log (GPS tracks), plugin_kyou (external plugin data — `typed_plugin` field non-null in TypeScript `Kyou` class; `data_type` is plugin-defined e.g. `claude_conversation`).

**Response structure:** All API responses include `messages []GkillMessage` and `errors []GkillError` (with `error_code` + `error_message`). HTTP 200 for normal responses (check `errors` array), 403 for access denied, 500 for unexpected errors.

### Frontend (Vue 3 + TypeScript) — `src/client/`

Stack: Vue 3 + Vuetify 4 + Vue Router 5 + vue-i18n 11 + Vite 8 + TypeScript 6 + PWA (vite-plugin-pwa + Workbox)

- `router/index.ts` — 13 routes (login, kftl, mi, rykv, kyou, mkfl, plaing, saihate, dashboard, set_new_password, regist_first_account, shared_page, shared_mi)
- `pages/views/` — 185 view components, `pages/dialogs/` — 100 dialog components (Escape key closes via `useFloatingDialog`), including ZIP contents browser, plugin HTML views (`plugin-html-view.vue`, `plugin-html-context-menu.vue`, `plugin-config-dialog.vue`), and Dnote trend graph components (`dnote-trend-graph-*` — client-side time-series aggregation via `classes/dnote/dnote-trend-aggregator.ts`, no server API)
- `classes/api/gkill-api.ts` — Singleton `GkillAPI` class (~3,500 lines), client-side API wrapper
- `classes/kftl/` — KFTL parser (44 statement types). Accepts the same Japanese/ASCII prefixes as the Go parser; ASCII constants and match/strip helpers centralized in `kftl-prefixes.ts`
- `serviceWorker.ts` — PWA service worker (Workbox precaching, POST caching, push notifications, Web Share Target; `/zip_cache/.*` on NavigationRoute denylist)

**State management:** Props/Emit only. No Pinia/Vuex. `GkillAPI` singleton for backend communication.

**Composable pattern:** Each view/dialog has a corresponding `classes/use-*.ts` composable with the component logic. Props and emits are defined in separate `-props.ts` / `-emits.ts` files alongside the `.vue` file. Template refs to child components use the `ComponentRef` type (`classes/component-ref.ts`).

**Naming convention:** `{action}-{feature}-{entity}-{component}` (e.g., `add-dnote-item-view.vue`, `confirm-delete-ryuu-item-dialog.vue`). Dnote and Ryuu follow the same pattern.

**i18n:** 7 languages (ja, en, zh, ko, es, fr, de) in `src/locales/`. 830 keys per locale. Flat key-value JSON. Shared between frontend (import) and backend (Go embed).

### MCP Server — `src/mcp/`

3つのMCPサーバーを提供。いずれもstdio (ローカル) / HTTP (OAuth 2.1認証付きリモート) の2モード対応。

| Server | Tools | stdio | HTTP | Port |
|---|---|---|---|---|
| Read | 8 (get_kyous, get_mi_board_list, get_all_tag_names, get_all_rep_names, get_gps_log, get_application_config, get_idf_file, get_idf_file_path) | `npm run mcp:gkill-read` | `npm run mcp:gkill-read-http` | 8808 |
| Write | 23 (9 add + 1 submit_kftl + 1 delete + 9 update + 3 read convenience) | `npm run mcp:gkill-write` | `npm run mcp:gkill-write-http` | 8809 |
| ReadWrite | 28 (read 8 + write 20) | `npm run mcp:gkill-readwrite` | `npm run mcp:gkill-readwrite-http` | 8810 |

**Environment variables:** `GKILL_BASE_URL`, `GKILL_USER`, `GKILL_PASSWORD_SHA256`, `MCP_LOG` (default `info`), `GKILL_MCP_MAX_FILE_BYTES` (default 8MB — cap for `get_idf_file` base64), `GKILL_MCP_FILE_LINK_TTL_MS` (default 1h — HTTP-mode file URL token TTL), and for HTTP mode: `MCP_TRANSPORT`, `MCP_PORT`, `MCP_OAUTH_ISSUER`.

**IDF file access for AI clients** (3 paths, preferred order): (1) `file_path` — `get_kyous` includes the absolute local path in IDF payloads for **stdio** clients (same machine); read it directly. (2) `file_url` / `file_url_full` — for **HTTP** clients, `get_kyous` includes a per-file, expiring, unguessable public URL served by the MCP server's own `GET /files/{token}` route (`FileLinkStore`, `lib/file-link-store.mjs`); fetch with no auth, any size (images: `file_url` is a thumbnail, `file_url_full` the original). (3) `gkill_get_idf_file` — base64 fallback, size-capped. Local paths never go to remote clients; the transport type (stdio vs HTTP) is the gate. gkill itself stays private — only the MCP server's public surface is used.

### Mobile — `src/android/`, `src/wear_os/`

**Android**: APK wrapper (WebView) bundling the gkill_server binary as `jniLibs/arm64-v8a/libgkill_server.so` and exec'ing it from `nativeLibraryDir` — required because targetSdk 29+ forbids executing files under the app's data dir (W^X). Needs `packaging { jniLibs { useLegacyPackaging = true } }` so the `.so` is extracted as a real file. compileSdk/targetSdk 36, minSdk 26. **Wear OS**: Gradle multi-module project (phone_companion + watch_app), communicates via Wearable Data Layer. Must copy `gradlew`/`gradlew.bat`/`gradle-wrapper.jar` from `src/android/` before building.

## Lint & Code Quality

ESLint 10 flat config (`eslint.config.js`). All rules enforced as error with zero violations.

| Rule | Level | Note |
|---|---|---|
| `@typescript-eslint/no-explicit-any` | error | `any` 使用禁止。`unknown` or 具体的型を使うこと |
| `@typescript-eslint/no-empty-object-type` | error | 空の `{}` 型禁止。`type X = ParentType` を使うこと |
| `@typescript-eslint/no-unused-vars` | warn | `_` プレフィックスで無視 (`argsIgnorePattern: '^_'`) |

Go: `slices.SortFunc` (not `sort.Slice`), `for range n` (not `for i := 0; i < n; i++`), `any` (not `interface{}`), `errors.Join` for multi-error collection.

## Language

The codebase (variable names, comments, commit messages) is primarily in Japanese. README and documentation are in Japanese.

## Documentation

- `resources/manual/` — HTML manuals (7 languages, 20 pages per language), embedded via `//go:embed` and served at `/resources/manual/`
- `documents/reverse/` — Reverse-engineered design documents (23 files). See `documents/reverse/README.md` for index. Key files: glossary.md (80 terms), api-endpoints.md (87 endpoints, 85 registered), usecase.md (78 use cases), sequence-diagrams.md (27 diagrams), testing-guide.md
- `src/ABOUT_TEST.md` — Test specification index, links to 22 subdirectory `ABOUT_TEST.md` files
