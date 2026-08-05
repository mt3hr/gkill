# 運用ガイド

## 1. ディレクトリ構成

gkill は `$HOME/gkill/`（デフォルト、`--gkill_home_dir` で変更可能）配下にデータ・設定・ログを保存する。

```
$HOME/gkill/
├── configs/                         # 設定データベース群（SQLite3）
│   ├── server_config.db            # サーバー設定
│   ├── account.db                  # アカウント情報
│   ├── account_state.db            # ログインセッション・ファイルアップロード履歴
│   ├── user_config.db              # アプリケーション設定・リポジトリ定義
│   ├── share_kyou_info.db          # 共有設定
│   └── gkill_notification_target.db # プッシュ通知ターゲット
├── datas/                           # ユーザーデータ（デフォルトデータディレクトリ）
├── caches/                          # キャッシュファイル
│   ├── thumb_cache/{user_id}/{rep_name}/ # サムネイル画像キャッシュ（利用者＋リポジトリ単位）
│   ├── video_cache/{user_id}/{rep_name}/ # 互換動画キャッシュ（利用者＋リポジトリ単位）
│   ├── zip_cache/{user_id}/{rep_name}/{sha1}/ # ZIP展開キャッシュ（利用者＋リポジトリ＋ハッシュ単位）
│   ├── local_rep_cache/            # ローカルリポジトリキャッシュDB
│   ├── git_commit_log_cache/       # Gitコミットログキャッシュ DB
│   ├── plugin_cache/{userID}/{pluginName}/ # プラグインのキャッシュDB（ユーザー＋プラグイン単位）
│   ├── temp_cache/                 # 一時キャッシュDB（非メモリモード時のみ）
│   └── latest_data_repository_address_cache/ # 最新データリポジトリアドレスキャッシュDB（非メモリモード時のみ）
├── logs/                            # ログファイル（JSON形式）
│   ├── gkill_error.log
│   ├── gkill_warn.log
│   ├── gkill_info.log
│   ├── gkill_access.log            # HTTPアクセスログ（--log access 以上で出力）
│   ├── gkill_debug.log
│   ├── gkill_trace.log
│   ├── gkill_trace_sql.log
│   ├── gkill.log                   # 全レベル統合
│   ├── gkill_mcp_read_access.log      # Read MCPサーバアクセスログ（MCP_LOG環境変数で制御）
│   ├── gkill_mcp_write_access.log     # Write MCPサーバアクセスログ
│   └── gkill_mcp_readwrite_access.log # Read/Write MCPサーバアクセスログ
├── lib/base_directory/              # ライブラリファイル
└── tls/                             # TLS証明書（オプション）
    ├── cert.cer
    └── key.pem
```

### IDF（Indexed Directory File）

監視対象ディレクトリには `.gkill/gkill_id.db` が作成され、ファイルのIDとメタデータを管理する。

---

## 2. 設定データベース一覧

全設定は SQLite3 データベースに保存される（`$HOME/gkill/configs/`）。初回起動時に自動作成。

| DB名 | 内容 | 主なテーブル |
|---|---|---|
| `server_config.db` | サーバー設定 | アドレス、TLS設定、デバイス名、VAPID鍵、URLogタイムアウト、ファイル操作コマンド |
| `account.db` | アカウント | ユーザーID、パスワードハッシュ（Argon2id）、パスワードリセットトークンとその有効期限 |
| `account_state.db` | セッション管理 | ログインセッション（SessionID, UserID, 有効期限等）、ファイルアップロード履歴 |
| `user_config.db` | ユーザー設定 | アプリケーション設定、リポジトリ定義、KFTLテンプレート |
| `share_kyou_info.db` | 共有設定 | 共有リスト設定 |
| `gkill_notification_target.db` | 通知ターゲット | Web Push通知の購読情報 |

### 主要サーバー設定項目

| 設定 | デフォルト値 | 説明 |
|---|---|---|
| Address | `:9999` | リッスンアドレス（全インターフェース、ポート9999） |
| EnableTLS | `false` | TLS有効化 |
| IsLocalOnlyAccess | `false` | ローカルアクセスのみ許可 |
| URLogTimeout | 1分 | URLog取得時のタイムアウト |
| URLogUserAgent | Chromeユーザーエージェント | URLog取得時のUA |
| UploadSizeLimitMonth | `-1` | 月間アップロード制限（-1=無制限） |
| OpenDirectoryCommand | `explorer /select,$filename` | ディレクトリを開くコマンド（Windows） |
| OpenFileCommand | `rundll32 url.dll,FileProtocolHandler $filename` | ファイルを開くコマンド（Windows） |

---

## 3. デプロイ手順

### 3.1 Windows（デスクトップアプリ）

1. ビルド:
   ```bash
   npm run install_app
   ```
2. `$GOPATH/bin/gkill.exe` が生成される
3. 実行:
   ```bash
   gkill.exe
   ```
4. go-astilectron ウィンドウ（450x750）が起動し、内蔵サーバーにアクセス

### 3.2 Windows/Linux（サーバーモード）

1. ビルド:
   ```bash
   npm run install_server
   ```
2. `$GOPATH/bin/gkill_server(.exe)` が生成される
3. 実行:
   ```bash
   gkill_server
   ```
4. ブラウザで `http://localhost:9999` にアクセス

### 3.3 Linux（クロスコンパイル）

```bash
npm run release
```

`release/` ディレクトリに各プラットフォーム向けバイナリが生成:
- `linux_amd64_gkill_server_*.zip`
- `linux_arm64_gkill_server_*.zip`
- `linux_arm_gkill_server_*.zip`

### 3.4 Android

1. gkill_server をクロスコンパイル（環境変数 `NDK` が必要）:
   ```bash
   npm run build_android_arm64
   ```
2. バイナリを `src/android/app/src/main/jniLibs/arm64-v8a/libgkill_server.so` に配置
   （`npm run copy_android_release`。targetSdk 29以降はアプリのデータディレクトリ配下を
   実行できないため、nativeLibraryDir から実行できるよう `lib*.so` 名で jniLibs に置く）
3. APKビルド:
   ```bash
   cd src/android
   ./gradlew assembleDebug
   ```
4. APKをインストール。WebView が `http://localhost:9999` で内蔵サーバーにアクセス

### 3.5 Wear OS

**前提:** Gradle Wrapper は `src/wear_os/` にコミット済みなので、事前のコピーは不要。
`src/android/` 側と揃え直したいときだけ `npm run setup_wear_os_gradle` を実行する。

ビルド:
```bash
cd src/wear_os
./gradlew :phone_companion:assembleDebug   # スマホ側コンパニオンアプリ
./gradlew :watch_app:assembleDebug          # ウォッチアプリ
```

**動作要件:**
- スマホ側: phone_companion APK + Android版gkillアプリ（gkill_serverが稼働していること）
- ウォッチ側: watch_app APK
- 同一Googleアカウントでペアリング済みであること
- 通信: Wearable Data Layer API経由

---

## 4. 初期セットアップ

### 4.1 初回起動

1. gkill_server（またはgkill）を起動
2. 自動的に以下が作成される:
   - `$HOME/gkill/` ディレクトリ構造
   - 全設定データベース（configs/配下）
   - `admin` アカウント（`PasswordHash = nil` かつ `PasswordResetToken` が設定された状態）
   - VAPID鍵ペア（Web Push用）
   - デフォルトデバイス `"gkill"`
3. ブラウザで `http://localhost:9999` にアクセス

### 4.2 初回アカウント登録とパスワード設定

初回起動時の `admin` アカウントには `PasswordResetToken` が設定されているため、**パスワードなしではログインできない**。`PasswordResetToken` が非nilのアカウントは、ログイン処理でパスワード照合より前に `ERR000004`（`AccountPasswordResetTokenIsNotNilError`）で拒否される（`handle_login.go`）。

**正しい初期導線：**
1. **サーバーと同じマシンの**ブラウザで `http://localhost:9999` にアクセスすると初回アカウント登録画面（`/register_first_account`）に誘導される
2. ここで `admin` のパスワードを設定して初回登録を完了する
3. 以降は SHA256(パスワード) を送って認証する（サーバ側はその値をArgon2idで保存・照合する）

**LAN の別マシンやスマートフォンからは自動誘導されない。** この 307 リダイレクトには
`admin` を丸ごと取れてしまうリセットトークンが載るため、**接続元がループバックで、かつ
転送ヘッダ（`X-Forwarded-For` / `X-Real-Ip` / `Forwarded` / `X-Forwarded-Host`）が付いていない
場合にだけ**返す（`utils.go` の `ifRedirectResetAdminAccountIsNotFound`）。
それ以外からアクセスすると通常のログイン画面が出る。

その場合は、**サーバー起動時に標準出力へ出るセットアップURL**を使う。

```
----------------------------------------------------------------
パスワードが未設定のアカウントがあります。
下記のURLからパスワードを設定してください。
期限が切れた場合は `gkill_server reset_password <user_id>` で再発行できます。
  admin : http://localhost:9999/set_new_password?user_id=admin&reset_token=...
----------------------------------------------------------------
```

`localhost` の部分を実際のホスト名やIPに読み替えてアクセスする。
Windows サービスなどで標準出力が見えない場合は、
`gkill_server reset_password admin` を実行すればURLを取り直せる。

### 4.3 スキーマ 1.1.0 への移行（アップグレード時の必読事項）

`account.db` のスキーマが `1.0.0` の状態で新しいバイナリを起動すると、**初回起動時に一度だけ移行が走る**。

移行の内容は次のとおり（単一トランザクション）。

1. `PASSWORD_SHA256` を `PASSWORD_HASH` にリネーム
2. `PASSWORD_RESET_TOKEN_EXPIRATION` を追加
3. **全アカウントのパスワードを無効化し、リセットトークンを発行しなおす**
4. スキーマ版を `1.1.0` に更新

旧方式では保存値（無塩SHA-256）がそのままログインに使えたため、Argon2id で包み直すだけでは
「DBを読めた者がログインできる」状態が続いてしまう。そのため包み直しではなく全員に再設定してもらう。

発行したリセットURLは起動時に標準出力へ出るので、**全アカウントぶんを控えてからパスワードを設定しなおす**。

| 注意点 | 内容 |
|---|---|
| **ダウングレード不可** | `1.1.0` の `account.db` を旧バイナリで開くと `invalid db schema version` で**起動を拒否する**。戻す場合は移行前の `account.db` をバックアップから書き戻す |
| **事前バックアップ必須** | 移行前に `$HOME/gkill/configs/account.db` をコピーしておく |
| **設定ディレクトリを端末間で同期している場合** | 移行済みの `account.db` が旧バイナリの端末へ届くと、その端末が起動できなくなる。**同期を止めて全端末のバイナリを入れ替えてから**移行を走らせる |
| **セッションが全部切れる** | パスワード設定のたびにそのユーザの全セッションが失効する。ブラウザ・Wear OS・MCP は再ログインが必要（Wear OS は保存済みパスワードで自動復帰する） |
| **URLog ブックマークレット** | ブックマークレットにはセッションIDが埋め込まれているため、再ログイン後に設定画面から取り直す |

### 4.4 リポジトリ設定

ライフログデータの保存先ディレクトリを設定画面から登録:
1. アプリケーション設定 → リポジトリ管理
2. データディレクトリのパスを追加
3. IDF（ファイル監視）対象ディレクトリの設定

---

## 5. バックアップ・リストア

### 5.1 バックアップ対象

gkill のデータは全て **ファイルベース**（SQLite3 + 通常ファイル）のため、ファイルコピーでバックアップ可能。

| 対象 | パス | 優先度 |
|---|---|---|
| 設定データベース群 | `$HOME/gkill/configs/*.db` | 必須 |
| ユーザーデータ | `$HOME/gkill/datas/` | 必須 |
| ユーザー登録リポジトリ | ユーザー設定で指定したディレクトリ群 | 必須 |
| TLS証明書 | `$HOME/gkill/tls/` | TLS使用時のみ |
| キャッシュ | `$HOME/gkill/caches/`（`zip_cache/` 含む） | 任意（再生成可能） |
| ログ | `$HOME/gkill/logs/` | 任意 |

### 5.2 バックアップ手順

```bash
# サーバーを停止してからバックアップ（SQLite3ロック回避）
# Ctrl+C でサーバー停止後:

# 設定のバックアップ
cp -r $HOME/gkill/configs/ /backup/gkill_configs_$(date +%Y%m%d)/

# データのバックアップ
cp -r $HOME/gkill/datas/ /backup/gkill_datas_$(date +%Y%m%d)/

# TLS証明書（使用時のみ）
cp -r $HOME/gkill/tls/ /backup/gkill_tls_$(date +%Y%m%d)/
```

**重要 — 外部ディレクトリのリポジトリを取りこぼさないこと:** 上記の `configs/` `datas/` `tls/` コピーだけでは**完全バックアップにならない**。ユーザーがリポジトリ設定で `$HOME/gkill/` の外（任意の外部ディレクトリ）を指定している場合、そのデータは含まれない。完全バックアップには、まず登録済みリポジトリのパス一覧を確認し、外部ディレクトリも個別にコピーする必要がある。

```bash
# 1. 登録済みリポジトリのパス一覧を設定DBから確認
#    リポジトリ定義は configs/ 配下の設定DB（rep 一覧）に保存されている。
#    アプリケーション設定画面（リポジトリ管理）でも一覧を確認できる。
sqlite3 $HOME/gkill/configs/*.db "SELECT file FROM REP;" 2>/dev/null | sort -u
#    （テーブル/カラム名は環境により異なる場合がある。設定画面での確認が確実）

# 2. $HOME/gkill/ の外にあるリポジトリを個別にコピー（例）
cp -r /path/to/external/repo/ /backup/gkill_external_repo_$(date +%Y%m%d)/
```

**バックアップ後の検証:**
- コピー先の SQLite3 ファイルが破損していないか確認する（`sqlite3 <file> "PRAGMA integrity_check;"` が `ok` を返すこと）。
- 外部リポジトリを含む全リポジトリパスがバックアップに含まれているか、手順1の一覧と突き合わせる。

**注意:** サーバー稼働中のSQLite3ファイルコピーはデータ破損のリスクがある。必ずサーバーを停止してからコピーすること。

### 5.3 リストア手順

1. gkill_server を停止
2. バックアップファイルを元のパスに上書きコピー
3. gkill_server を再起動

---

## 6. ログ確認方法

### 6.1 ログレベル

起動時に `--log` フラグでログレベルを指定:

```bash
gkill_server --log debug    # DEBUG以上を出力
gkill_server --log trace    # TRACE以上を出力
gkill_server --log trace_sql # SQL文も含め全出力
```

| レベル | 内容 | ファイル |
|---|---|---|
| `error` | エラーのみ | `gkill_error.log` |
| `warn` | 警告以上 | `gkill_warn.log` |
| `info` | 情報以上 | `gkill_info.log` |
| `access` | アクセスログ以上（INFO含む） | `gkill_access.log` |
| `debug` | デバッグ以上 | `gkill_debug.log` |
| `trace` | トレース以上 | `gkill_trace.log` |
| `trace_sql` | SQL文含む全て | `gkill_trace_sql.log` |
| `none` | ログ出力なし（**既定値**） | — |

`--log access` を指定すると、全HTTPリクエストのアクセスログ（リモートIP、メソッド、パス、ステータスコード、所要時間、ユーザID）が `gkill_access.log` に記録されます。

### 6.2 ログフォーマット

JSON形式。各行に以下のフィールド:
- タイムスタンプ
- ログレベル
- ソース位置（ファイル:行番号）
- メッセージ
- `{"app": "gkill"}`（静的フィールド）

### 6.3 統合ログ

`gkill.log` には全レベルのログが統合出力される。レベル別ファイルと統合ファイルの両方が同時に書き込まれる。

---

## 7. トラブルシューティング

### 7.1 ポート競合

**症状:** `bind: address already in use`

**対処:**
1. ポート9999を使用中のプロセスを確認:
   ```bash
   # Linux/macOS
   lsof -i :9999
   # Windows
   netstat -ano | findstr :9999
   ```
2. 該当プロセスを停止、または `server_config.db` でポートを変更

### 7.2 TLS証明書問題

**症状:** HTTPS接続エラー、証明書関連エラー

**対処:**
1. 証明書ファイルの存在確認: `$HOME/gkill/tls/cert.cer` と `key.pem`
2. `/api/generate_tls_file` で自己署名証明書を再生成
3. ブラウザで自己署名証明書の例外を許可
4. `--disable_tls` フラグでHTTPにフォールバック

### 7.3 Wear OS接続問題

**症状:** ウォッチからテンプレート取得やKFTL送信ができない

**チェックリスト:**
1. スマホ側で phone_companion APK がインストール・稼働しているか
2. スマホ側で gkill_server（Android版）が起動しているか
3. スマホとウォッチが同一Googleアカウントでペアリングされているか
4. Wearable Data Layer の接続状態を確認（Wear OS設定）
5. phone_companion のログイン情報（URL, ユーザーID, パスワード）が正しいか

### 7.4 キャッシュ問題

**症状:** 古いデータが表示される、メモリ使用量が大きい

**対処:**
- キャッシュ無効化: `--cache_in_memory=false` で起動
- キャッシュ上限調整: `--cache_clear_count_limit` でアイテム数を変更（デフォルト: 3000）
- キャッシュ更新間隔: `--cache_update_duration` で変更（デフォルト: 1分）
- API経由でキャッシュ更新: `POST /api/update_cache`
- CLI: `gkill_server update_cache ユーザーID...` サブコマンド（他サブコマンドと同様に対象ユーザーIDの文字列配列を受け取る。**認証情報の指定は不要**。サーバーと同一マシンで実行する前提で、ローカルの `configs/account.db` から有効な管理者アカウントを自動選択し、その名義で有効期限5分のログインセッションを `configs/account_state.db` へ直接発行して使う）

### 7.5 フロントエンドが表示されない

**症状:** `http://localhost:9999` にアクセスしても空白

**対処:**
1. ビルド確認: `npm run build` が成功しているか
2. embed確認: `src/server/gkill/api/embed/html/` にビルド成果物があるか
3. Go再ビルド: `npm run go_install` でembedを含めて再コンパイル
4. フルリビルド: `npm run install_server` で全工程をやり直す

### 7.6 SQLite3データベース破損

**症状:** 起動時やデータアクセス時に `database disk image is malformed`

**対処:**
1. サーバー停止
2. 破損DBの特定（ログで確認）
3. バックアップからリストア
4. バックアップがない場合: `sqlite3 broken.db ".recover" | sqlite3 repaired.db` で修復を試みる
5. `gkill_server optimize ユーザーID` でDB最適化

---

## 8. Web Push通知設定

### 8.1 自動セットアップ

VAPID鍵ペアは初回サーバー起動時に自動生成され、`server_config.db` に永続保存される。手動設定は不要。

### 8.2 ブラウザ側の許可

1. gkill にログイン
2. 通知設定画面で「通知を有効にする」
3. ブラウザの通知許可ダイアログで「許可」
4. Service Worker が `/api/register_gkill_notification` で購読情報を登録

### 8.3 通知が届かない場合

- ブラウザの通知権限を確認（サイト設定 → 通知 → 許可）
- Service Worker が登録されているか確認（DevTools → Application → Service Workers）
- サーバーのVAPID鍵が有効か確認（`server_config.db` の `GkillNotificationPrivateKey` / `GkillNotificationPublicKey`）

---

## 9. パフォーマンスチューニング

### 9.1 CLIフラグ

| フラグ | デフォルト | 説明 | チューニング指針 |
|---|---|---|---|
| `--cache_in_memory` | `true` | インメモリキャッシュ | メモリ不足時は `false` に |
| `--cache_clear_count_limit` | `3000` | キャッシュアイテム上限 | 大量データ時は増加、メモリ節約時は減少 |
| `--cache_update_duration` | `1m` | キャッシュ更新間隔 | リアルタイム性重視なら短く、負荷軽減なら長く |
| `--goroutine_pool` | `NumCPU()` | goroutineプールサイズ | CPU数に応じて自動調整。手動設定も可 |
| `--pre_load_users` | (なし) | 起動時にプリロードするユーザー | 頻繁にアクセスするユーザーを指定で起動後の初回アクセスが高速化 |

### 9.2 SQLite3の制約

- **同時書き込み:** SQLite3はライターロックを使用するため、高頻度の同時書き込みには不向き
- **データ量:** 単一テーブルに大量レコードがある場合、Append-Only設計のため履歴蓄積でサイズが増加
- **最適化:** `gkill_server optimize ユーザーID` でVACUUM実行

### 9.3 サムネイル・動画キャッシュ

大量のファイル（idf_kyou）がある場合、サムネイル/動画キャッシュの事前生成で表示速度を改善:

```bash
gkill_server generate_thumb_cache ユーザーID
gkill_server generate_video_cache ユーザーID
```

### 9.4 キャッシュアーキテクチャ詳細

gkillは複数層のキャッシュを組み合わせてパフォーマンスを確保している。

#### インメモリキャッシュ（CachedSQLite3Impl）

`--cache_in_memory=true`（デフォルト）の場合、各リポジトリは`CachedSQLite3Impl`でラップされる。

**設計方針:**
- `CachedSQLite3Impl`は内部にSQLite3の検索結果をメモリ上のキャッシュ（`[]KyouInterface`スライス）として保持する
- キャッシュの有効期間は`--cache_update_duration`（デフォルト: 1分）。期限切れ後は次回アクセス時に再取得
- キャッシュアイテム数が`--cache_clear_count_limit`（デフォルト: 3000）を超えるとキャッシュをクリアして再構築
- 書き込み（ADD/UPDATE）後はキャッシュを即時無効化し、次回読み取り時に再構築する（キャッシュ汚染防止）
- `--cache_in_memory=false`にするとキャッシュなしで毎回SQLite3に直接クエリ（メモリ不足時の代替手段）

#### ローカルキャッシュ（SQLite3ImplLocalCached）

`--cache_reps_local=true`の場合、リモートリポジトリのデータを`caches/local_rep_cache/`配下のSQLite3 DBにコピーしてローカルから高速検索する。ネットワーク遅延が大きい環境（NAS等）で有効。

#### サムネイル・動画キャッシュ（ファイルキャッシュ）

`caches/thumb_cache/{rep_name}/{sha1}.webp`（サムネイル）と`caches/video_cache/{rep_name}/{sha1}.webm`（互換動画）の形式でファイルを保存する。キャッシュキーにはファイルのSHA1ハッシュを使用するため、同一内容のファイルは1つのキャッシュエントリで共有される。

| キャッシュ種別 | パス形式 | 生成タイミング |
|---|---|---|
| サムネイル | `caches/thumb_cache/{rep_name}/{sha1}.webp` | 初回ブラウザアクセス時 or `generate_thumb_cache` 事前生成 |
| 互換動画 | `caches/video_cache/{rep_name}/{sha1}.webm` | 初回ブラウザアクセス時 or `generate_video_cache` 事前生成 |

#### ZIPキャッシュ（アトミック展開）

`/api/browse_zip_contents`リクエスト時、以下の手順でアトミックにZIPを展開する：

1. 一時ディレクトリ（`caches/zip_cache/{user_id}/{rep_name}/{sha1}_tmp_{uuid}/`）に展開
2. 展開完了後、`{sha1}_tmp_{uuid}` → `{sha1}` にアトミックリネーム
3. リネーム済みディレクトリが存在する場合は展開をスキップ（べき等）

この方式により、複数の同時リクエストがあっても競合状態を回避する（singleflightでさらに同時展開を抑制）。

#### キャッシュ更新API

`POST /api/update_cache`（または`gkill_server update_cache ユーザーID...` CLIコマンド）を呼び出すと、指定ユーザーのインメモリキャッシュを即時再構築する。サーバー再起動なしにリポジトリ変更を反映する際に使用する。**このエンドポイントは管理者セッション（`session_id`）を必須とする**（`wrapAuth` + `IsAdmin` 判定）。CLIサブコマンドは対象ユーザーIDの文字列配列を受け取り、**認証情報の指定は不要**（サーバーと同一マシンで実行する前提で、ローカルの `configs/account.db` から `IsAdmin && IsEnable` の管理者アカウントを自動選択し、その名義で有効期限5分のログインセッションを `configs/account_state.db` へ直接 INSERT して使い、終了時に削除する。`main/common/password_admin.go` の `issueLocalAdminSession`）。パスワードはArgon2idで保存されており DB から復元できないため、`/api/login` は経由しない。

#### キャッシュ削除

ディスク上の派生キャッシュ（サムネイル・互換動画・ZIP展開・プラグイン）を削除する手段は2つある：

- **CLI**: `gkill_server clear_cache <thumb|video|zip|plugin|all> <all|user_id...>` — 対象は必須指定（他サブコマンドと同様）。`all` を渡すとユーザーコンテキスト不要で `caches/thumb_cache` / `caches/video_cache` / `caches/zip_cache` / `caches/plugin_cache` ディレクトリを丸ごと削除する（全ユーザー対象。管理者メンテ向け）。user_id を1つ以上指定すると、各ユーザーのリポジトリを読み込み（`LoadIDFRepOnly`）、そのユーザーのIDFリポジトリ分のキャッシュのみ `IDFKyouReps.Clear{Thumb,Video,Zip}Cache(userID)` で削除する（3種とも `caches/{種別}/{user_id}/{rep_name}` を消す）。例: `clear_cache all mt3hr`（mt3hrの全種）、`clear_cache zip all`（全ユーザーのZIP）。`plugin` はディレクトリ（`caches/plugin_cache/{userID}`）を消すだけなのでリポジトリの読み込みを行わない。プラグインは次回起動時にキャッシュを作り直す。
- **画面のリロード（再読込）ボタン長押し**: `POST /api/reload_repositories` に `clear_thumb_cache` / `clear_video_cache` / `clear_zip_cache` フラグを立てて送信し、**ログイン中ユーザーのリポジトリ分のみ** サムネ・動画・ZIPキャッシュを削除してからリポジトリを再構築する（クリックはこれらフラグを立てず、キャッシュ削除は行わない）。

いずれも消したキャッシュは次回アクセス時に遅延再生成される。

---

## 10. サーバー管理

### 10.1 グレースフルシャットダウン

- `Ctrl+C`（SIGINT）または `SIGTERM` で30秒のグレースフルシャットダウン
- 処理中のリクエストは完了まで待機

### 10.2 サーバー再起動

設定変更（ServerConfig更新）時、サーバーは `RebootServerCh` チャネル経由で自動再起動する。手動再起動は不要。

### 10.3 CLIサブコマンド

| コマンド | 説明 |
|---|---|
| `gkill_server version` | バージョン・ビルド情報表示 |
| `gkill_server idf` | ディレクトリファイルのインデックス作成。`-i`/`--ignore` で除外指定。**`gkill_server` にのみ登録**されており、デスクトップアプリ `gkill` からは使えない |
| `gkill_server dvnf` | DVNF処理（`get [dvnfPath]` / `copy src target` / `move src target`）。共通フラグ `--new`/`-n`、`--auto_create`、`--device`。この端末に有効な ServerConfig が無い場合は `--device` が必須。詳細は [dvnf-rep-type-spec.md](dvnf-rep-type-spec.md) |
| `gkill_server generate_thumb_cache ユーザーID` | サムネイルキャッシュ生成 |
| `gkill_server generate_video_cache ユーザーID` | 動画キャッシュ生成 |
| `gkill_server optimize ユーザーID` | データベース最適化（VACUUM） |
| `gkill_server update_cache ユーザーID...` | HTTP API経由でキャッシュ更新（対象ユーザーIDの文字列配列。認証情報の指定は不要。管理者名義の短命セッションをローカルDBへ自己発行して使う） |
| `gkill_server clear_cache <thumb\|video\|zip\|plugin\|all> <all\|user_id...>` | ディスク上の派生キャッシュを削除。対象は必須で、`all`で全体、user_id指定で該当ユーザーのリポジトリ分のみ |
| `gkill_server reset_password ユーザーID...` | 指定アカウントのパスワードを無効化し、リセットトークンを再発行してURLを表示する。account.db を直接開くのでサーバー稼働中でも実行できる。パスワードはArgon2idで保存されておりDBから復元できないため、**管理者がパスワードを忘れたときやトークンが期限切れになったときの唯一の復帰経路**（`main/common/password_admin.go`） |

## 11. MCP HTTPサーバーのデプロイ

gkill MCP サーバーは3種類提供されている。いずれもOAuth 2.1認証で保護されたHTTPサーバーとして動作する。

| サーバー | ファイル | ツール数 | デフォルトポート | 用途 |
|---|---|---|---|---|
| Read専用 | `gkill-read-server.mjs` | 10 (8 read + 2 plugin) | 8808 | 読み取りのみ |
| Write専用 | `gkill-write-server.mjs` | 25 (20 write + 3 read convenience + 2 plugin) | 8809 | 書き込み中心 |
| Read/Write統合 | `gkill-readwrite-server.mjs` | 30 (8 read + 20 write + 2 plugin) | 8810 | 全機能 |

プラグインツール2つ（`gkill_get_plugin_list` / `gkill_get_plugin_content`）は
`src/mcp/lib/plugin-tools.mjs` の `PLUGIN_TOOLS` を各サーバの `TOOLS` に展開したもので、
3サーバ共通・読み取り専用（`post_plugin_config` は公開しない）。

### 11.1 起動

```bash
# Read専用
MCP_TRANSPORT=http MCP_PORT=8808 MCP_OAUTH_ISSUER="https://<公開ホスト名>" \
  node src/mcp/gkill-read-server.mjs

# Write専用
MCP_TRANSPORT=http MCP_PORT=8809 MCP_OAUTH_ISSUER="https://<公開ホスト名>" \
  node src/mcp/gkill-write-server.mjs

# Read/Write統合
MCP_TRANSPORT=http MCP_PORT=8810 MCP_OAUTH_ISSUER="https://<公開ホスト名>" \
  node src/mcp/gkill-readwrite-server.mjs
```

共通の環境変数 `GKILL_BASE_URL`, `GKILL_USER`, `GKILL_PASSWORD_SHA256` も必要。

### 11.2 環境変数

| 変数 | デフォルト | 説明 |
|---|---|---|
| `GKILL_BASE_URL` | `http://127.0.0.1:9999` | gkillバックエンドURL |
| `GKILL_USER` | — | gkillログインユーザーID |
| `GKILL_PASSWORD_SHA256` | — | パスワードのSHA-256ハッシュ（`GKILL_PASSWORD`でも可） |
| `MCP_TRANSPORT` | `stdio` | `http` でHTTPモード起動 |
| `MCP_PORT` | `8808`/`8809`/`8810` | HTTPサーバーポート（サーバーごとにデフォルト異なる） |
| `MCP_OAUTH_ISSUER` | `http://localhost:<port>` | OAuthメタデータのissuer URL。**リモートアクセス時は必須**（公開URL）|
| `GKILL_INSECURE` | `false` | `true` でgkillバックエンドへのTLS証明書検証をスキップ |
| `GKILL_HOME` | `$HOME/gkill` | MCPサーバがログ（`logs/`）とトークン永続化ファイル（`configs/`）を置く場所の解決に使う |
| `MCP_LOG` | `info` | MCPアクセスログレベル（`none`/`error`/`warn`/`info`/`debug`/`trace`） |

#### アクセスログファイル

| サーバー | ログファイル | トークン永続化ファイル |
|---|---|---|
| Read | `gkill_mcp_read_access.log` | `mcp_oauth_read_state.json` |
| Write | `gkill_mcp_write_access.log` | `mcp_oauth_write_state.json` |
| ReadWrite | `gkill_mcp_readwrite_access.log` | `mcp_oauth_readwrite_state.json` |

#### MCPアクセスログのイベント一覧

全サーバー共通で以下のイベントが記録される:

| msg | レベル | 記録内容 | 発生タイミング |
|---|---|---|---|
| `http_request` | INFO/WARN | remote_addr, method, path, status, reason, response_bytes | 全HTTPリクエスト（400以上はWARN） |
| `tool_call` | INFO | tool, user_id, remote_addr, duration | MCP ツールコール成功 |
| `tool_call_error` | ERROR | tool, user_id, remote_addr, duration, error | MCP ツールコール失敗 |
| `auth_success` | INFO | user_id | OAuth 認証成功 |
| `auth_failure` | WARN | user_id | OAuth 認証失敗 |
| `token_rejected` | WARN | remote_addr, method, path | Bearer トークン検証失敗 |
| `server_start` | INFO | transport, log_level, port | サーバ起動 |

ログの `source` フィールドでどのサーバーからの出力か識別可能（`gkill-read-server.mjs` / `gkill-write-server.mjs` / `gkill-readwrite-server.mjs`）。

> **注:** `POST /mcp` でツールが呼ばれた場合、`http_request`（HTTPレベル）と `tool_call`（ツールレベル）の2行が出力されます。

### 11.3 リモートアクセス（Cloudflare Tunnel等）

MCPサーバーをリモートから利用するには、外部からアクセス可能にする必要がある。各サーバーごとに個別のトンネルを設定する。

```yaml
# .cloudflared/<tunnel-name>.yml 例
ingress:
  - hostname: <公開ホスト名>
    service: http://localhost:<ポート>
    originRequest:
      noTLSVerify: true
      httpHostHeader: localhost
  - service: http_status:404
```

**重要**: `MCP_OAUTH_ISSUER` を公開URL（例: `https://example.com`）に設定すること。未設定だと OAuthメタデータ内のURLが `http://localhost` になり、Claude.ai/ChatGPT から認可エンドポイントに到達できない。

**重要**: Claude.ai等からのMCP接続URLは末尾に `/mcp` を付けること（例: `https://example.com/mcp`）。

### 11.4 トークン永続化

リフレッシュトークン（30日TTL）とDCRクライアント登録は `$GKILL_HOME/configs/` 配下の各サーバーのOAuth状態ファイルに自動保存される。サーバー再起動後も再認証不要。

### 11.4.1 ファイルURL配信ルート（HTTPモード）

HTTPモードのMCPサーバは `GET /files/{token}` を公開し、`gkill_get_kyous` のIDFペイロードに入る `file_url` / `file_url_full`（画像はサムネ＋原寸）のバイトを配信する。リモートのAIクライアント（ChatGPT等）が実パスを読めない代わりに、base64を経由せず任意サイズのファイルを取得できる。

- URLは `MCP_OAUTH_ISSUER`（公開URL）を基点に組み立てられる。Cloudflare Tunnelは hostname 単位で全パスをプロキシするため、`/mcp` と同じトンネルで `/files/…` も到達する。**追加のトンネル設定は不要**。
- トークンは特定1ファイルに束縛・期限付き（`GKILL_MCP_FILE_LINK_TTL_MS`、既定1時間）・推測不能で、メモリ保持（サーバ再起動で失効）。URLにセッションは載らない。gkill本体を公開する必要はない。
- 配信ルートはBearer不要（画像取得は認証ヘッダを付けられないため）。防御はトークン自体が担う。

### 11.5 既知の制限

- **ChatGPT**: OAuth認証・初回データ取得は成功するが、cursorベースのページング継続時にChatGPTプラットフォーム側で「Resource not found」が発生する（2026-03時点、ベータ版の制限）
- **Claude.ai**: 正常動作。OAuth自動検出・DCR・データ取得すべて安定
