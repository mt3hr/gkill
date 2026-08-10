# mcp テスト仕様

## 概要

MCP (Model Context Protocol) サーバのテスト。721テスト（20ファイル）で3種のMCPサーバ（Read専用・Write専用・Read/Write統合）の入力バリデーション、データ正規化、定数定義、ツールハンドラ（Read サーバ 8 + プラグイン1 = 9ツール、Write サーバ 23（書き込み20 + Read便利3）+ プラグイン1 = 24ツール、統合サーバ 28 + プラグイン1 = 29ツール。プラグインツールは3サーバ共通）、APIクライアント、サーバライフサイクル、OAuth 2.1認証（RFC 9728/8707/7591対応）、ファイルリンク配信、プラグイン本文の get_kyous へのインライン埋め込みとHTML→テキスト変換、アクセスログをカバーする。

## テストフレームワーク

Vitest（Node.js 環境）

## テストファイル一覧

### 共通・Read専用サーバ

| ファイル | テスト内容 |
|---------|-----------|
| `__tests__/validation.test.mjs` | MCP ツール入力のバリデーション |
| `__tests__/normalization.test.mjs` | クエリデータの正規化処理 |
| `__tests__/constants.test.mjs` | 定数定義の検証 |
| `__tests__/tool-handlers.test.mjs` | Read 7ツール分のハンドラ実行ロジック（ハンドコピーのツール名一覧・エンドポイント対応表・summarize） |
| `__tests__/client.test.mjs` | GkillReadClient（fetch モック使用、ログイン・認証リトライ等） |
| `__tests__/server.test.mjs` | McpServer のセットアップとトランスポート管理、セッションオーバーライド、プラグインツール振り分け |
| `__tests__/access-log.test.mjs` | McpAccessLog（レベルフィルタリング・JSON形式・lazy open・close・sourceパラメータ） |
| `__tests__/pkce.test.mjs` | PKCE (S256/plain) のコード検証、バリデーション |
| `__tests__/oauth-store.test.mjs` | OAuthストア（認可コード、アクセストークン、リフレッシュトークン、クライアント登録、TTL期限切れ、定期クリーンアップ、JSONファイル永続化） |
| `__tests__/oauth-server.test.mjs` | OAuth 2.1サーバ（メタデータ、認可フロー、トークン交換、PKCE検証、リフレッシュトークンローテーション、動的クライアント登録、E2Eフロー） |
| `__tests__/file-link.test.mjs` | FileLinkStore（HTTPモード用の期限付きファイルリンクトークンの発行・解決・失効、`GET /files/{token}` 配信） |

### プラグインツール（3サーバ共通）

| ファイル | テスト内容 |
|---------|-----------|
| `__tests__/plugin-tools.test.mjs` | プラグインツール（`gkill_get_plugin_list`）の定義とエンドポイント振り分け、`collectPluginPayloads` / `runGroupedWithConcurrency`（キー内直列・キー間並列・例外隔離）、`inlinePluginContents`（format別レスポンス・truncated・重複取得の集約・rep単位の失敗打ち切り・max_kyous/budget/deadline のskip・統計の整合・`file_name` を生やさない回帰）・summarize関数 |
| `__tests__/html-text.test.mjs` | `htmlToText` / `decodeHtmlEntities`（script/style/コメント破棄、ブロック境界の改行化、エンティティデコード順、エスケープ済みマークアップを復活させない、maxLength切り詰め） |

### Write専用サーバ

| ファイル | テスト内容 |
|---------|-----------|
| `__tests__/write-normalization.test.mjs` | Write入力の正規化（11 normalizer関数、mood範囲検証、data_type検証等） |
| `__tests__/write-client.test.mjs` | GkillWriteClient（環境変数、login、callWrite、認証リトライ） |
| `__tests__/write-server.test.mjs` | McpWriteServer（JSON-RPC、24ツールディスパッチ、プラグインツール振り分け、エンティティデフォルト値、レスポンス構造） |
| `__tests__/write-tool-handlers.test.mjs` | Write 23ツール定義（add系9 + update系9 + submit_kftl + delete_kyou + Read便利3）・summarize関数 |

### Read/Write統合サーバ

| ファイル | テスト内容 |
|---------|-----------|
| `__tests__/readwrite-client.test.mjs` | GkillClient（callApi統合メソッド、fetchFile、認証リトライ） |
| `__tests__/readwrite-server.test.mjs` | McpServer 統合（29ツール全ディスパッチ、プラグインツール振り分け、IDF画像ブロック、エンティティデフォルト値） |
| `__tests__/readwrite-tool-handlers.test.mjs` | 統合28ツール定義・summarize関数（Read+Write統合版） |

## テスト内容

- **Validation**: 各ツールの入力パラメータ検証（必須フィールド、型チェック、範囲検証）
- **Normalization**: 日付フォーマット、文字列トリム、デフォルト値補完
- **Write Normalization**: Write専用入力検証（mood 0-10範囲、amount数値型、data_type列挙値、unknown keys拒否等）
- **Constants**: ツール名、エラーコード、デフォルト設定値
- **Tool Handlers**: Read 8ツール + Write 20ツール（add系9 + update系9 + submit_kftl + delete_kyou）+ Read便利3ツール + プラグイン1ツール（3サーバ共通）
- **Plugin Tools**: `gkill_get_plugin_list` の定義・引数検証・エンドポイント振り分けと、`gkill_get_kyous` の `include_plugin_content` によるプラグイン本文のインライン埋め込み（並列度・予算・デッドライン・失敗隔離）、コンテンツHTMLのテキスト変換
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
