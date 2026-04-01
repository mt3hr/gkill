# gkill_server_api テスト仕様

## 概要

`gkill/api/gkill_server_api/` パッケージのテスト。`gkill/api/` から移動された HTTP API ハンドラ層（85+ファイル）に対する統合テストを含む。

## テストフレームワーク

Go `testing` パッケージ

## テストファイル一覧

| ファイル | テスト内容 |
|---------|-----------|
| `gkill_server_api_test.go` | API ハンドラ統合テスト（全エンドポイント） |
| `gkill_server_api_rate_limit_test.go` | ログインレート制限テスト（IP別カウント、ウィンドウ期限、IP抽出） |

## テスト内容

### `gkill_server_api_test.go`（統合テスト）

- **データ型別 CRUD**: 全11データ型（Kmemo, Mi, TimeIs, URLog, Nlog, Lantana, KC, Tag, Text, Notification, ReKyou）の Add / Update / Delete / Get
- **セッション管理**: ログイン、セッション検証、アカウント管理、セッション有効期限切れ検出（ERR000373）
- **認証ミドルウェア**: `authMiddleware` / `authWithReposMiddleware` による認証フローの検証
- **トランザクション**: 複数操作の一括処理
- **GetKyous 複合クエリ**: ワード検索、タグフィルタ、リポジトリフィルタ、カレンダー範囲、Mi チェック状態、複合条件
- **特殊エンドポイント**: GetKyousMCP, SubmitKFTLText, UpdateCache, BrowseZipContents
- **ZIPブラウズ**: BrowseZipContents エンドポイントのセキュリティテスト（パストラバーサル防止、Shift_JISエントリ名デコード、アトミック展開、zip_cache ファイルサーブ）
- **名前リスト**: ボード名一覧、タグ名一覧、リポジトリ名一覧
- **履歴**: タグ履歴、テキスト履歴、通知履歴
- **設定**: サーバ設定読み書き、アプリケーション設定更新、ユーザリポジトリ更新

### `gkill_server_api_rate_limit_test.go`（レート制限テスト）

- **レート制限**: 10回/15分のログイン試行制限、IP別独立カウント、ウィンドウ期限経過後のリセット
- **IP抽出**: IPv4/IPv6アドレスからのポート番号除去

## 実行方法

```bash
cd src/server && go test ./gkill/api/gkill_server_api/...
```

または:

```bash
npm run test_server
```
