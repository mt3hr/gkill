# reps テスト仕様

## 概要

全11データ型のリポジトリ SQLite3 実装テスト。CRUD 操作、キャッシュ層、一時リポジトリ層、キャッシュ更新処理を網羅する。

## テストフレームワーク

Go `testing` パッケージ（インメモリ SQLite3 使用）

## テストファイル一覧

### SQLite3 リポジトリ実装テスト（11ファイル）

| ファイル | テスト対象 |
|---------|-----------|
| `kmemo_repository_sqlite3_impl_test.go` | Kmemo（テキストメモ）リポジトリ |
| `mi_repository_sqlite3_impl_test.go` | Mi（タスク）リポジトリ |
| `lantana_repository_sqlite3_impl_test.go` | Lantana（気分値）リポジトリ |
| `time_is_repository_sqlite3_impl_test.go` | TimeIs（タイムスタンプ）リポジトリ |
| `ur_log_repository_sqlite3_impl_test.go` | URLog（ブックマーク）リポジトリ |
| `nlog_repository_sqlite3_impl_test.go` | Nlog（支出記録）リポジトリ |
| `kc_repository_sqlite3_impl_test.go` | KC（数値記録）リポジトリ |
| `tag_repository_sqlite3_impl_test.go` | Tag（タグ）リポジトリ |
| `text_repository_sqlite3_impl_test.go` | Text（テキスト注釈）リポジトリ |
| `notification_repository_sqlite3_impl_test.go` | Notification（通知）リポジトリ |
| `re_kyou_repository_sqlite3_impl_test.go` | ReKyou（リポスト）リポジトリ |

### キャッシュ・一時・ユーティリティテスト

| ファイル | テスト内容 |
|---------|-----------|
| `cached_and_temp_test.go` | キャッシュ層 / 一時リポジトリ層の動作検証 |
| `testhelper_test.go` | テストヘルパーユーティリティ |
| `cache/latest_data_repository_address_dao_sqlite3_impl_test.go` | キャッシュアドレス DAO |
| `cache/rep_cache_updater/rep_cache_updater_test.go` | キャッシュ更新処理 |
| `rep_cache_updater/rep_cache_updater_test.go` | キャッシュ更新処理（代替パス） |

## テスト内容

- **CRUD 操作**: 各データ型の Create / Read / Update / Delete
- **検索・フィルタ**: ID 検索、日付範囲検索、全件取得
- **4層リポジトリパターン**: interface → SQLite3 実装 → キャッシュ実装 → 一時実装の各層
- **キャッシュ管理**: キャッシュアドレスの解決、キャッシュの更新とクリア

## 実行方法

```bash
cd src/server && go test ./gkill/dao/reps/...
```

または:

```bash
npm run test_server
```
