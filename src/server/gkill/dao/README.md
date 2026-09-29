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
├── gkill_notification/          # 通知対象管理
├── hide_files/                  # ファイル隠蔽（OS 別実装）
├── reps/                        # メインリポジトリ → reps/README.md 参照
├── server_config/               # サーバ設定
├── share_kyou_info/             # Kyou 共有情報
├── skills/                      # AI 向けスキル（SKILL.md と付属ファイル）のファイルストア。rep ではない
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

書き込み先ではないリポジトリの組み立てに失敗した場合は、その1本だけを切り離し、利用可能なものだけで続行する。
設定の `IsEnable` は変更しない。切り離した各リポジトリと構築全体の集約を Error レベルで記録し、
呼び出し側が利用者向け警告へ変換できるよう `BrokenReps` を保持する。書き込み先の失敗は切り離さず全体を失敗させる。

`SkillStore`（`skills.Store`）も `GkillDAOManager` が持つ。利用者が AI 向けに書くスキル（`SKILL.md` と付属ファイル）を
`$GKILL_HOME/skills/<user_id>/<skill-name>/` に置く素のファイルストアで、記録（rep）ではないので 4 層構成にも `GkillRepositories` にも乗らない。
`NewGkillDAOManager` が `gkill_options.SkillsDir` を環境変数展開して `skills.NewStore` へ渡す。ファイルを触るのは gkill_server だけで、
MCP も画面も HTTP API 経由で使う（[ADR-0634](../../../../documents/adr/0634-per-user-skills-for-mcp.md)）。

## ルートファイル（11ファイル、資料を除く）

| ファイル | 役割 |
|---------|------|
| `config_da_os.go` | OS 別の設定ファイルパス定義。Windows / macOS / Linux で異なるパスを返す |
| `gkill_dao_manager.go` | `GkillDAOManager` 本体。全リポジトリの初期化・接続管理・ライフサイクル統括 |
| `gkill_dao_manager_broken_rep_test.go` | 読めないリポジトリの部分的切り離し、書き込み先のfail-closed、設定不変、個別/集約ログのテスト |
| `gkill_dao_manager_test.go` | `GkillDAOManager` のテスト |
| `gkill_dao_manager_git_rep_test.go` | git_commit_log の rep 定義（glob）に git リポジトリでないエントリが混ざっても `GetRepositories` 全体が落ちないことのテスト |
| `gkill_dao_manager_load_idf_rep_only_test.go` | IDFだけを必要とする経路で不要なリポジトリを組み立てないことのテスト |
| `gkill_notificater.go` | Web Push 通知の送信ロジック。VAPID 鍵を使用したブラウザ通知 |
| `plugin_manager.go` | プラグインバイナリの検出・起動管理。userID をパス要素として使用する前に検証する |
| `plugin_manager_rep_names_test.go` | `PluginManager.GetPluginByRepName` が manifest の `rep_name` でも `get_rep_name` で申告した `rep_names` でも引けることのテスト |
| `rep_file_glob.go` | リポジトリ定義のファイルパターンを、不要なツリー走査を避けて展開する |
| `rep_file_glob_test.go` | パターン展開の互換性とルート走査防止のテスト |

## サブディレクトリ一覧

### `account/`（4ファイル）— ユーザアカウント

| ファイル | 説明 |
|---------|------|
| `account.go` | `Account` エンティティ（user_id, password_hash 等）。パスワード照合とリセットトークン検証のメソッドを持つ |
| `password_hash.go` | Argon2id によるパスワードのハッシュ化・検証（PHC文字列）、資格情報とユーザIDの形式検証 |
| `account_dao.go` | `AccountDAO` インタフェース |
| `account_dao_sqlite3_impl.go` | SQLite3 実装。スキーマ 1.0.0 → 1.1.0 の移行もここ |

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

### `skills/`（7ファイル）— AI 向けスキルのファイルストア

利用者が AI 向けに書くスキル（`SKILL.md` と付属ファイル）を `$GKILL_HOME/skills/<user_id>/<skill-name>/` に置く。
記録（rep）ではないので、下の「エンティティ / DAO インタフェース / SQLite3 実装」の 3 ファイル構成にも `reps/` の 4 層にも乗らず、履歴も持たない
（[ADR-0634](../../../../documents/adr/0634-per-user-skills-for-mcp.md)）。フォルダは実体として扱わず、ファイルのパスの一部とみなす
（書けばできて、中身が無くなれば消える）。

| ファイル | 説明 |
|---------|------|
| `store.go` | `Store` 本体（一覧・取得・読み出し・書き込み・削除・zip の組み立てと置き換え）。書き込みは一時ファイルへ書いてから rename で置く。置き換えは `_tmp-*` へ展開 → 既存を `_old-*` へ退避 → rename の順で、途中で失敗したら元へ戻して既存のスキルに手を付けない（前回の中断で残った `_tmp-*` / `_old-*` は次の置き換えで掃除する）。利用者ごとの `RWMutex` で書き込み・置き換えと読み取りを直列化する。利用者ディレクトリは大文字小文字まで一致で引き、大小だけ違う利用者 ID は `ErrInvalidName` で拒否する（Windows では別の大小の名前が同じディレクトリ = 他人のスキルへ届くため） |
| `path.go` | スキル名（英小文字・数字・ハイフン、先頭と末尾は英数字）とスキル内パス（ASCII のみ、各要素の先頭は英数字、Windows の予約名は拒否）の規則、`joinWithin`（根の下だけを許す結合。`..` を含む正しい名前は別名にしない）、大小衝突とファイル/フォルダ衝突の検出 |
| `frontmatter.go` | `SKILL.md` の frontmatter（`name` / `description`）の解釈。両方必須で、`name` はディレクトリ名と一致していること |
| `zip.go` | アップロードされた zip の検査（1 段の包みフォルダの除去、OS の管理ファイルとドット始まりの無視、規則外の名前・絶対パス・`..`・シンボリックリンクの拒否、CRC の食い違いの拒否）と、ダウンロード用 zip の組み立て |
| `inspect.go` | ファイルの revision（内容のハッシュ）とテキスト判定。チャンク境界で UTF-8 の多バイト文字を割らない |
| `errors.go` | 種別エラー（`ErrSkillNotFound` / `ErrInvalidUserID` / `ErrInvalidName` / `ErrInvalidPath` / `ErrInvalidZip` / `ErrRevisionConflict` 等）と、詳細を添える `DetailError` |
| `store_test.go` | テスト。内容は [ABOUT_TEST.md](ABOUT_TEST.md) の「スキルのファイルストア」 |

### `sqlite3impl/`（2ファイル）— SQLite3 ユーティリティ

| ファイル | 説明 |
|---------|------|
| `localtime_check.go` | SQLite の `'localtime'` と Go の `time.Local` が同じ壁時計かの自己検査（`LocaltimeAgreement`）。食い違うと時間帯フィルタの SQL 段と Go 段が別の壁時計で判定し、検索が黙って0件になる |
| `sqlite3impl_util.go` | SQLite3 共通ユーティリティ関数（DB 接続、テーブル作成、検索 SQL の組み立て等） |

### `user_config/`（6ファイル）— ユーザ設定

| ファイル | 説明 |
|---------|------|
| `application_config.go` | `ApplicationConfig` エンティティ（KFTL テンプレート、表示設定等） |
| `application_config_dao.go` | `ApplicationConfigDAO` インタフェース |
| `application_config_dao_sqlite3_impl.go` | SQLite3 実装 |
| `repository.go` | `Repository` エンティティ（データ保存先定義） |
| `repository_dao.go` | `RepositoryDAO` インタフェース |
| `repository_dao_sqlite3_impl.go` | SQLite3 実装 |

### `reps/`（139ファイル。テストを含めると206）— メインリポジトリ

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
