# ADR-0042: 共有ページのファイル配信は共有クエリを再評価した許可パス集合にだけ許す

| | |
|---|---|
| Status | Accepted |
| Date | 2026-08-21 |
| Sources | `bb364253`（監査 C-03） / `CLAUDE.md`「共有ページのファイル配信は共有クエリの結果に含まれるファイルだけ」節 |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/api/gkill_server_api/shared_file_authz.go` |

## Context

共有ページのファイル配信（`handle_file_serve.go` の `sharedID != ""` 経路）は、**rep名が一致すれば IDF rep へ委譲していた**。

共有URLが見せるのは「その共有クエリの検索結果」だけのはずなのに、**結果の外にあるファイルまで取れていた**。同じ rep に入っていれば、共有していないファイルの相対パスを指定するだけで取得できる。

## Decision

`sharedID != ""` の経路では、**共有 `FindQuery` を再評価して許可パス集合を作り**（`shared_file_authz.go` の `collectSharedIDFFilePaths`）、要求パスが集合に無ければ403にする。

判定手順は `handle_get_shared_kyous.go` と**同一のものを共有する**。

## Rejected alternatives

- **rep名の一致だけで許す（改修前）** — 共有の単位は「クエリの結果」であって「rep」ではない。rep 単位で許すと、共有した1件と同じ rep にある全ファイルが取れる。

- **許可パスの一覧を共有作成時に固定して保存する** — 共有クエリは期間や条件で書かれるので、**あとから追加された記録も共有に含まれる**のが仕様。固定すると共有の意味が変わる。

- **判定手順を配信側で書き直す** — 一覧を2箇所で維持することになり、片方だけ直したときに**配信側だけ緩い**状態が生まれる。`handle_get_shared_kyous` と同じ手順を共有する。

- **セッション経路にも同じゲートをかける** — 不要。セッションを持つ利用者は自分の全記録にアクセスできるのが仕様。ゲートは `sharedKyouInfo != nil` のときだけ。

## Consequences

パス正規化は `idf_file_url.go` の `cleanRelativeURLPath` と**同一**でなければならない。ずれると、正規化前と後で別のパスとして扱われて**ゲートをすり抜ける**。

共有クエリの再評価が配信のたびに走るので、共有ファイルの取得は素の配信より重い。頻度が低い経路なので許容する。

## Evidence

実測なし — 脅威モデルからの判断（外部監査 C-03 の指摘）。

## Related tests

- `src/server/gkill/api/gkill_server_api/shared_file_authz_test.go`
- `src/server/gkill/api/gkill_server_api/handle_get_shared_kyous_test.go`
