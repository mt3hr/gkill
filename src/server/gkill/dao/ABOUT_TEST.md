# dao テスト仕様

## 概要

データアクセス層（DAO）全体のテスト。GkillDAOManager、アカウント管理、セッション管理、設定管理、共有情報、通知ターゲット、ファイル非表示、SQLite3ユーティリティ、リポジトリ実装を網羅する。

## テストフレームワーク

Go `testing` パッケージ（インメモリ SQLite3 使用）

## テストファイル一覧

### DAO マネージャ

| ファイル | テスト内容 |
|---------|-----------|
| `gkill_dao_manager_test.go` | GkillDAOManager のライフサイクルと初期化 |

### アカウント / セッション

| ファイル | テスト内容 |
|---------|-----------|
| `account/account_dao_sqlite3_impl_test.go` | ユーザアカウント CRUD |
| `account_state/login_session_dao_sqlite3_impl_test.go` | ログインセッション管理 |
| `account_state/file_upload_history_dao_sqlite3_impl_test.go` | ファイルアップロード履歴 |

### 設定

| ファイル | テスト内容 |
|---------|-----------|
| `server_config/server_config_dao_sqlite3_impl_test.go` | サーバ設定（アドレス、TLS、デバイス名） |
| `user_config/application_config_dao_sqlite3_impl_test.go` | アプリケーション設定 |
| `user_config/repository_dao_sqlite3_impl_test.go` | リポジトリ定義 |

### 共有・通知

| ファイル | テスト内容 |
|---------|-----------|
| `share_kyou_info/share_kyou_info_dao_sqlite3_impl_test.go` | Kyou 共有設定 CRUD |
| `gkill_notification/gkill_notificate_target_dao_sqlite3_impl_test.go` | プッシュ通知ターゲット DAO |

### ユーティリティ

| ファイル | テスト内容 |
|---------|-----------|
| `sqlite3impl/sqlite3impl_util_test.go` | SQLite3 ユーティリティ関数 |
| `hide_files/file_hider_test.go` | ファイル非表示ロジック |

## テスト内容

- **GkillDAOManager**: 全 DAO の初期化・接続管理・ライフサイクル
- **アカウント**: ユーザ作成、パスワードハッシュ検証、アカウント更新・削除
- **セッション**: セッション発行、有効期限検証、セッション破棄
- **設定管理**: サーバ設定・ユーザ設定・リポジトリ定義の CRUD
- **共有**: Kyou 共有情報の作成・更新・削除・取得
- **通知**: VAPID プッシュ通知ターゲットの管理
- **SQLite3**: 接続ユーティリティ、テーブル存在確認、マイグレーション

## 実行方法

```bash
cd src/server && go test ./gkill/dao/...
```

または:

```bash
npm run test_server
```

## 関連ドキュメント

| サブディレクトリ | テスト仕様 |
|----------------|-----------|
| `reps/` | [reps/ABOUT_TEST.md](reps/ABOUT_TEST.md) |
