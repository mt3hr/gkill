# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

gkill is a lifelogging application that records and reviews life events (memos, bookmarks, tasks, mood values, expenses, timestamps, files, git commits, etc.). It is a standalone client-server app with a Go backend and Vue 3 frontend, compiled into a single binary with the frontend embedded.

## Build & Development Commands

All commands are npm scripts. CGO is required (sqlite3 dependency).

| Command | Purpose |
|---|---|
| `npm run dev` | Start Vite dev server (frontend only) |
| `npm run build` | Build frontend (type-check + vite build) |
| `npm run lint` | ESLint with auto-fix for .vue/.ts/.js files |
| `npm run install_server` | Full build: frontend → embed → `go install` (headless server) |
| `npm run install_app` | Full build: frontend → embed → `go install` (desktop app with window) |
| `npm run go_install` | Go install only (skip frontend rebuild) |
| `npm run go_mod` | Regenerate go.mod and go.sum from scratch |
| `npm run release` | Cross-compile release for all platforms |

**Build pipeline** (`install_server`): clean embed dir → write version.json → vite build → copy dist/ to `src/server/gkill/api/embed/html/` → copy locales to embed → `go install`

**Prerequisites:** Go 1.25.5+, C compiler (CGO), Node.js 20.15.1+, `npm i`

**No automated tests exist.** No Go test files or JS test files.

## Architecture

### Two deployment modes

- **`gkill_server`** (`src/server/gkill/main/gkill_server/main.go`): HTTP server accessed via browser/PWA
- **`gkill`** (`src/server/gkill/main/gkill/main.go`): Desktop app wrapping the server with go-astilectron

Both use cobra for CLI subcommands (version, idf, dvnf, generate_thumb_cache, generate_video_cache, optimize, update_cache).

### Backend (Go) — `src/server/`

Module: `github.com/mt3hr/gkill/src/server` (Go 1.26.0)

Key packages:
- `gkill/api/` — HTTP API handlers via gorilla/mux, all POST endpoints under `/api/`. The main handler file `gkill_server_api.go` is ~557KB with 60+ endpoints.
- `gkill/api/embed.go` — `//go:embed embed` directive serves the compiled Vue SPA at root `/`
- `gkill/api/req_res/` — Request/response structs for every endpoint
- `gkill/api/kftl/` — KFTL custom text format parser
- `gkill/dao/` — Data access layer with `GkillDAOManager` managing SQLite3 databases
- `gkill/dao/reps/` — Repository interfaces and implementations for each data type
- `gkill/main/common/` — Shared CLI commands, server initialization, logging

**Repository pattern:** Each data type has 4 implementation layers:
1. `*_repository.go` — interface
2. `*_repository_sqlite3_impl.go` — SQLite3 implementation
3. `*_repository_cached_sqlite3_impl.go` — cached variant
4. `*_repository_temp_sqlite3_impl.go` — temp/transactional variant

**Core entity — "Kyou"** (record). Data types: kmemo (text), timeis (timestamps), lantana (mood 0-10), kc (numeric), nlog (expense), urlog (bookmark), mi (task), idf_kyou (file), re_kyou (repost), tag, text, notification, git_commit_log.

### Frontend (Vue 3 + TypeScript) — `src/client/`

Stack: Vue 3 + Vuetify 3 + Vue Router + vue-i18n + Vite + PWA

- `main.ts` — App entry, registers Vuetify, router, i18n, custom `v-long-press` directive
- `pages/` — Route pages (login, kftl, mi, rykv, kyou, plaing, saihate, etc.)
- `pages/views/` — 200+ view components for add/edit/list operations per data type
- `pages/dialogs/` — 100+ dialog components
- `classes/api/gkill-api.ts` — Singleton `GkillAPI` class (~171KB), client-side API wrapper
- `classes/datas/` — TypeScript data models mirroring Go structs

i18n: 7 languages (ja, en, zh, ko, es, fr, de) in `src/locales/`

### MCP Server — `src/mcp/`

Read-only MCP server (`gkill-read-server.mjs`) exposing 5 tools for AI integration. Run with `npm run mcp:gkill-read`.

### Mobile — `src/android/`, `src/wear_os/`

Android APK wrapper bundling gkill_server as an asset. Wear OS companion + watch app pair. Both are Gradle projects.

## Language

The codebase (variable names, comments, commit messages) is primarily in Japanese. README and documentation are in Japanese.
