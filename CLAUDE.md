# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

gkill is a lifelogging application (version 1.1.0) that records and reviews life events (memos, bookmarks, tasks, mood values, expenses, timestamps, files, git commits, etc.). It is a standalone client-server app with a Go backend and Vue 3 frontend, compiled into a single binary with the frontend embedded via `//go:embed`. Licensed under MIT.

## Build & Development Commands

All commands are npm scripts defined in `package.json`. No CGO required (pure Go SQLite driver).

| Command | Purpose |
|---|---|
| `npm run dev` | Start Vite dev server (frontend only) |
| `npm run build` | Build frontend (`vue-tsc --build` + `vite build` in parallel) |
| `npm run lint` | ESLint with auto-fix for `.vue`/`.ts`/`.js` files |
| `npm run install_server` | Full build: frontend → embed → `go install` (headless HTTP server) |
| `npm run install_app` | Full build: frontend → embed → `go install` (desktop app with go-astilectron window) |
| `npm run go_install` | Go install only (skip frontend rebuild) |
| `npm run go_mod` | Regenerate `go.mod` and `go.sum` from scratch |
| `npm test` | Run all tests (server + client + MCP + Android + Wear OS) |
| `npm run release` | Cross-compile release for all platforms (Windows, Linux, Android, Wear OS) |

**Build pipeline** (`install_server`): clean embed dir → write `version.json` (commit hash + build time + version) → `vue-tsc` type-check → `vite build` → copy `dist/` to `src/server/gkill/api/embed/html/` → copy `src/locales/` to `src/server/gkill/api/embed/i18n/locales/` → `go install`

**Prerequisites:**
- Go 1.26.0+ (see `src/server/go.mod`)
- Node.js 20.15.1+
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

**Test coverage (~1,887 tests total, all 29 Go packages covered):**
- Go (~531 tests, 47 files across all 29 packages): API handler integration (CRUD + Update + Get for all 11 data types, session validation, account management, transactions, session expiry detection (ERR000373), login rate limiting (ERR000374, IP-based sliding window), GetKyous query with complex filters (word/tag/rep/calendar/Mi check state/combined), GetKyousMCP, SubmitKFTLText for all data types (Kmemo/Lantana/Mi/Nlog/TimeIs/URLog/KC + tag/time combos + TimeIsEnd variants), UpdateCache, board/tag/rep name lists, tag/text/notification histories, server config read/update + TLS path change, application config update + search state preservation + structure deletion timeout, user repository update + duplicate write detection, share CRUD, Notification edit/soft-delete/history regression, Tag/Text delete with Kyou persistence, local access restriction, TLS file generation), KFTL parser (81 tests: all data type line generation + request map application for Lantana/Mi/Nlog/TimeIs variants/URLog/KC/text blocks/tags, ASCII prefix variants for all data types including !/,/,,/#/?/--//mi//mood//expense//num//url//start//end//timeis//end?//endt//endt?, Mi ASCII ? time field parsing, Nlog title/amount mismatch handling, Lantana mood 0-10 range validation), DAO layer (account, session, server config, user config, share_kyou_info, gkill_notification, 11 SQLite3 repos + 11 cached repos + 11 temp repos, cache address DAO, rep_cache_updater, GkillDAOManager), find package (all filter flags, MiCheckState/MiSortType enum values, nil dates, empty slices, data type filter), req_res package (18 JSON roundtrip tests including IDFPayloadMCPDTO), message package, sqlite3 utilities, gpslogs parsing, dvnf file management, hide_files, threads pool, gkill_log routing, gkill_options defaults, CLI entry points, fitbit batch, legacy data transformer
- Frontend unit (~676 tests, 49 files): GkillAPI methods (add/update/delete/get for all data types, config, sharing, upload, transactions, notifications, error handling, session management, endpoint address verification), 22 data models (Kmemo, Tag, Kyou, Mi, TimeIs, URLog, Nlog, Lantana, KC, Text, GitCommitLog, GPSLog, IDFKyou, Notification, ReKyou, InfoBase, InfoIdentifier, MetaInfoBase, CircleOptions, LatLng, KftlTemplateElementData, ShareKyousInfo), D-note module (52 predicates, 17 aggregate targets, 12 key getters, DnoteAgregator/DnoteListAggregator, serialization dictionaries), KFTL parser, composition functions (context menus for 10 entity types, add views for Mi/Tag/Nlog/URLog/Lantana/TimeIs/KC, edit views for Kmemo/Mi/Nlog/URLog/TimeIs/Lantana/KC), page composables, query composable patterns, router (12 routes), i18n completeness (7 locales), utility functions (deep-equals, looks-like-url, format-date-time, long-press directive, save-as, delete-gkill-cache, service-worker-utils, use-dialog-history-stack)
- Frontend E2E (187 tests, 29 spec files): All 12 routes covered + CRUD interaction flows. Login session persistence/auth redirect/password masking/logout/no-password-login-rejection/Rep check state. KFTL CRUD flows (submit all data types via KFTL, verify in RYKV/Mi/Plaing) + KFTL TimeIs End flows (end by title/title-if-exist/tag/tag-if-exist). GUI add dialog flows (Mi/Lantana/Nlog/TimeIs/URLog/KC via FAB, Tag/Text via context menu) + minimal-input Mi add, full-field TimeIs/URLog add. GUI edit dialog flows (Kmemo/Mi/Nlog/URLog/Lantana/TimeIs/Tag content editing + empty content validation) + running TimeIs end button, ReKyou edit, Text edit. GUI delete flows (Kmemo/Mi/Lantana/Nlog/URLog/TimeIs/Tag/Text deletion + ReKyou add/delete). View/browse history flows (Lantana/Mi/Nlog/URLog/ReKyou/Tag/Text history + repost + NoImage fallback). Notification CRUD (add/edit/delete/view/history via context menu). Search & summary (keyword search on RYKV/Mi, D-note toggle). Settings page CRUD (server config/user config/tag/rep/device/KFTL template structure sections) + server config functional tests (profile/TLS/address/account/rep management) + user config functional tests (GoogleMap API/image viewer/board names/hot reload/tag/rep/device/reptype/KFTL template structure CRUD). Mi operations (board move/completion state/share view/share stop). Regression tests (Kmemo required field/local access/tag/device/reptype structure add/ApplicationConfig apply/file upload). Misc (Notification/Text visual distinction, TimeIs history, context menu deduplication, bookmarklet, GPS upload, invalid share link, server restart)
- MCP (~367 tests, 9 files): validation, normalization (including normalizeIdfFileArgs), constants, tool handlers, GkillReadClient (including fetchFile), McpServer (including gkill_get_idf_file tool dispatch/image content block), PKCE (S256/plain verification), OAuth store (token/code/client CRUD, TTL expiration, JSON file persistence), OAuth server (metadata, authorize, token exchange, PKCE, refresh token rotation, DCR with client_id_issued_at, RFC 8707 resource parameter, redirect_uri validation, full E2E flow)
- Android (12 tests, 2 files): unit tests (server URL/port/binary name/socket timeout/retry interval/PID extraction regex/process line filter), instrumented tests (package name/app context validation with filesDir/assets/cacheDir)
- Wear OS (114 tests, 9 files): credential store (SharedPreferences CRUD), API client (MockWebServer: login/KFTL submit/template fetch), wearable listener service (message path handling), activity lifecycle, template cache management, wear client (Data Layer), data models (TemplateNode/PlaingTimeIsNode)

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
| `--log` | (none) | Log level: `none`, `error`, `warn`, `info`, `debug`, `trace`, `trace_sql` |

### Backend (Go) — `src/server/`

Module: `github.com/mt3hr/gkill/src/server` (Go 1.26.0)

Key packages:
- `gkill/api/` — HTTP API handlers via gorilla/mux, **79 POST endpoints** under `/api/`. Main handler: `gkill_server_api.go` (~14,000 lines). Route definitions: `gkill_server_api_address.go`.
- `gkill/api/embed.go` — `//go:embed embed` directive serves the compiled Vue SPA at root `/`
- `gkill/api/req_res/` — Request/response structs for every endpoint (171 types)
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

**Core entity — "Kyou"** (record). Data types: kmemo (text), timeis (timestamps), lantana (mood 0-10), kc (numeric), nlog (expense), urlog (bookmark), mi (task), idf_kyou (file), re_kyou (repost), tag, text, notification, git_commit_log, gps_log (GPS tracks).

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

Stack: Vue 3 + Vuetify 3 + Vue Router + vue-i18n + Vite + PWA (vite-plugin-pwa + Workbox)

- `main.ts` — App entry, registers Vuetify, router, i18n, custom `v-long-press` directive
- `router/index.ts` — 12 routes (login, kftl, mi, rykv, kyou, mkfl, plaing, saihate, set_new_password, regist_first_account, shared_page, shared_mi)
- `pages/` — Route page components
- `pages/views/` — 175 view components for add/edit/list operations per data type
- `pages/dialogs/` — 92 dialog components (Escape key closes via `useFloatingDialog`)
- `classes/api/gkill-api.ts` — Singleton `GkillAPI` class (~3,400 lines), client-side API wrapper
- `classes/datas/` — TypeScript data models mirroring Go structs
- `classes/kftl/` — KFTL parser (44 statement types)
- `serviceWorker.ts` — PWA service worker (Workbox precaching, POST caching, push notifications, Web Share Target)
- `plugins/vuetify.ts` — Vuetify config with light (`gkill_theme`) and dark (`gkill_dark_theme`) themes

**State management:** Props/Emit only. No Pinia/Vuex. `GkillAPI` singleton for backend communication.

**i18n:** 7 languages (ja, en, zh, ko, es, fr, de) in `src/locales/`. ~765 keys per locale. Flat key-value JSON. Shared between frontend (import) and backend (Go embed).

### MCP Server — `src/mcp/`

Read-only MCP server (`gkill-read-server.mjs`) exposing **7 tools** for AI integration:
1. `gkill_get_kyous` — Get records with filters, pagination, type data inlined
2. `gkill_get_mi_board_list` — Get task board names
3. `gkill_get_all_tag_names` — Get all tags
4. `gkill_get_all_rep_names` — Get all repository names
5. `gkill_get_gps_log` — Get GPS logs by date range
6. `gkill_get_application_config` — Get tag hierarchy, board structure, templates
7. `gkill_get_idf_file` — Get IDF file content by rep_name + file_name (proxies backend `/files/` endpoint; images returned as MCP image content blocks)

Transport: stdio (default, `npm run mcp:gkill-read`) or HTTP (`npm run mcp:gkill-read-http`, with OAuth 2.1 auth).

**Environment variables:** `GKILL_BASE_URL`, `GKILL_USER`, `GKILL_PASSWORD_SHA256`, and for HTTP mode: `MCP_TRANSPORT`, `MCP_PORT`, `MCP_OAUTH_ISSUER` (required for remote access — set to public URL e.g. `https://example.com`). OAuth 2.1 (Authorization Code + PKCE) is always enabled for HTTP mode. Supports ChatGPT and Claude.ai MCP connectors via RFC 9728 (Protected Resource Metadata), RFC 8414 (AS Metadata), RFC 7591 (DCR), RFC 8707 (Resource Indicators). Token persistence: refresh tokens and DCR registrations saved to `$GKILL_HOME/configs/mcp_oauth_state.json`.

### Mobile — `src/android/`, `src/wear_os/`

**Android** (`src/android/`): APK wrapper (WebView) bundling gkill_server binary in assets. Copies binary to internal storage, launches it, opens WebView at `http://localhost:9999`. compileSdk=35, targetSdk=28, minSdk=26.

**Wear OS** (`src/wear_os/`): Gradle multi-module project (phone_companion + watch_app). Communicates via Wearable Data Layer. Must copy `gradlew`/`gradlew.bat`/`gradle-wrapper.jar` from `src/android/` before building. Both modules use applicationId `com.gkill_android.mobile_app.src.gkill.mt3hr.gkill`.

## Language

The codebase (variable names, comments, commit messages) is primarily in Japanese. README and documentation are in Japanese.

## Documentation

Reverse-engineered design documents are in `documents/reverse/`:
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
- `api-endpoints.md` — API endpoint reference (79 endpoints)
- `error-handling-and-security.md` — Error handling policy and security design
- `frontend-architecture.md` — Frontend design guide
- `operations-guide.md` — Deployment, backup, troubleshooting, logging
- `dev-setup.md` — Development environment setup guide
- `dvnf-rep-type-spec.md` — DVNF repository type specification
- `folder-structure.md` — Project folder structure reference
- `program-spec.md` — Program specification details
- `user-guide.md` — End-user guide
- `testing-guide.md` — Test execution, architecture, and guidelines (~1,887 tests)

### Test Documentation

Each `src/` subdirectory that has a `README.md` also has an `ABOUT_TEST.md` describing the tests for that area in Japanese. The top-level index is `src/ABOUT_TEST.md`, which links to all 19 subdirectory test specification documents.

- `documents/reverse/testing-guide.md` — Comprehensive test guide (execution, architecture, troubleshooting)
- `src/ABOUT_TEST.md` — Test specification index (links to all subdirectory ABOUT_TEST.md files)
