# server テスト仕様

## 概要

Go バックエンドのテスト。約534テスト、47テストファイル、29パッケージで構成される。API ハンドラ統合テスト、DAO 層テスト、KFTL パーサテスト、CLI テストを網羅する。

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
│   └── req_res/                  # JSON 往復テスト
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
│   └── reps/                     # リポジトリ実装 (16ファイル)
├── dvnf/                    # DVNF ファイル管理 (2ファイル)
└── main/                    # CLI エントリポイント (8ファイル)
    ├── common/                   # 共有ロジック・オプション・ログ・スレッド
    ├── gkill/                    # デスクトップアプリ
    ├── gkill_server/             # HTTP サーバ
    ├── gkill_fitbit_kc_convert_batch/  # Fitbit 変換
    └── old_data_transfomer_for_old_rykv/ # 旧データ移行
```

## テスト内容

| カテゴリ | テストファイル数 | 内容 |
|---------|----------------|------|
| API 統合 | 5 | 全11データ型 CRUD、セッション管理、複合クエリ、特殊エンドポイント |
| KFTL パーサ | 3 | Factory、Statement、Request Map |
| req_res | 1 | 18種の JSON 往復テスト（IDFPayloadMCPDTO含む） |
| DAO 管理 | 1 | GkillDAOManager ライフサイクル |
| アカウント | 3 | アカウント CRUD、セッション、アップロード履歴 |
| 設定 | 3 | サーバ設定、アプリ設定、リポジトリ定義 |
| 共有・通知 | 2 | 共有情報 CRUD、通知ターゲット |
| ユーティリティ | 2 | SQLite3 ユーティリティ、ファイル非表示 |
| リポジトリ | 16 | 11データ型の SQLite3 実装 + キャッシュ + 一時 |
| DVNF | 2 | ファイル管理、CLI コマンド |
| CLI/Main | 8 | 共有ロジック、オプション、ログ、スレッド、エントリポイント、バッチ |

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
| `gkill/api/kftl/` | [gkill/api/kftl/ABOUT_TEST.md](gkill/api/kftl/ABOUT_TEST.md) |
| `gkill/api/req_res/` | [gkill/api/req_res/ABOUT_TEST.md](gkill/api/req_res/ABOUT_TEST.md) |
| `gkill/dao/` | [gkill/dao/ABOUT_TEST.md](gkill/dao/ABOUT_TEST.md) |
| `gkill/dao/reps/` | [gkill/dao/reps/ABOUT_TEST.md](gkill/dao/reps/ABOUT_TEST.md) |
| `gkill/dvnf/` | [gkill/dvnf/ABOUT_TEST.md](gkill/dvnf/ABOUT_TEST.md) |
| `gkill/main/` | [gkill/main/ABOUT_TEST.md](gkill/main/ABOUT_TEST.md) |
