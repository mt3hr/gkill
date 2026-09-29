# dao テスト仕様

## 概要

データアクセス層（DAO）全体のテスト。GkillDAOManager、アカウント管理、セッション管理、設定管理、共有情報、通知ターゲット、ファイル非表示、SQLite3ユーティリティ、スキルのファイルストア、リポジトリ実装を網羅する。

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
| `server_config/server_config_dao_sqlite3_impl_test.go` | サーバ設定（アドレス、TLS、デバイス名）。`GetDefaultServerConfig` と行が無い端末の `GetServerConfig` が既定の定数（`DefaultListenAddress` = `127.0.0.1:9999`、`DefaultIsLocalOnlyAccess` = true）を返すこと |
| `user_config/application_config_dao_sqlite3_impl_test.go` | アプリケーション設定 |
| `user_config/repository_dao_sqlite3_impl_test.go` | リポジトリ定義 |

### 共有・通知

| ファイル | テスト内容 |
|---------|-----------|
| `share_kyou_info/share_kyou_info_dao_sqlite3_impl_test.go` | Kyou 共有設定 CRUD |
| `share_kyou_info/share_kyou_info_schema_migration_test.go` | スキーマ 1.0.0 → 1.1.0 の移行。保存済み検索条件JSONを新形式へ書き換えること、再実行しても壊れないこと。共有URLは配布済みで再発行できないため、読み出し時の互換層ではなく保存データ自体を移行する |
| `gkill_notification/gkill_notificate_target_dao_sqlite3_impl_test.go` | プッシュ通知ターゲット DAO |

### スキルのファイルストア

| ファイル | テスト内容 |
|---------|-----------|
| `skills/store_test.go` | 利用者が AI 向けに書くスキルのファイルストア（`skills.Store`。rep ではない素の置き場。ADR-0634）。**名前の検証**（`TestValidateSkillName`: 英小文字・数字・ハイフンで先頭と末尾は英数字）、**パスの正規化**（`TestNormalizeFilePath`: ASCII のみ、ドット始まり・`..` 要素・絶対パス・空要素・Windows の予約名を拒否し、`a..b.txt` のような正しい名前は通す）、**`joinWithin`**（根の下だけを許し、`..` を含む正しい名前を別名にしない。`../evil` や絶対パスは `ErrInvalidPath`）、**frontmatter**（`TestParseManifest`: `name` / `description` の両方が必要で `name` はディレクトリ名と一致すること。食い違うスキルは一覧から消さず `InvalidReason` 付きで出す）、テキスト判定が 64KiB のチャンク境界で多バイト文字を割らないこと、書き込みの作成・更新・衝突（スキルが無ければ `SKILL.md` 以外は書けない、revision 省略は新規作成だけ、食い違う revision は拒否して今の revision を伝える、大小だけ違うパスとファイル/フォルダの衝突は重複、改行コードはそのまま、一時ファイルを残さない）、一覧・取得・読み出し（規則外のディレクトリ・予約名・ドット始まりは一覧に出ない、上限ちょうどのサイズは省かない、大小だけ違うパスでは読めない）、`..` を含む正しいファイル名が書き込みでも zip の置き換えでも同じ名前のまま保存されること、ファイルとスキルの削除（空になったフォルダも消える）、**zip の検査**（`TestParseUploadedZip`: 1 段の包みフォルダを剥がし不要物を無視する、根に `SKILL.md` があれば剥がさない、トップレベルが 2 つなら剥がさない。項目の中身がヘッダの CRC と食い違う zip は `SKILL.md` でも付属ファイルでも `ErrInvalidZip`。読み出しの戻り値を見落とすと壊れた中身が revision を持って取り込まれる）、置き換えの計画と実行（計画だけでは何も書かない、手で置いたドットファイルも消える、前回の中断で残った `_tmp-*` / `_old-*` を掃除する）、ダウンロードした zip をそのまま上げ直しても何も変わらないこと、**利用者の分離**（`TestUsersAreIsolated`: 他の利用者からは見えず、`..` や区切り・`:` を含む利用者 ID は `ErrInvalidUserID`）、**大小だけ違う利用者 ID の拒否**（`TestUserDirCaseConflictIsRejected`: 読み取り系も書き込み系も `ErrInvalidName` で止まり、元の利用者のスキルは 1 バイトも変わらず、根の直下に別名のディレクトリも作業用の残骸も残らない。Windows のファイルシステムは大小を区別しないので、`exactChildDir` の突き合わせを外すと `WriteFile` と `Replace` が他人のスキルを書き換える）、**`Replace` の巻き戻し**（`TestReplaceKeepsExistingSkillWhenSwapFails`: 誰かがファイルを開いていて rename が失敗しても既存のスキルは変わらず、`_tmp-*` / `_old-*` を残さない。Windows 専用）、後始末ログのパスが `%q` で 1 行に収まること |

### ユーティリティ

| ファイル | テスト内容 |
|---------|-----------|
| `sqlite3impl/sqlite3impl_util_test.go` | 検索SQLの組み立て。LIKE のエスケープ、列をまたぐAND、検索対象列を持たないリポジトリ、曜日フィルタの nil ガード。ID の照合は肯定語の前方一致だけで、**7文字未満の語には ID の `LIKE` を付けない**（`TestGenerateFindSQLCommon_ShortWordDoesNotMatchID`。気分の `8` で git のコミットが出ていた件の再発防止。見る列が無い内部クエリでも短い語は何にも一致せず、完全一致の経路は語の長さを問わず ID を見る）。語長の境界は `TestGenerateFindSQLCommon_IDPrefixMatchWordLengthBoundary` が固定する（6文字は ID を見ずバインド値は対象列の数だけ、7文字ちょうどで ID の `LIKE` が出る。`%` のエスケープで LIKE パターンが伸びても語そのものの長さで数える。Go 側の `find_word.MinIDPrefixMatchLength` と対。[ADR-0114](../../../../documents/adr/0114-word-filter-id-prefix-needs-seven-chars.md)）。除外語は ID を見ず、`WordsSkipIDMatch` なら肯定語でも見ない |
| `sqlite3impl/index_usage_test.go` | 主要クエリがインデックスを使うこと（EXPLAIN QUERY PLAN で確認） |
| `sqlite3impl/unixepoch_index_test.go` | 時刻比較を `unixepoch()` に統一してもインデックスが効くこと |
| `sqlite3impl/sqlite_connection_test.go` | SQLite3 接続の設定（PRAGMA 等） |
| `sqlite3impl/localtime_check_test.go` | SQLite の `'localtime'` と Go の `time.Local` が同じ壁時計かの自己検査（4テスト + 子プロセス用の `TestMain`。中核の3本は Linux 専用で、Windows の開発機では `t.Skip` になり CI の Linux でだけ走る）。実行環境で一致すること、Linux では Go 側だけ Asia/Tokyo に固定し `TZ=:/nonexistent` の子プロセスで libc が UTC へ落ちる Android の状態を再現して「不一致」と言えること、`TZ=:<TZif ファイル>` と POSIX 文字列（`JST-9` / `<+09>-9`）のどちらでも一致に戻ること、`TZ=:<相対名>` は CWD にそのファイルが実在しても musl が zoneinfo ディレクトリでしか探さず UTC に落ちること（同じファイルの絶対パスなら一致。Termux で `TZ=:$HOME/...` のリテラルになっていた事故の再現）。食い違うと時間帯フィルタの SQL 段と Go 段が別の壁時計で判定し、検索がエラーも警告も出ないまま0件になる |
| `sqlite3impl/latest_data_index_test.go` | 最新版判定の相関サブクエリ（`UPDATE_TIME_UNIX = (SELECT MAX(...) WHERE ID = T.ID)`）が ID の索引で引けていること（EXPLAIN QUERY PLAN）。索引が外れても1行ごとの全表走査になって「ただ遅い」だけでエラーは出ない |
| `sqlite3impl/timeis_range_index_test.go` | TIMEIS キャッシュ表の `START_TIME_UNIX` / `END_TIME_UNIX` 索引を採る判断の根拠。期間絞り込み（全検索で走る）が索引の無いときは全表走査で、索引を張ると `START_TIME_UNIX` の索引を使い全表走査が残らないことをクエリプランで表明する（実行時間は計測環境でぶれるのでプランで固定する）。実行中判定の経路は開いた範囲なので索引が不利になるが、それを承知で採った理由は冒頭コメントに残してあるだけで、この経路のプランは表明していない |
| `sqlite3impl/bulk_insert_bench_test.go` | キャッシュ再構築の INSERT を1行ずつ実行するのと multi-row にまとめるのの比較ベンチ。**multi-row にしてはいけない**根拠（modernc.org/sqlite はプレースホルダを数千個持つ文の準備とバインドが高くつく）。`go test` の既定では走らない |
| `sqlite3impl/synchronous_bench_test.go` | `synchronous` を NORMAL から FULL へ上げたときの書き込みコストの実測（ADR-0215 の根拠。実ファイルを開いて fsync のコストを測る）。`go test` の既定では走らない |
| `hide_files/file_hider_test.go` | ファイル非表示ロジック |

## テスト内容

- **GkillDAOManager**: 全 DAO の初期化・接続管理・ライフサイクル
- **部分的な保管場所障害**: 利用可能な rep だけでの継続、書き込み先の fail-closed、個別/集約 Error ログ、正常時の無用な集約ログ抑止
- **アカウント**: ユーザ作成、パスワードハッシュ検証、アカウント更新・削除
- **セッション**: セッション発行、有効期限検証、セッション破棄
- **設定管理**: サーバ設定・ユーザ設定・リポジトリ定義の CRUD
- **共有**: Kyou 共有情報の作成・更新・削除・取得
- **通知**: VAPID プッシュ通知ターゲットの管理
- **スキルのファイルストア**: 名前・パスの規則、frontmatter、zip の検査、一時ファイル + rename の書き込みと置き換えの巻き戻し、利用者の分離（大小だけ違う利用者 ID の拒否）
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
