# テスト仕様書 索引

## 概要

gkill プロジェクト全体で約2,310件のテストが存在する（Go テスト関数 588 + フロントエンドユニット 790 + E2E 206 + MCP 602 + Android 12 + Wear OS 114）。Go バックエンド、Vue 3 フロントエンド、MCP サーバ、Android、Wear OS の各コンポーネントをカバーしている。

## テスト統計

数値は **トップレベルのテスト宣言数**（静的カウント）であり、実行時のケース数とは一致しない。`t.Run` のサブテストや `it.each` / `test.each` のパラメタライズド展開により、実行時のケース数はこれより多くなる。

| コンポーネント | テスト宣言数 | フレームワーク |
|--------------|---------|---------------|
| Go バックエンド (`server/`) | 588 | Go `testing` |
| フロントエンド ユニット (`client/`) | 790 | Vitest |
| フロントエンド E2E (`client/`) | 206 | Playwright |
| MCP サーバ (`mcp/`) | 602 | Vitest |
| Android (`android/`) | 12 | JUnit 4 |
| Wear OS (`wear_os/`) | 114 | JUnit 4 + MockK |
| **合計** | **~2,312** | |

<details>
<summary>カウント方法（再現手順）</summary>

```bash
# Go: トップレベルのテスト関数（t.Run サブテストは含まない）
grep -rhE '^func Test' src/server --include=*.go | wc -l

# フロントエンド ユニット / MCP / E2E: 行頭のテスト宣言
grep -rhE '^\s*(it|test)(\.each)?\(' src/client/__tests__/unit --include=*.ts | wc -l
grep -rhE '^\s*(it|test)(\.each)?\(' src/mcp --include=*.mjs | wc -l
grep -rhE '^\s*test\(' src/client/__tests__/e2e --include=*.ts | wc -l
```

</details>

## テスト実行コマンド

| コマンド | 対象 |
|---------|------|
| `npm test` | 全テスト（server + client + MCP + Android + Wear OS） |
| `npm run test_server` | Go バックエンド (`cd src/server && go test ./...`) |
| `npm run test_client` | フロントエンド（ユニット + E2E） |
| `npm run test_client_unit` | フロントエンドユニットテストのみ |
| `npm run test_client_e2e` | フロントエンド E2E テストのみ |
| `npm run test_mcp` | MCP サーバ |
| `npm run test_android` | Android |
| `npm run test_wear_os` | Wear OS |

## 各ディレクトリの ABOUT_TEST.md

| ディレクトリ | テスト仕様 | 概要 |
|-------------|-----------|------|
| `client/` | [client/ABOUT_TEST.md](client/ABOUT_TEST.md) | フロントエンド全体（unit 800 + E2E 207） |
| `client/classes/` | [client/classes/ABOUT_TEST.md](client/classes/ABOUT_TEST.md) | ユーティリティクラス |
| `client/classes/api/` | [client/classes/api/ABOUT_TEST.md](client/classes/api/ABOUT_TEST.md) | GkillAPI クライアント |
| `client/classes/datas/` | [client/classes/datas/ABOUT_TEST.md](client/classes/datas/ABOUT_TEST.md) | 23データモデル |
| `client/classes/dnote/` | [client/classes/dnote/ABOUT_TEST.md](client/classes/dnote/ABOUT_TEST.md) | D-note 集計・フィルタ |
| `client/classes/kftl/` | [client/classes/kftl/ABOUT_TEST.md](client/classes/kftl/ABOUT_TEST.md) | KFTL パーサ (TypeScript) |
| `client/pages/` | [client/pages/ABOUT_TEST.md](client/pages/ABOUT_TEST.md) | E2E + Composable + Router |
| `locales/` | [locales/ABOUT_TEST.md](locales/ABOUT_TEST.md) | i18n 完全性検証（7言語） |
| `server/` | [server/ABOUT_TEST.md](server/ABOUT_TEST.md) | Go バックエンド全体（588テスト） |
| `server/gkill/api/` | [server/gkill/api/ABOUT_TEST.md](server/gkill/api/ABOUT_TEST.md) | API 共通基盤（FindFilter等） |
| `server/gkill/api/gkill_server_api/` | [server/gkill/api/gkill_server_api/ABOUT_TEST.md](server/gkill/api/gkill_server_api/ABOUT_TEST.md) | API ハンドラ統合テスト（handle_*.go 88ファイル） |
| `server/gkill/api/kftl/` | [server/gkill/api/kftl/ABOUT_TEST.md](server/gkill/api/kftl/ABOUT_TEST.md) | KFTL パーサ (Go) |
| `server/gkill/api/req_res/` | [server/gkill/api/req_res/ABOUT_TEST.md](server/gkill/api/req_res/ABOUT_TEST.md) | JSON 往復テスト |
| `server/gkill/dao/` | [server/gkill/dao/ABOUT_TEST.md](server/gkill/dao/ABOUT_TEST.md) | DAO 層全体 |
| `server/gkill/dao/reps/` | [server/gkill/dao/reps/ABOUT_TEST.md](server/gkill/dao/reps/ABOUT_TEST.md) | 12リポジトリ実装（IDFKyou含む） |
| `server/gkill/usecase/` | [server/gkill/usecase/ABOUT_TEST.md](server/gkill/usecase/ABOUT_TEST.md) | ビジネスロジック層（間接テスト） |
| `server/gkill/dvnf/` | [server/gkill/dvnf/ABOUT_TEST.md](server/gkill/dvnf/ABOUT_TEST.md) | DVNF ファイル管理 |
| `server/gkill/main/` | [server/gkill/main/ABOUT_TEST.md](server/gkill/main/ABOUT_TEST.md) | CLI エントリポイント |
| `mcp/` | [mcp/ABOUT_TEST.md](mcp/ABOUT_TEST.md) | MCP サーバ（602テスト） |
| `android/` | [android/ABOUT_TEST.md](android/ABOUT_TEST.md) | Android APK テスト |
| `wear_os/` | [wear_os/ABOUT_TEST.md](wear_os/ABOUT_TEST.md) | Wear OS テスト（114テスト） |
| `plugins/` | [plugins/ABOUT_TEST.md](plugins/ABOUT_TEST.md) | プラグイン（独自テストなし、API統合テストでカバー） |
