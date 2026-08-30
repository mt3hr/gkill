---
name: gkill-build-test
description: "gkill のビルド・テスト・E2E 実行の約束。ビルドパイプライン（dist と i18n とマニュアルの embed への3コピー）、embed 無しの裸 go build が起動即死バイナリを作る罠、dev サーバの --api プロキシ指定、E2E の専用ホームと空きポート自動採番（ポートを固定しない）を扱う。package.json・src/tools/・vite.config.ts・playwright.config.ts・src/client/__tests__/e2e/・.github/workflows/ を編集するとき、ビルドやテストの実行を頼まれたとき必読。「ビルドは通るのに起動即死する」「E2E が本番サーバと衝突する」の調査でも必読。"
---

# ビルド・テスト・E2E の約束

対象: `package.json` / `src/tools/**` / `vite.config.ts` / `playwright.config.ts` / `src/client/__tests__/e2e/**` / `.github/workflows/**`
（npm スクリプトの一覧表は `AGENTS.md` にある）

**このファイルは全文が、実際に起きた事故の再発防止である。該当作業では飛ばさずに読むこと。**
多くは「例外もエラーも出さずに静かに壊れる」種類で、破っても目の前ではエラーにならない。

**Build pipeline** (`install_server` → `prepare_install`): `clean_dist` → clean embed dir → write `version.json` (commit hash + build time + version) → `vue-tsc` type-check → `vite build` → copy `dist/` to `src/server/gkill/api/embed/html/` → copy `src/locales/` to `src/server/gkill/api/embed/i18n/locales/` → `build_manuals` (`src/tools/build_manuals.mjs`: `resources/manual_src/` → `resources/manual/`) → copy `resources/manual/` to `src/server/gkill/api/embed/manual/` (3 copies: html + i18n + manual) → `go install`

**Bare `go build` caveat:** any gkill binary panics at startup if `embed/i18n/locales` is empty (`gkill/api`'s `init()` reads it), so a `go build` without a preceding `npm run copy_i18n_to_app_embed` produces a binary that compiles but dies immediately.

**APK リリース署名（2026-08-30、監査 F-006）:** `npm run release` の APK 3本（Android / Wear companion / watch）は `assembleRelease` + リリース署名で作る（以前は `assembleDebug` の成果物を配布名へ rename しており、debug 鍵署名の APK が正式版として公開されていた）。鍵はリポジトリへ入れず、`~/.gradle/gradle.properties` か環境変数で `GKILL_RELEASE_KEYSTORE` / `GKILL_RELEASE_KEYSTORE_PASSWORD` / `GKILL_RELEASE_KEY_ALIAS` / `GKILL_RELEASE_KEY_PASSWORD` を渡す（初回は `keytool -genkeypair -v -keystore 〈保管先〉.jks -alias gkill -keyalg RSA -keysize 4096 -validity 10950` で作成し、keystore とパスワードはリポジトリ外で保管する）。未設定の `assembleRelease` は未署名 APK（`app-release-unsigned.apk`）になり、配布名への rename が見つからずパイプラインが止まる — 黙って debug 鍵の配布物へ戻ることはない。`verify_release_artifacts` が `apksigner verify --print-certs` で署名の実在と「debug 証明書（CN=Android Debug）でないこと」を検査し、apksigner が見つからない環境では fail-closed で止まる。**署名鍵が debug から変わった版へ更新する既存インストール端末は、署名不一致でそのまま更新できず、アンインストール→再インストールが必要**（リリースノートに明記すること）。

**Dev server backend target:** `npm run dev` runs `src/tools/dev.mjs`, a thin wrapper that strips `--api` / `--api-target` (Vite errors on unknown CLI options) and passes the value to Vite as `GKILL_API_PROXY_TARGET`; all other args pass through untouched. Accepts a full URL, a bare port (`--api=19999` → `http://127.0.0.1:19999`), or `host:port`. Precedence: `--api` > `GKILL_API_PROXY_TARGET` > `http://localhost:9999`. `vite.config.ts` proxies `/api`, `/files`, `/zip_cache` and `/resources/manual` to that target — SPA routes (`/rykv`, `/kftl`, `/mi`, `/rudbeckia`, …) are deliberately not proxied since vue-router owns them in dev.

**E2E test environment:** `$HOME/gkill_test` as dedicated gkill home. `npm run test_client_e2e` automatically cleans, starts gkill_server (fresh admin, no password) and a Vite dev server, runs Playwright, and stops both. See `src/client/__tests__/e2e/run-e2e.mjs`.

**Ports are never fixed in tests.** `run-e2e.mjs` allocates free ports from the OS (`src/client/__tests__/e2e/free-port.mjs`) and passes them as `--address 127.0.0.1:<port>` (gkill_server) and `--port <port> --strictPort` (Vite), so a production gkill_server occupying `:9999` never collides. The ports reach the tests via `GKILL_E2E_BASE_URL` / `GKILL_E2E_VITE_URL` (read by `playwright.config.ts`, `check-server.ts`, `auth.setup.ts`), and Vite's `/api` proxy target via `GKILL_API_PROXY_TARGET` (`vite.config.ts`) — which also prevents E2E writes from ever reaching a production server. Leftover-process cleanup only kills `gkill_server` processes whose command line contains `gkill_test`. Nothing else in `npm test` binds a fixed port: Go tests use `httptest` (ephemeral), and MCP/Android/Wear OS tests bind none.

**Test details:** See `documents/reverse/testing-guide.md`, `src/ABOUT_TEST.md` for per-directory test specs.

## 関連スキル

- [gkill-docs](../gkill-docs/SKILL.md) — `npm run verify_docs` が検査する内容（`npm test` に含まれる）
