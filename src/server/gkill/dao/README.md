# dao - データアクセス層

## 概要

gkill のデータアクセス層。SQLite3 をバックエンドとし、`GkillDAOManager` が全リポジトリの初期化・管理を統括する。
Repository パターンを採用し、各データ型に対して複数の実装層を提供する。

## ディレクトリ構造

```
dao/
├── gkill_dao_manager.go         # DAO 全体の管理・初期化（未確認: config_da_os.go で検出）
├── gkill_notificater.go         # Web Push 通知送信
├── config_da_os.go              # OS 別設定パス定義
├── account/                     # ユーザアカウント
├── account_state/               # ログインセッション・ファイルアップロード履歴
├── cache/                       # キャッシュ管理（空、reps/cache/ に実体あり）
├── gkill_notification/          # 通知対象管理
├── hide_files/                  # ファイル隠蔽（OS 別実装）
├── reps/                        # メインリポジトリ → reps/README.md 参照
├── server_config/               # サーバ設定
├── share_kyou_info/             # Kyou 共有情報
├── sqlite3impl/                 # SQLite3 ユーティリティ
└── user_config/                 # ユーザ設定（アプリ設定、リポジトリ）
```

## 設計思想

### Repository パターン（4層実装）

各データ型に対して以下の層で実装:

```
1. エンティティ定義     xxx.go                — データ構造体
2. DAO インタフェース   xxx_dao.go             — Go interface
3. SQLite3 実装         xxx_dao_sqlite3_impl.go — DB 直接操作
```

メインデータ型（reps/ 配下）はさらに:
```
4. キャッシュ付き実装   xxx_cached_sqlite3_impl.go
5. ローカルキャッシュ   xxx_sqlite3_impl_local_cached.go
6. 一時リポジトリ       xxx_temp_repository.go + xxx_temp_sqlite3_impl.go
```

詳細は [reps/README.md](reps/README.md) を参照。

### Append-Only 方式

データの更新は既存レコードの上書きではなく、新しいレコードの追加で表現。
最新レコードが有効データとなり、変更履歴が自然に保持される。

### GkillDAOManager

全リポジトリの初期化・接続管理・ライフサイクルを統括する中心的な構造体。
ユースケース層（`usecase/`）および API ハンドラ層は `GkillDAOManager` 経由でリポジトリにアクセスする。

## ルートファイル（3ファイル）

| ファイル | 役割 |
|---------|------|
| `config_da_os.go` | OS 別の設定ファイルパス定義。Windows / macOS / Linux で異なるパスを返す |
| `gkill_notificater.go` | Web Push 通知の送信ロジック。VAPID 鍵を使用したブラウザ通知 |
| *(gkill_dao_manager は reps/gkill_repositories.go 等で管理)* | |

## サブディレクトリ一覧

### `account/`（3ファイル）— ユーザアカウント

| ファイル | 説明 |
|---------|------|
| `account.go` | `Account` エンティティ（user_id, password_sha256 等） |
| `account_dao.go` | `AccountDAO` インタフェース |
| `account_dao_sqlite3_impl.go` | SQLite3 実装 |

### `account_state/`（6ファイル）— セッション・アップロード履歴

| ファイル | 説明 |
|---------|------|
| `login_session.go` | `LoginSession` エンティティ |
| `login_session_dao.go` | `LoginSessionDAO` インタフェース |
| `login_session_dao_sqlite3_impl.go` | SQLite3 実装 |
| `file_upload_history.go` | `FileUploadHistory` エンティティ |
| `file_upload_history_dao.go` | `FileUploadHistoryDAO` インタフェース |
| `file_upload_history_dao_sqlite3_impl.go` | SQLite3 実装 |

### `gkill_notification/`（3ファイル）— 通知対象管理

| ファイル | 説明 |
|---------|------|
| `gkill_notificate_target.go` | `GkillNotificateTarget` エンティティ（Web Push 登録情報） |
| `gkill_notificate_target_dao.go` | `GkillNotificateTargetDAO` インタフェース |
| `gkill_notificate_target_dao_sqlite3_impl.go` | SQLite3 実装 |

### `hide_files/`（3ファイル）— ファイル隠蔽

| ファイル | 説明 |
|---------|------|
| `file_hider.go` | `FileHider` インタフェース |
| `file_hider_windows.go` | Windows 実装（ファイル属性で隠し設定） |
| `file_hider_other.go` | その他 OS 実装（`.` プレフィックスで隠し設定） |

### `server_config/`（3ファイル）— サーバ設定

| ファイル | 説明 |
|---------|------|
| `server_config.go` | `ServerConfig` エンティティ（ポート、パス等） |
| `server_config_dao.go` | `ServerConfigDAO` インタフェース |
| `server_config_dao_sqlite3_impl.go` | SQLite3 実装 |

### `share_kyou_info/`（3ファイル）— Kyou 共有情報

| ファイル | 説明 |
|---------|------|
| `share_kyou_info.go` | `ShareKyouInfo` エンティティ |
| `share_kyou_info_dao.go` | `ShareKyouInfoDAO` インタフェース |
| `share_kyou_info_dao_sqlite3_impl.go` | SQLite3 実装 |

### `sqlite3impl/`（1ファイル）— SQLite3 ユーティリティ

| ファイル | 説明 |
|---------|------|
| `sqlite3impl_util.go` | SQLite3 共通ユーティリティ関数（DB 接続、テーブル作成等） |

### `user_config/`（6ファイル）— ユーザ設定

| ファイル | 説明 |
|---------|------|
| `application_config.go` | `ApplicationConfig` エンティティ（KFTL テンプレート、表示設定等） |
| `application_config_dao.go` | `ApplicationConfigDAO` インタフェース |
| `application_config_dao_sqlite3_impl.go` | SQLite3 実装 |
| `repository.go` | `Repository` エンティティ（データ保存先定義） |
| `repository_dao.go` | `RepositoryDAO` インタフェース |
| `repository_dao_sqlite3_impl.go` | SQLite3 実装 |

### `reps/`（126ファイル）— メインリポジトリ

全 Kyou データ型のリポジトリ。詳細は [reps/README.md](reps/README.md) を参照。

## 開発ガイドライン

### DAO の共通パターン

各サブディレクトリは以下の3ファイル構成が基本:
1. `xxx.go` — エンティティ構造体定義
2. `xxx_dao.go` — DAO インタフェース定義
3. `xxx_dao_sqlite3_impl.go` — SQLite3 実装

### OS 別実装

`hide_files/` のように OS 別の実装が必要な場合は、Go のビルドタグ/ファイル名規約を使用:
- `*_windows.go` — Windows 用
- `*_other.go` — その他 OS 用

### SQLite3 接続

`sqlite3impl/sqlite3impl_util.go` の共通ユーティリティを使用して DB 接続を管理する。
`modernc.org/sqlite`（pure Go）を使用しており、CGO は不要。
