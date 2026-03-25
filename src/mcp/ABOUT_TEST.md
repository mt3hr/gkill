# mcp テスト仕様

## 概要

MCP (Model Context Protocol) サーバのテスト。~367テストで入力バリデーション、データ正規化、定数定義、ツールハンドラ（gkill_get_idf_file含む7ツール）、API クライアント、サーバライフサイクル、OAuth 2.1認証（RFC 9728/8707/7591対応）をカバーする。

## テストフレームワーク

Vitest（Node.js 環境）

## テストファイル一覧

| ファイル | テスト内容 |
|---------|-----------|
| `__tests__/validation.test.mjs` | MCP ツール入力のバリデーション |
| `__tests__/normalization.test.mjs` | クエリデータの正規化処理 |
| `__tests__/constants.test.mjs` | 定数定義の検証 |
| `__tests__/tool-handlers.test.mjs` | 7ツールのハンドラ実行ロジック |
| `__tests__/client.test.mjs` | GkillReadClient（fetch モック使用、ログイン・getKyous・テンプレート取得等） |
| `__tests__/server.test.mjs` | McpServer のセットアップとトランスポート管理、セッションオーバーライド |
| `__tests__/pkce.test.mjs` | PKCE (S256/plain) のコード検証、バリデーション |
| `__tests__/oauth-store.test.mjs` | OAuthストア（認可コード、アクセストークン、リフレッシュトークン、クライアント登録、TTL期限切れ、定期クリーンアップ、JSONファイル永続化） |
| `__tests__/oauth-server.test.mjs` | OAuth 2.1サーバ（メタデータ、認可フロー、トークン交換、PKCE検証、リフレッシュトークンローテーション、動的クライアント登録、E2Eフロー） |

## テスト内容

- **Validation**: 各ツールの入力パラメータ検証（必須フィールド、型チェック、範囲検証）
- **Normalization**: 日付フォーマット、文字列トリム、デフォルト値補完
- **Constants**: ツール名、エラーコード、デフォルト設定値
- **Tool Handlers**: `gkill_get_kyous`, `gkill_get_mi_board_list`, `gkill_get_all_tag_names`, `gkill_get_all_rep_names`, `gkill_get_gps_log`, `gkill_get_application_config`, `gkill_get_idf_file` の7ツール
- **Client**: GkillReadClient の API ラッパー（認証、エラーハンドリング、レスポンスパース）
- **Server**: stdio / HTTP トランスポートの初期化、OAuth セッションオーバーライド、gkill_get_idf_file ツール（ディスパッチ/画像image block/非画像/ネストパス/セッションフォールバック）
- **PKCE**: S256/plain のコード検証、verifier フォーマット検証、チャレンジメソッド検証
- **OAuth Store**: 認可コード/アクセストークン/リフレッシュトークン/クライアント登録の CRUD、TTL 期限切れ、定期クリーンアップ、JSONファイル永続化（save/load/期限切れスキップ/ファイル不在・不正JSON耐性/自動保存）
- **OAuth Server**: メタデータエンドポイント、認可 GET/POST、トークン交換（コード→トークン）、PKCE 検証失敗、コード再利用拒否、期限切れコード拒否、リフレッシュトークンローテーション、動的クライアント登録（client_id_issued_at含む）、トークン検証、Bearer トークン抽出、RFC 8707 resource パラメータ（保存/一致/不一致/省略/再描画保持）、DCR redirect_uri 検証、完全 E2E フロー

## 設定ファイル

`vitest.config.mcp.ts` — Node.js 環境、.mjs ファイルの shebang 除去

## 実行方法

```bash
npm run test_mcp
```
