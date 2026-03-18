# dvnf - データバージョニング・ファイル管理

## 概要

DVNF（Data Versioning and Notification Framework）は、ファイルのバージョン管理とコピー・移動を行うユーティリティ。
gkill の CLI サブコマンドとして提供され、`gkill dvnf` コマンドで呼び出される。

## ディレクトリ構造

```
dvnf/
├── dvnf.go          # コアロジック（バージョニング・ファイル操作）
└── cmd/             # cobra CLIサブコマンド
    ├── dvnf_cmd.go  # dvnf ルートコマンド定義
    ├── copy.go      # dvnf copy サブコマンド
    ├── get.go       # dvnf get サブコマンド
    └── move.go      # dvnf move サブコマンド
```

## ファイル一覧（5ファイル）

| ファイル | 役割 |
|---------|------|
| `dvnf.go` | DVNF のコアロジック。ファイルのバージョン管理、コピー・移動の実装 |
| `cmd/dvnf_cmd.go` | cobra による `dvnf` ルートコマンドの定義。サブコマンドの登録 |
| `cmd/copy.go` | `dvnf copy` サブコマンド — ファイルのバージョン付きコピー |
| `cmd/get.go` | `dvnf get` サブコマンド — バージョン情報の取得 |
| `cmd/move.go` | `dvnf move` サブコマンド — ファイルのバージョン付き移動 |

## CLI コマンド

```
gkill dvnf copy <src> <dst>   # バージョン付きファイルコピー
gkill dvnf get <path>         # ファイルのバージョン情報取得
gkill dvnf move <src> <dst>   # バージョン付きファイル移動
```

## 設計思想

- cobra のサブコマンドパターンに従い、`cmd/` 配下にコマンド定義を配置
- コアロジック（`dvnf.go`）と CLI 定義（`cmd/`）を分離
- `src/server/gkill/main/common/` から cobra ルートコマンドに登録される
