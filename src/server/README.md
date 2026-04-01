# server - Go バックエンド

## 概要

gkill の Go バックエンドサーバ。HTTP API、データアクセス層（SQLite3）、CLI ツール群を提供する。
フロントエンドの Vue 3 SPA をバイナリに埋め込み、単一バイナリで配信可能。

## モジュール情報

- **Go モジュール**: `github.com/mt3hr/gkill/src/server`
- **Go バージョン**: 1.26.0
- **go.mod の場所**: `src/server/go.mod`（このディレクトリがモジュールルート）

## ディレクトリ構造

```
server/
├── go.mod                     # Go モジュール定義
├── go.sum                     # 依存関係チェックサム
└── gkill/
    ├── api/                   # HTTP API 共通基盤 → api/README.md
    │   ├── gkill_server_api/  # HTTP ハンドラ（79 handle_*.go + 認証・ルーティング）
    │   ├── embed/             # ビルド生成物（Vue SPA 埋め込み）
    │   ├── find/              # 検索クエリ型
    │   ├── gpslogs/           # GPS ログパーサ
    │   ├── kftl/              # KFTL パーサ → api/kftl/README.md
    │   ├── message/           # エラー/メッセージコード
    │   └── req_res/           # Request/Response 構造体 → api/req_res/README.md
    ├── usecase/               # ユースケース層（HTTP 非依存ビジネスロジック）→ usecase/README.md
    ├── dao/                   # データアクセス層 → dao/README.md
    │   ├── account/           # アカウント
    │   ├── account_state/     # セッション・履歴
    │   ├── gkill_notification/# 通知対象
    │   ├── hide_files/        # ファイル隠蔽
    │   ├── reps/              # メインリポジトリ → dao/reps/README.md
    │   ├── server_config/     # サーバ設定
    │   ├── share_kyou_info/   # 共有情報
    │   ├── sqlite3impl/       # SQLite3 ユーティリティ
    │   └── user_config/       # ユーザ設定
    ├── dvnf/                  # データバージョニング → dvnf/README.md
    └── main/                  # CLI エントリポイント → main/README.md
        ├── common/            # 共有初期化・ログ・オプション
        ├── gkill/             # デスクトップアプリ (go-astilectron)
        ├── gkill_server/      # HTTP サーバ
        ├── gkill_fitbit_kc_convert_batch/
        └── old_data_transfomer_for_old_rykv/
```

## 設計思想

### Repository パターン（4層実装）

各 Kyou データ型に対して一貫した4層構造:

| 層 | ファイル命名 | 役割 |
|---|---|---|
| インタフェース | `xxx_repository.go` | Go interface 定義 |
| SQLite3 実装 | `xxx_repository_sqlite3_impl.go` | DB 直接操作 |
| キャッシュ付き | `xxx_repository_cached_sqlite3_impl.go` | キャッシュラッパー |
| 一時リポジトリ | `xxx_temp_repository.go` / `xxx_repository_temp_sqlite3_impl.go` | トランザクション用 |

### Append-Only DAO

データ更新は既存レコードの上書きではなく、新レコードの追加で表現。
変更履歴が自然に保持される設計。

### 単一バイナリ配信

`api/embed.go` の `//go:embed` ディレクティブで Vue SPA をバイナリに埋め込み。
ビルドパイプライン: `npm run build` → `dist/` を `embed/html/` にコピー → `go install`

## 主要依存ライブラリ

| ライブラリ | 用途 |
|-----------|------|
| `gorilla/mux` | HTTP ルーティング |
| `modernc.org/sqlite` | SQLite3 ドライバ（pure Go） |
| `spf13/cobra` | CLI コマンド構造 |
| `spf13/viper` | 設定ファイル管理 |
| `asticode/go-astilectron` | デスクトップアプリラッパー |
| `go-git/go-git/v5` | Git コミットログ読み取り |
| `google/uuid` | UUID 生成 |
| `SherClockHolmes/webpush-go` | Web Push 通知 |
| `PuerkitoBio/goquery` | HTML パース（URLog サムネイル取得等） |
| `twpayne/go-gpx` | GPX ファイルパース |
| `rwcarlsen/goexif` | EXIF 情報読み取り |
| `fsnotify/fsnotify` | ファイル変更監視 |
| `nicksnyder/go-i18n/v2` | サーバ側 i18n |
| `axgle/mahonia` | 文字コード変換 |

## ビルド方法

```bash
# モジュールルートへ移動
cd src/server

# HTTP サーバのみビルド
go install ./gkill/main/gkill_server/

# デスクトップアプリビルド
go install ./gkill/main/gkill/

# 全パッケージビルド確認
go build ./...
```

**前提条件:**
- Go 1.26.0+
- フロントエンドビルド済み（`npm run build` → `embed/html/` にコピー済み）

## コード規約

### ファイル命名

- snake_case（例: `gkill_server_api.go`, `kmemo_repository_sqlite3_impl.go`）
- OS 別実装: `*_windows.go`, `*_other.go`
- テスト: `*_test.go`（`api/find_filter_test.go`, `api/gkill_server_api/` 内等）

### パッケージ構成

- `api/` — HTTP API 共通基盤（FindFilter、エラー/メッセージ型、リクエスト/レスポンス型、KFTL パーサ）と HTTP ハンドラ（`gkill_server_api/`）
- `usecase/` — HTTP 非依存のビジネスロジック。ハンドラから呼び出される共通処理（追加・更新・取得）
- `dao/` — データアクセス。SQL クエリとエンティティ定義
- `main/` — エントリポイントと CLI。初期化ロジック
- `dvnf/` — 独立したユーティリティ機能

### ハンドラパターン

全ハンドラは `gkill_server_api/` パッケージ内の `GkillServerAPI` 構造体のメソッドとして実装。
リクエストの JSON デコード → バリデーション → ユースケース層呼び出し → レスポンス返却の流れ。

## 関連ドキュメント

- [api/README.md](gkill/api/README.md) — HTTP API 共通基盤・ハンドラ層（全78エンドポイント一覧）
- [api/gkill_server_api/README.md](gkill/api/gkill_server_api/README.md) — HTTP ハンドラ（94ファイル）
- [api/kftl/README.md](gkill/api/kftl/README.md) — KFTL パーサ
- [api/req_res/README.md](gkill/api/req_res/README.md) — Request/Response 構造体
- [api/message/README.md](gkill/api/message/README.md) — エラーコード・メッセージコード
- [api/find/README.md](gkill/api/find/README.md) — 検索クエリ型定義
- [usecase/README.md](gkill/usecase/README.md) — ユースケース層（HTTP 非依存ビジネスロジック）
- [dao/README.md](gkill/dao/README.md) — データアクセス層
- [dao/reps/README.md](gkill/dao/reps/README.md) — メインリポジトリ
- [main/README.md](gkill/main/README.md) — CLI エントリポイント
- [main/common/README.md](gkill/main/common/README.md) — CLI 共通基盤（ログ、オプション、スレッド管理）
- [dvnf/README.md](gkill/dvnf/README.md) — データバージョニング
