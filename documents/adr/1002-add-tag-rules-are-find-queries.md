# ADR-1002: add_tag のルールは検索条件 JSON そのもの。接頭辞は画面の「記録種別」で表し、曖昧な指定はサーバに触る前に落とす

| | |
|---|---|
| Status | Accepted |
| Date | 2026-09-11 |
| Sources | `.claude/skills/gkill-cli-ops/SKILL.md`「CLI サブコマンドと運用の約束」 / `src/server/gkill/main/common/add_tag.go` / `src/client/classes/api/find_query/find-kyou-query.ts` |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/main/common/add_tag_test.go` |

## Context

**タグを自動で付ける CLI が、対象を rep 名でしか選べなかった。**

旧 `auto_tag` は「rep 名の接頭辞 → 固定タグ」（`--tag_by_rep_prefix 'AutoScreenshot_=autolog_screenshot'`）と
「rep 種別 → その rep 名をタグに」（`--tag_by_rep_name git_commit_log`）の2種類の固定ルールだけを持っていた。
「この期間の写真」「このタグが付いていて別のタグが無いもの」のような条件では対象を選べず、
ルールを1つ足すたびに Go のコードとフラグを増やす構造だった。利用者の要望は
**検索条件（画面の `FindKyouQuery`）を JSON で渡し、その結果に指定タグが付いていなければ付ける**こと。
あわせてサブコマンド名を `add_tag` へ改名した。

ただし素の検索条件では旧ルールを表現できない。実環境の rep は `AutoScreenshot_<端末>_<YYYYMMDD>` の形で
端末追加や `dvnf --new` のたびに増える（2026-09-11 時点で AutoScreenshot 4 個・AutoAudio 2 個）のに、
Go の `FindQuery.Reps` は完全一致（`find_filter.go` の `selectMatchRepsFromQuery`）で、
サーバの `rep_types` は kmemo / directory / git_commit_log のような保管形式の語彙。AutoScreenshot は
user_config.db 上 `TYPE=directory` で、DCIM や Foods と同じ種別になる。

一方、**画面の「記録種別」は別の概念**だった。クライアントの `FindKyouQuery.rep_types_in_sidebar` は
rep 名を `_` で3分割した先頭要素（`rep_to_struct`）で、`apply_rep_summary_sets_to_detaul` が
「種別 ∈ rep_types_in_sidebar かつ端末 ∈ devices_in_sidebar」の rep 名で `reps` を上書きしてから送る。
利用者が「既存の RepType でできないか」と言ったのはこちらを指していた。

## Decision

**ルールは `{"tag": "<タグ名>", "query": {<FindKyouQuery JSON>}}` で、`query` は画面の検索条件 JSON をそのまま受ける。**
`--rule '<json>'`（繰り返し可）と `--rules_file <path>`（配列）の両方を持つ。

1. **接頭辞は `rep_types_in_sidebar` で表す。** CLI が `/api/get_all_rep_names` の一覧をクライアントと同じ規則で
   分解し、`reps` を上書きしてから送る。`devices_in_sidebar` 省略＝全端末（画面と違い、省略できる）。
   運用スクリプトのルールは `{"tag": "autolog_screenshot", "query": {"rep_types_in_sidebar": ["AutoScreenshot"]}}` になる。
2. **「タグ＝rep 名」のルールは廃止する。** 固定タグ＋検索条件では表現できず、残すには別のルール型が要る。
   Windows の git コミットへのリポジトリ名タグはやめる（過去に付いたタグは残る）。
3. **曖昧な指定はサーバに触る前に exit 1 にする。** 未知のキー（`DisallowUnknownFields`）、絞り込みが1つも無い
   `query`、常に0件になる非 null 空配列、非空の `keywords`、`update_cache: true`、`devices_in_sidebar` 単独、
   揃っていない地図3値、`timeis_words` 無しの `timeis_tags`。どれも「エラーにならず意図と違う範囲に静かに当たる」。
   先頭 BOM と旧 `use_*` 形式は受理する。
4. **タグ ID の名前空間と `CREATE_APP` の値（`gkill_auto_tag`）は改名後も据え置く。** 過去に付けたタグと ID が
   一致し続けることが、再実行の冪等性（`ERR000056` でスキップ）と「手で消したタグが復活しない」ことの根拠。
5. 「付いているか」の判定は旧実装どおり `/api/get_kyous` 2回の差分（条件そのまま vs. `tags: [<tag>], tags_and: true`）。
   タグ以外の条件が残るので差分がそのまま未付与分になる。応答本文の上限は 8 MiB から 1 GiB へ
   （検索応答を受ける以上、実用的な上限は張れない）。

## Rejected alternatives

- **サーバの `rep_types` で切る** — AutoScreenshot も DCIM も `directory` で、152 行ある directory rep の
  全部に当たる。切り出しにならない。
- **rep 名を JSON に列挙する** — 純粋な FindQuery で済むが、端末追加や `dvnf --new` で増えた rep を
  **静かに取りこぼす**。エラーも警告も出ず、「新しい端末の分だけタグが付かない」で気付くことになる。
- **`reps` にワイルドカード（`AutoScreenshot_*`）を許す** — 自己完結するが、FindQuery の意味論から外れた
  CLI だけの方言になり、画面の JSON と互換でなくなる。画面の `rep_types_in_sidebar` が同じことを既に表しており、
  その規則を CLI で再現するほうが利用者の語彙に合う。
- **`tag_by_rep_name`（タグ＝rep 名）を残す** — ルールに `"tag_by_rep_name": true` のような別型を足せば残せるが、
  利用者が廃止を選んだ。Linux 版は元から使っておらず、Windows 版だけが使っていた。
- **`keywords` をクライアントと同じ規則で `words` へ解析する** — 画面から貼った JSON に `keywords` が残っていても
  そのまま通せるが、TS の解析（半角/全角スペース分割・`-` 接頭辞）を Go に写す保守が要る。
  `words` を書くよう促すほうが単純で、拒否する理由もエラー文で伝わる。
- **未知キーを無視する（サーバの `HandleGetKyous` と同じ素の `Decode`）** — `"rep"` の綴り誤りが `reps` 未使用に
  化け、利用者の全記録にタグが付く。タグは論理削除で行が残り、一括で取り消す手段が無い。
- **旧フラグを互換のために残す** — 「指定したのに効かない」を静かに起こす。呼び出し元は3スクリプトだけで、
  同時に直せる。
- **応答本文の上限を外す** — 暴走時の歯止めとして残し、既定 1 GiB にした。上限はクライアント構造体の
  フィールドで、テストは小さな値で「上限超過はデコード失敗として現れる」壊れ方を固定する。

## Consequences

- 運用スクリプト（Windows / Linux / Android の3本）はルール JSON ファイルを持ち、`--rules_file` で渡す。
  PowerShell 5.1 はネイティブコマンドへ渡す引数の二重引用符を落とすことがあるので、Windows では `--rule` を使わない。
- 配布は新バイナリが先。古いバイナリに `add_tag` は無く、`unknown command` で exit 1 になる（loud だが、その回はタグが付かない）。
- クライアントの `FindKyouQuery` にキーが増えると、そのキーを含む JSON が CLI で「未知のキー」になる。
  `TestAddTagQueryJSONAcceptsEveryClientFindKyouQueryKey` が TS のソースを走査して検出する。
- 画面から貼った JSON は `tags`（チェック済み一覧）・`hide_tags`・`calendar_*` で範囲が狭い。差分計算は正しいので
  「target = 0」が静かに出るだけになる。各ルールが印字する有効な FindQuery と `kyous = N` を `--dry_run` で見る運用。
- 端末名に `_` が入る rep は「記録種別」に分類されない（画面と同じ）。旧接頭辞照合なら拾えていた分が対象外になる。
- ルールの書式誤りは RunE のエラーに加えて stderr にも出す。main の `log.Fatal` はログファイルにしか書かず、
  端末には usage と exit 1 しか見えなかった。

## Evidence

- 実環境の rep 名の形: `$HOME/Kyou/` 配下に `AutoScreenshot_<端末>_<日付>` 4 個・`AutoAudio_<端末>_<日付>` 2 個、
  いずれも `_` で3要素（2026-09-11）。user_config.db の REPOSITORY は `~/Kyou/AutoScreenshot_*` を `TYPE=directory` で登録。
- 旧実装と新実装の `--dry_run` の一致（同じ稼働中サーバ、2026-09-11）:
  AutoScreenshot `reps = 4 kyous = 1236 tagged = 1231 target = 5`、AutoAudio `reps = 2 kyous = 400 tagged = 382 target = 18`。
  旧 `--tag_by_rep_prefix` と新 `rep_types_in_sidebar` で4つの数字がすべて一致した。
- 拒否する指定の実測: `{}` / `"rep"` / `tags: []` / `keywords` / `update_cache: true` / 末尾の余分な JSON / 存在しないファイル は
  いずれも exit 1 で、理由が stderr の `add_tag: ...` 行に出る。BOM 付きファイルは読める。
- 応答の `messages` には毎回 `MSG000025 検索完了` が載る。印字するのは rep 読み込み失敗（`MSG000090`）と
  プラグイン失敗（`MSG000088`）だけに絞った。

## Related tests

- `src/server/gkill/main/common/add_tag_test.go` — ルール JSON の厳格復号、拒否/通過する指定の網羅表、
  クライアントの全キー受理（TS ソース走査）、`rep_types_in_sidebar` の展開、タグ ID の過去値固定、HTTP 応答判定
- `src/server/gkill/main/common/common_test.go` — `add_tag` が RunE + SilenceUsage/SilenceErrors であること
