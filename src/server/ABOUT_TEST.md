# server テスト仕様

## 概要

Go バックエンドのテスト。794テスト関数、95テストファイル、30パッケージで構成される。API ハンドラ統合テスト、DAO 層テスト、プラグインのサブプロセス管理テスト、プラグインSDKテスト、KFTL パーサテスト、CLI テストを網羅する。

## テストフレームワーク

Go `testing` パッケージ（インメモリ SQLite3 使用）

## パッケージ構成

```
src/server/gkill/
├── api/                     # API ハンドラ層
│   ├── gkill_server_api_test.go  # 統合テスト（全エンドポイント）
│   ├── find_filter_test.go       # 検索フィルタ
│   ├── find/                     # クエリビルダー
│   ├── gpslogs/                  # GPS ログ解析
│   ├── message/                  # メッセージフォーマット
│   ├── kftl/                     # KFTL パーサ (3ファイル)
│   └── req_res/                  # ワイヤ契約（JSONタグ名 / omitempty）
├── usecase/                 # ユースケース層（専用テストは持たず、gkill_server_api の統合テストでカバー）
├── plugin/                  # プラグイン作者向け SDK
│   └── sdk/                      # Run() の stdio ループ + EnsureConfig (2ファイル / 18テスト)
├── dao/                     # データアクセス層
│   ├── gkill_dao_manager_test.go # DAO マネージャ
│   ├── account/                  # アカウント管理
│   ├── account_state/            # セッション・アップロード履歴
│   ├── server_config/            # サーバ設定
│   ├── user_config/              # ユーザ設定・リポジトリ定義
│   ├── share_kyou_info/          # 共有設定
│   ├── gkill_notification/       # 通知ターゲット
│   ├── hide_files/               # ファイル非表示
│   ├── sqlite3impl/              # SQLite3 ユーティリティ
│   └── reps/                     # リポジトリ実装 (34ファイル)
├── dvnf/                    # DVNF ファイル管理 (2ファイル)
└── main/                    # CLI エントリポイント (7ファイル)
    ├── common/                   # 共有ロジック・オプション・ログ・スレッド
    ├── gkill/                    # デスクトップアプリ
    ├── gkill_server/             # HTTP サーバ
    └── gkill_fitbit_kc_convert_batch/  # Fitbit 変換
```

## テスト内容

| カテゴリ | テストファイル数 | 内容 |
|---------|----------------|------|
| API 統合 | 10 | 全データ型 CRUD、セッション管理、複合クエリ、特殊エンドポイント、SSRF・レート制限、デバイス名キャッシュ、プラグイン本文HTMLキャッシュ |
| 検索フィルタ・クエリ | 4 | `api/` 直下の検索フィルタ・タグ絞り込み・対象リポジトリ選択キャッシュ + `api/find/` のクエリビルダー |
| KFTL パーサ | 3 | Factory、Statement、Request Map |
| req_res | 1 | ワイヤ契約の検証5本（JSONタグ名・MCP DTO の omitempty・プラグインpayload）。旧「JSON往復テスト」は削除済み |
| メッセージ・GPS | 2 | メッセージフォーマット、GPS ログ解析 |
| DAO 管理 | 1 | GkillDAOManager ライフサイクル |
| アカウント | 5 | アカウント CRUD、セッション、アップロード履歴 |
| 設定 | 3 | サーバ設定、アプリ設定、リポジトリ定義 |
| 共有・通知 | 2 | 共有情報 CRUD、通知ターゲット |
| ユーティリティ | 5 | SQLite3 ユーティリティ、ファイル非表示 |
| リポジトリ | 31 | `reps/` 直下28（13データ型の SQLite3 実装 + キャッシュ + 一時 + プラグイン + デッドロック/ネスト並列回帰）+ `reps/cache/`・`reps/cache/rep_cache_updater/`・`reps/rep_cache_updater/` の3 |
| DVNF | 2 | ファイル管理、CLI コマンド |
| CLI/Main | 7 | 共有ロジック（`clear_cache` の各モード・サブコマンド登録を含む）、オプション、ログ、スレッド、エントリポイント、バッチ |
| プラグイン SDK | 2 | `Run()` の stdio ループ（14本）+ `EnsureConfig`（4本） |

**合計 78 ファイル**（上表の合計。`node src/tools/verify_docs.mjs --list` が出す Go テストファイル数と一致する）。

## 実行方法

```bash
npm run test_server
```

手動実行:
```bash
cd src/server && go test ./...
```

## 関連ドキュメント

| サブディレクトリ | テスト仕様 |
|----------------|-----------|
| `gkill/api/` | [gkill/api/ABOUT_TEST.md](gkill/api/ABOUT_TEST.md) |
| `gkill/api/gkill_server_api/` | [gkill/api/gkill_server_api/ABOUT_TEST.md](gkill/api/gkill_server_api/ABOUT_TEST.md) |
| `gkill/api/kftl/` | [gkill/api/kftl/ABOUT_TEST.md](gkill/api/kftl/ABOUT_TEST.md) |
| `gkill/api/req_res/` | [gkill/api/req_res/ABOUT_TEST.md](gkill/api/req_res/ABOUT_TEST.md) |
| `gkill/dao/` | [gkill/dao/ABOUT_TEST.md](gkill/dao/ABOUT_TEST.md) |
| `gkill/dao/reps/` | [gkill/dao/reps/ABOUT_TEST.md](gkill/dao/reps/ABOUT_TEST.md) |
| `gkill/usecase/` | [gkill/usecase/ABOUT_TEST.md](gkill/usecase/ABOUT_TEST.md) |
| `gkill/plugin/sdk/` | [gkill/plugin/sdk/ABOUT_TEST.md](gkill/plugin/sdk/ABOUT_TEST.md) |
| `gkill/dvnf/` | [gkill/dvnf/ABOUT_TEST.md](gkill/dvnf/ABOUT_TEST.md) |
| `gkill/main/` | [gkill/main/ABOUT_TEST.md](gkill/main/ABOUT_TEST.md) |
