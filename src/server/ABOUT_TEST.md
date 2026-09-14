# server テスト仕様

## 概要

Go バックエンドのテスト。1254テスト関数、183テストファイル、32パッケージで構成される。API ハンドラ統合テスト、DAO 層テスト、プラグインのサブプロセス管理テスト、プラグインSDKテスト、KFTL パーサテスト、CLI テストを網羅する。

## テストフレームワーク

Go `testing` パッケージ（インメモリ SQLite3 使用）

## パッケージ構成

```
src/server/gkill/
├── api/                     # API ハンドラ層
│   ├── gkill_server_api/         # 統合テスト（全エンドポイント。35ファイル）
│   ├── find_filter_test.go       # 検索フィルタ
│   ├── gkill_sample_data_test.go # 配布サンプルデータの現行コード互換性
│   ├── find/                     # クエリビルダー
│   ├── gpslogs/                  # GPS ログ解析
│   ├── message/                  # メッセージフォーマット・HTTPステータス表
│   ├── kftl/                     # KFTL パーサ (6ファイル)
│   ├── req_res/                  # ワイヤ契約（JSONタグ名 / omitempty）
│   └── safefetch/                # SSRF 対策の共有フェッチ
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
│   └── reps/                     # リポジトリ実装 (61ファイル)
├── dvnf/                    # DVNF ファイル管理 (3ファイル)
└── main/                    # CLI エントリポイント (10ファイル)
    ├── common/                   # 共有ロジック・オプション・ログ・スレッド
    ├── gkill/                    # デスクトップアプリ
    ├── gkill_server/             # HTTP サーバ
```

## テスト内容

| カテゴリ | テストファイル数 | 内容 |
|---------|----------------|------|
| API 統合 | 38 | 全データ型 CRUD、セッション管理、複合クエリ、特殊エンドポイント、SSRF・レート制限、デバイス名キャッシュ、プラグイン本文HTMLキャッシュ、rep名絞り込み（キャッシュON/OFF・tx確定後）、タグ絞り込み、時間帯フィルタ、MCP用取得、壊れたrepのWeb/MCP警告、認証時rep取得失敗のHTTP 500とErrorログ、無認証経路のボディ上限・読み取り期限（wrapNoAuthCapped）、ルート表（apiRoutes）とハンドラ・doc コメント・認証区分の突き合わせ、URLog登録の取得抑止フラグ配線のソース走査 |
| 検索フィルタ・クエリ | 17 | `api/` 直下の検索フィルタ・タグ絞り込み・対象リポジトリ選択キャッシュ・rep名での結果絞り込み・並び替え・rep種別語彙の網羅 + `api/find/` のクエリビルダー・時間帯の秒解釈・検索語の正規化 + `api/find_word/` のワード判定（SQL とプラグイン SDK と揃える規則） |
| 埋め込み配信 | 1 | `embed.go` の `init()` が登録する `.webmanifest` のMIME型（登録を落とすと `http.FileServer` が中身を見て `text/plain` で配ってしまい、PWAのmanifestが仕様どおりのMIMEでなくなる。ビルドもvetも通るのでここでしか気付けない） |
| サンプルデータ | 1 | 配布サンプル `resources/gkill_sample_data` の現行コード互換性（アカウント認証・rep定義14件のパス実在と種別・全件検索で主要repから記録が返る・秘密鍵が空で配布されている） |
| KFTL パーサ | 6 | Factory、Statement、Request Map、MiReKyou、NLog、時刻 |
| req_res | 1 | ワイヤ契約の検証5本（JSONタグ名・MCP DTO の omitempty・プラグインpayload）。旧「JSON往復テスト」は削除済み |
| safefetch | 1 | SSRF 対策の共有フェッチ9本（接続直前の実IP検査・スキーム/サイズ上限・画像寸法・2xx判定） |
| メッセージ・GPS | 5 | メッセージコード体系、`EnsureNotEmpty`、エラーコード→HTTPステータス対応表、端末固有情報の伏せ処理、GPS ログ解析 |
| プラグインプロトコル | 1 | `gkill_plugin` の stdio メッセージ型 |
| ユースケース | 3 | write-through のキャッシュ反映、キャッシュ実装の INSERT 列と引数の対応、規約のソース走査（下記） |
| DAO 管理 | 5 | GkillDAOManager ライフサイクル、壊れた索引DBを持つrepの扱い（切り離し、書き込み先のfail-closed、`IsEnable`不変、失敗時のClose、rep名予測、個別/集約Errorログ、正常時の集約ログ抑止）、Git rep のディレクトリ限定 glob（非リポジトリの混入で全体を殺さない）、IDF rep だけの読み込み経路（ADR-0307）、リポジトリ定義のパターン展開（rep_file_glob） |
| アカウント | 5 | アカウント CRUD、セッション、アップロード履歴 |
| 設定 | 3 | サーバ設定、アプリ設定、リポジトリ定義 |
| 共有・通知 | 3 | 共有情報 CRUD、通知ターゲット |
| ユーティリティ | 9 | SQLite3 ユーティリティ、ファイル非表示 |
| リポジトリ | 63 | `reps/` 直下61（13データ型の SQLite3 実装 + キャッシュ + 一時 + プラグイン + プラグイン診断 + プラグインの GitCommitLog 型別アダプタ + デッドロック/ネスト並列回帰 + gitキャッシュUNIQUE + GPS集約dedup + 最新版アドレスのrep名走査 + URLog取得抑止フラグ + get_kyou の版履歴がキャッシュrepを回ること）+ `reps/cache/`・`reps/rep_cache_updater/` の2 |
| DVNF | 3 | ファイル管理、CLI コマンドの引数解析、copy/move 共用 `copyFile` の実ファイル操作（内容一致・copyLastMod の mtime 保存） |
| CLI/Main | 10 | 共有ロジック（`clear_cache` の各モード・サブコマンド登録を含む）、オプション、ログ、スレッド、エントリポイント、パスワード管理、add_tag バッチ（ルール JSON の検証・HTTP投稿と応答判定を含む） |
| プラグイン SDK | 5 | `Run()` の stdio ループ（18本）+ `EnsureConfig`（4本）+ ZIP走査（18本）+ キャッシュDBパス（5本）+ ワード判定 `Query.MatchText`（2本） |

**合計 183 ファイル**（上表の合計。`node src/tools/verify_docs.mjs --list` が出す `goTestFiles` と一致する。
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
| `TestWriteThroughSetsWriteRepName` | write-through の13型すべてが、キャッシュへ INSERT する直前に書き込み先repの実名を入れる（IDFKyou はリクエストが書き込み先を選ぶため除外。漏れた型は追加直後だけ rep絞り込みから消える） |

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
