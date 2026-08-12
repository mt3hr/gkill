# usecase テスト仕様

## 概要

`gkill/usecase/` パッケージのテスト仕様。HTTP 非依存のビジネスロジック層（17ファイル）。

## テストファイル

| ファイル | 内容 |
|---|---|
| `write_through_cache_test.go` | 書き込み後のキャッシュrepへの反映（`WriteThroughXxxCache`）と、個数判定の再発防止 |

## テスト状態

上記1本を除き、usecase パッケージに専用のテストファイルは無い。ビジネスロジックは
`gkill/api/gkill_server_api/gkill_server_api_test.go` のハンドラ統合テストを通して
実行されており、**HTTPレイヤ込みで実際の経路が通っている**。

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

### 専用テストを置いていない理由

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
