# mcp テスト仕様

## 概要

MCP (Model Context Protocol) サーバのテスト。~563テスト（17ファイル）で3種のMCPサーバ（Read専用・Write専用・Read/Write統合）の入力バリデーション、データ正規化、定数定義、ツールハンドラ（Read 7ツール + Write 11ツール + 統合18ツール）、APIクライアント、サーバライフサイクル、OAuth 2.1認証（RFC 9728/8707/7591対応）、アクセスログをカバーする。

## テストフレームワーク

Vitest（Node.js 環境）

## テストファイル一覧

### 共通・Read専用サーバ

| ファイル | テスト内容 |
|---------|-----------|
| `__tests__/validation.test.mjs` | MCP ツール入力のバリデーション |
| `__tests__/normalization.test.mjs` | クエリデータの正規化処理 |
| `__tests__/constants.test.mjs` | 定数定義の検証 |
| `__tests__/tool-handlers.test.mjs` | Read 7ツールのハンドラ実行ロジック |
| `__tests__/client.test.mjs` | GkillReadClient（fetch モック使用、ログイン・認証リトライ等） |
| `__tests__/server.test.mjs` | McpServer のセットアップとトランスポート管理、セッションオーバーライド |
| `__tests__/access-log.test.mjs` | McpAccessLog（レベルフィルタリング・JSON形式・lazy open・close・sourceパラメータ） |
| `__tests__/pkce.test.mjs` | PKCE (S256/plain) のコード検証、バリデーション |
| `__tests__/oauth-store.test.mjs` | OAuthストア（認可コード、アクセストークン、リフレッシュトークン、クライアント登録、TTL期限切れ、定期クリーンアップ、JSONファイル永続化） |
| `__tests__/oauth-server.test.mjs` | OAuth 2.1サーバ（メタデータ、認可フロー、トークン交換、PKCE検証、リフレッシュトークンローテーション、動的クライアント登録、E2Eフロー） |

### Write専用サーバ

| ファイル | テスト内容 |
|---------|-----------|
| `__tests__/write-normalization.test.mjs` | Write入力の正規化（11 normalizer関数、mood範囲検証、data_type検証等） |
| `__tests__/write-client.test.mjs` | GkillWriteClient（環境変数、login、callWrite、認証リトライ） |
| `__tests__/write-server.test.mjs` | McpWriteServer（JSON-RPC、14ツールディスパッチ、エンティティデフォルト値、レスポンス構造） |
| `__tests__/write-tool-handlers.test.mjs` | Write 14ツール定義・summarize関数 |

### Read/Write統合サーバ

| ファイル | テスト内容 |
|---------|-----------|
| `__tests__/readwrite-client.test.mjs` | GkillClient（callApi統合メソッド、fetchFile、認証リトライ） |
| `__tests__/readwrite-server.test.mjs` | McpServer 統合（18ツール全ディスパッチ、IDF画像ブロック、エンティティデフォルト値） |
| `__tests__/readwrite-tool-handlers.test.mjs` | 統合18ツール定義・summarize関数（Read+Write統合版） |

## テスト内容

- **Validation**: 各ツールの入力パラメータ検証（必須フィールド、型チェック、範囲検証）
- **Normalization**: 日付フォーマット、文字列トリム、デフォルト値補完
- **Write Normalization**: Write専用入力検証（mood 0-10範囲、amount数値型、data_type列挙値、unknown keys拒否等）
- **Constants**: ツール名、エラーコード、デフォルト設定値
- **Tool Handlers**: Read 7ツール + Write 11ツール + Read便利3ツール
- **Client**: GkillReadClient / GkillWriteClient / GkillClient（統合）のAPIラッパー（認証、エラーハンドリング、レスポンスパース）
- **Server**: Read / Write / ReadWrite各サーバのツールディスパッチ、JSON-RPCプロトコル、IDF画像ブロック、Writeエンティティデフォルト値
- **Access Log**: McpAccessLog のレベルフィルタリング・JSON形式・lazy open・close・configurable source
- **PKCE**: S256/plain のコード検証、verifier フォーマット検証、チャレンジメソッド検証
- **OAuth Store**: 認可コード/アクセストークン/リフレッシュトークン/クライアント登録の CRUD、TTL 期限切れ、定期クリーンアップ、JSONファイル永続化
- **OAuth Server**: メタデータ、認可フロー、トークン交換、PKCE検証、リフレッシュトークンローテーション、動的クライアント登録、RFC 8707 resource パラメータ、完全 E2E フロー

## 設定ファイル

`vitest.config.mcp.ts` — Node.js 環境、.mjs ファイルの shebang 除去

## 実行方法

```bash
npm run test_mcp
```
