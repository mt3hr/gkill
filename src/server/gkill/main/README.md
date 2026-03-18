# main - CLI エントリポイント

## 概要

gkill アプリケーションのエントリポイントと CLI サブコマンド定義。
cobra ライブラリを使用したコマンドライン構造を持ち、2つのデプロイモードを提供する。

## ディレクトリ構造

```
main/
├── common/                    # 共有初期化ロジック
│   ├── common.go              # サーバ初期化・共通 cobra コマンド登録
│   ├── gkill_log/             # ログシステム
│   │   ├── gkill_log.go       # ログ初期化
│   │   ├── level.go           # ログレベル定義
│   │   ├── router.go          # ログルーティング
│   │   ├── routing_handler.go # ルーティングハンドラ
│   │   ├── sinks.go           # ログ出力先
│   │   └── switch_writer.go   # 出力先切り替え
│   ├── gkill_options/
│   │   └── option.go          # CLI オプション定義
│   └── threads/
│       └── threads.go         # スレッド管理
├── gkill/
│   └── main.go                # デスクトップアプリエントリポイント
├── gkill_server/
│   └── main.go                # HTTP サーバエントリポイント
├── gkill_fitbit_kc_convert_batch/
│   └── gkill_fitbit_kc_convert_batch.go  # Fitbit データ変換バッチ
└── old_data_transfomer_for_old_rykv/
    └── old_data_transfomer_for_old_rykv.go  # 旧 rykv データ移行
```

## デプロイモード

### `gkill_server/main.go` — HTTP サーバモード

ヘッドレスな HTTP サーバとして起動。ブラウザや PWA からアクセスする。
ビルド: `go install ./gkill/main/gkill_server/`

### `gkill/main.go` — デスクトップアプリモード

go-astilectron を使用して Electron ライクなデスクトップウィンドウ内で gkill_server を起動。
ビルド: `go install ./gkill/main/gkill/`

## cobra CLI サブコマンド

`common/common.go` で以下のサブコマンドがルートコマンドに登録される:

| サブコマンド | 説明 |
|-------------|------|
| `version` | バージョン情報表示 |
| `idf` | IDF（ファイル）関連操作 |
| `dvnf` | データバージョニング・ファイル管理 → `dvnf/README.md` 参照 |
| `generate_thumb_cache` | サムネイルキャッシュ生成 |
| `generate_video_cache` | 動画キャッシュ生成 |
| `optimize` | データベース最適化 |
| `update_cache` | キャッシュ更新 |

## common/ 配下

### `common.go`

サーバの初期化ロジック:
- cobra ルートコマンドの設定
- サブコマンドの登録
- GkillDAOManager の初期化
- HTTP サーバの起動

### `gkill_log/`（6ファイル）— ログシステム

| ファイル | 役割 |
|---------|------|
| `gkill_log.go` | ログシステムの初期化・グローバルロガー設定 |
| `level.go` | ログレベル定義（DEBUG, INFO, WARN, ERROR 等） |
| `router.go` | ログメッセージのルーティング |
| `routing_handler.go` | ルーティングハンドラ実装 |
| `sinks.go` | ログ出力先定義（ファイル、コンソール等） |
| `switch_writer.go` | 出力先の動的切り替え |

### `gkill_options/option.go` — CLI オプション

cobra のフラグ定義。サーバポート、データディレクトリ等のオプション。

### `threads/threads.go` — スレッド管理

goroutine の管理ユーティリティ。

## ユーティリティバッチ

### `gkill_fitbit_kc_convert_batch/`

Fitbit のヘルスデータ（歩数等）を gkill の KC（数値記録）形式に変換するバッチ処理。

### `old_data_transfomer_for_old_rykv/`

旧バージョンの rykv（gkill の前身アプリ）のデータを gkill 形式に移行するツール。
レガシー用途のため、新規開発での使用は想定されていない。

## 開発ガイドライン

### 新しいサブコマンドの追加

1. `common/common.go` の cobra ルートコマンドに `AddCommand()` でサブコマンドを登録
2. サブコマンドのロジックは別パッケージ（例: `dvnf/cmd/`）に分離
3. 共通の初期化ロジック（DB 接続等）は `common.go` の既存関数を再利用
