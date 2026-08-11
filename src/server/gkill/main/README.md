# main - CLI エントリポイント

## 概要

gkill アプリケーションのエントリポイントと CLI サブコマンド定義。
cobra ライブラリを使用したコマンドライン構造を持ち、2つのデプロイモードを提供する。

## ディレクトリ構造

```
main/
├── common/                    # 共有初期化ロジック → common/README.md 参照
│   ├── common.go              # サーバ初期化・共通 cobra コマンド登録
│   ├── common_test.go         # common.go のテスト
│   ├── gkill_log/             # ログシステム（7ファイル: 実装6 + テスト1）
│   │   ├── gkill_log.go       # ログ初期化
│   │   ├── level.go           # ログレベル定義（8レベル）
│   │   ├── router.go          # ログルーティング
│   │   ├── routing_handler.go # ルーティングハンドラ
│   │   ├── sinks.go           # ログ出力先
│   │   ├── switch_writer.go   # 出力先切り替え
│   │   └── gkill_log_test.go  # ログシステムテスト
│   ├── gkill_options/         # CLI オプション（2ファイル）
│   │   ├── option.go          # 全 CLI フラグ定義（70+ パラメータ）
│   │   └── option_test.go     # オプションテスト
│   └── threads/               # ゴルーチンプール（2ファイル）
│       ├── threads.go         # セマフォベースのゴルーチン制限
│       └── threads_test.go    # スレッド管理テスト
├── gkill/
│   └── main.go                # デスクトップアプリエントリポイント
├── gkill_server/
│   └── main.go                # HTTP サーバエントリポイント
```

## デプロイモード

### `gkill_server/main.go` — HTTP サーバモード

ヘッドレスな HTTP サーバとして起動。ブラウザや PWA からアクセスする。
ビルド: `go install ./gkill/main/gkill_server/`

### `gkill/main.go` — デスクトップアプリモード

go-astilectron を使用して Electron ライクなデスクトップウィンドウ内で gkill_server を起動。
ビルド: `go install ./gkill/main/gkill/`

## cobra CLI サブコマンド

サブコマンドの定義は `common/common.go` にあり、各バイナリの `main.go`（`gkill_server/main.go`, `gkill/main.go`）が
`AddCommand()` でルートコマンドに登録する。`idf` は `gkill_server` のみが登録する（`gkill` には無い）:

| サブコマンド | 説明 |
|-------------|------|
| `version` | バージョン情報表示 |
| `idf` | IDF（ファイル）関連操作（gkill_server のみ） |
| `dvnf` | ファイル命名規則・バージョニング → `dvnf/README.md` 参照 |
| `generate_thumb_cache` | サムネイルキャッシュ生成 |
| `generate_video_cache` | 動画キャッシュ生成 |
| `optimize` | データベース最適化 |
| `update_cache` | キャッシュ更新 |
| `clear_cache` | ディスク上の派生キャッシュ削除 |
| `reset_password` | パスワードを無効化してリセットURLを再発行（`common/password_admin.go`） |

## common/ 配下

詳細は [common/README.md](common/README.md) を参照。

### `common.go`

サーバの初期化ロジック:
- cobra ルートコマンドの設定
- サブコマンドの登録
- GkillDAOManager の初期化
- HTTP サーバの起動

### `gkill_log/`（7ファイル）— ログシステム

`log/slog` ベースのカスタムマルチレベルログ（TraceSQL, Trace, Debug, Access, Info, Warn, Error, None の8レベル）。
レベル別ファイル分割、統合ログ、stdout ミラーリング、ホットスワップに対応。

| ファイル | 役割 |
|---------|------|
| `gkill_log.go` | ログシステムの初期化・グローバルロガー設定 |
| `level.go` | ログレベル定義（8レベル） |
| `router.go` | ログメッセージのルーティング（分割/統合ファイル） |
| `routing_handler.go` | ルーティングハンドラ実装 |
| `sinks.go` | FileSink（ファイルベースログ出力） |
| `switch_writer.go` | スレッドセーフな出力先の動的切り替え |
| `gkill_log_test.go` | ログシステムのテスト |

### `gkill_options/`（2ファイル）— CLI オプション

cobra/viper 連携のグローバル変数群（70+ パラメータ）。GkillHomeDir, CacheDir, LogDir, ConfigDir, GoroutinePool, CacheXxxReps フラグ等。

### `threads/`（2ファイル）— ゴルーチンプール

セマフォベースの並行度制限。`gkill_options.GoroutinePool`（デフォルト: `runtime.NumCPU()`）に基づき `Acquire()` / `Release()` で管理。

## 開発ガイドライン

### 新しいサブコマンドの追加

1. `common/common.go` にサブコマンド（cobra.Command）を定義し、各バイナリの `main.go`（`gkill_server/main.go`・`gkill/main.go`）で `AddCommand()` する
2. サブコマンドのロジックは別パッケージ（例: `dvnf/cmd/`）に分離
3. 共通の初期化ロジック（DB 接続等）は `common.go` の既存関数を再利用
