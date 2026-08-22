---
name: gkill-cli-ops
description: "gkill の CLI サブコマンドと運用（src/server/gkill/main/）の約束。gkill_server / gkill の2配布形態、update_cache と auto_tag が起動中サーバの HTTP クライアントであること、configs 直書きの自己発行セッションとその権限、auto_tag のタグID採番と再実行、clear_cache のユーザー別派生キャッシュ（thumb/video/zip/plugin）、CLI フラグ一覧、RunE + SilenceUsage/SilenceErrors と errors.Join を扱う。src/server/gkill/main/・gkill_options/・local_rep_cache_path.go を編集するとき、サブコマンドを足すとき必読。「削除したタグが復活する」「キャッシュがディスクを埋める」の調査でも必読。"
---

# CLI サブコマンドと運用の約束

対象: `src/server/gkill/main/**` / `src/server/gkill/main/common/gkill_options/**` / `src/server/gkill/dao/reps/local_rep_cache_path.go`

**このファイルは全文が、実際に起きた事故の再発防止である。該当作業では飛ばさずに読むこと。**
多くは「例外もエラーも出さずに静かに壊れる」種類で、破っても目の前ではエラーにならない。

### Two Deployment Modes

- **`gkill_server`** (`src/server/gkill/main/gkill_server/main.go`): Headless HTTP server (browser/PWA)
- **`gkill`** (`src/server/gkill/main/gkill/main.go`): Desktop app wrapping server with go-astilectron

Both use cobra for CLI with shared subcommands: `version`, `dvnf`, `generate_thumb_cache`, `generate_video_cache`, `optimize`, `update_cache`, `clear_cache`, `reset_password`, `auto_tag`. `idf` is registered by `gkill_server` only (`main/gkill/main.go` does not register `IDFCmd`). Default listen address: `:9999`, TLS disabled, initial user `admin` with no password.

`update_cache` and `auto_tag` are **HTTP clients of the running server**, not offline operations — they need gkill_server to be up. Neither can log in with a password (Argon2id), so they mint a short-lived session by writing a row into `configs/account_state.db` directly (`main/common/password_admin.go`: `issueLocalSession` / `issueLocalAdminSession`); the trust basis is being able to write the config dir on the same machine. `update_cache` uses an admin session; `auto_tag` uses **the target user's own session**, because `/api/get_kyous` and `/api/add_tag` act as the session's user and an admin session would see admin's repositories. Both resolve the endpoint through `common.ResolveLocalServerEndpoint` (reads `server_config.db`, skips TLS verification for the localhost self-signed cert).

`auto_tag <user_id...>` adds tags by repository rule — `--tag_by_rep_prefix '<rep name prefix>=<tag>'` and `--tag_by_rep_name '<rep type>'` (tag = the rep's own name), both repeatable, plus `--dry_run`. It decides "already tagged?" by diffing two `/api/get_kyous` calls (unfiltered vs. `tags` non-null + **`tags_and: true`**). Both the OR and AND branches of `filterTagsKyous` match tag names exactly (case-insensitive) — `gkill` never matches `gkill_autolog` — so either would work; `tags_and` is kept because a single-tag "has this tag" query is what AND expresses directly. Tag IDs are a UUIDv5 of (target id, tag name), so reruns never duplicate: the server rejects an existing ID with `ERR000056` `AlreadyExistTagError`, which the command counts as a skip. That also means a **manually deleted tag is never resurrected**, since the deleted row keeps the same ID.

`clear_cache <thumb|video|zip|plugin|all> <all|user_id...>` deletes the on-disk derived caches (`thumb_cache` / `video_cache` / `zip_cache` / `plugin_cache`). The target is required (matching `generate_thumb_cache`/`optimize` etc. which require positional user args): pass the literal `all` to remove the whole cache dirs under `$HOME/gkill/caches/` globally (no user context needed), or one or more user_ids to load each user's repositories (`LoadIDFRepOnly`) and clear only that user's IDF-rep caches via `IDFKyouReps.Clear{Thumb,Video,Zip}Cache(userID)`. `plugin` mode is a plain directory removal (`ClearPluginCache`) and skips the repository load entirely. Missing target or unknown mode prints usage. All three derived caches are keyed per user — `caches/zip_cache/{userID}/{repName}/{sha1(zipPath)}/` and `caches/{thumb,video}_cache/{userID}/{repName}/` — built by `derivedCacheDirForUser` in `dao/reps/local_rep_cache_path.go`. Rep names are NOT unique across users (`filepath.Base(contentDir)`, no UNIQUE constraint), so name matching alone cannot isolate them; that's why all three `Clear*Cache(userID)` take the user id. Thumb/video have no dedicated route — they are only reachable via `/files/{repName}/...?thumb=`. rep名の照合では原理的に守れない理由は [ADR-0044](../../../documents/adr/0044-per-user-derived-cache-dir.md)。

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

- `gkill/main/common/` — Shared CLI commands, server initialization, logging
- `gkill/main/common/gkill_options/` — CLI flag definitions and directory structure

**CLIサブコマンドは `RunE` + `SilenceUsage/SilenceErrors`**（2026-08-21、監査 M-8）。失敗で exit 1（main の `log.Fatal` が唯一のエラー出力）。ユーザごとのループは `errors.Join` で集約し、途中失敗でも成功分の結果（reset_password の URL 等）はその場で即出力してから続行する（`os.Exit` を足すと defer のセッション後始末が飛ぶので RunE 経由で返す）。**互換注意**: `SyncDatas` 等が update_cache の失敗を初めて exit code で観測する。`auto_tag` は長時間実行向けに `issueLocalSession` の `refresh` でセッション期限を延長する。

## 関連スキル

- [gkill-go-backend](../gkill-go-backend/SKILL.md) — 派生キャッシュの構造・rep名で利用者を分離できない理由
- [gkill-build-test](../gkill-build-test/SKILL.md) — ビルドと E2E の実行手順

## 詳しい設計と却下案（ADR）

- [ADR-0044 派生キャッシュはユーザー別ディレクトリ](../../../documents/adr/0044-per-user-derived-cache-dir.md)
