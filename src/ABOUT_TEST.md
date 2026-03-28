# テスト仕様書 索引

## 概要

gkill プロジェクト全体で約1,904件のテストが存在する。Go バックエンド、Vue 3 フロントエンド、MCP サーバ、Android、Wear OS の各コンポーネントをカバーしている。

## テスト統計

| コンポーネント | テスト数 | フレームワーク |
|--------------|---------|---------------|
| Go バックエンド (`server/`) | ~534 | Go `testing` |
| フロントエンド ユニット (`client/`) | ~676 | Vitest |
| フロントエンド E2E (`client/`) | 187 | Playwright |
| MCP サーバ (`mcp/`) | ~381 | Vitest |
| Android (`android/`) | 12 | JUnit 4 |
| Wear OS (`wear_os/`) | 114 | JUnit 4 + MockK |
| **合計** | **~1,904** | |

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
| `client/` | [client/ABOUT_TEST.md](client/ABOUT_TEST.md) | フロントエンド全体（unit ~676 + E2E 187） |
| `client/classes/` | [client/classes/ABOUT_TEST.md](client/classes/ABOUT_TEST.md) | ユーティリティクラス |
| `client/classes/api/` | [client/classes/api/ABOUT_TEST.md](client/classes/api/ABOUT_TEST.md) | GkillAPI クライアント |
| `client/classes/datas/` | [client/classes/datas/ABOUT_TEST.md](client/classes/datas/ABOUT_TEST.md) | 22データモデル |
| `client/classes/dnote/` | [client/classes/dnote/ABOUT_TEST.md](client/classes/dnote/ABOUT_TEST.md) | D-note 集計・フィルタ |
| `client/classes/kftl/` | [client/classes/kftl/ABOUT_TEST.md](client/classes/kftl/ABOUT_TEST.md) | KFTL パーサ (TypeScript) |
| `client/pages/` | [client/pages/ABOUT_TEST.md](client/pages/ABOUT_TEST.md) | E2E + Composable + Router |
| `locales/` | [locales/ABOUT_TEST.md](locales/ABOUT_TEST.md) | i18n 完全性検証（7言語） |
| `server/` | [server/ABOUT_TEST.md](server/ABOUT_TEST.md) | Go バックエンド全体（~534テスト） |
| `server/gkill/api/` | [server/gkill/api/ABOUT_TEST.md](server/gkill/api/ABOUT_TEST.md) | API ハンドラ統合テスト |
| `server/gkill/api/kftl/` | [server/gkill/api/kftl/ABOUT_TEST.md](server/gkill/api/kftl/ABOUT_TEST.md) | KFTL パーサ (Go) |
| `server/gkill/api/req_res/` | [server/gkill/api/req_res/ABOUT_TEST.md](server/gkill/api/req_res/ABOUT_TEST.md) | JSON 往復テスト |
| `server/gkill/dao/` | [server/gkill/dao/ABOUT_TEST.md](server/gkill/dao/ABOUT_TEST.md) | DAO 層全体 |
| `server/gkill/dao/reps/` | [server/gkill/dao/reps/ABOUT_TEST.md](server/gkill/dao/reps/ABOUT_TEST.md) | 11リポジトリ実装 |
| `server/gkill/dvnf/` | [server/gkill/dvnf/ABOUT_TEST.md](server/gkill/dvnf/ABOUT_TEST.md) | DVNF ファイル管理 |
| `server/gkill/main/` | [server/gkill/main/ABOUT_TEST.md](server/gkill/main/ABOUT_TEST.md) | CLI エントリポイント |
| `mcp/` | [mcp/ABOUT_TEST.md](mcp/ABOUT_TEST.md) | MCP サーバ（~381テスト） |
| `android/` | [android/ABOUT_TEST.md](android/ABOUT_TEST.md) | Android APK テスト |
| `wear_os/` | [wear_os/ABOUT_TEST.md](wear_os/ABOUT_TEST.md) | Wear OS テスト（114テスト） |
