# ADR-0509: メモ帳の打刻終了は「開始時刻が最新の実行中1件」を終え、削除済みは候補にしない

| | |
|---|---|
| Status | Accepted |
| Date | 2026-09-16 |
| Sources | 2026-09-16 の利用者報告（「`ーいたえ` などが機能しない。画面で処理していたときは動いていた」）と、利用者の同期済み打刻 DB・タグ DB の件数集計、旧 TS（`99998673^` のタグ指定・題名指定の終了リクエスト）の読み合わせ。[ADR-0507](0507-kftl-single-implementation-on-server.md) / [ADR-0508](0508-kftl-blank-records-are-input-errors.md) |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/api/kftl/kftl_timeis.go`（`findPlayingTimeIsEntries` / `sortPlayingTimeIsEntries` / `addSearchTag` / `endTargetIDInheritingPrototype`） |

## Context

ADR-0507 で Web のメモ帳も Go の `/api/submit_kftl_text` へ送るようにした翌日、利用者から「`ーいたえ`
（タグで打刻終了・存在時）などが機能しない。いま走っている打刻が終わらない」と報告があった。
利用者のメモ帳テンプレートは 35 本すべてが `ーいたえ` / `<カテゴリタグ>` / `、、` / `。<タグ>` / `ーた` / `<題名>` / `！`
（走っている X カテゴリの打刻を終えて、次の打刻を始める）の形で、終了系はメモ帳の使い方の中心にある。

旧 TS と新 Go を読み合わせると、終了対象の**探し方**は同じ（設定の playing 検索条件を写し、候補ごとにタグを引く）だが、
**選び方**が違っていた。

| | 旧 TS（Web、2026-09-15 まで） | Go（Wear / MCP は以前から。Web は ADR-0507 から） |
|---|---|---|
| 候補の取得 | `get_kyous`（`api.FindFilter`） | `TimeIsReps.FindTimeIs` を直接 |
| 候補の並び | RelatedTime（＝開始時刻）降順・ID 昇順（`sortResultKyousByKey`） | **不定**（map を舐めて slice を組む。契約にも「順序を保証しない」） |
| 削除済み | 落とす（`findKyous` の `IncludeDeletedData` 分岐） | **含む**（`FindTimeIs` は `IS_DELETED` を見ない） |
| 終える件数 | 先頭一致の1件で `break` | 先頭一致の1件で `break` |

つまり Go は「不定順の先頭に一致した1件」を終えていた。利用者の同期済み DB を件数だけ集計すると、テンプレートの
検索タグを持つ実行中（終了時刻なし・最新版）の打刻は、いま走っている1件のほかに **2026-03〜05 開始の終え忘れが 3〜4 件、
削除済みなのに終了時刻の無い打刻が 4〜6 件**あった。10 件から不定順で1件を選ぶので、走っている打刻に当たるのは
おおむね 1/10。削除済みに当たれば削除済みの新しい版が1つ増えるだけで、見えている打刻は走ったまま。
旧 TS は最新の1件を決定的に選んでいたので、終え忘れが何件あっても走っている打刻が終わっていた。

あわせて、同じ関数の読み合わせで Go だけの取りこぼしが2つ見えた。

- `kftlTimeIsEndByTagRequest.AddTag` の override が検索タグを本体の `Tags` にも積んでいた。`doBaseRequest` は本体タグを
  `r.RequestID`（新規 UUID。Kyou は無い）へ Tag 行として書くので、終了に成功するたびに**付け先の無い Tag が1行**増え、
  `Analyze().Tags` にも検索タグが載って Web の未知タグ確認が余計に出ていた。旧 TS は `add_target_tag_name` を別に持っていた
- 終了4行（`ーえ` `ーいえ` `ーたえ` `ーいたえ`）だけが直前のプロトタイプ（`？時刻` / `。タグ`）の target_id を引き継がず、
  常に新しい UUID を採っていた。`？18:00` / `ーいたえ` / `X` は旧 TS では 18:00 に終わったが、Go は関連時刻のプロトタイプが
  宙に浮き、ADR-0508 からは「タグ・テキスト・関連時刻を付ける記録がありません」の行エラー（それ以前は黙って「今」で終了）。
  他の全型（kmemo / kc / mi / lantana / nlog / timeis / urlog）は Go でも引き継いでいる

既存テスト（`TestHandleSubmitKFTLText_TimeIsEndByTagIfExist_WithMatch`、E2E `kftl-timeis-end.spec.ts`）は
「エラーが出ない」しか見ておらず、実行中の打刻が1件しか無いテスト環境では不定順でも当たるので捕まらなかった。

## Decision

- **候補は `findPlayingTimeIsEntries` の出口で、削除済みを落とし、開始時刻降順・同着 ID 昇順に並べる**
  （`sortPlayingTimeIsEntries`。鍵は `find_filter.go` の `sortResultKyousByKey` と同じ）。閉包 `FindKyous` によるタグ絞り込みの
  有無に関わらず同じ後処理を通す。呼び出し側（`ーえ` / `ーたえ` 系の `DoRequest`）は今までどおり先頭一致の1件を終える。
  終えるのは**最新の1件だけ**（旧 TS と同じ）
- **検索タグは `addSearchTag` で `searchTags` だけに積む。** `AddTag` の override は消す。本体 `Tags` に混ぜないので
  付け先の無い Tag 行は書かれず、`Analyze().Tags` にも載らない
- **終了4行も直前のプロトタイプを引き継ぐ**（`endTargetIDInheritingPrototype`。他の型と同じ判定
  `prevLine.GetContext().ThisIsPrototype`）。`requestMap.Set` がプロトタイプの関連時刻・タグを終了リクエストへ移す
- 変えないもの: 複数タグ指定（`A、B`）の照合は片方一致（OR）のまま（旧 TS は AND だったが利用者のテンプレートに複数指定は無く、
  今回は範囲外）。終了ブロックの後ろに書いた `。タグ` / `ーー` の付け先も今までどおり `r.RequestID`

## Rejected alternatives

- **一致する実行中の打刻を全件終える** — 終え忘れが自動で片付くように見えるが、数か月前の開始に「今」の終了時刻が入り、
  数か月の長さの打刻が生まれる。旧 TS も1件だけだった。終え忘れは利用者が実行中画面から手で終える
- **利用者に選ばせる（候補を返して確認）** — Wear OS / MCP に確認の UI が無い。ADR-0507 の「全経路で同じ1実装」に反する
- **rep 層 `FindTimeIs` で並べる** — 契約が「順序を保証しない」で、並べ替えは呼び出し側の責務。旧 TS 時代も
  `find_filter` が並べていた。rep の集約は map で組む前提の設計（`time_is_repository.go`）
- **`FindTimeIs` で削除済みを落とす** — `FindTimeIs` は履歴取得（`IncludeDeletedData`）にも使われ、rep 層で一律に落とすと
  そちらが壊れる。終了対象の1箇所で落とす
- **検索タグを `Analyze().Tags` に残して未知タグを警告する** — 検索タグの打ち間違いは `ーたえ` なら「終了対象の打刻が
  存在しませんでした」で分かる。本体タグと混ぜたまま「付け先の無い Tag 行」を書き続ける害のほうが大きい

## Consequences

- 同じタグ／同じ題名の実行中の打刻が何件あっても、**開始時刻が最新の1件**が終わる（Web / Wear / MCP で同じ）
- 削除済みの打刻は終了の対象にならない
- `ーたえ` / `ーいたえ` を送っても付け先の無い Tag 行は増えない。`parse_kftl_text` の `tags` に検索タグは載らない
- `？時刻` / `ーいたえ` / `X` はその時刻で終わる（`ーえ` 系も同じ）。`。タグ` / `ーいたえ` / `X` はタグが終了リクエストに
  引き継がれ、今までどおり `r.RequestID` へ書かれる（旧 TS と同じ振る舞い。付け先の見直しは別の判断）
- 契約（`SubmitKFTLTextRequest/Response`、`parse_kftl_text`）は変えていない。配布中のバイナリ（2026-09-15）には
  ADR-0508 も入っていないので、両方を同じ配置で出す

## Evidence

- 2026-09-16、利用者の同期済み `TimeIs_*.db` / `Tag_*.db`（13 + 12 ファイル）を件数だけ集計: 実行中（最新版・未削除）9 件、
  削除済みで終了時刻無し 10 件。テンプレートの検索タグ別では「生きている実行中 3〜4 件（うち現在のものは 1 件）＋
  削除済み実行中 4〜6 件」
- 修正前の `findPlayingTimeIsEntries` に 4 件の実行中 + 1 件の削除済み実行中を流すと、3 回の実行で
  `second-oldest,newest,oldest,deleted-running,third` / `newest,oldest,deleted-running,third,second-oldest` /
  `second-oldest,newest,oldest,deleted-running,third`（`TestFindPlayingTimeIsEntries_NewestFirstAndSkipsDeleted` の変異検査）
- 利用者の設定（この PC と配布スナップショット）は `playing_timeis_find_kyou_query` が `null` で、検索条件の写し漏れではなかった

## Related tests

- `src/server/gkill/api/kftl/kftl_analyze_test.go` `TestFindPlayingTimeIsEntries_NewestFirstAndSkipsDeleted`（並びと削除済み除外）
- `src/server/gkill/api/kftl/kftl_statement_test.go` `TestApply_AsciiEndByTagTagNames`（検索タグは本体 Tags に混ざらない）
- `src/server/gkill/api/gkill_server_api/handle_submit_kftl_text_test.go` `TestHandleSubmitKFTLText_TimeIsEndTargetsLatestRunning`
  （cache_in_memory の両方で: 最新の1件だけ終わる／削除済みは触らない／Tag 行が増えない／`？時刻` が引き継がれる）
- `src/server/gkill/api/gkill_server_api/gkill_server_api_test.go` `TestHandleSubmitKFTLText_TimeIsEndByTagIfExist_WithMatch`
  （終わったことまで見る）
- `src/client/__tests__/e2e/kftl-timeis-end.spec.ts`（終了後に実行中画面からラベルが消える）
