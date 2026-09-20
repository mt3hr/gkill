# テスト仕様書 索引

## 概要

gkill プロジェクトには Go バックエンド、Vue 3 フロントエンド、MCP サーバ、Android、Wear OS の各コンポーネントにテストが存在する。加えて `src/plugins/` の各プラグイン（独立した Go モジュール）にもテストがある。

## テスト統計

数値は **トップレベルのテスト宣言数**（静的カウント）であり、実行時のケース数とは一致しない。`t.Run` のサブテストや `it.each` / `test.each` のパラメタライズド展開により、実行時のケース数はこれより多くなる。MCP サーバだけは旧 vitest の `test` を `t.Run` に 1:1 で移してあるので `t.Run` を数える（Go バックエンドの表には含めない）。
この表の値は `npm run verify_docs -- --list` が算出するメトリクスと同じ定義で、`verify_docs` が実測と突合している。

| コンポーネント | テスト宣言数 | テストファイル数 | フレームワーク |
|--------------|---------|----------------|---------------|
| Go バックエンド (`server/`) | 1350 | 202 | Go `testing` |
| フロントエンド ユニット (`client/`) | 2019 | 178 | Vitest |
| フロントエンド E2E (`client/`) | 253 | 46 | Playwright |
| MCP サーバ (`server/gkill/mcp/`) | 1121 | 36 | Go `testing` |
| ツール (`tools/`) | 55 | 2 | Vitest |
| Android (`android/`) | 15 | 2 | JUnit 4 |
| Wear OS (`wear_os/`) | 228 | 18 | JUnit 4 + MockK |
| **合計** | **5,041** | **484** | |

`src/plugins/` の Go テスト 196件は独立モジュールのため上表（`src/server` 基準の集計）には含まれない。実行は `npm run test_plugins` が担当し、`npm test` からも呼ばれる（[plugins/ABOUT_TEST.md](plugins/ABOUT_TEST.md) 参照）。

### テストの書き方の方針

宣言数は指標ではない。次の方針で「数を稼ぐだけのテスト」を実効あるテストに置き換えている。

- **型やコンパイラが保証済みのものは書かない** — `typeof api.foo === 'function'`、コンストラクタで代入した既定値、`instanceof` の確認など。`vue-tsc --build` と Go のコンパイラが既に保証している
- **同じ経路の繰り返しはテーブル駆動に畳む** — 例: エンドポイントごとに書かれていた認証エラーテスト48本は、フィクスチャを1回だけ作る `TestAuthMiddleware_RejectsInvalidSession` の96サブテストになった（粒度は `t.Run` で維持、実行時間は約40秒短縮）
- **手書きの一覧はソースから機械的に導出する** — 例: エラーコードの重複チェックは、手書きした29個だけでなく `go/parser` で読み取った全定数（導入時406個。定数を足せば対象も自動で増える）が対象になった
- **条件で包んで「静かに成功する」テストを作らない** — E2E の `if (await x.count() > 0) { ...本体... }` は、要素が見つからないと何も検証せずパスする。移行は完了しており、`eslint.config.js` の `playwright/no-conditional-in-test` / `no-wait-for-timeout` は**全 spec に error** で効く（免除リストは無い）

<details>
<summary>カウント方法（再現手順）</summary>

```bash
# Go: トップレベルのテスト関数（t.Run サブテストは含まない）
grep -rhE '^func Test' src/server --include=*.go | wc -l

# フロントエンド ユニット / E2E: 行頭のテスト宣言
grep -rhE '^\s*(it|test)(\.each)?\(' src/client/__tests__/unit --include=*.test.ts | wc -l
# MCP: t.Run のサブテスト宣言（旧 vitest の test と 1:1）
grep -rhE '^\s*t\.Run\(' src/server/gkill/mcp --include=*_test.go | wc -l
grep -rhE '^\s*(it|test)(\.each)?\(' src/tools/__tests__ --include=*.test.mjs | wc -l
grep -rhE '^\s*test\(' src/client/__tests__/e2e --include=*.spec.ts | wc -l

# 上と同じ定義を機械的に出す（doc の数値はこれと突合される）
npm run verify_docs -- --list
```

</details>

## テスト実行コマンド

| コマンド | 対象 |
|---------|------|
| `npm test` | 全テスト。先に `install_server`（ビルド）と `verify_docs`（docs CI）を実行してから server + client + MCP + tools + plugins + Android + Wear OS。各 `test_*` は `src/tools/run_test_suite.mjs` 経由で、成功したスイートを `test_attestation.local.json` に記録する（`npm run release` のゲートが読む） |
| `npm run test_plugins` | `src/plugins/` の各プラグイン（独立 Go モジュール） |
| `npm run vet_plugins` | 同梱プラグインへ `go vet`（CI の `plugins` ジョブが `test_plugins` の前に回す。`npm test` には入っていない） |
| `npm run test_server` | Go バックエンド (`src/server` で `go test ./...`) |
| `npm run test_client` | フロントエンド（ユニット + E2E） |
| `npm run test_client_unit` | フロントエンドユニットテストのみ |
| `npm run test_client_e2e` | フロントエンド E2E テストのみ |
| `npm run test_mcp` | MCP サーバ（`src/server` で `go test ./gkill/mcp/...`。`test_server` にも含まれる） |
| `npm run test_tools` | `src/tools/` のリリースゲート・attestation ランナー |
| `npm run test_android` | Android |
| `npm run test_wear_os` | Wear OS |

## 各ディレクトリの ABOUT_TEST.md

| ディレクトリ | テスト仕様 | 概要 |
|-------------|-----------|------|
| `client/` | [client/ABOUT_TEST.md](client/ABOUT_TEST.md) | フロントエンド全体（unit 2019 + E2E 253） |
| `client/classes/` | [client/classes/ABOUT_TEST.md](client/classes/ABOUT_TEST.md) | ユーティリティクラス |
| `client/classes/api/` | [client/classes/api/ABOUT_TEST.md](client/classes/api/ABOUT_TEST.md) | GkillAPI クライアント |
| `client/classes/datas/` | [client/classes/datas/ABOUT_TEST.md](client/classes/datas/ABOUT_TEST.md) | 35ファイル（データモデル + 横断検証） |
| `client/classes/dnote/` | [client/classes/dnote/ABOUT_TEST.md](client/classes/dnote/ABOUT_TEST.md) | D-note 集計・フィルタ |
| `client/classes/kftl/` | [client/classes/kftl/ABOUT_TEST.md](client/classes/kftl/ABOUT_TEST.md) | KFTL 行分類器 (TypeScript。行ラベル専用) と送信経路 |
| `client/pages/` | [client/pages/ABOUT_TEST.md](client/pages/ABOUT_TEST.md) | E2E + Composable + Router |
| `locales/` | [locales/ABOUT_TEST.md](locales/ABOUT_TEST.md) | i18n 完全性検証（7言語） |
| `server/` | [server/ABOUT_TEST.md](server/ABOUT_TEST.md) | Go バックエンド全体（1350テスト / 32パッケージ） |
| `server/gkill/api/` | [server/gkill/api/ABOUT_TEST.md](server/gkill/api/ABOUT_TEST.md) | API 共通基盤（FindFilter等） |
| `server/gkill/api/gkill_server_api/` | [server/gkill/api/gkill_server_api/ABOUT_TEST.md](server/gkill/api/gkill_server_api/ABOUT_TEST.md) | API ハンドラ統合テスト（handle_*.go 実装91ファイル） |
| `server/gkill/api/kftl/` | [server/gkill/api/kftl/ABOUT_TEST.md](server/gkill/api/kftl/ABOUT_TEST.md) | KFTL パーサ (Go) |
| `server/gkill/api/req_res/` | [server/gkill/api/req_res/ABOUT_TEST.md](server/gkill/api/req_res/ABOUT_TEST.md) | ワイヤ契約（JSONタグ名 / omitempty） |
| `server/gkill/dao/` | [server/gkill/dao/ABOUT_TEST.md](server/gkill/dao/ABOUT_TEST.md) | DAO 層全体 |
| `server/gkill/dao/reps/` | [server/gkill/dao/reps/ABOUT_TEST.md](server/gkill/dao/reps/ABOUT_TEST.md) | 13リポジトリ実装（IDFKyou・MiReKyou含む）+ プラグインのサブプロセス管理 |
| `server/gkill/usecase/` | [server/gkill/usecase/ABOUT_TEST.md](server/gkill/usecase/ABOUT_TEST.md) | ビジネスロジック層（ハンドラ経由で33〜50%到達） |
| `server/gkill/dvnf/` | [server/gkill/dvnf/ABOUT_TEST.md](server/gkill/dvnf/ABOUT_TEST.md) | DVNF ファイル管理 |
| `server/gkill/main/` | [server/gkill/main/ABOUT_TEST.md](server/gkill/main/ABOUT_TEST.md) | CLI エントリポイント |
| `server/gkill/mcp/` | [server/gkill/mcp/ABOUT_TEST.md](server/gkill/mcp/ABOUT_TEST.md) | MCP サーバ（1121テスト）。旧 Node 実装とのゴールデン一致を含む |
| `android/` | [android/ABOUT_TEST.md](android/ABOUT_TEST.md) | Android APK テスト |
| `wear_os/` | [wear_os/ABOUT_TEST.md](wear_os/ABOUT_TEST.md) | Wear OS テスト（226テスト） |
| `server/gkill/plugin/sdk/` | [server/gkill/plugin/sdk/ABOUT_TEST.md](server/gkill/plugin/sdk/ABOUT_TEST.md) | プラグイン SDK（stdio ループ + 単独モード --gkill-build-cache + EnsureConfig + ZIP走査 + キャッシュDBパス + ワード判定 + rep_names、56テスト） |
| `plugins/` | [plugins/ABOUT_TEST.md](plugins/ABOUT_TEST.md) | 同梱プラグイン（独立モジュール。`npm run test_plugins` で実行） |
| `tools/` | [tools/ABOUT_TEST.md](tools/ABOUT_TEST.md) | リリースゲート・attestation ランナーとリリース工程の書く側（55テスト、2ファイル） |
