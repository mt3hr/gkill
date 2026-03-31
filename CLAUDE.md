# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

gkill is a lifelogging application (version 1.1.1-dev) that records and reviews life events (memos, bookmarks, tasks, mood values, expenses, timestamps, files, git commits, etc.). It is a standalone client-server app with a Go backend and Vue 3 frontend, compiled into a single binary with the frontend embedded via `//go:embed`. Licensed under MIT.

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
| `npm run release` | Cross-compile release for all platforms (Windows, Linux, Android, Wear OS) |

**Build pipeline** (`install_server`): clean embed dir → write `version.json` (commit hash + build time + version) → `vue-tsc` type-check → `vite build` → copy `dist/` to `src/server/gkill/api/embed/html/` → copy `src/locales/` to `src/server/gkill/api/embed/i18n/locales/` → `go install`

**Prerequisites:**
- Go 1.26.0+ (see `src/server/go.mod`)
- Node.js 20.19+ (Vite 7 requires 20.19+; 24.x recommended)
- `npm i` (install JS dependencies)

**Test commands:**

| Command | Purpose |
|---|---|
| `npm test` | Run all tests (server + client + MCP + Android + Wear OS) |
| `npm run test_server` | Go tests (`cd src/server && go test ./...`) |
| `npm run test_client` | Frontend unit + E2E tests |
| `npm run test_client_unit` | Vitest unit tests |
| `npm run test_client_e2e` | Playwright E2E tests (gkill_server自動起動・停止、`$HOME/gkill_test`使用) |
| `npm run test_e2e_server` | E2E用gkill_server起動 (`$HOME/gkill_test`、TLS無効) |
| `npm run test_mcp` | MCP server tests (Vitest) |
| `npm run test_android` | Android Gradle tests |
| `npm run test_wear_os` | Wear OS Gradle tests |

**Test coverage (~2,086 tests total, all 29 Go packages covered):**

| Area | Tests | Files | Scope |
|---|---|---|---|
| Go backend | ~534 | 47 | API handlers, KFTL parser, DAO layer, find filters, req_res, utilities |
| Frontend unit | ~676 | 49 | GkillAPI, data models, D-note, KFTL, composables, router, i18n, utils |
| Frontend E2E | 187 | 29 | All 12 routes, CRUD flows, settings, Mi operations, regressions |
| MCP | ~563 | 17 | Tools (read/write/readwrite), OAuth (PKCE/DCR/RFC 8707), access-log, write-normalization |
| Android | 12 | 2 | Unit + instrumented |
| Wear OS | 114 | 9 | Credential, API client, Data Layer, lifecycle |

See `documents/reverse/testing-guide.md` for details, `src/ABOUT_TEST.md` for per-directory test specs.

**E2E test environment:** E2E tests use `$HOME/gkill_test` as a dedicated gkill home directory. `npm run test_client_e2e` automatically cleans this directory, starts gkill_server (fresh admin with no password), runs Playwright tests, and stops the server. See `src/client/__tests__/e2e/run-e2e.mjs` for the orchestration logic.

## Architecture

### Source Directory Structure

```
src/
  client/     # Vue 3 + TypeScript frontend (App.vue, main.ts, classes/, pages/)
  server/     # Go backend (go.mod at src/server/, packages under gkill/)
  mcp/        # MCP server (gkill-read-server.mjs)
  android/    # Android APK wrapper (WebView + bundled gkill_server binary)
  wear_os/    # Wear OS project (phone_companion/ + watch_app/)
  locales/    # i18n JSON files (ja, en, zh, ko, es, fr, de) — shared by frontend & backend
  tools/      # Utility scripts
```

### Two Deployment Modes

- **`gkill_server`** (`src/server/gkill/main/gkill_server/main.go`): Headless HTTP server accessed via browser/PWA
- **`gkill`** (`src/server/gkill/main/gkill/main.go`): Desktop app wrapping the server with go-astilectron (450x750 window)

Both use cobra for CLI with shared subcommands: `version`, `idf`, `dvnf`, `generate_thumb_cache`, `generate_video_cache`, `optimize`, `update_cache`.

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

Module: `github.com/mt3hr/gkill/src/server` (Go 1.26.0)

Key packages:
- `gkill/api/` — HTTP API handlers via gorilla/mux, **80 POST endpoints** under `/api/`, plus `/zip_cache/` file server for extracted ZIP contents. Main handler: `gkill_server_api.go` (~14,000 lines). Route definitions: `gkill_server_api_address.go`. ZIP browsing: `handle_browse_zip_contents.go` (extraction with path traversal prevention, Shift_JIS→UTF-8 filename conversion, singleflight dedup).
- `gkill/api/embed.go` — `//go:embed embed` directive serves the compiled Vue SPA at root `/`
- `gkill/api/req_res/` — Request/response structs for every endpoint (173 types)
- `gkill/api/kftl/` — KFTL custom text format parser (single package, no sub-packages). Supports both Japanese (。！？、ーー etc.) and ASCII (#!?,-- /mi /mood /expense /num /url /start /end /timeis /end? /endt /endt?) prefixes
- `gkill/dao/` — Data access layer with `GkillDAOManager` managing SQLite3 databases
- `gkill/dao/reps/` — Repository interfaces and implementations for each data type
- `gkill/main/common/` — Shared CLI commands, server initialization, logging
- `gkill/main/common/gkill_options/` — CLI flag definitions and directory structure

**Key dependencies:** gorilla/mux (router), modernc.org/sqlite (DB, pure Go), spf13/cobra+viper (CLI/config), asticode/go-astilectron (desktop), SherClockHolmes/webpush-go (VAPID), go-git/go-git (git integration), twpayne/go-gpx (GPS), nicksnyder/go-i18n/v2 (i18n), google/uuid

**Repository pattern:** Each data type has 4 implementation layers:
1. `*_repository.go` — interface
2. `*_repository_sqlite3_impl.go` — SQLite3 implementation
3. `*_repository_cached_sqlite3_impl.go` — cached variant (in-memory)
4. `*_repository_temp_sqlite3_impl.go` — temp/transactional variant

**Core entity — "Kyou"** (record). Data types: kmemo (text), timeis (timestamps), lantana (mood 0-10), kc (numeric), nlog (expense), urlog (bookmark), mi (task), idf_kyou (file, with `is_zip` flag for .zip/.cbz), re_kyou (repost), tag, text, notification, git_commit_log, gps_log (GPS tracks).

**Response structure:** All API responses include `messages []GkillMessage` and `errors []GkillError` (with `error_code` + `error_message`). HTTP 200 for normal responses (check `errors` array), 403 for access denied, 500 for unexpected errors.

### Runtime Directory Structure

```
$HOME/gkill/
├── configs/          # SQLite3 config databases
│   ├── server_config.db        # Server address, TLS, device name
│   ├── account.db              # User accounts, password hashes
│   ├── account_state.db        # Login sessions, file upload history
│   ├── user_config.db          # App settings, repository definitions
│   ├── share_kyou_info.db      # Sharing settings
│   └── gkill_notification_target.db  # Push notification targets
├── datas/            # User data (default data directory)
├── caches/           # Cache files
│   ├── thumb_cache/{rep_name}/  # Thumbnail image cache per repository
│   ├── video_cache/{rep_name}/  # Compat video cache per repository
│   ├── zip_cache/{rep_name}/{sha1}/  # Extracted ZIP contents for browsing
│   ├── local_rep_cache/         # Local repository cache DBs
│   ├── git_commit_log_cache/    # Git commit log cache DBs
│   ├── temp_cache/              # Temp cache DBs (non-memory mode only)
│   └── latest_data_repository_address_cache/  # Latest data repository address cache DBs (non-memory mode only)
├── logs/             # Log files (JSON format, split by level)
│   ├── gkill_error.log / gkill_warn.log / gkill_info.log / ...
│   └── gkill.log               # Merged log file
├── lib/base_directory/  # Library files
└── tls/              # TLS certificates (optional)
    ├── cert.cer
    └── key.pem
```

### Server Defaults

| Setting | Default |
|---|---|
| Listen address | `:9999` (all interfaces, port 9999) |
| TLS | Disabled |
| IsLocalOnlyAccess | `false` (remote access allowed) |
| Session expiration | 30 days |
| Upload size limit | Unlimited (`-1`) |
| Initial user | `admin` with no password (created on first run) |

### Frontend (Vue 3 + TypeScript) — `src/client/`

Stack: Vue 3 + Vuetify 4 + Vue Router 5 + vue-i18n 11 + Vite 7 + TypeScript 6 + PWA (vite-plugin-pwa + Workbox)

- `main.ts` — App entry, registers Vuetify, router, i18n, custom `v-long-press` directive
- `router/index.ts` — 12 routes (login, kftl, mi, rykv, kyou, mkfl, plaing, saihate, set_new_password, regist_first_account, shared_page, shared_mi)
- `pages/` — Route page components
- `pages/views/` — 175 view components for add/edit/list operations per data type
- `pages/dialogs/` — 93 dialog components (Escape key closes via `useFloatingDialog`), including ZIP contents browser with inline image viewer
- `classes/api/gkill-api.ts` — Singleton `GkillAPI` class (~3,400 lines), client-side API wrapper
- `classes/datas/` — TypeScript data models mirroring Go structs
- `classes/kftl/` — KFTL parser (43 statement types)
- `serviceWorker.ts` — PWA service worker (Workbox precaching, POST caching, push notifications, Web Share Target; `/zip_cache/.*` on NavigationRoute denylist)
- `plugins/vuetify.ts` — Vuetify config with light (`gkill_theme`) and dark (`gkill_dark_theme`) themes

**State management:** Props/Emit only. No Pinia/Vuex. `GkillAPI` singleton for backend communication.

**Composable pattern:** Each view/dialog has a corresponding `classes/use-*.ts` composable with the component logic. Props and emits are defined in separate `-props.ts` / `-emits.ts` files alongside the `.vue` file. Template refs to child components use the `ComponentRef` type (`classes/component-ref.ts`).

**Naming convention:** `{action}-{feature}-{entity}-{component}` (e.g., `add-dnote-item-view.vue`, `confirm-delete-ryuu-item-dialog.vue`). Dnote and Ryuu follow the same pattern.

**i18n:** 7 languages (ja, en, zh, ko, es, fr, de) in `src/locales/`. ~768 keys per locale. Flat key-value JSON. Shared between frontend (import) and backend (Go embed).

### MCP Server — `src/mcp/`

3つのMCPサーバーを提供。いずれもstdio (ローカル) / HTTP (OAuth 2.1認証付きリモート) の2モード対応。

**Read専用** (`gkill-read-server.mjs`) — **7 tools**, port 8808:
1. `gkill_get_kyous` — Get records with filters, pagination, type data inlined
2. `gkill_get_mi_board_list` — Get task board names
3. `gkill_get_all_tag_names` — Get all tags
4. `gkill_get_all_rep_names` — Get all repository names
5. `gkill_get_gps_log` — Get GPS logs by date range
6. `gkill_get_application_config` — Get tag hierarchy, board structure, templates
7. `gkill_get_idf_file` — Get IDF file content (images returned as MCP image content blocks)

**Write専用** (`gkill-write-server.mjs`) — **14 tools** (11 write + 3 read convenience), port 8809:
1. `gkill_add_kmemo` — Create text memo
2. `gkill_add_urlog` — Create bookmark
3. `gkill_add_nlog` — Create expense record
4. `gkill_add_lantana` — Create mood record (0-10)
5. `gkill_add_timeis` — Create time interval
6. `gkill_add_mi` — Create task
7. `gkill_add_kc` — Create numeric record
8. `gkill_add_tag` — Add tag to existing entry
9. `gkill_add_text` — Add text annotation to existing entry
10. `gkill_submit_kftl` — Submit KFTL formatted text (batch creation)
11. `gkill_delete_kyou` — Soft-delete an entry
12-14. Read convenience: `gkill_get_all_rep_names`, `gkill_get_mi_board_list`, `gkill_get_all_tag_names`

**Read/Write統合** (`gkill-readwrite-server.mjs`) — **18 tools** (read 7 + write 11), port 8810:
Read専用の全7ツール + Write専用の全11ツールを1サーバーで提供。

Transport: stdio (default) or HTTP (OAuth 2.1).

| Server | stdio | HTTP | Default Port |
|---|---|---|---|
| Read | `npm run mcp:gkill-read` | `npm run mcp:gkill-read-http` | 8808 |
| Write | `npm run mcp:gkill-write` | `npm run mcp:gkill-write-http` | 8809 |
| ReadWrite | `npm run mcp:gkill-readwrite` | `npm run mcp:gkill-readwrite-http` | 8810 |

**Environment variables:** `GKILL_BASE_URL`, `GKILL_USER`, `GKILL_PASSWORD_SHA256`, `MCP_LOG` (access log level, default `info`), and for HTTP mode: `MCP_TRANSPORT`, `MCP_PORT`, `MCP_OAUTH_ISSUER` (required for remote access — set to public URL e.g. `https://example.com`). OAuth 2.1 (Authorization Code + PKCE) is always enabled for HTTP mode. Supports ChatGPT and Claude.ai MCP connectors via RFC 9728 (Protected Resource Metadata), RFC 8414 (AS Metadata), RFC 7591 (DCR), RFC 8707 (Resource Indicators).

| Server | Token persistence | Access log |
|---|---|---|
| Read | `$GKILL_HOME/configs/mcp_oauth_state.json` | `$GKILL_HOME/logs/gkill_mcp_read_access.log` |
| Write | `$GKILL_HOME/configs/mcp_oauth_write_state.json` | `$GKILL_HOME/logs/gkill_mcp_write_access.log` |
| ReadWrite | `$GKILL_HOME/configs/mcp_oauth_readwrite_state.json` | `$GKILL_HOME/logs/gkill_mcp_readwrite_access.log` |

### Mobile — `src/android/`, `src/wear_os/`

**Android** (`src/android/`): APK wrapper (WebView) bundling gkill_server binary in assets. Copies binary to internal storage, launches it, opens WebView at `http://localhost:9999`. compileSdk=35, targetSdk=28, minSdk=26.

**Wear OS** (`src/wear_os/`): Gradle multi-module project (phone_companion + watch_app). Communicates via Wearable Data Layer. Must copy `gradlew`/`gradlew.bat`/`gradle-wrapper.jar` from `src/android/` before building. Both modules use applicationId `com.gkill_android.mobile_app.src.gkill.mt3hr.gkill`.

## Lint & Code Quality

ESLint 9 flat config (`eslint.config.js`). All rules enforced as error with zero violations.

| Rule | Level | Note |
|---|---|---|
| `@typescript-eslint/no-explicit-any` | error | `any` 使用禁止���`unknown` or 具体的型を使うこと |
| `@typescript-eslint/no-empty-object-type` | error | 空の `{}` 型禁止。`type X = ParentType` ���使うこと |
| `@typescript-eslint/no-unused-vars` | warn | `_` プレフィックスで無視 (`argsIgnorePattern: '^_'`) |

Go: `slices.SortFunc` (not `sort.Slice`), `for range n` (not `for i := 0; i < n; i++`), `any` (not `interface{}`), `errors.Join` for multi-error collection.

## Language

The codebase (variable names, comments, commit messages) is primarily in Japanese. README and documentation are in Japanese.

## Documentation

### User Manual

HTML manuals in `resources/manual/` (7 languages: ja, en, zh, ko, es, fr, de). 17 pages per language covering login, KFTL, Mi, RYKV, Plaing, settings, etc. Embedded into the binary via `//go:embed` and served at `/manual/`.

### Reverse-Engineered Design Documents

`documents/reverse/`:
- `README.md` — Overview and index of reverse-engineered documents
- `glossary.md` — Domain term definitions (71 terms)
- `design-philosophy.md` — Architecture decisions and rationale
- `er-diagram.md` — Entity-relationship diagrams (Mermaid)
- `class-diagrams.md` — Go/TypeScript class hierarchies
- `sequence-diagrams.md` — Interaction sequences (21 diagrams: 16 normal flows + 5 error scenarios)
- `activity-diagrams.md` — Processing flowcharts
- `state-machines.md` — Entity state transitions
- `screen-transition.md` — Page/dialog navigation
- `screen-specs.md` — Screen specifications and layouts
- `usecase.md` — Use case catalog (74 use cases)
- `api-endpoints.md` — API endpoint reference (80 endpoints)
- `error-handling-and-security.md` — Error handling policy and security design
- `frontend-architecture.md` — Frontend design guide
- `operations-guide.md` — Deployment, backup, troubleshooting, logging
- `dev-setup.md` — Development environment setup guide
- `dvnf-rep-type-spec.md` — DVNF repository type specification
- `folder-structure.md` — Project folder structure reference
- `program-spec.md` — Program specification details
- `user-guide.md` — End-user guide
- `testing-guide.md` — Test execution, architecture, and guidelines (~1,904 tests)

### Test Documentation

- `documents/reverse/testing-guide.md` — Comprehensive test guide (execution, architecture, troubleshooting)
- `src/ABOUT_TEST.md` — Test specification index, links to 19 subdirectory `ABOUT_TEST.md` files
