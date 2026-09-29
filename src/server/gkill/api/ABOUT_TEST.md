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
| `select_match_reps_cache_test.go` | 検索対象リポジトリの選定。種別フィルタとの和集合、非nil空指定（＝候補0件）の扱い、そして**rep名を指定してもキャッシュrepを剥がさない**こと（`UnWrap()` は枝刈り判定にだけ使う。剥がすとGUIの検索が毎回キャッシュをバイパスする）。配下の一部だけが指定された部分一致も見る。リーフが `RepNamesProvider`（複数の rep 名を名乗るプラグイン）なら申告名のどれか1つが `Reps` にあれば選ばれ、manifest の代表名では選ばれないこと |
| `find_kyou_rep_name_filter_test.go` | rep名での**結果側**の絞り込み（`filterKyousByRepName`）。指定repだけ残る／全部落ちたIDはキーごと消える（空スライスを残すと後段が `kyous[0]` で panic）／`Reps == nil` は未指定／`RepName` が空の行は残す（追加直後の行がこれ）／本文ヒット由来の2本目の検索にも効く |
| `sort_result_kyous_test.go` | 検索結果の並べ替え |
| `find_filter_pipeline_bench_test.go` | 検索パイプラインのベンチマーク（`go test` の既定では走らない） |
| `gkill_sample_data_test.go` | 配布サンプル `resources/gkill_sample_data` が現行コードで動くこと（account.db のスキーマと Argon2id 認証、REPOSITORY 14件の `$GKILL_HOME` 展開後パス実在と種別が既知集合に含まれること、`FindKyous` で主要repから記録が返ること、Web Push 鍵が空で配布されていること）。**コミット済みDBを直接開かず、必ずテンポラリへコピーしてから検証する**（DAOは開くだけでスキーマ移行・IDF走査によりDBを変異させる） |
| `embed_mime_test.go` | `embed.go` の `init()` が登録する MIME 型の固定。登録行を落としても build / vet は通り、「manifest が text/plain で配られる」形で静かに壊れる |
| `rep_types_coverage_test.go` | rep_types の正準語彙（`find.KyouRepTypes`）と実装の switch 群の対応。どちらかに値を足し忘れると「その rep 種別だけ検索から静かに消える」 |

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
| `find/period_of_time_test.go` | 時間帯フィルタの秒の二重解釈の境界（0..86399 は秒オブデイ、86400以上は絶対epoch秒としてローカル時分秒へ変換） |
| `gpslogs/gpslogs_test.go` | GPS ログファイル解析 |
| `message/message_test.go` | GkillMessage / GkillError フォーマット |
| `message/gkill_error_test.go` | `EnsureNotEmpty`（失敗したのに GkillError が1つも無い応答を潰す。内部 error だけで返すと HTTP 200 + 0件になり「成功・該当0件」と見分けが付かない）と、ワイヤの形: 成功時の `errors` / `messages` が `null` でなく `[]`、`error_kind` / `reason` の付与（分類できないときは `reason` を省く）、`Cause` の端末固有情報が `MarshalJSON` で伏せられること、`warning` レベルが残ること |
| `message/error_kind_test.go` | エラーコード→`error_kind`（クライアントがヒント文を引く鍵）。全コードに kind が決まり語彙が `errorKinds` に閉じていること、上書き表のコードが実在すること、既知の割り当てと分布 |
| `message/error_reason_test.go` | Go の error→`reason` の分類（`ReasonOf`）。modernc.org/sqlite が実際に返す CANTOPEN / NOTADB / READONLY / BUSY で確認、分類できないときのエラーコードからの既定、語彙 `reasonTokens` |
| `message/http_status_test.go` | エラーコード→HTTP ステータスの対応表。全コードが分類されていること（漏れると `HTTPStatusOf` が 0 を返して 500 扱いになるだけで、目の前ではエラーにならない）、既知の割り当てと分布、`HTTPStatusForErrors`（errors 配列からの決定） |
| `message/redact_test.go` | `RedactEnvironmentSpecific` が端末のローカル絶対パス・メールアドレスを伏せること、何度かけても同じ結果になること（伏せ損なうとプラグインの診断文に載ったパスが API 経由で AI へ届き、資料へ引き写される。ADR-0707） |
| `safefetch/safefetch_test.go` | SSRF 対策の共有フェッチ。接続直前の実 IP 検査（ループバック・私有アドレスは既定で拒否、許可フラグで通す）、スキームとサイズ上限、2xx 以外の拒否、画像の寸法と種別の判定 |
| `gkill_plugin/plugin_manifest_test.go` | manifest の `emits_kyou` の既定。書いていない既存プラグインは従来どおり Kyou を返す扱いであること（false 側に倒れると manifest を書き換えていない全プラグインが記録保管場所の一覧から消える）、未設定なら JSON に出さないこと |
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
