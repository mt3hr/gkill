# common - CLI 共通基盤

## 概要

gkill の CLI 共有初期化ロジック、ログシステム、グローバル設定オプション、ゴルーチンプール管理を提供する。
`gkill_server` と `gkill`（デスクトップアプリ）の両方から共通で使用される。

## ディレクトリ構造

```
common/
├── common.go              # cobra CLI コマンド定義・サーバ初期化
├── common_test.go         # common.go のテスト
├── add_tag.go             # add_tag サブコマンド（稼働中サーバの HTTP クライアント）
├── add_tag_test.go        # add_tag のテスト
├── generate_plugin_cache.go      # generate_plugin_cache サブコマンド（プラグインを単独起動して同期構築）
├── generate_plugin_cache_test.go # generate_plugin_cache のテスト（偽プラグインをヘルパープロセスで起動）
├── password_admin.go      # reset_password サブコマンド／ローカル管理者セッションの発行
├── password_admin_test.go # password_admin.go のテスト
├── gkill_log/             # ログシステム（10ファイル）
│   ├── gkill_log.go       # ロガー初期化・レベルルーティング
│   ├── level.go           # カスタム slog.Level 定義（8レベル）
│   ├── router.go          # ログルーター（分割/統合ファイル）
│   ├── routing_handler.go # slog.Record 処理ハンドラ
│   ├── sinks.go           # FileSink（ファイル出力・サイズ上限での世代回転）
│   ├── switch_writer.go   # スレッドセーフなライター切り替え
│   ├── sql_log.go         # TRACE_SQL の唯一の入口（引数の先行評価を避ける）
│   ├── gkill_log_test.go  # ログシステムのテスト
│   ├── no_eager_sql_format_test.go     # TRACE_SQL の引数を先行評価していないか
│   └── log_level_source_scan_test.go   # ログレベルが内容と合っているか
├── gkill_options/         # グローバル設定オプション（2ファイル）
│   ├── option.go          # 全 CLI フラグ定義（70+ パラメータ）
│   └── option_test.go     # オプションのテスト
└── threads/               # ゴルーチンプール管理（2ファイル）
    ├── threads.go         # セマフォベースのゴルーチン制限
    └── threads_test.go    # スレッド管理のテスト
```

**合計: 22ファイル**（実装13 + テスト9）

## サブパッケージ

### `gkill_log/` — ログシステム

`log/slog` をベースにしたカスタムマルチレベルログシステム。

#### ログレベル（低→高）

| レベル | slog.Level 値 | 用途 |
|--------|-------------|------|
| `TraceSQL` | 最低 | SQL クエリトレース |
| `Trace` | | 詳細トレース |
| `Debug` | | 開発時の詳細。**エラーの置き場ではない**（呼び出し元へ返り、境界が1行出すものだけ） |
| `Access` | | HTTP アクセスログ |
| `Info` | | 起動・終了・構築完了などの節目。1事象1行で流れ続けない |
| `Warn` | | 動き続けるが結果が痩せる。監査に要る利用者由来の事象（認証失敗・認可拒否・レート制限） |
| `Error` | | 運用者がいま知るべきサーバ側の障害。利用者の操作では起こらない |
| `None` | 最高 | ログ無効化 |

**既定は `error`。** 「`gkill_error.log` に出ていなければ起きていない」と言えることを既定にしてある。
どの事象をどのレベルで出すかの判断基準は [ADR-1001](../../../../../documents/adr/1001-log-level-by-severity.md)、
機械検査は `log_level_source_scan_test.go`。

#### 特徴

- **レベル別ファイル分割**: 各レベルごとに個別のログファイルに出力
- **統合ログファイル**: 全レベルを1つのファイルにも統合出力
- **ログローテーション**: 統合・レベル別の各ファイルを既定32 MiBで回転し、5世代を保持（CLIで変更可能）
- **stdout ミラーリング**: オプションで標準出力にも出力
- **ホットスワップ**: `SwitchWriter` によりログファイルの出力先を無停止で切り替え可能
- **ルーティング**: `Router` が各 `slog.Record` を適切なファイルに振り分け

### `gkill_options/` — グローバル設定オプション

cobra/viper と連携する CLI フラグのグローバル変数群。構造体ではなくパッケージレベル変数として定義。

#### 主要オプション

| 変数 | デフォルト | 説明 |
|------|-----------|------|
| `GkillHomeDir` | `$HOME/gkill` | gkill ホームディレクトリ |
| `CacheDir` | `$GkillHomeDir/cache` | キャッシュディレクトリ |
| `LogDir` | `$GkillHomeDir/log` | ログ出力ディレクトリ |
| `ConfigDir` | `$GkillHomeDir/config` | 設定ファイルディレクトリ |
| `DataDirectory` | `$GkillHomeDir/data` | データ保存ディレクトリ |
| `TLSCertFile` | | TLS 証明書ファイルパス |
| `TLSKeyFile` | | TLS 秘密鍵ファイルパス |
| `GoroutinePool` | `runtime.NumCPU()` | ゴルーチンプールサイズ |
| `CacheClearCountLimit` | `3000` | キャッシュクリアしきい値 |
| `CacheUpdateDuration` | `1m` | キャッシュ更新間隔 |
| `PreLoadUserNames` | （なし） | 起動時プリロード対象ユーザ |
| `CacheXxxReps` | `true` | 各データ型のキャッシュ有効フラグ（Kmemo, TimeIs, Mi 等） |
| `IDFIgnore` | | IDF 無視リスト |

### `threads/` — ゴルーチンプール

セマフォベースのゴルーチン数制限。`gkill_options.GoroutinePool` で設定されたサイズに基づき、
`Acquire()` / `Release()` で並行度を管理する。`sync.Once` パターンで初期化。

## common.go — CLI コマンド

cobra のルートコマンドに登録されるサブコマンド群を定義。サーバ初期化ロジックも含む:
- `version` — バージョン情報表示
- `idf` — IDF 操作
- `dvnf` — データバージョニングファイル操作
- `generate_thumb_cache` — サムネイルキャッシュ生成
- `generate_video_cache` — 動画キャッシュ生成
- `optimize` — データベース最適化
- `update_cache` — キャッシュ手動更新
- `clear_cache` — ディスク上の派生キャッシュ削除
- `generate_plugin_cache` — プラグインのキャッシュを稼働中サーバ無しで同期構築（定義は `generate_plugin_cache.go`。各プラグインバイナリを `--gkill-build-cache` で単独起動し、stdout の結果行 `built` / `no_cache` で成否を判定）
- `reset_password` — パスワードを無効化してリセットURLを再発行（定義は `password_admin.go`）
- `add_tag` — 検索条件 JSON（FindKyouQuery）に一致する Kyou へタグを付与（定義は `add_tag.go`。稼働中サーバの HTTP クライアント）

## 関連ドキュメント

- [main/README.md](../README.md) — CLI エントリポイント
- [dvnf/README.md](../../dvnf/README.md) — DVNF サブコマンド詳細
