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
| `plugin_manager_rep_names_test.go` | `PluginManager.GetPluginByRepName` が manifest の `rep_name` でも `get_rep_name` で申告した `rep_names` でも引けること。manifest 名で引けるときは申告名を取りに行かない（既存プラグインの引き当てで stdio へ行かない） |
| `gkill_dao_manager_git_rep_test.go` | git_commit_log の rep 定義（`$HOME/Git/*` のような glob）の展開先に git リポジトリでないディレクトリやファイルが混ざっていても、`GetRepositories` 全体が失敗せず本物の git リポジトリだけを読み込むこと。**ここが崩れると認証ミドルウェアが ERR000018 を返し、対象利用者はログイン直後から全 API が「内部エラー」になる** |
| `gkill_dao_manager_broken_rep_test.go` | 読み込めない書き込み先以外の rep を1本だけ切り離し、利用可能な rep で継続すること。書き込み先は切り離さず失敗し、設定の `IsEnable` は書き換えない。切り離しの個別行と集約行が Error レベルで出て、正常時には集約行が出ないことも固定する |
| `gkill_dao_manager_load_idf_rep_only_test.go` | `LoadIDFRepOnly` が立っているとき IDF の rep だけが読み込まれ、それ以外の rep 定義はパターンの展開もディレクトリ作成もされないこと。**判定が `os.MkdirAll` より後ろへ戻ると、読み込まない rep のパターン展開ぶんの走査を丸ごと払う** |
| `rep_file_glob_test.go` | REPOSITORY の `FILE` パターンの展開が go-zglob と同じ集合を返すこと（`*` が `/` をまたがない・`?` と `[` はリテラル・`**` は go-zglob へフォールバック・区切りの二連・ファイルシステムのルートを走査しない）。**ずれると rep が黙って増減する**（[ADR-0211](../../../../documents/adr/0211-expand-rep-patterns-without-walking.md)） |

### アカウント / セッション

| ファイル | テスト内容 |
|---------|-----------|
| `account/account_dao_sqlite3_impl_test.go` | ユーザアカウント CRUD |
| `account/password_hash_test.go` | Argon2id のラウンドトリップ、誤った資格情報の否認、ソルトが毎回変わること、改竄・不正な PHC 文字列の拒否、パスワード未設定アカウントが常に不一致になること（fail-closed）、資格情報とユーザIDの形式検証 |
| `account/account_schema_migration_test.go` | スキーマ 1.0.0 → 1.1.0 の移行。**全アカウントのパスワードが無効化されリセットトークンが再発行されること**、カラムのリネームと追加、版の更新、再起動しても移行が二度走らないこと。**版を記録する行そのものが無い旧DB（版管理の仕組みが入る前のもの）を新規DBと誤認せず移行すること**と、本当の新規DBは移行しないことも含む |
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
| `share_kyou_info/share_kyou_info_schema_migration_test.go` | スキーマ 1.0.0 → 1.1.0 の移行。保存済み検索条件JSONを新形式へ書き換えること、再実行しても壊れないこと。共有URLは配布済みで再発行できないため、読み出し時の互換層ではなく保存データ自体を移行する |
| `gkill_notification/gkill_notificate_target_dao_sqlite3_impl_test.go` | プッシュ通知ターゲット DAO |

### ユーティリティ

| ファイル | テスト内容 |
|---------|-----------|
| `sqlite3impl/sqlite3impl_util_test.go` | 検索SQLの組み立て。LIKE のエスケープ、列をまたぐAND、検索対象列を持たないリポジトリ、曜日フィルタの nil ガード |
| `sqlite3impl/index_usage_test.go` | 主要クエリがインデックスを使うこと（EXPLAIN QUERY PLAN で確認） |
| `sqlite3impl/unixepoch_index_test.go` | 時刻比較を `unixepoch()` に統一してもインデックスが効くこと |
| `sqlite3impl/sqlite_connection_test.go` | SQLite3 接続の設定（PRAGMA 等） |
| `hide_files/file_hider_test.go` | ファイル非表示ロジック |

## テスト内容

- **GkillDAOManager**: 全 DAO の初期化・接続管理・ライフサイクル
- **部分的な保管場所障害**: 利用可能な rep だけでの継続、書き込み先の fail-closed、個別/集約 Error ログ、正常時の無用な集約ログ抑止
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
