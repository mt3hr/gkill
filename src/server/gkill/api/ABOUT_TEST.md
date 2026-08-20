# api テスト仕様

## 概要

Go バックエンドの API 共通基盤層のテスト。検索フィルタ、サブパッケージ（find, gpslogs, message, kftl, req_res）のテストを含む。HTTP API ハンドラのテストは `gkill_server_api/` サブパッケージに移動済み。

## テストフレームワーク

Go `testing` パッケージ

## テストファイル一覧

### api パッケージ直下テスト

| ファイル | テスト内容 |
|---------|-----------|
| `find_filter_test.go` | 検索フィルタのテスト（最新版への差し替え、リネーム済みタグの除外、並行取得のエラー回収、曜日フィルタの nil ガード） |
| `find_filter_helpers_test.go` | 複数リポジトリからの収集ヘルパ。1つが失敗しても他のエラーを取りこぼさず `errors.Join` で束ねること |
| `find_filter_location_test.go` | 地図検索（同一座標での NaN 回避、区間の端点、`map_radius <= 0` の素通し、3値そろわないときの不成立） |
| `find_filter_mi_test.go` | Mi 抽出（チェック状態が空文字・未知値のときの既定、`_create` へのフォールバック、同着のタイブレーク） |
| `find_filter_sort_trim_test.go` | 並べ替えと重複排除。同一IDでも版・データ型・関連日時が違えば別entryとして扱うこと |
| `find_filter_tags_test.go` | タグ絞り込みの意味論（AND/OR、存在しないタグ名、大小無視、「タグなし」の扱い、空指定） |
| `find_filter_timeis_test.go` | 打刻タグでの絞り込みと、打刻検索が最新版だけを見ること |
| `filter_tags_kyous_test.go` | タグ AND 検索を map 参照へ置き換えた性能改修の結果不変（`find_filter_tags_test.go` と同じ関数を別観点で検査） |
| `select_match_reps_cache_test.go` | 検索対象リポジトリの選定。種別フィルタとの和集合、非nil空指定（＝候補0件）の扱い、そして**rep名を指定してもキャッシュrepを剥がさない**こと（`UnWrap()` は枝刈り判定にだけ使う。剥がすとGUIの検索が毎回キャッシュをバイパスする）。配下の一部だけが指定された部分一致も見る |
| `find_kyou_rep_name_filter_test.go` | rep名での**結果側**の絞り込み（`filterKyousByRepName`）。指定repだけ残る／全部落ちたIDはキーごと消える（空スライスを残すと後段が `kyous[0]` で panic）／`Reps == nil` は未指定／`RepName` が空の行は残す（追加直後の行がこれ）／本文ヒット由来の2本目の検索にも効く |
| `sort_result_kyous_test.go` | 検索結果の並べ替え |
| `find_filter_pipeline_bench_test.go` | 検索パイプラインのベンチマーク（`go test` の既定では走らない） |

#### `find_filter_test.go` の内容

- **最新版への差し替え** (`TestReplaceLatestKyouInfos_ExcludeStaleKeepLatest`):
  グローバル最新でない版を除外し、最新版のentryだけ残すこと。
  最新版アドレス表に載らない種別（プラグイン・git・GPS 由来）は除外せず素通しすること。
  `DisableLatestDataRepositoryCache` では分岐しないので、キャッシュ設定は1通りだけ流す
- **リネーム済みタグの除外** (`TestFindTags_ExcludeRenamedAwayVersion`):
  TAGテーブルはappend-onlyなので旧名の版が残る。旧名で検索してもヒットしないこと
- **並行取得のエラー回収** (`TestDrainFindErrors_*`):
  `FindKyous` はタグ取得・非表示タグ・タグ検索・テキスト検索・TimeIsテキスト・
  TimeIsタグの6経路をgoroutineで並行実行する。`drainFindErrors` が
  goroutineの完了を待ってからエラーを回収すること、エラーが無ければ nil を返すこと。
  以前は待ち合わせより前に吸い出していたため6経路のエラーが常に捨てられ、
  検索が成功扱いで不完全な結果を返していた。待ち合わせを関数の内側に置いたので、
  呼び出し順を誤っても再発しない

### サブパッケージテスト

| ファイル | テスト内容 |
|---------|-----------|
| `find/find_query_test.go` | FindQuery ビルダー（ゲートヘルパ、`null` とキー欠落と `[]` の復元差、MiCheckState/MiSortType enum の JSON 往復、nil日付、データ型フィルタ） |
| `find/find_query_legacy_json_test.go` | 旧形式（`use_*` フラグ入り）JSON の新形式への移行。無効化・有効化・打刻グループの従属、ネスト構造、冪等性、数値精度、壊れたJSON |
| `gpslogs/gpslogs_test.go` | GPS ログファイル解析 |
| `message/message_test.go` | GkillMessage / GkillError フォーマット |
| `kftl/*_test.go` | メモ帳構文のパーサ。詳細は [kftl/ABOUT_TEST.md](kftl/ABOUT_TEST.md) |
| `req_res/req_res_test.go` | 入出力構造体。詳細は [req_res/ABOUT_TEST.md](req_res/ABOUT_TEST.md) |

## 実行方法

```bash
cd src/server && go test ./gkill/api/...
```

または:

```bash
npm run test_server
```

## 関連ドキュメント

| サブディレクトリ | テスト仕様 |
|----------------|-----------|
| `gkill_server_api/` | [gkill_server_api/ABOUT_TEST.md](gkill_server_api/ABOUT_TEST.md) |
| `kftl/` | [kftl/ABOUT_TEST.md](kftl/ABOUT_TEST.md) |
| `req_res/` | [req_res/ABOUT_TEST.md](req_res/ABOUT_TEST.md) |
