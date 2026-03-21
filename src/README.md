# src - gkill ソースコード

## 概要

gkill はライフログアプリケーション。テキストメモ、ブックマーク、タスク、気分値、支出、タイムスタンプ、ファイル、Git コミットログなど、
日常のあらゆる記録を一元管理する。Go バックエンド + Vue 3 フロントエンドを1つのバイナリにコンパイルし、
クライアント・サーバ一体型で動作する。

## アーキテクチャ

### 2つのデプロイモード

| モード | バイナリ | 説明 |
|-------|---------|------|
| `gkill_server` | HTTP サーバ | ブラウザ / PWA からアクセス |
| `gkill` | デスクトップアプリ | go-astilectron でウィンドウ表示 |

### 技術構成

```
┌─────────────────────────────────────────┐
│  フロントエンド (Vue 3 + Vuetify 3 + TS) │  ← src/client/
│  ・PWA 対応                              │
│  ・vue-i18n (7言語)                      │  ← src/locales/
├─────────────────────────────────────────┤
│  //go:embed  (1バイナリに埋め込み)         │
├─────────────────────────────────────────┤
│  バックエンド (Go + gorilla/mux)         │  ← src/server/
│  ・76 POST API エンドポイント             │
│  ・SQLite3 (pure Go)                      │
│  ・cobra CLI                             │
├─────────────────────────────────────────┤
│  モバイル                                │
│  ・Android APK ラッパー                  │  ← src/android/
│  ・Wear OS (Pixel Watch) KFTL 入力       │  ← src/wear_os/
├─────────────────────────────────────────┤
│  AI 連携                                 │
│  ・MCP Server (読み取り専用)              │  ← src/mcp/
└─────────────────────────────────────────┘
```

## 中核概念「Kyou」

「Kyou（きょう）」は gkill の基本データ単位。全データ型は Kyou を基底として共通フィールド（ID, 作成日時, 更新日時等）を持つ。

### データ型一覧

| データ型 | 説明 | 例 |
|---------|------|---|
| **Kmemo** | テキストメモ | 「今日は天気が良かった」 |
| **KC** | 数値記録 | 体重: 65.5, 歩数: 8000 |
| **Lantana** | 気分値（0-10） | 気分: 7 |
| **Mi** | タスク | 「レポート提出」（チェック状態、期限付き） |
| **Nlog** | 支出記録 | 「コンビニ ¥500」 |
| **URLog** | ブックマーク | URL + タイトル + サムネイル |
| **TimeIs** | タイムスタンプ | 「作業中」（開始〜終了時刻） |
| **IDFKyou** | ファイル | 画像、動画、ドキュメント等 |
| **ReKyou** | リポスト | 既存 Kyou の再投稿 |
| **GitCommitLog** | Git コミットログ | リポジトリの自動収集 |
| **GPSLog** | GPS ログ | 位置情報記録 |

### メタ情報

各 Kyou には以下のメタ情報を添付可能:
- **Tag**: タグ付け
- **Text**: テキスト注釈
- **Notification**: 通知設定

## ディレクトリ構造

```
src/
├── client/          # Vue 3 フロントエンド (PWA)
│   ├── classes/     #   TypeScript ユーティリティ・Composable
│   │   ├── api/     #     GkillAPI クライアント
│   │   ├── datas/   #     データモデル
│   │   ├── dnote/   #     Dynamic Note (集計・フィルタ)
│   │   └── kftl/    #     KFTL パーサ
│   └── pages/       #   Vue ページ・ビュー・ダイアログ
│       ├── views/   #     View コンポーネント (172個)
│       └── dialogs/ #     Dialog コンポーネント (89個)
├── server/          # Go バックエンド
│   └── gkill/
│       ├── api/     #   HTTP API (76エンドポイント)
│       │   ├── kftl/    #   KFTL パーサ
│       │   └── req_res/ #   Request/Response 構造体 (155ファイル)
│       ├── dao/     #   データアクセス層 (SQLite3)
│       │   └── reps/#   メインリポジトリ (4層実装)
│       ├── dvnf/    #   データバージョニング
│       └── main/    #   CLI エントリポイント
├── android/         # Android APK ラッパー
├── wear_os/         # Wear OS (Pixel Watch) KFTL 入力アプリ
│   ├── phone_companion/  # スマホ側コンパニオン
│   └── watch_app/        # ウォッチ側アプリ
├── mcp/             # MCP Server (AI 連携)
├── locales/         # i18n 翻訳ファイル (7言語)
└── tools/           # ユーティリティスクリプト
```

## ビルドパイプライン

```bash
# フロントエンド開発サーバ
npm run dev

# フルビルド（フロントエンド → 埋め込み → Go install）
npm run install_server    # HTTP サーバ版
npm run install_app       # デスクトップアプリ版

# フロントエンドのみビルド
npm run build

# Go のみインストール（フロントエンド再ビルドなし）
npm run go_install
```

ビルドフロー:
1. `dist/` をクリーン
2. `version.json` 生成
3. `vite build`（型チェック + フロントエンドビルド）
4. `dist/` → `server/gkill/api/embed/html/` にコピー
5. `locales/` → `server/gkill/api/embed/` にコピー
6. `go install`

**前提条件:**
- Go 1.25.5+、Node.js 20.15.1+

## `tools/` ディレクトリ

| ファイル | 説明 |
|---------|------|
| `license_getter.ps1` | 依存ライブラリのライセンス情報を収集する PowerShell スクリプト |

## 開発ガイドライン

### 言語

コードベース（変数名、コメント、コミットメッセージ）は基本的に**日本語**。

### テスト

約1,400件の自動テストが存在する。詳細は [ABOUT_TEST.md](ABOUT_TEST.md) を参照。

| コマンド | 対象 |
|---------|------|
| `npm test` | 全テスト |
| `npm run test_server` | Go バックエンド (~453テスト) |
| `npm run test_client_unit` | フロントエンド ユニット (669テスト) |
| `npm run test_client_e2e` | フロントエンド E2E (49テスト) |
| `npm run test_mcp` | MCP サーバ (226テスト) |
| `npm run test_android` | Android |
| `npm run test_wear_os` | Wear OS (9テスト) |

### 設計パターン

- **バックエンド**: Repository パターン（4層: interface → SQLite3 → cached → temp）、Append-Only DAO
- **フロントエンド**: Vue 3 Composable パターン（`use-*.ts`）、Props/Emits 分離、Page → View → Dialog 階層

### データ型の追加手順

新しいデータ型を追加する場合、以下の全箇所に実装が必要:

1. **サーバ DAO**: `dao/reps/` に8ファイル（エンティティ + 4層リポジトリ）
2. **サーバ API**: `api/req_res/` にリクエスト/レスポンス、`gkill_server_api.go` にハンドラ
3. **クライアント データモデル**: `classes/datas/` にモデル
4. **クライアント API**: `classes/api/req_res/` に型、`gkill-api.ts` にメソッド
5. **クライアント UI**: `pages/views/` と `pages/dialogs/` にコンポーネント
6. **KFTL**: `api/kftl/` と `classes/kftl/` にパーサ（KFTL 対応する場合）

## 各サブディレクトリの README

| ディレクトリ | README |
|-------------|--------|
| `client/` | [client/README.md](client/README.md) |
| `client/classes/` | [client/classes/README.md](client/classes/README.md) |
| `client/classes/api/` | [client/classes/api/README.md](client/classes/api/README.md) |
| `client/classes/datas/` | [client/classes/datas/README.md](client/classes/datas/README.md) |
| `client/classes/dnote/` | [client/classes/dnote/README.md](client/classes/dnote/README.md) |
| `client/classes/kftl/` | [client/classes/kftl/README.md](client/classes/kftl/README.md) |
| `client/pages/` | [client/pages/README.md](client/pages/README.md) |
| `server/` | [server/README.md](server/README.md) |
| `server/gkill/api/` | [server/gkill/api/README.md](server/gkill/api/README.md) |
| `server/gkill/api/kftl/` | [server/gkill/api/kftl/README.md](server/gkill/api/kftl/README.md) |
| `server/gkill/api/req_res/` | [server/gkill/api/req_res/README.md](server/gkill/api/req_res/README.md) |
| `server/gkill/dao/` | [server/gkill/dao/README.md](server/gkill/dao/README.md) |
| `server/gkill/dao/reps/` | [server/gkill/dao/reps/README.md](server/gkill/dao/reps/README.md) |
| `server/gkill/main/` | [server/gkill/main/README.md](server/gkill/main/README.md) |
| `server/gkill/dvnf/` | [server/gkill/dvnf/README.md](server/gkill/dvnf/README.md) |
| `android/` | [android/README.md](android/README.md) |
| `wear_os/` | [wear_os/README.md](wear_os/README.md) |
| `locales/` | [locales/README.md](locales/README.md) |
| `mcp/` | [mcp/README.md](mcp/README.md)（既存） |

## 各サブディレクトリの ABOUT_TEST.md

テスト仕様書の索引: [ABOUT_TEST.md](ABOUT_TEST.md)

| ディレクトリ | テスト仕様 |
|-------------|-----------|
| `client/` | [client/ABOUT_TEST.md](client/ABOUT_TEST.md) |
| `client/classes/` | [client/classes/ABOUT_TEST.md](client/classes/ABOUT_TEST.md) |
| `client/classes/api/` | [client/classes/api/ABOUT_TEST.md](client/classes/api/ABOUT_TEST.md) |
| `client/classes/datas/` | [client/classes/datas/ABOUT_TEST.md](client/classes/datas/ABOUT_TEST.md) |
| `client/classes/dnote/` | [client/classes/dnote/ABOUT_TEST.md](client/classes/dnote/ABOUT_TEST.md) |
| `client/classes/kftl/` | [client/classes/kftl/ABOUT_TEST.md](client/classes/kftl/ABOUT_TEST.md) |
| `client/pages/` | [client/pages/ABOUT_TEST.md](client/pages/ABOUT_TEST.md) |
| `locales/` | [locales/ABOUT_TEST.md](locales/ABOUT_TEST.md) |
| `server/` | [server/ABOUT_TEST.md](server/ABOUT_TEST.md) |
| `server/gkill/api/` | [server/gkill/api/ABOUT_TEST.md](server/gkill/api/ABOUT_TEST.md) |
| `server/gkill/api/kftl/` | [server/gkill/api/kftl/ABOUT_TEST.md](server/gkill/api/kftl/ABOUT_TEST.md) |
| `server/gkill/api/req_res/` | [server/gkill/api/req_res/ABOUT_TEST.md](server/gkill/api/req_res/ABOUT_TEST.md) |
| `server/gkill/dao/` | [server/gkill/dao/ABOUT_TEST.md](server/gkill/dao/ABOUT_TEST.md) |
| `server/gkill/dao/reps/` | [server/gkill/dao/reps/ABOUT_TEST.md](server/gkill/dao/reps/ABOUT_TEST.md) |
| `server/gkill/dvnf/` | [server/gkill/dvnf/ABOUT_TEST.md](server/gkill/dvnf/ABOUT_TEST.md) |
| `server/gkill/main/` | [server/gkill/main/ABOUT_TEST.md](server/gkill/main/ABOUT_TEST.md) |
| `mcp/` | [mcp/ABOUT_TEST.md](mcp/ABOUT_TEST.md) |
| `android/` | [android/ABOUT_TEST.md](android/ABOUT_TEST.md) |
| `wear_os/` | [wear_os/ABOUT_TEST.md](wear_os/ABOUT_TEST.md) |
