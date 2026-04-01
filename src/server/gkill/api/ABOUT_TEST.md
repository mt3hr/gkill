# api テスト仕様

## 概要

Go バックエンドの API 共通基盤層のテスト。検索フィルタ、サブパッケージ（find, gpslogs, message, kftl, req_res）のテストを含む。HTTP API ハンドラのテストは `gkill_server_api/` サブパッケージに移動済み。

## テストフレームワーク

Go `testing` パッケージ

## テストファイル一覧

### api パッケージ直下テスト

| ファイル | テスト内容 |
|---------|-----------|
| `find_filter_test.go` | 検索フィルタのテスト |

### サブパッケージテスト

| ファイル | テスト内容 |
|---------|-----------|
| `find/find_query_test.go` | FindQuery ビルダー（フィルタフラグ、MiCheckState/MiSortType enum、nil日付、空スライス、データ型フィルタ） |
| `gpslogs/gpslogs_test.go` | GPS ログファイル解析 |
| `message/message_test.go` | GkillMessage / GkillError フォーマット |

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
| `gkill_server_api/` | [gkill_server_api/ABOUT_TEST.md](gkill_server_api/ABOUT_TEST.md) |
| `kftl/` | [kftl/ABOUT_TEST.md](kftl/ABOUT_TEST.md) |
| `req_res/` | [req_res/ABOUT_TEST.md](req_res/ABOUT_TEST.md) |
