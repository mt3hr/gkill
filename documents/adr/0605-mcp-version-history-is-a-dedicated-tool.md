# ADR-0605: 1件の版履歴と削除の取り消しは専用ツールで返す（only_latest_data は開けない）

| | |
|---|---|
| Status | Accepted |
| Date | 2026-08-24 |
| Sources | `98afedb7` / `483c3139` |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/mcp/lib/read-handlers.mjs`（`gkill_get_kyou_history`）/ `src/mcp/lib/write-handlers.mjs`（`gkill_restore_kyou`） |

## Context

gkill は追記型で、update は新しい版の追加、delete は `is_deleted=true` の版の追加として保存される。
一方 `gkill_get_kyous` は最新版かつ未削除のものしか返さない。

その結果 MCP からは、誤って delete した記録について
「何を消したのか確認する手段」も「取り消す手段」も無かった。
`gkill_delete_kyou` の説明は
"The data remains in the database and can potentially be recovered by clearing the is_deleted flag"
と書いていたが、その flag を戻す手段はどのツールにも無かった。

`query.only_latest_data: false` を送れば旧版が見えるはず、というのが素直な期待だが、
これは成立しない（Rejected alternatives 参照）。

## Decision

1件の版履歴は `gkill_get_kyou_history`（read）、削除の取り消しは `gkill_restore_kyou`（write）
という専用ツールで返す。どちらも型別の `/api/get_*` が返す histories を使い、Go 側は変更しない。
`query.only_latest_data` は今後も常に true 固定とし、スキーマの説明にその旨を書く。

## Rejected alternatives

- **`only_latest_data: false` を利用者指定に開放する** — 旧版を落としているゲートは
  監査が最初に見つけた2箇所ではなく**最低6箇所**あり、うち3つは静かに0件を返す。実測で確認した内訳:
  - `handle_get_kyous_mcp.go:114` の無条件上書きと `:341` の `findQueryForBatch` 固定（既知の2つ）
  - `find_filter.go` の `replaceLatestKyouInfos` が `OnlyLatestData` で分岐せず、
    最新 `UpdateTime` 以外の entry を無条件に落とす。**この1つだけで、既知の2箇所を外しても件数は1件も変わらない**
  - `dao/reps/repositories.go` が `!query.OnlyLatestData` のときバケットキーを `ID + UpdateTime` に変えるが、
    `filterTagsKyous` は素の ID で引く。→ **タグフィルタ付きクエリが全部0件になる**
  - `dedupAdjacentResultKyous` が `(ID, DataType, RelatedTime)` で畳む。`UpdateTime` を見ないので、
    related_time を動かさない普通の内容編集は1件に戻る
  - v2 の複合カーソルは `(RelatedTime, ID)`。同一 ID の複数版は両方同値なので版のランの内側を指せず、
    ページングが飛ぶかループする
- **型非依存の `/api/get_kyou` へフォールバックして `data_type` を省略可能にする** —
  `Repositories.GetKyouHistoriesByRepName` は冒頭で `UnWrap()` を呼び、キャッシュ rep を丸ごと
  バイパスする。これは実データで 11rep → 約940rep・20.7秒になった経路そのもの。
  型別の `XxxRepositories.GetXxxHistoriesByRepName` はキャッシュ rep を直接回るので安全
- **`gkill_delete_kyou` に `restore: true` フラグを足す** — delete という名のツールが un-delete するのは
  悪い affordance で、要約行も「Deleted (soft)」と報告してしまう。
  誤って削除を取り消す事故より、誤って削除する事故のほうが重い
- **`update_*` 9本に `is_deleted` を出す** — 誤字を直すだけの AI が現在値の `is_deleted:true` を
  エコーして記録を消す。現行の `update_*` が `histories[0]` の `is_deleted` を黙って保持する挙動が正しい
- **履歴を無制限に返す** — 履歴は編集のたびに1件伸びる。上限を掛けないと1件の記録で応答が埋まる

## Consequences

- **`update_time` は1秒解像度でしか保存されない**（`sqlite3impl.TimeLayout`）。
  履歴の取得は `ID + UpdateTime` で dedup し、検索の最新版判定は厳密な `UpdateTime.After` で行うので、
  **同じ秒の中で delete → restore すると新しい版が最新と見なされず、黙って何も起きない**。
  `nextUpdateTime` で必ず1秒以上進める。既存の `delete_kyou` にも同じバグがあった
- 同じ理由で、**同じ秒に書かれた2つの版は履歴から1つに畳まれる**。履歴に版が欠けることがある
- 復活した記録は書き込み先 rep へ着くので、**元と違う rep に戻ることがある**
- 削除・復活を繰り返すと版が積み上がる（追記型なので正しいが、履歴は伸び続ける）
- `gkill_get_kyou_history` は書き込みサーバにも載せる。`gkill_delete_kyou` がそちらにあるので、
  「今なにを消したのか」を同じサーバから確かめられないと取り消しが当てずっぽうになる

## Evidence

2026-08-23 の readwrite MCP 監査（実測）:

- D-1 の件数が、`only_latest_data` 省略（=true）でも `false` でも **22件で変わらない**。
  その日には update を通した記録が5件あり、旧版が存在するはずだった
- 削除した kmemo を `ids` 指定 + `is_deleted:true` で引いても `total_count: 0`
- 削除済みへ `update_kmemo` を当てると**成功する**が `is_deleted` は true のまま（復活しない）

## Related tests

- `src/mcp/__tests__/read-handlers.test.mjs`
- `src/mcp/__tests__/write-handlers.test.mjs`
- `src/mcp/__tests__/write-tool-handlers.test.mjs`
