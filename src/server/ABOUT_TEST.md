# server テスト仕様

## 概要

Go バックエンドのテスト。910テスト関数、125テストファイル、30パッケージで構成される。API ハンドラ統合テスト、DAO 層テスト、プラグインのサブプロセス管理テスト、プラグインSDKテスト、KFTL パーサテスト、CLI テストを網羅する。

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
├── usecase/                 # ユースケース層 (3ファイル。write-through 反映・INSERT列ずれ・規約のソース走査)
├── plugin/                  # プラグイン作者向け SDK
│   └── sdk/                      # Run() の stdio ループ + EnsureConfig + ZIP走査 + キャッシュDBパス (4ファイル / 45テスト)
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
│   └── reps/                     # リポジトリ実装 (43ファイル)
├── dvnf/                    # DVNF ファイル管理 (2ファイル)
└── main/                    # CLI エントリポイント (7ファイル)
    ├── common/                   # 共有ロジック・オプション・ログ・スレッド
    ├── gkill/                    # デスクトップアプリ
    ├── gkill_server/             # HTTP サーバ
```

## テスト内容

| カテゴリ | テストファイル数 | 内容 |
|---------|----------------|------|
| API 統合 | 18 | 全データ型 CRUD、セッション管理、複合クエリ、特殊エンドポイント、SSRF・レート制限、デバイス名キャッシュ、プラグイン本文HTMLキャッシュ、rep名絞り込み（キャッシュON/OFF・tx確定後）、タグ絞り込み、MCP用取得 |
| 検索フィルタ・クエリ | 14 | `api/` 直下の検索フィルタ・タグ絞り込み・対象リポジトリ選択キャッシュ・rep名での結果絞り込み・並び替え + `api/find/` のクエリビルダー |
| KFTL パーサ | 5 | Factory、Statement、Request Map、MiReKyou、時刻 |
| req_res | 1 | ワイヤ契約の検証5本（JSONタグ名・MCP DTO の omitempty・プラグインpayload）。旧「JSON往復テスト」は削除済み |
| メッセージ・GPS | 3 | メッセージコード体系、`EnsureNotEmpty`、GPS ログ解析 |
| プラグインプロトコル | 1 | `gkill_plugin` の stdio メッセージ型 |
| ユースケース | 3 | write-through のキャッシュ反映、キャッシュ実装の INSERT 列と引数の対応、規約のソース走査（下記） |
| DAO 管理 | 2 | GkillDAOManager ライフサイクル |
| アカウント | 5 | アカウント CRUD、セッション、アップロード履歴 |
| 設定 | 3 | サーバ設定、アプリ設定、リポジトリ定義 |
| 共有・通知 | 3 | 共有情報 CRUD、通知ターゲット |
| ユーティリティ | 8 | SQLite3 ユーティリティ、ファイル非表示 |
| リポジトリ | 45 | `reps/` 直下43（13データ型の SQLite3 実装 + キャッシュ + 一時 + プラグイン + デッドロック/ネスト並列回帰）+ `reps/cache/`・`reps/rep_cache_updater/` の2 |
| DVNF | 2 | ファイル管理、CLI コマンド |
| CLI/Main | 8 | 共有ロジック（`clear_cache` の各モード・サブコマンド登録を含む）、オプション、ログ、スレッド、エントリポイント、バッチ |
| プラグイン SDK | 4 | `Run()` の stdio ループ（18本）+ `EnsureConfig`（4本）+ ZIP走査（18本）+ キャッシュDBパス（5本） |

**合計 125 ファイル**（上表の合計。`node src/tools/verify_docs.mjs --list` が出す `goTestFiles` と一致する。
ずれたら `checkCounts` が落とす）。

## 規約のソース走査（`usecase/source_conventions_scan_test.go`）

製品コードを1行も実行しない、**ソースの書き方を見張るだけ**のテスト群。
どれも「外れても `go build` も `go vet` も通り、実行時にエラーも出ずに静かに間違った結果を返す」
種類のズレなので、機械検査でしか気付けない。

| テスト | 見張っている規約 |
|-------|----------------|
| `TestWriteThroughCacheErrorIsNotDiscarded` | キャッシュ反映のエラーを `_ =` で捨てない |
| `TestPeriodOfTimeWeekOfDaysHasNilGuard` | `PeriodOfTimeWeekOfDays` を nil ガード無しで `len()` しない（nil を「7曜日ではない」と判定すると全件消える） |
| `TestNoUnreachablePackages` | どこからも import されないパッケージを残さない（バイト単位の複製が2重管理されていた） |
| `TestAggregateFindChunksIDs` | 集約リポジトリの Find 入口が `findChunkedByIDs` を通る（IDリストがSQLのバインド上限を超えると**エラーが立たないまま0件**になる） |
| `TestNoRepNameFilterInDaoReps` | `dao/reps` で `FindQuery.Reps` による絞り込みをしない（ReKyou/MiReKyou のワード委譲が利用者のクエリをそのまま渡すので、ここで絞ると未チェックrepに参照先を持つリポストが語句検索に当たらなくなる） |
| `TestCommitTxSetsRealRepNameBeforeWriteThrough` | `commit_tx` の13型すべてが、キャッシュへ書き戻す直前に実rep名を入れる（一時リポジトリの合成名が入ると、確定した記録が rep絞り込みから漏れる） |
| `TestCommitTxRestoresIDFTargetRepNameBeforeRealWrite` | IDF だけは `TargetRepName` を**実DBへ書く前に**戻す（合成名が入るとファイルの所在が実データごと壊れ、`UpdateCache` でも直らない） |

## 実行方法

```bash
npm run test_server
```

手動実行:
```bash
cd src/server && go test ./...
```

ベンチマーク（`go test` の既定では走らない）:
```bash
cd src/server && go test -run '^$' -bench . ./gkill/api/
```
`api/` の `BenchmarkInclusiveTimeIntervalIndex` だけが対象。
TimeIs・位置情報の区間判定を総当たりから二分探索へ変えたときの効果を測るためのもので、
実装を戻したくなったときに数字で比べられるように残してある。

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
