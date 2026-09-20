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
│   ├── gkill.log.1 ... .5          # サイズ上限で回転した統合ログ（レベル別ログも同様）
│   ├── gkill_mcp_read.log             # Read MCPサーバの統合ログ（レベル別の gkill_mcp_read_<level>.log も同じ接頭辞。回転も本体と同じ）
│   ├── gkill_mcp_write.log            # Write MCPサーバ（同上）
│   └── gkill_mcp_readwrite.log        # Read/Write MCPサーバ（同上）
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
| Address | `127.0.0.1:9999` | リッスンアドレス（**ループバック限定**、ポート9999。`:9999` のようにホスト部を空にすると全インターフェース） |
| EnableTLS | `false` | TLS有効化 |
| IsLocalOnlyAccess | `true` | ローカルアクセスのみ許可（ループバック以外からのリクエストを 403 にする） |
| URLogTimeout | 1分 | URLog取得時のタイムアウト |
| URLogUserAgent | Chromeユーザーエージェント | URLog取得時のUA |
| UploadSizeLimitMonth | `-1` | 月間アップロード制限（-1=無制限） |
| OpenDirectoryCommand | `explorer /select,$filename` | ディレクトリを開くコマンド（Windows） |
| OpenFileCommand | `rundll32 url.dll,FileProtocolHandler $filename` | ファイルを開くコマンド（Windows） |

既定の2値（Address / IsLocalOnlyAccess）の正本は `src/server/gkill/dao/server_config/server_config.go` の
`DefaultListenAddress` / `DefaultIsLocalOnlyAccess` で、初回起動ブロックと DAO の既定マップが両方これを引く
（[ADR-0708](../adr/0708-local-only-listen-by-default.md)）。**既存の `server_config.db` はこの既定へ移行されない。**
2026-09-14 より前に作られた設定は `:9999` / `false` のままなので、LAN 公開をやめたい場合は設定画面で変える。

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

> **リリースゲート（2026-09-14。2026-08-30 監査 F-009 の運用規約を機械強制に置き換え）**:
> `npm run release` は先頭で `npm run verify_release_gate`（`src/tools/verify_release_gate.mjs`）を
> 通し、次が全部そろわないと止まる。**抜け道のフラグは無い。**
>
> 1. 作業ツリーがクリーン（untracked も不可）
> 2. 必須9スイート（`verify_docs` / `test_server` / `test_client_unit` / `test_client_e2e` /
>    `test_mcp` / `test_tools` / `test_plugins` / `test_android` / `test_wear_os`）が **今の HEAD の tree** に
>    対して `test_attestation.local.json` に記録済み。記録は `npm run test_*` の実体
>    `src/tools/run_test_suite.mjs` が成功時にだけ書く（`npm test` 一発でも、段階ごとに回しても同じ）
> 3. E2E は PATH 上の `gkill_server` も同じ tree から作られている（`gkill_server version` の `tree:`）
> 4. GitHub Actions の CI がこの HEAD で success、直近の Nightly が success かつ 48 時間以内・HEAD の祖先・
>    以後に依存の宣言（`package.json` / `package-lock.json` / `go.mod` / `go.sum`）が変わっていない
>
> 通ると `release/RELEASE_ATTESTATION_<version>.json` が書かれ、`verify_release_artifacts` が SHA256SUMS に
> 載せて、ビルド後にも HEAD が動いていないことを再検査する。
>
> **実際の流れ**: バージョンを上げてコミット → push → CI の完了を待つ → その最終ツリーで
> `npm run test_*` を全部通す（`npm test` でも可。バージョン bump も tree を変えるので bump の後に回す）
> → WSL で `npm run release`。段階ごとに回すときの並列度指定は `npm run test_client_e2e -- --workers=2` の
> ように渡せる（並列度・レポータは記録される。`-run` / `--grep` / ファイル名の絞り込みは記録されない）。
>
> NG の読み方: 「記録なし / tree 不一致」→ そのスイートを今のツリーで回す。「gkill_server が別のツリー」→
> `npm run put_version_info_embed` からビルドし直して E2E を回す。「CI run が無い」→ push していない。
> 「Nightly 以後に依存の宣言が変わっている」→ GitHub Actions 画面で Nightly を手動実行して待つ。
> ネットワーク不通・API エラーもそのまま NG（fail-closed）。
>
> Nightly の run 単位の success が保証するのは govulncheck / npm audit（E2E と Android/Wear は
> continue-on-error）。E2E・Android/Wear の実証はローカルの attestation（2.）が担う。
>
> リポジトリ外の配置スクリプトは
> `npm run release` を経由せずビルドスクリプトを直接呼ぶため**このゲートを通らない**。そちらからも
> 先頭で `npm run verify_release_gate` を呼ぶこと。

> MCP のツール説明を変えたら `gkill_server mcp schema-budget` で tools/list のバイト量が予算内かを見る。
> 意図した変更なら `gkill_server mcp schema-budget --update` で予算ファイル（`src/server/gkill/mcp/tool_schema_budget.json`）を
> 書き直してから同じコミットに入れる（`test_mcp` の予算テストが照合する）。応答の形を変えたら `golden_test.go` の
> ゴールデンも当該行を更新する。MCP は gkill_server の一部なので、本番へは本体と同じ exe を配り直す（停止窓が要る）。
>
> APK 3本はリリース署名でビルドされる。署名鍵の受け渡しと未設定時の挙動は
> `.claude/skills/gkill-build-test/SKILL.md` の「APK リリース署名」を参照。

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

**既定ではそもそも LAN の別マシンから届かない**（待受が `127.0.0.1:9999` で、届いても
「ローカルアクセスのみ許可」が 403 にする）。他の端末から使うには、同じ端末で初回登録を済ませてから
下の 4.3 の手順でサーバ設定を開く。

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

### 4.3 LAN の他端末から使う（LAN 公開）

既定ではサーバと同じ端末からしか使えない。スマートフォンや別の PC から使うには、
管理者がサーバ設定画面（設定 → サーバ設定）で次の2つを**明示的に**開く。

1. **アドレス** を `:9999`（全インターフェース）または LAN 側の IP（例: `192.168.0.10:9999`）にする
2. **ローカルアクセスのみ許可** をオフにする
3. 適用するとサーバが再起動する（`HandleUpdateServerConfigs` → `LaunchGkillServerAPI` のループが張り直す）

TLS を有効にすることを推奨する（「オレオレTLSファイル生成」→「TLS有効」→ 証明書・鍵のパスを確認）。
TLS 無しで非ループバック待受にすると、起動時に標準出力へ警告が出る（`printInsecureBindWarning`）が、
起動は妨げない。TLS を使うかは利用者の判断で、gkill は強制しない。

`--address` フラグは設定 DB を書き換えない実行時上書きなので、設定画面の値と実際の待受がずれる
（Android 同梱サーバは `--address 127.0.0.1:9999` で常にループバック）。

### 4.4 スキーマ 1.1.0 への移行（アップグレード時の必読事項）

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

### 4.5 リポジトリ設定

ライフログデータの保存先ディレクトリを設定画面から登録:
1. アプリケーション設定 → リポジトリ管理
2. データディレクトリのパスを追加
3. IDF（ファイル監視）対象ディレクトリの設定

---

## 5. バックアップ・リストア

### 5.1 バックアップ対象

gkill のデータは全て **ファイルベース**（SQLite3 + 通常ファイル）である。ただし SQLite3 は `journal_mode=DELETE`（ロールバックジャーナル方式）で動作するため、**サーバー稼働中に単純ファイルコピーすると書き込み途中の不整合な状態が写る**。しかも設定・データが複数の DB ファイルと外部リポジトリに分かれているため、稼働中コピーでは各ファイルの「時点」が揃わず、**復元して起動するまで不整合に気づけない**。整合したバックアップを取るには「サーバー停止」か「スナップショット」のどちらかで時点を揃える必要がある（→ 5.2）。

| 対象 | パス | 優先度 |
|---|---|---|
| 設定データベース群 | `$HOME/gkill/configs/*.db` | 必須 |
| ユーザーデータ | `$HOME/gkill/datas/` | 必須 |
| ユーザー登録リポジトリ | ユーザー設定で指定したディレクトリ群（`$HOME/gkill/` の外に置ける） | 必須 |
| TLS証明書 | `$HOME/gkill/tls/` | TLS使用時のみ |
| キャッシュ | `$HOME/gkill/caches/`（`zip_cache/` 含む） | 任意（再生成可能） |
| ログ | `$HOME/gkill/logs/` | 任意 |

### 5.2 バックアップ手順（時点整合の取り方）

**時点整合は必ず次のどちらかの方法で確保すること。** 稼働中の単純コピーは公式手順としない。

#### 方法A（推奨）: サーバーを停止してからコピー

最も確実。全 DB と外部リポジトリを同一時点でコピーできる。

```bash
# 1. gkill_server を停止（Ctrl+C など）してから実行

# 2. 設定・データ・TLS をコピー
cp -r $HOME/gkill/configs/ /backup/gkill_configs_$(date +%Y%m%d)/
cp -r $HOME/gkill/datas/   /backup/gkill_datas_$(date +%Y%m%d)/
cp -r $HOME/gkill/tls/     /backup/gkill_tls_$(date +%Y%m%d)/   # TLS使用時のみ
```

#### 方法B: ファイルシステムのスナップショット（無停止）

サーバーを止められない場合は、全ファイルを一瞬で写すファイルシステムスナップショット（LVM / ZFS / btrfs / Windows VSS 等）を使う。スナップショットは複数 DB と外部リポジトリを**同一時点**で捉えるため、稼働中でも整合したバックアップになる。スナップショットを取ってから、そのスナップショットに対して 5.1 の対象をコピーする。

> **SQLite online backup（`.backup` / バックアップAPI）について:** `sqlite3 src.db ".backup dst.db"` は稼働中の**単一 DB** を安全に写せるが、gkill は複数 DB と外部リポジトリにまたがるため、**各ファイルを個別に online backup しても相互の時点は揃わない**。全体整合が要るバックアップでは方法A（停止）か方法B（スナップショット）を用いること。online backup は「特定の1 DB だけを稼働中に取り出したい」用途に限る。

#### 外部リポジトリを取りこぼさないこと

上記の `configs/` `datas/` `tls/` コピーだけでは**完全バックアップにならない**。ユーザーがリポジトリ設定で `$HOME/gkill/` の外（任意の外部ディレクトリ）を指定している場合、そのデータは含まれない。まず登録済みリポジトリのパス一覧を確認し、外部ディレクトリも**本体と同じ時点で**コピーする（停止中、または同一スナップショット内で）。

```bash
# 1. 登録済みリポジトリのパス一覧を設定DBから確認
#    リポジトリ定義は configs/ 配下の設定DB（rep 一覧）に保存されている。
#    アプリケーション設定画面（リポジトリ管理）でも一覧を確認できる。
sqlite3 $HOME/gkill/configs/*.db "SELECT file FROM REP;" 2>/dev/null | sort -u
#    （テーブル/カラム名は環境により異なる場合がある。設定画面での確認が確実）

# 2. $HOME/gkill/ の外にあるリポジトリを個別にコピー（例）
cp -r /path/to/external/repo/ /backup/gkill_external_repo_$(date +%Y%m%d)/
```

#### バックアップ後の検証（integrity_check + ハッシュマニフェスト）

バックアップは**取っただけでは検証にならない**。次の2つを毎回実施する。

```bash
# (1) SQLite3 ファイルの内部整合性を確認（各 DB が ok を返すこと）
find /backup/gkill_configs_$(date +%Y%m%d) /backup/gkill_datas_$(date +%Y%m%d) \
  -name '*.db' -print0 \
| while IFS= read -r -d '' db; do
    echo "$db: $(sqlite3 "$db" 'PRAGMA integrity_check;')"
  done

# (2) ハッシュマニフェストを作成（後日の破損検知・復元前照合に使う）
( cd /backup && find gkill_configs_$(date +%Y%m%d) gkill_datas_$(date +%Y%m%d) \
    -type f -exec sha256sum {} + ) > /backup/manifest_$(date +%Y%m%d).sha256
```

- `PRAGMA integrity_check` が全 DB で `ok` を返すこと（`ok` 以外は破損。稼働中コピーで時点がずれると出やすい）。
- 外部リポジトリを含む全リポジトリパスがバックアップに含まれているか、上のパス一覧と突き合わせる。
- ハッシュマニフェストは復元前の照合（`sha256sum -c manifest_YYYYMMDD.sha256`）に使う。

### 5.3 リストア手順と復元訓練

**リストア:**

1. gkill_server を停止
2. 復元前にハッシュマニフェストで破損がないか照合（`( cd /backup && sha256sum -c manifest_YYYYMMDD.sha256 )`）
3. バックアップファイルを元のパスに上書きコピー（**外部リポジトリのパスも忘れずに**）
4. gkill_server を再起動し、ログにエラーが無いこと・記録が検索できることを確認

**復元訓練（restore drill）を定期的に行うこと:** バックアップは「復元できて初めて成功」である。本番とは別のホームへ実際に復元して gkill_server を起動し、記録が読めるかを定期的に確認する。これにより、外部リポジトリの取りこぼしや時点不整合を**復元が必要になる前に**発見できる。

```bash
# 別ホームへ復元して起動確認（本番に影響しない）
mkdir -p /tmp/gkill_restore_test/gkill
cp -r /backup/gkill_configs_YYYYMMDD/ /tmp/gkill_restore_test/gkill/configs/
cp -r /backup/gkill_datas_YYYYMMDD/   /tmp/gkill_restore_test/gkill/datas/
gkill_server --gkill_home_dir /tmp/gkill_restore_test/gkill --address 127.0.0.1:19999
# 起動後、ログイン・検索が正常なら訓練成功。確認後は破棄してよい。
```

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
| `error` | エラーのみ（**既定値**） | `gkill_error.log` |
| `warn` | 警告以上 | `gkill_warn.log` |
| `info` | 情報以上 | `gkill_info.log` |
| `access` | アクセスログ以上（INFO含む） | `gkill_access.log` |
| `debug` | デバッグ以上 | `gkill_debug.log` |
| `trace` | トレース以上 | `gkill_trace.log` |
| `trace_sql` | SQL文含む全て | `gkill_trace_sql.log` |
| `none` | ログ出力なし | — |

`--log access` を指定すると、全HTTPリクエストのアクセスログ（リモートIP、メソッド、パス、ステータスコード、所要時間、ユーザID）が `gkill_access.log` に記録されます。

**既定が `error` なのは、「`gkill_error.log` に出ていなければ起きていない」と言えるようにするためです。**
どの事象をどのレベルで出すかの判断基準は [ADR-1001](../adr/1001-log-level-by-severity.md) にあります。
要点は「呼び出し元へ返らないエラーは Debug に置かない」「利用者の入力・認証・認可の失敗は Warn 以下」の2つです。
失敗したリクエストは応答を書く1箇所（`writeErrorStatus`）が、ステータスから決まるレベルで1行残します
（5xx は Error、401・403・429 は Warn、その他の4xx は Debug）。エラーコード・メソッド・パス・ユーザIDが載ります。

### 6.2 ログフォーマット

JSON形式。各行に以下のフィールド:
- タイムスタンプ
- ログレベル
- ソース位置（ファイル:行番号）
- メッセージ
- `{"app": "gkill"}`（静的フィールド）

### 6.3 統合ログ

`gkill.log` には全レベルのログが統合出力される。レベル別ファイルと統合ファイルの両方が同時に書き込まれる。

### 6.4 ログローテーション

統合ログと各レベル別ログは、1ファイルが既定の 32 MiB を超える直前に回転する。
現在のファイルを `.1`、それ以前を `.2` 以降へ送り、既定では5世代まで保持する。

| フラグ | 既定値 | 意味 |
|---|---:|---|
| `--log_rotate_max_bytes` | `33554432` | 1ファイルの上限バイト数。0以下は回転を無効化する |
| `--log_rotate_keep` | `5` | 保持する旧世代数。0以下は旧ファイルを保持せず破棄する |

回転に失敗しても本体処理は止めず、現在のログへの書き込みを続ける。容量監視では現行ファイルだけでなく
`.1` 以降も含めること。障害調査中に世代数を減らすと必要な時刻のログが消えるため、先に退避してから変更する。

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
1. 画面に「記録保管場所を読み込めない」旨が出ても、まず他の場所の記録が表示されているか確認する。書き込み先でない1本だけが読めない場合、利用可能な場所で処理は継続する
2. 外部ストレージやネットワーク共有の接続、読み取り権限を確認し、再接続後に画面の再読み込みまたは記録保管場所の再読み込みを行う
3. 継続する場合は `gkill_error.log` を確認する。`detach broken repository` が個別の場所、`repositories loaded with detached reps` が同一構築処理の集約を示す。正常な構築では集約行は出ない
4. 書き込み先が読めない場合や設定データベース自体が壊れた場合は処理を継続せず HTTP 500 になる。サーバーを停止し、ログから対象DBを特定する
5. バックアップからリストアする。バックアップがない場合は、原本を退避したうえで `sqlite3 broken.db ".recover" | sqlite3 repaired.db` による救出を別ファイルへ試す
6. 復旧後に `PRAGMA integrity_check;` が `ok` を返すこと、記録保管場所を再読み込みして警告が消えること、読み書きできることを確認する

`optimize` は健全なDBの最適化であり、破損DBの修復手段ではない。破損した原本へ直接書き込まず、
バックアップまたはコピーに対して復旧作業を行うこと。

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

どちらも生成済みのものは作り直さない。判定は、生成対象の親ディレクトリとキャッシュディレクトリをそれぞれ1回ずつ列挙して行う（1件ずつ `os.Stat` はしない。[ADR-0212](../adr/0212-derived-cache-scan-lists-directories.md)）。

互換動画は全件を変換するわけではない。ffprobe でコンテナ・映像コーデック・画素形式・プロファイル・音声コーデックを見て、**原本のままブラウザで再生できると言い切れるものは変換せずそのまま配信する**（[ADR-0213](../adr/0213-transcode-only-what-the-browser-cannot-play.md)）。変換に失敗した動画には `<キャッシュ名>.failed` という印が残り、次回以降はやり直さずに原本へフォールバックする。印を消すには `gkill_server clear_cache video ユーザーID` でキャッシュごと削除する。

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

`--cache_reps_local=true`の場合、リモートリポジトリのデータを`caches/local_rep_cache/{user_id}/`配下のSQLite3 DBにコピーしてローカルから高速検索する。ネットワーク遅延が大きい環境（NAS等）で有効。

**コピーするかどうかは元DBとキャッシュDBの mtime + サイズ で決める**（`localRepCacheNeedsCopy`）。判定は必ず「閉じる・消す」より**前**に行うこと。`os.Remove`のあとに`os.Stat`すると、消した直後なので必ず「要コピー」と判定され、

- 変更のないrepまで毎回まるごとコピーし直す
- `LastUpdateCacheChanged()`が常に`true`になり、上位のキャッシュrep（`CachedSQLite3Impl`）のフルリビルド抑止が丸ごと効かなくなる

の2つが同時に起きる。実データ（rep約940・約83万行・外付けUSB上のDB 818本 1.3GB）では、これで`update_cache`1回のphase1が **0.2秒から1〜2分** へ悪化していた。回帰は`local_rep_cache_granular_test.go`が検出する。

再構築成功の通知（`CommitCacheRebuild`）を受けるまで基準を進めない点も他repと同じ。途中で失敗した回のぶんは次回も再構築される。

ただし**ReKyou / MiReKyouのローカルキャッシュrepは常に「変更あり」を返す**。この2つのキャッシュ内容は自分のDBファイルだけでなく他repのターゲット解決結果にも依存し、`GkillRepositories.UpdateCache`がアドレス確定後にもう一度更新するため、mtimeで判定するとその2回目が飛んでターゲット未解決の中身が残る（`db_file_change_detector.go`の説明と同じ理由）。コピーの省略だけは行う。

**上の判定が効いていれば、この層の定常コストはほぼゼロ**（実データの反復`update_cache`で、有効26.8秒 / 無効30.3秒。むしろ有効なほうが速い。再構築の読み出し元が外付けUSBではなくC:になるため）。逆に**元ファイルのmtimeがまとめて動く運用（同期直後など）では毎回1.3GBのコピーが上乗せされる**ので、その場合だけは`--cache_reps_local`を外したほうが速くなる。効かせたいなら同期側でmtimeを保存させるのが本筋。

#### サムネイル・動画キャッシュ（ファイルキャッシュ）

`caches/thumb_cache/{user_id}/{rep_name}/`（サムネイル）と `caches/video_cache/{user_id}/{rep_name}/`（互換動画）の配下にファイルを保存する。ファイル名は `{sha1(リポジトリ内相対パス)}_{ファイルサイズ}_{幅}x{高さ}.jpg`（サムネイル）／`{sha1(相対パス)}_{ファイルサイズ}_compat_{高さ}p.mp4`（互換動画）。キャッシュキーは**ファイル内容ではなく相対パス**のSHA1なので、同一内容の別ファイルはキャッシュを共有しない。ファイルサイズをキーに含めることで、同じパスのファイルが差し替わったときに自動で別エントリになる。rep名は利用者間で一意でないため、利用者IDを1階層挟んで混ざらないようにしている（`derivedCacheDirForUser`）。

| キャッシュ種別 | パス形式 | 生成タイミング |
|---|---|---|
| サムネイル | `caches/thumb_cache/{user_id}/{rep_name}/{sha1}_{size}_{W}x{H}.jpg` | 初回ブラウザアクセス時 or `generate_thumb_cache` 事前生成 |
| 互換動画 | `caches/video_cache/{user_id}/{rep_name}/{sha1}_{size}_compat_{H}p.mp4` | 初回ブラウザアクセス時 or `generate_video_cache` 事前生成 |

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

- **CLI**: `gkill_server clear_cache <thumb|video|zip|plugin|all> <all|user_id...>` — 対象は必須指定（他サブコマンドと同様）。`all` を渡すとユーザーコンテキスト不要で `caches/thumb_cache` / `caches/video_cache` / `caches/zip_cache` / `caches/plugin_cache` ディレクトリを丸ごと削除する（全ユーザー対象。管理者メンテ向け）。user_id を1つ以上指定すると、各ユーザーのリポジトリを読み込み（`LoadIDFRepOnly`）、そのユーザーのIDFリポジトリ分のキャッシュのみ `IDFKyouReps.Clear{Thumb,Video,Zip}Cache(userID)` で削除する（3種とも `caches/{種別}/{user_id}/{rep_name}` を消す）。例: `clear_cache all 〈ユーザーID〉`（そのユーザーの全種）、`clear_cache zip all`（全ユーザーのZIP）。`plugin` はディレクトリ（`caches/plugin_cache/{userID}`）を消すだけなのでリポジトリの読み込みを行わない。プラグインは次回起動時にキャッシュを作り直す。
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
| `gkill_server generate_plugin_cache <plugin_name\|all> ユーザーID...` | プラグインのキャッシュ（`caches/plugin_cache/`）を同期構築する。稼働中サーバは不要で、各プラグインバイナリを `--gkill-build-cache` 付きで単独起動して終わるまで待つ（逐次・タイムアウト無し）。`plugin_name` は manifest.json の `name`（＝フォルダ名）。同期スクリプトでは新しいデータを置いた直後・`update_cache` の前に置く |
| `gkill_server add_tag ユーザーID... --rules_file <path>` | 検索条件 JSON に一致する記録へタグを付ける（稼働中サーバの HTTP クライアント。`main/common/add_tag.go`） |
| `gkill_server reset_password ユーザーID...` | 指定アカウントのパスワードを無効化し、リセットトークンを再発行してURLを表示する。account.db を直接開くのでサーバー稼働中でも実行できる。パスワードはArgon2idで保存されておりDBから復元できないため、**管理者がパスワードを忘れたときやトークンが期限切れになったときの唯一の復帰経路**（`main/common/password_admin.go`） |

## 11. MCP HTTPサーバーのデプロイ

gkill MCP サーバーは3種類提供されている。いずれもOAuth 2.1認証で保護されたHTTPサーバーとして動作する。

| サーバー | ファイル | ツール数 | デフォルトポート | 用途 |
|---|---|---|---|---|
| Read専用 | `gkill_server mcp --kind read` | 12 (11 read + 1 plugin) | 8808 | 読み取りのみ |
| Write専用 | `gkill_server mcp --kind write` | 29 (21 write + 7 read convenience + 1 plugin) | 8809 | 書き込み中心 |
| Read/Write統合 | `gkill_server mcp --kind readwrite` | 33 (11 read + 21 write + 1 plugin) | 8810 | 全機能 |

プラグインツール `gkill_get_plugin_list` は `src/server/gkill/mcp/plugin_tools.go` の `PluginTools` を
各サーバの `composeTools(...)` に連結したもので、3サーバ共通・読み取り専用（`post_plugin_config` は公開しない）。
MCP は `gkill_server mcp` サブコマンド（stdio / http）で、起動中の gkill_server への HTTP クライアント。設定は
`$GKILL_HOME/configs/gkill_mcp.json`（初回起動時に生成）と環境変数（フラグ > 環境変数 > ファイル > 既定値）。
プラグインKyouの本文は `gkill_get_kyous` に `include_plugin_content:true` を渡すと
レスポンスへ直接埋め込まれる（Write専用サーバには `gkill_get_kyous` が無いため本文は読めない）。

### 11.1 起動

```bash
# Read専用
MCP_TRANSPORT=http MCP_PORT=8808 MCP_OAUTH_ISSUER="https://<公開ホスト名>" \
  gkill_server mcp --kind read

# Write専用
MCP_TRANSPORT=http MCP_PORT=8809 MCP_OAUTH_ISSUER="https://<公開ホスト名>" \
  gkill_server mcp --kind write

# Read/Write統合
MCP_TRANSPORT=http MCP_PORT=8810 MCP_OAUTH_ISSUER="https://<公開ホスト名>" \
  gkill_server mcp --kind readwrite
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
| `MCP_BIND_ADDR` | `0.0.0.0` | 待ち受けアドレス。トンネル・リバースプロキシの背後では `127.0.0.1` |
| `GKILL_HOME` | `$HOME/gkill` | MCPサーバがログ（`logs/`）・設定（`configs/gkill_mcp.json`）・トークン永続化ファイル（`configs/`）を置く場所の解決に使う（`--gkill_home_dir` が明示されていないときだけ） |
| `MCP_LOG` | `access` | MCP のログレベル（`none`/`error`/`warn`/`info`/`access`/`debug`/`trace`/`trace_sql`。gkill_server の `--log` と同じ語彙。未知の値は起動を止める） |

#### ログファイルとトークン永続化ファイル

ログは gkill_log の別名ファイル群（`logs/gkill_mcp_<kind>.log` = 全レベル統合、`gkill_mcp_<kind>_{error,warn,info,access,debug,trace,trace_sql}.log` = レベル別。形式・回転は本体と同じで、静的フィールドは `app=gkill_mcp` と `kind`）。

| サーバー | 統合ログ | トークン永続化ファイル |
|---|---|---|
| Read | `gkill_mcp_read.log` | `mcp_oauth_read_state.json` |
| Write | `gkill_mcp_write.log` | `mcp_oauth_write_state.json` |
| ReadWrite | `gkill_mcp_readwrite.log` | `mcp_oauth_readwrite_state.json` |

#### MCPログのイベント一覧

全サーバー共通で以下のイベントが記録される:

| msg | レベル | 記録内容 | 発生タイミング |
|---|---|---|---|
| `http_request` | ACCESS/WARN | remote_addr, method, path（クエリは落とす）, status, reason, response_bytes | 全HTTPリクエスト（400以上はWARN） |
| `tool_call` | ACCESS | tool, user_id, remote_addr, duration | MCP ツールコール成功 |
| `tool_call_error` | WARN/ERROR | tool, user_id, remote_addr, duration, error | MCP ツールコール失敗（引数の誤り = 呼び出し側の問題は WARN、内部エラーは ERROR） |
| `auth_success` | INFO | user_id | OAuth 認証成功 |
| `auth_failure` | WARN | user_id, reason | OAuth 認証失敗 |
| `token_rejected` | WARN | remote_addr, method, path | Bearer トークン検証失敗 |
| `token_scope_rejected` | WARN | remote_addr, method, path, token_scope, required_scope | scope の不一致（403） |
| `server_start` | INFO | transport, log_level, port, pid, schema_revision, tool_count | サーバ起動 |
| `http_listening` | INFO | port, issuer | HTTP の待ち受け開始 |
| `config_created` | INFO | path | 設定ファイルの初回生成 |
| `status_gkill_unreachable` | WARN | error | `gkill_status` が gkill へ届かなかった |
| `file_fetch_error` / `http_handler_error` / `oauth_state_save_error` | ERROR | error ほか | 内部障害 |

どのサーバーからの出力かはファイル名（接頭辞 `gkill_mcp_<kind>`）と静的フィールド `kind` で識別できる。

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
