# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

gkill is a lifelogging application (version 1.1.0-dev) that records and reviews life events (memos, bookmarks, tasks, mood values, expenses, timestamps, files, git commits, etc.). It is a standalone client-server app with a Go backend and Vue 3 frontend, compiled into a single binary with the frontend embedded via `//go:embed`. Licensed under MIT.

## Build & Development Commands

All commands are npm scripts defined in `package.json`. CGO is required (sqlite3 dependency).

| Command | Purpose |
|---|---|
| `npm run dev` | Start Vite dev server (frontend only) |
| `npm run build` | Build frontend (`vue-tsc --build` + `vite build` in parallel) |
| `npm run lint` | ESLint with auto-fix for `.vue`/`.ts`/`.js` files |
| `npm run install_server` | Full build: frontend → embed → `go install` (headless HTTP server) |
| `npm run install_app` | Full build: frontend → embed → `go install` (desktop app with go-astilectron window) |
| `npm run go_install` | Go install only (skip frontend rebuild) |
| `npm run go_mod` | Regenerate `go.mod` and `go.sum` from scratch |
| `npm run release` | Cross-compile release for all platforms (Windows, Linux, Android, Wear OS) |

**Build pipeline** (`install_server`): clean embed dir → write `version.json` (commit hash + build time + version) → `vue-tsc` type-check → `vite build` → copy `dist/` to `src/server/gkill/api/embed/html/` → copy `src/locales/` to `src/server/gkill/api/embed/i18n/locales/` → `go install`

**Prerequisites:**
- Go 1.26.0+ (see `src/server/go.mod`)
- C compiler (CGO required for mattn/go-sqlite3)
- Node.js 20.15.1+
- `npm i` (install JS dependencies)

**No automated tests exist.** No Go test files or JS test files.

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
- `gkill/api/` — HTTP API handlers via gorilla/mux, **77 POST endpoints** under `/api/`. Main handler: `gkill_server_api.go` (~12,300 lines). Route definitions: `gkill_server_api_address.go`.
- `gkill/api/embed.go` — `//go:embed embed` directive serves the compiled Vue SPA at root `/`
- `gkill/api/req_res/` — Request/response structs for every endpoint (150+ types)
- `gkill/api/kftl/` — KFTL custom text format parser (single package, no sub-packages)
- `gkill/dao/` — Data access layer with `GkillDAOManager` managing SQLite3 databases
- `gkill/dao/reps/` — Repository interfaces and implementations for each data type
- `gkill/main/common/` — Shared CLI commands, server initialization, logging
- `gkill/main/common/gkill_options/` — CLI flag definitions and directory structure

**Key dependencies:** gorilla/mux (router), mattn/go-sqlite3 (DB), spf13/cobra+viper (CLI/config), asticode/go-astilectron (desktop), SherClockHolmes/webpush-go (VAPID), go-git/go-git (git integration), twpayne/go-gpx (GPS), nicksnyder/go-i18n/v2 (i18n), google/uuid

**Repository pattern:** Each data type has 4 implementation layers:
1. `*_repository.go` — interface
2. `*_repository_sqlite3_impl.go` — SQLite3 implementation
3. `*_repository_cached_sqlite3_impl.go` — cached variant (in-memory)
4. `*_repository_temp_sqlite3_impl.go` — temp/transactional variant

**Core entity — "Kyou"** (record). Data types: kmemo (text), timeis (timestamps), lantana (mood 0-10), kc (numeric), nlog (expense), urlog (bookmark), mi (task), idf_kyou (file), re_kyou (repost), tag, text, notification, git_commit_log.

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
- `router/index.ts` — 13 routes (login, kftl, mi, rykv, kyou, mkfl, plaing, saihate, set_new_password, regist_first_account, shared_page, shared_mi)
- `pages/` — Route page components
- `pages/views/` — 200+ view components for add/edit/list operations per data type
- `pages/dialogs/` — 100+ dialog components
- `classes/api/gkill-api.ts` — Singleton `GkillAPI` class (~2,900 lines), client-side API wrapper
- `classes/datas/` — TypeScript data models mirroring Go structs
- `classes/kftl/` — KFTL parser (35+ statement types)
- `serviceWorker.ts` — PWA service worker (Workbox precaching, POST caching, push notifications, Web Share Target)
- `plugins/vuetify.ts` — Vuetify config with light (`gkill_theme`) and dark (`gkill_dark_theme`) themes

**State management:** Props/Emit only. No Pinia/Vuex. `GkillAPI` singleton for backend communication.

**i18n:** 7 languages (ja, en, zh, ko, es, fr, de) in `src/locales/`. ~743 keys per locale. Flat key-value JSON. Shared between frontend (import) and backend (Go embed).

### MCP Server — `src/mcp/`

Read-only MCP server (`gkill-read-server.mjs`) exposing **6 tools** for AI integration:
1. `gkill_get_kyous` — Get records with filters, pagination, type data inlined
2. `gkill_get_mi_board_list` — Get task board names
3. `gkill_get_all_tag_names` — Get all tags
4. `gkill_get_all_rep_names` — Get all repository names
5. `gkill_get_gps_log` — Get GPS logs by date range
6. `gkill_get_application_config` — Get tag hierarchy, board structure, templates

Transport: stdio (default, `npm run mcp:gkill-read`) or HTTP (`npm run mcp:gkill-read-http`, with API key auth).

**Environment variables:** `GKILL_BASE_URL`, `GKILL_USER`, `GKILL_PASSWORD_SHA256`, and for HTTP mode: `MCP_TRANSPORT`, `MCP_API_KEY`, `MCP_PORT`.

### Mobile — `src/android/`, `src/wear_os/`

**Android** (`src/android/`): APK wrapper (WebView) bundling gkill_server binary in assets. Copies binary to internal storage, launches it, opens WebView at `http://localhost:9999`. compileSdk=35, targetSdk=28, minSdk=26.

**Wear OS** (`src/wear_os/`): Gradle multi-module project (phone_companion + watch_app). Communicates via Wearable Data Layer. Must copy `gradlew`/`gradlew.bat`/`gradle-wrapper.jar` from `src/android/` before building. Both modules use applicationId `com.gkill_android.mobile_app.src.gkill.mt3hr.gkill`.

## Language

The codebase (variable names, comments, commit messages) is primarily in Japanese. README and documentation are in Japanese.

## Documentation

Reverse-engineered design documents are in `documents/reverse/`:
- `glossary.md` — Domain term definitions (80+ terms)
- `design-philosophy.md` — Architecture decisions and rationale
- `er-diagram.md` — Entity-relationship diagrams (Mermaid)
- `class-diagrams.md` — Go/TypeScript class hierarchies
- `sequence-diagrams.md` — Interaction sequences (16 diagrams)
- `activity-diagrams.md` — Processing flowcharts
- `state-machines.md` — Entity state transitions
- `screen-transition.md` — Page/dialog navigation
- `usecase.md` — Use case catalog (80+ use cases)
- `api-endpoints.md` — API endpoint reference (77 endpoints)
- `error-handling-and-security.md` — Error handling policy and security design
- `frontend-architecture.md` — Frontend design guide
- `operations-guide.md` — Deployment, backup, troubleshooting, logging
