# api テスト仕様

## 概要

Go バックエンドの API 共通基盤層のテスト。検索フィルタ、サブパッケージ（find, gpslogs, message, kftl, req_res）のテストを含む。HTTP API ハンドラのテストは `gkill_server_api/` サブパッケージに移動済み。

## テストフレームワーク

Go `testing` パッケージ

## テストファイル一覧

### api パッケージ直下テスト

| ファイル | テスト内容 |
|---------|-----------|
| `find_filter_test.go` | 検索フィルタのテスト（最新版への差し替え、リネーム済みタグの除外、並行取得のエラー回収） |

#### `find_filter_test.go` の内容

- **最新版への差し替え** (`TestReplaceLatestKyouInfos_ExcludeStaleKeepLatest`):
  グローバル最新でない版を除外し、最新版のentryだけ残すこと。
  `DisableLatestDataRepositoryCache` の両ブランチで確認する
- **リネーム済みタグの除外** (`TestFindTags_ExcludeRenamedAwayVersion`):
  TAGテーブルはappend-onlyなので旧名の版が残る。旧名で検索してもヒットしないこと
- **並行取得のエラー回収** (`TestDrainFindErrors_*`):
  `FindKyous` はタグ取得・非表示タグ・タグ検索・テキスト検索・TimeIsテキスト・
  TimeIsタグの6経路をgoroutineで並行実行する。`drainFindErrors` が
  goroutineの完了を待ってからエラーを回収すること、エラーが無ければ nil を返すこと。
  以前は待ち合わせより前に吸い出していたため6経路のエラーが常に捨てられ、
  検索が成功扱いで不完全な結果を返していた。待ち合わせを関数の内側に置いたので、
  呼び出し順を誤っても再発しない

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
