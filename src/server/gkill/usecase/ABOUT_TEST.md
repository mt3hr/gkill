# usecase テスト仕様

## 概要

`gkill/usecase/` パッケージのテスト仕様。HTTP 非依存のビジネスロジック層（17ファイル）。

## テストファイル

| ファイル | 内容 |
|---|---|
| `write_through_cache_test.go` | 書き込み後のキャッシュrepへの反映（`WriteThroughXxxCache`）と、個数判定の再発防止 |
| `cached_rep_insert_alignment_test.go` | キャッシュ実装の `INSERT` の**列並びと引数の並びが一致している**こと。ずれてもコンパイルは通り、SQLもエラーにならず、値が隣の列へ入るだけなので静かに壊れる |
| `source_conventions_scan_test.go` | 規約のソース走査（下記） |

## テスト状態

ビジネスロジックそのものは
`gkill/api/gkill_server_api/gkill_server_api_test.go` のハンドラ統合テストを通して
実行されており、**HTTPレイヤ込みで実際の経路が通っている**。
このパッケージに置いてあるのは、そのやり方では捕まえられない3種類だけ。

## 規約のソース走査（`source_conventions_scan_test.go`）

製品コードを1行も実行しない、**ソースの書き方を見張るだけ**のテスト群。
どれも「外れても `go build` も `go vet` も通り、実行時にエラーも出ずに静かに間違った結果を返す」
種類のズレで、ハンドラ統合テストでは再現できない。

| テスト | 見張っている規約 | 破ったときの症状 |
|-------|----------------|----------------|
| `TestWriteThroughCacheErrorIsNotDiscarded` | キャッシュ反映のエラーを `_ =` で捨てない | 反映が落ちても呼び出し元に何も伝わらず、次の `UpdateCache` まで古い応答が見える |
| `TestPeriodOfTimeWeekOfDaysHasNilGuard` | `PeriodOfTimeWeekOfDays` を nil ガード無しで `len()` しない | nil が「7曜日ではない」と判定され、許可曜日が1つも立たず**全件消える** |
| `TestNoUnreachablePackages` | どこからも import されないパッケージを残さない | バイト単位の複製が2重管理され、片方だけテストが薄いまま放置される |
| `TestAggregateFindChunksIDs` | 集約リポジトリの Find 入口が `findChunkedByIDs` を通る | IDリストがSQLのバインド上限(32766)を超えたとき、**GkillError が立たないまま「HTTP 200・errors:null・0件」**で返る |
| `TestNoRepNameFilterInDaoReps` | `dao/reps` で `FindQuery.Reps` による絞り込みをしない | ReKyou / MiReKyou のワード委譲が利用者のクエリをそのまま `FindKyousSequential` へ渡すので、チェックしていないrepに参照先を持つリポストが**黙って語句検索に当たらなくなる** |
| `TestCommitTxSetsRealRepNameBeforeWriteThrough` | `commit_tx` の13型すべてが、キャッシュへ書き戻す直前に実rep名を入れる | 一時リポジトリの合成名（`"KmemoTemp"` 等）がキャッシュへ入り、確定した記録が rep絞り込みから漏れて**一覧から丸ごと消える** |
| `TestCommitTxRestoresIDFTargetRepNameBeforeRealWrite` | IDF だけは `TargetRepName` を**実DBへ書く前に**戻す | leaf の `AddIDFKyouInfo` が `TARGET_REP_NAME` として永続化するので、合成名が入るとファイルの所在が実データごと壊れる。キャッシュではないので `UpdateCache` でも直らない |

> 13型・457メソッドのようにコピペで増える形は、**1つだけ抜けても他が緑のまま通る**。
> 型ごとに振る舞いのテストを書くより、書き方をソースで見張るほうが確実で速い。

## `cached_rep_insert_alignment_test.go`

`*_repository_cached_sqlite3_impl.go` を `go/parser` で読み、`INSERT INTO ... (列, 列, ...)` の
列並びと、対応する `queryArgs []any{...}` の並びを突き合わせる。
とくに `CreateUser` / `CreateDevice` / `CreateApp` / `UpdateUser` / `UpdateDevice` / `UpdateApp` の
6つは**どれも文字列で型が同じ**なので、入れ替わってもSQLは成功し、値だけが隣の列へ入る。
突き合わせられた `INSERT` が10件を下回ったらパーサがずれたとみなして落とす（空振りで緑になるのを防ぐ）。

### `write_through_cache_test.go` を例外として置いている理由

キャッシュrepへの書き戻しは、かつて
`len(repositories.TagReps) == 1 && *gkill_options.CacheTagReps` という
**要素数による代用判定**で守られていた。プラグインの型別アダプタは
キャッシュrepでの差し替えより後に `XxxReps` へ append されるので、
`provides` を持つプラグイン（Fitbitの `["kc","tag"]`）が1つ入るだけで
長さが2になり、判定が常に false になって書き戻しが丸ごと飛んでいた。
読み取りはキャッシュrepしか見ない（下層repへフォールバックしない）ため、
付けたタグが次の `UpdateCache` まで最大1分間見えず、
その間にPWAが古い応答をキャッシュし直すと恒久的に古いまま焼き付く。

ハンドラ統合テストの `setupTestRouterWithRepos` はプラグインを積まないので
`len(XxxReps)` が1のままになり、**この壊れ方を再現できない**。
そのため「アダプタ相当のrepを1つ足した `GkillRepositories`」を
このテストだけが自前で組み立てている。

`TestNoRepsCountCacheGuard` は `gkill/` 配下のソースを走査し、
個数判定が復活したら落とす。

デフォルトの `go test -cover` は「テスト対象パッケージ自身」しか計測しないため、
このパッケージ単体で測ると 0.0% と出るが、これは実測値ではない。
`-coverpkg` を付けて計測すると、主要な関数（Add/Update/Get系）は
**33〜50% のステートメントに到達している**（残りはDB障害時などのエラー分岐）。

```bash
cd src/server && go test ./gkill/api/gkill_server_api/... \
  -coverpkg=./gkill/usecase/... -coverprofile=/tmp/c.out
go tool cover -func=/tmp/c.out | grep usecase/
```

### 関数ごとの専用テストを置いていない理由

usecase の各関数はリポジトリ群（`*reps.GkillRepositories`）を引数に取るため、
単体テストを書くにも結局リポジトリ一式を組み立てることになる。
それは既にハンドラ統合テストの `setupTestRouterWithRepos` がやっていることで、
同じ準備を二重に持つと保守コストだけが増える。

そのため「usecase の振る舞いを確認したい」場合は
`gkill_server_api_test.go` にハンドラ経由のテストを足す方針を取っている。
例: MiReKyou（`usecase/mirekyou.go`）は
`TestHandleAddMiReKyou_AndGetMiReKyou` / `TestHandleUpdateMiReKyou` /
`TestHandleUpdateMiReKyou_Nonexistent_ReturnsError` /
`TestHandleGetKyous_MiReKyouResolvesTarget` /
`TestHandleGetMiBoardList_IncludesMiReKyouOnlyBoard` /
`TestHandleGetKyous_WordFilter_ReKyouTargetingMiReKyou` でカバーしている。

## 実行方法

```bash
cd src/server && go test ./gkill/usecase/...
```

または:

```bash
npm run test_server
```
