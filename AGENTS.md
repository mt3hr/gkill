# AGENTS.md

このリポジトリで作業する AI エージェント共通の入口（Codex CLI / Cursor / Claude Code / Gemini CLI / Copilot 等）。
Claude Code は `CLAUDE.md` の import 経由でこのファイルを読み込む。**規約の編集は必ずこのファイル側で行うこと。**

## Project Overview

gkill is a lifelogging application that records and reviews life events (memos, bookmarks, tasks, mood values, expenses, timestamps, files, git commits, etc.). Standalone client-server app: Go backend + Vue 3 frontend, compiled into a single binary with the frontend embedded via `//go:embed`. Licensed under MIT.

## 触る場所ごとの必読資料（ルーティング表）

下の表で**触るファイルに一致する行があれば、編集を始める前に「先に読む」のスキルファイルを Read すること**。
gkill の不変条件の多くは「例外もエラーも出さずに静かに壊れる」種類で、読まずに書くと3列目が起きる。
パス連動のスキル機構を持たないエージェントも、これらはただの Markdown なので同じパスを Read すればよい。

<!-- ROUTING-TABLE:BEGIN 手書きの表。スキルの増減時は必ずここも更新する（verify_docs の checkSkills が双方向網羅を検査） -->

| 触るファイル | 先に読む | 読み落とすと |
|---|---|---|
| `src/server/gkill/api/**`・`src/server/gkill/dao/**`・`src/server/gkill/usecase/**`・`req_res/**` | [gkill-go-backend](.claude/skills/gkill-go-backend/SKILL.md) | rep名絞り込みがキャッシュを丸ごとバイパスし検索が11rep→約940rep・20.7秒になる／`FindQuery.IDs` が6553件でエラーも立てず0件になる／追加したタグが最大1分消え PWA に焼き付く |
| `src/server/gkill/api/find/**`・`find_filter.go`・`src/client/classes/api/find_query/**`・`src/mcp/lib/constants.mjs` | [gkill-find-query](.claude/skills/gkill-find-query/SKILL.md) | null と `[]` の意味を取り違えて例外もエラーも出ずに全件 or 0件になる／TS 側の `undefined` が localStorage 往復で既定値を復活させる |
| `src/server/gkill/main/**`・`gkill_options/**`・`local_rep_cache_path.go` | [gkill-cli-ops](.claude/skills/gkill-cli-ops/SKILL.md) | update_cache / auto_tag をオフライン操作と誤解する（実体は起動中サーバの HTTP クライアント）／派生キャッシュの削除で他ユーザーの分を巻き込む |
| `package.json`・`src/tools/**`・`vite.config.ts`・`playwright.config.ts`・`src/client/__tests__/e2e/**` | [gkill-build-test](.claude/skills/gkill-build-test/SKILL.md) | embed 無しの裸 `go build` が「コンパイルは通るのに起動即死する」バイナリを作る／E2E がポート固定で本番サーバと衝突する |
| `src/plugins/**`・`src/server/gkill/plugin/**`・`api/gkill_plugin/**`・`dao/reps/plugin_*.go`・`plugin-html-view.vue` | [gkill-plugin](.claude/skills/gkill-plugin/SKILL.md) | 1件ずつプラグインへ聞きに行き、一覧の行数ぶんの直列 stdio でプロセスが殺され続ける／Takeout の歩数が2倍になる |
| `src/client/**`（どのファイルでも） | [gkill-client-foundation](.claude/skills/gkill-client-foundation/SKILL.md) | 中継束・再読込手順の手書きで「タグを足しても表示が変わらない」が再発する／成功時 `errors` は null をスプレッドして TypeError |
| `use-rykv-view.ts`・`use-mi-view.ts`（**対称実装。修正は必ず両方へ**）・`use-dashboard-page.ts`・`kyou-local-insert.ts`・`kyou-change-bus.ts` ほか列まわり | [gkill-client-columns](.claude/skills/gkill-client-columns/SKILL.md) | 検索結果が別の列に出る／初期化が永久スピナーで固まる／追加した記録がエラーも警告も出ないまま一覧から消える |
| `src/client/classes/kftl/**`・`kftl-tabs.ts`・`use-kftl-*.ts`・`kftl-view.vue`・`mkfl-view.vue`・`kftl-dialog.vue`・`src/server/gkill/api/kftl/**` | [gkill-client-kftl](.claude/skills/gkill-client-kftl/SKILL.md) | メモ帳が二重登録される／別のタブへ誤配送される／行ラベルが NaN で丸ごと消える |
| `edit-kyou-tags-view.vue`・`kyou-tags.ts`・`use-add-*.ts`・`use-edit-*.ts` | [gkill-client-tags](.claude/skills/gkill-client-tags/SKILL.md) | `add_tag` 完了前に `registered_kyou` を emit すると、タグ付きで追加した記録がエラーも警告も出ないまま一覧に現れない（順序が唯一の防御線） |
| `rudbeckia-page.vue`・`use-rudbeckia-page.ts`・ホストされる `rykv-view.vue` / `mi-view.vue` / `dashboard-view.vue` / `plaing-time-is-view.vue` | [gkill-client-rudbeckia](.claude/skills/gkill-client-rudbeckia/SKILL.md) | バーが画面最上部へ飛ぶ／4枚のウィンドウが完全に重なって1枚に見える／2枚目が1枚目の保存条件を上書きする |
| `src/mcp/**` | [gkill-mcp](.claude/skills/gkill-mcp/SKILL.md) | 並行リクエストで別ユーザーの session/user が混線し、他人のセッションに紐づく file-link URL を発行する |
| `src/android/**`・`src/wear_os/**`・`handle_submit_kftl_text.go` | [gkill-mobile](.claude/skills/gkill-mobile/SKILL.md) | 同梱サーバが全インターフェース待受になり LAN の第三者が無認証で全記録を読み書きできる／打刻が二重登録される |
| `AGENTS.md`・`CLAUDE.md`・`.claude/skills/**`・`documents/**`・`resources/manual_src/**`・各 `ABOUT_TEST.md` | [gkill-docs](.claude/skills/gkill-docs/SKILL.md) | 件数・リンク・生成鮮度の機械検査（verify_docs）が落ちる／マニュアルの7言語セットが崩れる |

<!-- ROUTING-TABLE:END -->

### 症状から引く

| 症状 | 読む |
|---|---|
| 検索が黙って0件になる／HTTP 200 で `errors` が null なのに結果が空 | gkill-go-backend, gkill-find-query |
| タグの個数で検索結果が変わる | gkill-go-backend |
| 追加したタグ・記録が最大1分見えない／PWA が古いまま焼き付く | gkill-go-backend |
| 全種類の検索が数分止まる | gkill-go-backend |
| 検索結果が別の列に出る／列クリックで検索がやり直しになる | gkill-client-columns |
| 追加した記録が一覧に出ない・直後に消える | gkill-client-columns, gkill-client-tags |
| 画面が読み込み中のまま固まる／スピナーが消えない | gkill-client-columns |
| タグを足しても表示が変わらない | gkill-client-foundation（Kyou の再読込） |
| メモ帳が二重登録される／別のタブへ保存された | gkill-client-kftl |
| ダイアログが後ろに隠れる／FAB が押せない／集計リストだけ消える | gkill-client-rudbeckia |
| Android 共有が2件保存される／ログイン失敗でエラー表示が消える | gkill-client-foundation |
| プラグインの本文がときどき空白／プロセスが殺され続ける／歩数が2倍 | gkill-plugin |
| MCP で他人のセッションの URL が発行される／`ErrPluginBusy` | gkill-mcp |
| LAN の第三者が無認証で読める／打刻が二重登録される | gkill-mobile |
| `npm run verify_docs` が落ちた | gkill-docs |

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

**Prerequisites:** Go 1.26.6+ (`src/server/go.mod` declares `go 1.26.6`), Node.js 20.19+ (24.x recommended — `package.json` has no `engines` field, so this is not enforced), `npm i`

ビルドパイプラインの内訳・`go build` の罠・dev プロキシ・E2E 環境の詳細は [gkill-build-test](.claude/skills/gkill-build-test/SKILL.md)。

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
  plugins/    # 独立プラグインバイナリ群（1プラグイン=1 go.mod。実測仕様と約束は gkill-plugin スキル）
```

**Core entity — "Kyou"** (record). Data types: kmemo (text), timeis (timestamps), lantana (mood 0-10), kc (numeric), nlog (expense), urlog (bookmark), mi (task), idf_kyou (file, with `is_zip` flag for .zip/.cbz), re_kyou (repost), mi_re_kyou (an existing Kyou turned into a task: `target_id` + Mi scheduling fields, no title; `data_type` is `mirekyou_create` / `_check` / `_limit` / `_start` / `_end`, so prefix checks must test `mirekyou` **before** `mi`; the client's `load_typed_datas` guarantees this structurally by sorting its prefix table longest-first rather than relying on the order the branches are written in — `kyou-typed-data-dispatch.test.ts`), tag, text, notification, git_commit_log, gps_log (GPS tracks), plugin_kyou (external plugin data — `typed_plugin` field non-null in TypeScript `Kyou` class; `data_type` is plugin-defined e.g. `claude_conversation`).

**Response structure:** All API responses include `messages []GkillMessage` and `errors []GkillError` (with `error_code` + `error_message`). HTTP 200 for normal responses (check `errors` array), 403 for access denied, 500 for unexpected errors. The Go struct tags carry no `omitempty`, so **on success `messages` / `errors` come back as `null`, not as an empty array**. On the client always go through `res.errors ?? []` **before spreading or `concat`** (a bare `[...res.errors]` throws `TypeError`; `[].concat(null)` silently pushes `null` as an element). `if (res.errors && res.errors.length !== 0) { ... }` のガードで囲うのも同じく正しく、そちらが約180箇所と多数派。**ガード方式をわざわざ `?? []` へ書き換える必要はない** —— 守るべきなのは「null のまま展開しない」の1点だけ。

## Cross-cutting conventions

**Naming convention (files):** `{action}-{feature}-{entity}-{component}` (e.g., `add-dnote-item-view.vue`, `confirm-delete-ryuu-item-dialog.vue`), kebab-case. Dnote and Ryuu follow the same pattern.

**Frozen spellings:** `plaing` / `Plaing` は "playing" の誤綴りではなく**製品綴りとして凍結**（ルート `/plaing`、`FindQuery` の `plaing_time`、MCP ツールスキーマ、Wear OS データレイヤーパス、マニュアルのページ名、`default_page` の保存値に浸透しているため）。綴り修正の提案はしないこと。詳細は `documents/reverse/glossary.md` の「凍結された綴り」節。 判定基準（永続データ／外部契約に乗っているか）と境界の一覧は [ADR-0060](documents/adr/0060-freeze-plaing-spelling.md)。

**Naming convention (identifiers):** データクラスのプロパティ/メソッド・ローカル変数・通常関数は snake_case（Go 側 JSON タグとの写像）。コンポーザブルは `useXxx`、イベントコールバックは `onXxx`、CRUD リレーハンドラ束は `xxxHandlers`（束の生成は `kyou-view-relay.ts` に一元化。いずれも camelCase）。型は PascalCase、enum メンバーは snake_case。`@typescript-eslint/naming-convention` で機械検査される（`eslint.config.js` の `app/naming-convention` ブロック。対象は `src/client` 本体のみで、`__tests__`・`src/mcp`・`src/tools`・`*.d.ts` は別流儀として対象外）。

**i18n:** 7 languages (ja, en, zh, ko, es, fr, de) in `src/locales/`. 919 keys per locale. Flat key-value JSON. Shared between frontend (import) and backend (Go embed).

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

- 領域別の禁止文・不変条件の正本: `.claude/skills/gkill-*/SKILL.md`（上のルーティング表から引く）
- なぜそうなっているか（却下案・実測値・事件譚）: `documents/adr/`（索引: [documents/adr/README.md](documents/adr/README.md)）
- 現在どうなっているか（リバース資料）: `documents/reverse/`（索引: [documents/reverse/README.md](documents/reverse/README.md)）
- テスト仕様の目次: `src/ABOUT_TEST.md`。利用者向けマニュアル原稿: `resources/manual_src/`（7言語セット、`npm run build_manuals` で生成）
- 資料の件数・リンク・ファイル名実在・スキル索引は `npm run verify_docs` が機械検査する。
  資料層の保守手順は [gkill-docs](.claude/skills/gkill-docs/SKILL.md) スキルにある

## AI エージェントへの約束

- **個人情報・実環境の情報をリポジトリへ入れない（最重要）。** 実在の利用者ID・人名・メールアドレス・
  端末のローカル絶対パス（`C:\Users\〈名前〉` など）・実データの中身を、コード・資料・テストデータ・
  コミットメッセージのどこにも書かない。テストの利用者IDは `testuser` 系、例示パスは `$HOME` や
  `〈ユーザー名〉` のプレースホルダで書く。`npm run verify_docs` が資料への混入をパターン検査するが、
  検査は網でしかない — 書く前に止めることがすべて。
- **このファイルと `CLAUDE.md` に領域別の規約本文を書き足さない。** 正本は `.claude/skills/*/SKILL.md`。
  ここが太ると全タスクの常時コンテキストを食う。サイズ上限（verify_docs が検査）に当たったら、
  上限を上げるのではなく中身をスキルへ落とすこと。
- 資料に書いた件数・リンク・ファイル名は `npm run verify_docs`（`npm test` に含まれる）が機械検査する。
  数字を書いたら `src/tools/verify_docs.mjs` の検査にも載せること。
- 作業報告・コミットメッセージ・新規コメントは日本語で書く（コメント言語の詳細は Language 節）。
