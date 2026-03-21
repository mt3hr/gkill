# mcp テスト仕様

## 概要

MCP (Model Context Protocol) サーバのテスト。226テストで入力バリデーション、データ正規化、定数定義、ツールハンドラ、API クライアント、サーバライフサイクルをカバーする。

## テストフレームワーク

Vitest（Node.js 環境）

## テストファイル一覧

| ファイル | テスト内容 |
|---------|-----------|
| `__tests__/validation.test.mjs` | MCP ツール入力のバリデーション |
| `__tests__/normalization.test.mjs` | クエリデータの正規化処理 |
| `__tests__/constants.test.mjs` | 定数定義の検証 |
| `__tests__/tool-handlers.test.mjs` | 6ツールのハンドラ実行ロジック |
| `__tests__/client.test.mjs` | GkillReadClient（fetch モック使用、ログイン・getKyous・テンプレート取得等） |
| `__tests__/server.test.mjs` | McpServer のセットアップとトランスポート管理 |

## テスト内容

- **Validation**: 各ツールの入力パラメータ検証（必須フィールド、型チェック、範囲検証）
- **Normalization**: 日付フォーマット、文字列トリム、デフォルト値補完
- **Constants**: ツール名、エラーコード、デフォルト設定値
- **Tool Handlers**: `gkill_get_kyous`, `gkill_get_mi_board_list`, `gkill_get_all_tag_names`, `gkill_get_all_rep_names`, `gkill_get_gps_log`, `gkill_get_application_config` の6ツール
- **Client**: GkillReadClient の API ラッパー（認証、エラーハンドリング、レスポンスパース）
- **Server**: stdio / HTTP トランスポートの初期化、API キー認証

## 設定ファイル

`vitest.config.mcp.ts` — Node.js 環境、.mjs ファイルの shebang 除去

## 実行方法

```bash
npm run test_mcp
```
