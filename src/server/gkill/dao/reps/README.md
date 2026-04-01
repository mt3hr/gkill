# reps - メインリポジトリ層

## 概要

gkill の全データ型に対する Repository インタフェースと実装を提供するパッケージ。
SQLite3 をバックエンドとし、4層のアーキテクチャで構成されている。
フロントエンド側の対応データモデル: `src/client/classes/datas/`

## 設計思想

### 4層リポジトリアーキテクチャ

各データ型に対して以下の4層（+エンティティ定義）でファイルが構成される:

```
1. エンティティ定義:        xxx.go               — データ構造体
2. リポジトリインタフェース:  xxx_repository.go    — Go interface
3. SQLite3 実装:            xxx_repository_sqlite3_impl.go — DB 直接操作
4. キャッシュ付き実装:       xxx_repository_cached_sqlite3_impl.go — キャッシュラッパー
                           xxx_repository_sqlite3_impl_local_cached.go — ローカルキャッシュ
5. 一時リポジトリ:           xxx_temp_repository.go — トランザクション用インタフェース
                           xxx_repository_temp_sqlite3_impl.go — トランザクション用実装
```

加えて、複数リポジトリの集約型:
```
6. 集約リポジトリ:           xxx_repositories.go  — 複数リポジトリをまとめるインタフェース
```

### Append-Only 方式

データの更新は既存レコードの上書きではなく、**新しいレコードの追加**で表現される。
最新のレコードが有効なデータとなり、履歴が自然に保持される。

### GkillRepositories

`gkill_repositories.go` で定義される `GkillRepositories` 構造体が、
全リポジトリの集約アクセスポイントとして機能する。
ユースケース層（`usecase/`）からはこの構造体を経由して各リポジトリにアクセスする。

## データ型別ファイル一覧

### 基底エンティティ

| ファイル | 説明 |
|---------|------|
| `kyou.go` | **Kyou** — 全データ型の基底エンティティ。ID, CreateTime, UpdateTime 等の共通フィールド |
| `repository.go` | `Repository` インタフェース — 全リポジトリの基底インタフェース |
| `repositories.go` | `Repositories` — 複数リポジトリ集約の基底インタフェース |
| `gkill_repositories.go` | `GkillRepositories` — 全データ型のリポジトリを統合管理する構造体 |
| `temp_reps.go` | `TempReps` — トランザクション用一時リポジトリの集約 |

### Kmemo（テキストメモ）— 8ファイル

| ファイル | 層 |
|---------|---|
| `kmemo.go` | エンティティ定義 |
| `kmemo_repository.go` | インタフェース |
| `kmemo_repositories.go` | 集約インタフェース |
| `kmemo_repository_sqlite3_impl.go` | SQLite3 実装 |
| `kmemo_repository_cached_sqlite3_impl.go` | キャッシュ付き実装 |
| `kmemo_repository_sqlite3_impl_local_cached.go` | ローカルキャッシュ実装 |
| `kmemo_temp_repository.go` | 一時リポジトリインタフェース |
| `kmemo_repository_temp_sqlite3_impl.go` | 一時リポジトリ実装 |

### KC（数値記録）— 8ファイル

| ファイル | 層 |
|---------|---|
| `kc.go` | エンティティ定義 |
| `kc_repository.go` | インタフェース |
| `kc_repositories.go` | 集約インタフェース |
| `kc_repository_sqlite3_impl.go` | SQLite3 実装 |
| `kc_repository_cached_sqlite3_impl.go` | キャッシュ付き実装 |
| `kc_repository_sqlite3_impl_local_cached.go` | ローカルキャッシュ実装 |
| `kc_temp_repository.go` | 一時リポジトリインタフェース |
| `kc_repository_temp_sqlite3_impl.go` | 一時リポジトリ実装 |

### Lantana（気分値）— 8ファイル

| ファイル | 層 |
|---------|---|
| `lantana.go` | エンティティ定義 |
| `lantana_repository.go` | インタフェース |
| `lantana_repositories.go` | 集約インタフェース |
| `lantana_repository_sqlite3_impl.go` | SQLite3 実装 |
| `lantana_repository_cached_sqlite3_impl.go` | キャッシュ付き実装 |
| `lantana_repository_sqlite3_impl_local_cached.go` | ローカルキャッシュ実装 |
| `lantana_temp_repository.go` | 一時リポジトリインタフェース |
| `lantana_repository_temp_sqlite3_impl.go` | 一時リポジトリ実装 |

### Mi（タスク）— 8ファイル

| ファイル | 層 |
|---------|---|
| `mi.go` | エンティティ定義 |
| `mi_repository.go` | インタフェース |
| `mi_repositories.go` | 集約インタフェース |
| `mi_repository_sqlite3_impl.go` | SQLite3 実装 |
| `mi_repository_cached_sqlite3_impl.go` | キャッシュ付き実装 |
| `mi_repository_sqlite3_impl_local_cached.go` | ローカルキャッシュ実装 |
| `mi_temp_repository.go` | 一時リポジトリインタフェース |
| `mi_repository_temp_sqlite3_impl.go` | 一時リポジトリ実装 |

### Nlog（支出記録）— 8ファイル

| ファイル | 層 |
|---------|---|
| `nlog.go` | エンティティ定義 |
| `nlog_repository.go` | インタフェース |
| `nlog_repositories.go` | 集約インタフェース |
| `nlog_repository_sqlite3_impl.go` | SQLite3 実装 |
| `nlog_repository_cached_sqlite3_impl.go` | キャッシュ付き実装 |
| `nlog_repository_sqlite3_impl_local_cached.go` | ローカルキャッシュ実装 |
| `nlog_temp_repository.go` | 一時リポジトリインタフェース |
| `nlog_repository_temp_sqlite3_impl.go` | 一時リポジトリ実装 |

### URLog（ブックマーク）— 8ファイル

| ファイル | 層 |
|---------|---|
| `ur_log.go` | エンティティ定義 |
| `ur_log_repository.go` | インタフェース |
| `ur_log_repositories.go` | 集約インタフェース |
| `ur_log_repository_sqlite3_impl.go` | SQLite3 実装 |
| `ur_log_repository_cached_sqlite3_impl.go` | キャッシュ付き実装 |
| `ur_log_repository_sqlite3_impl_local_cached.go` | ローカルキャッシュ実装 |
| `ur_log_temp_repository.go` | 一時リポジトリインタフェース |
| `ur_log_repository_temp_sqlite3_impl.go` | 一時リポジトリ実装 |

### TimeIs（タイムスタンプ）— 8ファイル

| ファイル | 層 |
|---------|---|
| `time_is.go` | エンティティ定義 |
| `time_is_repository.go` | インタフェース |
| `time_is_repositories.go` | 集約インタフェース |
| `time_is_repository_sqlite3_impl.go` | SQLite3 実装 |
| `time_is_repository_cached_sqlite3_impl.go` | キャッシュ付き実装 |
| `time_is_repository_sqlite3_impl_local_cached.go` | ローカルキャッシュ実装 |
| `time_is_temp_repository.go` | 一時リポジトリインタフェース |
| `time_is_repository_temp_sqlite3_impl.go` | 一時リポジトリ実装 |

### Tag — 8ファイル

| ファイル | 層 |
|---------|---|
| `tag.go` | エンティティ定義 |
| `tag_repository.go` | インタフェース |
| `tag_repositories.go` | 集約インタフェース |
| `tag_repository_sqlite3_impl.go` | SQLite3 実装 |
| `tag_repository_cached_sqlite3_impl.go` | キャッシュ付き実装 |
| `tag_repository_sqlite3_impl_local_cached.go` | ローカルキャッシュ実装 |
| `tag_temp_repository.go` | 一時リポジトリインタフェース |
| `tag_repository_temp_sqlite3_impl.go` | 一時リポジトリ実装 |

### Text — 8ファイル

| ファイル | 層 |
|---------|---|
| `text.go` | エンティティ定義 |
| `text_repository.go` | インタフェース |
| `text_repositories.go` | 集約インタフェース |
| `text_repository_sqlite3_impl.go` | SQLite3 実装 |
| `text_repository_cached_sqlite3_impl.go` | キャッシュ付き実装 |
| `text_repository_sqlite3_impl_local_cached.go` | ローカルキャッシュ実装 |
| `text_temp_repository.go` | 一時リポジトリインタフェース |
| `text_repository_temp_sqlite3_impl.go` | 一時リポジトリ実装 |

### Notification — 8ファイル

| ファイル | 層 |
|---------|---|
| `notification.go` | エンティティ定義 |
| `notification_repository.go` | インタフェース |
| `notification_repositories.go` | 集約インタフェース |
| `notification_repository_sqlite3_impl.go` | SQLite3 実装 |
| `notification_repository_cached_sqlite3_impl.go` | キャッシュ付き実装 |
| `notification_repository_sqlite3_impl_local_cached.go` | ローカルキャッシュ実装 |
| `notification_temp_repository.go` | 一時リポジトリインタフェース |
| `notification_repository_temp_sqlite3_impl.go` | 一時リポジトリ実装 |

### ReKyou（リポスト）— 8ファイル

| ファイル | 層 |
|---------|---|
| `re_kyou.go` | エンティティ定義 |
| `re_kyou_repository.go` | インタフェース |
| `re_kyou_repositories.go` | 集約インタフェース |
| `re_kyou_repository_sqlite3_impl.go` | SQLite3 実装 |
| `re_kyou_repository_cached_sqlite3_impl.go` | キャッシュ付き実装 |
| `re_kyou_repository_sqlite3_impl_local_cached.go` | ローカルキャッシュ実装 |
| `re_kyou_temp_repository.go` | 一時リポジトリインタフェース |
| `re_kyou_repository_temp_sqlite3_impl.go` | 一時リポジトリ実装 |

### IDFKyou（ファイル）— 9ファイル

| ファイル | 層 |
|---------|---|
| `idf_kyou.go` | エンティティ定義 |
| `idf_kyou_repository.go` | インタフェース |
| `idf_kyou_repositories.go` | 集約インタフェース |
| `idf_kyou_repository_sqlite3_impl.go` | SQLite3 実装 |
| `idf_kyou_repository_cached_sqlite3_impl.go` | キャッシュ付き実装 |
| `idf_kyou_repository_sqlite3_impl_local_cached.go` | ローカルキャッシュ実装 |
| `idf_kyou_temp_repository.go` | 一時リポジトリインタフェース |
| `idf_kyou_temp_repository_sqlite3_impl.go` | 一時リポジトリ実装 |
| `idf_file_url.go` | IDF ファイル URL 生成ユーティリティ |

### Git Commit Log — 5ファイル

| ファイル | 層 |
|---------|---|
| `git_commit_log.go` | エンティティ定義 |
| `git_commit_log_repository.go` | インタフェース |
| `git_commit_log_repositories.go` | 集約インタフェース |
| `git_commit_log_repository_cached_sqlite3_impl.go` | キャッシュ付き実装 |
| `git_commit_log_repository_local_dir_impl.go` | ローカルディレクトリからの読み取り実装 |

### GPS Log — 4ファイル

| ファイル | 層 |
|---------|---|
| `gps_log.go` | エンティティ定義 |
| `gps_log_repository.go` | インタフェース |
| `gps_log_repositories.go` | 集約インタフェース |
| `gps_log_repository_gpx_dir_impl.go` | GPX ディレクトリからの読み取り実装 |

### ファイルサーバ — 2ファイル

| ファイル | 説明 |
|---------|------|
| `idf_thumb_file_server.go` | サムネイル画像のファイルサーバ |
| `idf_video_file_server.go` | 動画ファイルのファイルサーバ |

## サブディレクトリ

### `cache/`（3ファイル）

リポジトリアドレスの最新データキャッシュ管理。

| ファイル | 説明 |
|---------|------|
| `latest_data_repository_address.go` | 最新データリポジトリアドレスのエンティティ |
| `latest_data_repository_address_dao.go` | DAO インタフェース |
| `latest_data_repository_address_dao_sqlite3_impl.go` | SQLite3 実装 |

### `cache/rep_cache_updater/` および `rep_cache_updater/`（5ファイル）

キャッシュの自動更新ロジック。

| ファイル | 説明 |
|---------|------|
| `cache_updatable.go` | キャッシュ更新可能インタフェース |
| `latest_repository_address_cache_updater.go` | 最新リポジトリアドレスのキャッシュ更新 |
| `rep_cache_updater.go` | キャッシュ更新メインロジックのインタフェース |
| `rep_cache_updater_impl.go` | キャッシュ更新メインロジックの実装 |
| `watch_target_entry.go` | 監視対象エントリ定義 |

## 開発ガイドライン

### 新しいデータ型を追加する場合

1. `xxx.go` でエンティティ構造体を定義（`Kyou` を埋め込む）
2. `xxx_repository.go` で Repository インタフェースを定義
3. `xxx_repositories.go` で集約インタフェースを定義
4. `xxx_repository_sqlite3_impl.go` で SQLite3 実装を作成
5. `xxx_repository_cached_sqlite3_impl.go` でキャッシュ付き実装を作成
6. `xxx_repository_sqlite3_impl_local_cached.go` でローカルキャッシュ実装を作成
7. `xxx_temp_repository.go` で一時リポジトリインタフェースを定義
8. `xxx_repository_temp_sqlite3_impl.go` で一時リポジトリ実装を作成
9. `gkill_repositories.go` に新しいリポジトリフィールドを追加
10. `temp_reps.go` に一時リポジトリフィールドを追加

### 命名規則

- エンティティ名: PascalCase（例: `Kmemo`, `TimeIs`, `URLog`）
- ファイル名: snake_case（例: `kmemo_repository.go`, `time_is_repository.go`）
- 集約型: `XxxRepositories`（複数形）
- 実装型: `XxxRepositorySqlite3Impl`, `XxxRepositoryCachedSqlite3Impl` 等

### Git Commit Log / GPS Log の特殊性

これらは SQLite3 を直接使わず、ローカルディレクトリや GPX ファイルから読み取る実装:
- `git_commit_log_repository_local_dir_impl.go` — git リポジトリから直接読み取り
- `gps_log_repository_gpx_dir_impl.go` — GPX ファイルディレクトリから読み取り
