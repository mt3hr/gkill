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
├── logs/                            # ログファイル（JSON形式）
│   ├── gkill_error.log
│   ├── gkill_warn.log
│   ├── gkill_info.log
│   ├── gkill_debug.log
│   ├── gkill_trace.log
│   ├── gkill_trace_sql.log
│   └── gkill.log                   # 全レベル統合
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
| `account.db` | アカウント | ユーザーID、パスワードSHA256、パスワードリセットトークン |
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

1. gkill_server をクロスコンパイル:
   ```bash
   GOOS=linux GOARCH=arm64 go build -o gkill_server ./src/server/gkill/main/gkill_server/
   ```
2. バイナリを `src/android/app/src/main/assets/gkill_server` に配置
3. APKビルド:
   ```bash
   cd src/android
   ./gradlew assembleDebug
   ```
4. APKをインストール。WebView が `http://localhost:9999` で内蔵サーバーにアクセス

### 3.5 Wear OS

**前提:** `src/android/` から Gradle Wrapper をコピーする必要がある。

```bash
cp src/android/gradlew src/wear_os/
cp src/android/gradlew.bat src/wear_os/
mkdir -p src/wear_os/gradle/wrapper
cp src/android/gradle/wrapper/gradle-wrapper.jar src/wear_os/gradle/wrapper/
cp src/android/gradle/wrapper/gradle-wrapper.properties src/wear_os/gradle/wrapper/
```

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
   - `admin` アカウント（パスワード未設定）
   - VAPID鍵ペア（Web Push用）
   - デフォルトデバイス `"gkill"`
3. ブラウザで `http://localhost:9999` にアクセス

### 4.2 パスワード設定

初回ログイン時、`admin` アカウントはパスワードなしでログイン可能。

**速やかにパスワードを設定すること：**
1. ログイン後、設定画面でパスワードを変更
2. 以降は SHA256(パスワード) で認証

### 4.3 リポジトリ設定

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
| キャッシュ | `$HOME/gkill/caches/` | 任意（再生成可能） |
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
| `debug` | デバッグ以上 | `gkill_debug.log` |
| `trace` | トレース以上 | `gkill_trace.log` |
| `trace_sql` | SQL文含む全て | `gkill_trace_sql.log` |
| `none` | ログ出力なし | — |

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

### 7.3 TLS証明書問題

**症状:** HTTPS接続エラー、証明書関連エラー

**対処:**
1. 証明書ファイルの存在確認: `$HOME/gkill/tls/cert.cer` と `key.pem`
2. `/api/generate_tls_file` で自己署名証明書を再生成
3. ブラウザで自己署名証明書の例外を許可
4. `--disable_tls` フラグでHTTPにフォールバック

### 7.4 Wear OS接続問題

**症状:** ウォッチからテンプレート取得やKFTL送信ができない

**チェックリスト:**
1. スマホ側で phone_companion APK がインストール・稼働しているか
2. スマホ側で gkill_server（Android版）が起動しているか
3. スマホとウォッチが同一Googleアカウントでペアリングされているか
4. Wearable Data Layer の接続状態を確認（Wear OS設定）
5. phone_companion のログイン情報（URL, ユーザーID, パスワード）が正しいか

### 7.5 キャッシュ問題

**症状:** 古いデータが表示される、メモリ使用量が大きい

**対処:**
- キャッシュ無効化: `--cache_in_memory=false` で起動
- キャッシュ上限調整: `--cache_clear_count_limit` でアイテム数を変更（デフォルト: 3000）
- キャッシュ更新間隔: `--cache_update_duration` で変更（デフォルト: 1分）
- API経由でキャッシュ更新: `POST /api/update_cache`
- CLI: `gkill_server update_cache` サブコマンド

### 7.6 フロントエンドが表示されない

**症状:** `http://localhost:9999` にアクセスしても空白

**対処:**
1. ビルド確認: `npm run build` が成功しているか
2. embed確認: `src/server/gkill/api/embed/html/` にビルド成果物があるか
3. Go再ビルド: `npm run go_install` でembedを含めて再コンパイル
4. フルリビルド: `npm run install_server` で全工程をやり直す

### 7.7 SQLite3データベース破損

**症状:** 起動時やデータアクセス時に `database disk image is malformed`

**対処:**
1. サーバー停止
2. 破損DBの特定（ログで確認）
3. バックアップからリストア
4. バックアップがない場合: `sqlite3 broken.db ".recover" | sqlite3 repaired.db` で修復を試みる
5. `gkill_server optimize --user ユーザーID` でDB最適化

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
- **最適化:** `gkill_server optimize --user ユーザーID` でVACUUM実行

### 9.3 サムネイル・動画キャッシュ

大量のファイル（idf_kyou）がある場合、サムネイル/動画キャッシュの事前生成で表示速度を改善:

```bash
gkill_server generate_thumb_cache --user ユーザーID
gkill_server generate_video_cache --user ユーザーID
```

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
| `gkill_server idf` | ディレクトリファイルのインデックス作成 |
| `gkill_server dvnf` | DVNF処理 |
| `gkill_server generate_thumb_cache --user ユーザーID` | サムネイルキャッシュ生成 |
| `gkill_server generate_video_cache --user ユーザーID` | 動画キャッシュ生成 |
| `gkill_server optimize --user ユーザーID` | データベース最適化（VACUUM） |
| `gkill_server update_cache` | HTTP API経由でキャッシュ更新 |

## 10. MCP HTTPサーバーのデプロイ

gkill MCP サーバー（`src/mcp/gkill-read-server.mjs`）は、Claude.ai / ChatGPT 等のAI MCPクライアントからgkillデータを読み取るためのHTTPサーバー。OAuth 2.1認証で保護されている。

### 10.1 起動

```bash
# 環境変数を設定して起動
GKILL_BASE_URL=http://127.0.0.1:9999 \
GKILL_USER=admin \
GKILL_PASSWORD_SHA256="<sha256 hex>" \
MCP_TRANSPORT=http \
MCP_OAUTH_ISSUER="https://<公開ホスト名>" \
MCP_PORT=8808 \
node src/mcp/gkill-read-server.mjs
```

### 10.2 環境変数

| 変数 | デフォルト | 説明 |
|---|---|---|
| `GKILL_BASE_URL` | `http://127.0.0.1:9999` | gkillバックエンドURL |
| `GKILL_USER` | — | gkillログインユーザーID |
| `GKILL_PASSWORD_SHA256` | — | パスワードのSHA-256ハッシュ（`GKILL_PASSWORD`でも可） |
| `MCP_TRANSPORT` | `stdio` | `http` でHTTPモード起動 |
| `MCP_PORT` | `8808` | HTTPサーバーポート |
| `MCP_OAUTH_ISSUER` | `http://localhost:<port>` | OAuthメタデータのissuer URL。**リモートアクセス時は必須**（公開URL）|
| `GKILL_INSECURE` | `false` | `true` でgkillバックエンドへのTLS証明書検証をスキップ |

### 10.3 リモートアクセス（Cloudflare Tunnel等）

MCPサーバーをリモートから利用するには、外部からアクセス可能にする必要がある。

```yaml
# .cloudflared/config.yml 例
ingress:
  - hostname: example.com
    service: http://localhost:8808
    originRequest:
      noTLSVerify: true
  - service: http_status:404
```

**重要**: `MCP_OAUTH_ISSUER` を公開URL（例: `https://example.com`）に設定すること。未設定だと OAuthメタデータ内のURLが `http://localhost` になり、Claude.ai/ChatGPT から認可エンドポイントに到達できない。

### 10.4 トークン永続化

リフレッシュトークン（30日TTL）とDCRクライアント登録は `$GKILL_HOME/configs/mcp_oauth_state.json` に自動保存される。サーバー再起動後も再認証不要。

### 10.5 既知の制限

- **ChatGPT**: OAuth認証・初回データ取得は成功するが、cursorベースのページング継続時にChatGPTプラットフォーム側で「Resource not found」が発生する（2026-03時点、ベータ版の制限）
- **Claude.ai**: 正常動作。OAuth自動検出・DCR・データ取得すべて安定
