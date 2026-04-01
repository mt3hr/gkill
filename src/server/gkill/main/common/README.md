# common - CLI 共通基盤

## 概要

gkill の CLI 共有初期化ロジック、ログシステム、グローバル設定オプション、ゴルーチンプール管理を提供する。
`gkill_server` と `gkill`（デスクトップアプリ）の両方から共通で使用される。

## ディレクトリ構造

```
common/
├── common.go              # cobra CLI コマンド定義・サーバ初期化
├── common_test.go         # common.go のテスト
├── gkill_log/             # ログシステム（7ファイル）
│   ├── gkill_log.go       # ロガー初期化・レベルルーティング
│   ├── level.go           # カスタム slog.Level 定義（8レベル）
│   ├── router.go          # ログルーター（分割/統合ファイル）
│   ├── routing_handler.go # slog.Record 処理ハンドラ
│   ├── sinks.go           # FileSink（ファイルベースログ出力）
│   ├── switch_writer.go   # スレッドセーフなライター切り替え
│   └── gkill_log_test.go  # ログシステムのテスト
├── gkill_options/         # グローバル設定オプション（2ファイル）
│   ├── option.go          # 全 CLI フラグ定義（70+ パラメータ）
│   └── option_test.go     # オプションのテスト
└── threads/               # ゴルーチンプール管理（2ファイル）
    ├── threads.go         # セマフォベースのゴルーチン制限
    └── threads_test.go    # スレッド管理のテスト
```

**合計: 13ファイル**（実装8 + テスト5）

## サブパッケージ

### `gkill_log/` — ログシステム

`log/slog` をベースにしたカスタムマルチレベルログシステム。

#### ログレベル（低→高）

| レベル | slog.Level 値 | 用途 |
|--------|-------------|------|
| `TraceSQL` | 最低 | SQL クエリトレース |
| `Trace` | | 詳細トレース |
| `Debug` | | デバッグ情報 |
| `Access` | | HTTP アクセスログ |
| `Info` | | 一般情報 |
| `Warn` | | 警告 |
| `Error` | | エラー |
| `None` | 最高 | ログ無効化 |

#### 特徴

- **レベル別ファイル分割**: 各レベルごとに個別のログファイルに出力
- **統合ログファイル**: 全レベルを1つのファイルにも統合出力
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

## 関連ドキュメント

- [main/README.md](../README.md) — CLI エントリポイント
- [dvnf/README.md](../../dvnf/README.md) — DVNF サブコマンド詳細
