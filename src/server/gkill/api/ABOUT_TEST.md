# api テスト仕様

## 概要

Go バックエンドの HTTP API ハンドラ層のテスト。全11データ型に対する CRUD 統合テスト、セッション管理、複合クエリフィルタ、サブパッケージ（find, gpslogs, message, kftl, req_res）のテストを含む。

## テストフレームワーク

Go `testing` パッケージ

## テストファイル一覧

### メイン API テスト

| ファイル | テスト内容 |
|---------|-----------|
| `gkill_server_api_test.go` | API ハンドラ統合テスト（全エンドポイント） |
| `find_filter_test.go` | 検索フィルタのテスト |

### サブパッケージテスト

| ファイル | テスト内容 |
|---------|-----------|
| `find/find_query_test.go` | FindQuery ビルダー（フィルタフラグ、MiCheckState/MiSortType enum、nil日付、空スライス、データ型フィルタ） |
| `gpslogs/gpslogs_test.go` | GPS ログファイル解析 |
| `message/message_test.go` | GkillMessage / GkillError フォーマット |

## テスト内容

### `gkill_server_api_test.go`（統合テスト）

- **データ型別 CRUD**: 全11データ型（Kmemo, Mi, TimeIs, URLog, Nlog, Lantana, KC, Tag, Text, Notification, ReKyou）の Add / Update / Delete / Get
- **セッション管理**: ログイン、セッション検証、アカウント管理
- **トランザクション**: 複数操作の一括処理
- **GetKyous 複合クエリ**: ワード検索、タグフィルタ、リポジトリフィルタ、カレンダー範囲、Mi チェック状態、複合条件
- **特殊エンドポイント**: GetKyousMCP, SubmitKFTLText, UpdateCache
- **名前リスト**: ボード名一覧、タグ名一覧、リポジトリ名一覧
- **履歴**: タグ履歴、テキスト履歴、通知履歴
- **設定**: サーバ設定読み書き、アプリケーション設定更新、ユーザリポジトリ更新

## 実行方法

```bash
cd src/server && go test ./gkill/api/...
```

または:

```bash
npm run test_server
```

## 関連ドキュメント

| サブディレクトリ | テスト仕様 |
|----------------|-----------|
| `kftl/` | [kftl/ABOUT_TEST.md](kftl/ABOUT_TEST.md) |
| `req_res/` | [req_res/ABOUT_TEST.md](req_res/ABOUT_TEST.md) |
