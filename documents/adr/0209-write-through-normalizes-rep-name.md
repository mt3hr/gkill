# ADR-0209: キャッシュへ書き戻す rep 名はクライアントの値を信用せず書き込み側で正規化する

| | |
|---|---|
| Status | Accepted |
| Date | 2026-08-24 |
| Sources | `7e1f8023` / `.claude/skills/gkill-go-backend/SKILL.md`「`len(XxxReps) == 1` でキャッシュrepを判定してはいけない」節 |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/dao/reps/gkill_repositories.go`（`writeRepNameOrEmpty`） |

## Context

gkill の Kyou は2層に書かれる。実体は leaf rep（`XxxRepositorySQLite3Impl`）へ、
検索用の写しはキャッシュ rep（`XxxRepositoryCachedSQLite3Impl`）へ入る。

この2層で `RepName` の扱いが非対称になっている。

- leaf rep のテーブルに **REP_NAME 列は無い**。読むときに Go 側が
  `kyou.RepName = repName` を注入するので、書き込み時に何が入っていようと結果は常に正しい
- キャッシュ rep は **REP_NAME 列を持ち**、`AddXxxInfo` に渡された `RepName` を
  そのまま INSERT し、SELECT でそのまま返す

`--cache_in_memory`（既定 true）では検索がキャッシュ rep しか見ないため、
キャッシュへ書いた値がそのまま利用者の目に触れる。

一方 `RepName` の値は呼び出し側から来る。MCP の追加経路は空文字を送り
（クライアントは書き込み先 rep を知りようがない）、MCP の更新経路は
`/api/get_kmemo` で読んだ現在値をそのままエコーするので **取得元 rep** の名前を送る。
端末ごとに rep を分けている環境では、取得元と書き込み先が食い違う。

この決定が無いと、追加した記録の `rep_name` が空のまま、あるいは別 rep の名前のまま
検索結果に出る。`filterKyousByRepName` は空を残すので記録自体は消えないが、
`query.reps` での絞り込みが効かず、`group_by:"rep_name"` は集計にならない。
更新の場合は「非空だが指定 rep に無い名前」になるので**更新直後だけ一覧から消える**。
いずれも次の `UpdateCache`（既定1分）で直るため、再現しようとすると消えている。

## Decision

`GkillRepositories.WriteThroughXxxCache` の 13 メソッドが、キャッシュ rep へ渡す直前に
`RepName` を書き込み先 rep の実名（`g.WriteXxxRep.GetRepName`）で上書きする。
取得に失敗したときは空文字にする。`WriteThroughIDFKyouCache` だけは例外とする。

## Rejected alternatives

- **呼び出し側（usecase の `Add*` / `Update*`）へ同じ5行を貼る** — `Update*` 側12ファイルには
  既にこの形があり、`Add*` 側だけが欠けていた。対称にコピーすれば直るが、
  60行の重複コメントが増えるうえ **`api/kftl/` の11箇所と bookmarklet の1箇所は依然として空のまま残る**。
  write-through 呼び出しは非テストだけで54箇所あり、そのすべてが必ず通る1点で直すほうが漏れない
- **MCP 側が正しい rep 名を送る** — クライアントは書き込み先 rep を推測できない
  （サーバの `USE_TO_WRITE` 設定で決まる）。推測を送らせると「実在しない rep 名が
  キャッシュへ入り、記録が黙って消える」という重い方の壊れ方に化ける。
  `find_filter.go` の `filterKyousByRepName` に「実在しない rep 名を書き込み側に入れさせるな」
  という明示的な禁止コメントがあるのはこのため
- **`filterKyousByRepName` に例外を足して食い違いを吸収する** — 検索側で辻褄を合わせると、
  今度は「本当に別 rep にある記録」まで拾ってしまう。壊れているのは書き込み側なので書き込み側で直す
- **`GetRepName` が失敗したらエラーにする** — キャッシュへの書き戻しは
  「失敗しても次の UpdateCache で直る」補助経路で、ここで追加そのものを失敗させる価値は無い。
  空文字は `filterKyousByRepName` が残すので安全側に倒れる

## Consequences

- **`WriteThroughIDFKyouCache` だけは上書きしてはいけない。** `handle_upload_files.go` が
  `request.TargetRepName` でリクエストごとに書き込み先 rep を選ぶため、
  `g.WriteIDFKyouRep` とは別の rep になりうる。呼び出し側が既に正しい値を入れている。
  doc コメントに理由を書き、走査テストの allowlist にも理由つきで載せてある
- `api/kftl/` の11箇所が `""` → 実 rep 名になる。正しい方向だが可視の挙動変化で、
  今日は「rep 絞り込みで外した rep に KFTL 保存しても一覧に出る」、以後は「出ない」。
  更新側は元からこの挙動なので整合する
- **壊れ方は静か。** 1型でも抜けると、その型だけが「追加直後の1分間だけ rep 絞り込みから漏れる」
  という時限付きの状態になり、他の12型のテストは緑のまま通る
- **キャッシュ OFF では再現しない。** leaf rep が読むときに自分の名前を注入するため、
  この回帰はキャッシュ ON でしか観測できない。テストを既定の OFF だけで走らせていると
  「rep_name は常時付与」というアサーションを持ちながら見逃す（実際に見逃していた）

## Evidence

2026-08-23 の readwrite MCP 監査（実測）:

- 8月の全記録で `group_by:"rep_name"` を撃つと `[{"(unknown)":40},{"Files":2},{"Mi":1}]`。
  43件中40件が `(unknown)`。rep_name が入っていた唯一の Mi は、その監査中に update を通した1件だけ
- `gkill_get_kyous` の結果で kmemo / nlog / kc / lantana / urlog / timeis のすべてが `rep_name: ""`
- 同じ id を update 経由で読み直すと `rep_name: "Mi"` になる（Update 側には正規化があるため）

修正の検証（実測）: `TestHandleGetKyousMCP_IDAndRepNameAlwaysPresent` を
キャッシュ ON/OFF の両方で走らせると、修正前は `cacheInMemory=true` 側だけが
`data_type "kmemo" のrep_nameが空(常時付与のはず)` で落ち、修正後は両方通る。

## Related tests

- `src/server/gkill/api/gkill_server_api/handle_get_kyous_mcp_test.go`
- `src/server/gkill/usecase/source_conventions_scan_test.go`
- `src/server/gkill/usecase/write_through_cache_test.go`
