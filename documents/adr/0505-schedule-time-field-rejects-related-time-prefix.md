# ADR-0505: Mi / MiReKyou の予定日時欄では関連時刻の接頭辞「？」を入力エラーにする

| | |
|---|---|
| Status | Accepted |
| Date | 2026-09-05 |
| Sources | 2026-09-05 の設計相談（繰り返し記法「？？」の下ごしらえ）。[ADR-0502](0502-kftl-errors-are-per-line.md) / [ADR-0503](0503-kftl-prefix-misuse-is-an-input-error.md) |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/api/kftl/kftl_related_time_statement_line.go`（`parseScheduleFieldTime`）/ `src/client/classes/kftl/kftl-schedule-field-time.ts`（`parse_schedule_field_time`） |

## Context

Mi と MiReKyou の予定日時欄（見積開始・見積終了・期限）は、行頭の `？` / `?` を
関連時刻と同じ接頭辞として黙って剥がしていた。剥がしたあとにパースへ失敗しても、
Go は `return nil // invalid → skip silently`、TS は `if (time !== null)` で
未設定として握り潰す。

つまり **`？` で始まり日時として読めない行は、エラーも警告も出ないまま日付だけが入らない。**
Mi 本体は正常に1件作られるので、利用者からは保存が成功したようにしか見えない。
同じ形が6欄（Mi 3 + MiReKyou 3）すべてにあった。

そもそも `reps.Mi` に RelatedTime 列は無いので、この欄に関連時刻の接頭辞を書けること自体に
意味が無い。利用者向けマニュアルも既に「日時に `？` は不要です」と書いていた。

決定を急ぐ理由は繰り返し記法 `？？` にある。`？？` を導入すると、それが予定日時欄の位置へ
落ちたときに `？` が1つだけ剥がれ、残った `？` が必ずパースに失敗して上の無音経路へ入る。
**書き損じが「エラーも出ないのに繰り返しだけ効かない」形になる。**

## Decision

Mi / MiReKyou の予定日時欄では、行頭の `？` / `?` を入力エラーにする。剥がさない。
判定は Go の `parseScheduleFieldTime` と TS の `parse_schedule_field_time` の
**1箇所ずつに集約**し、6欄すべてがそれを通る。

空行は未設定。日時として読めないその他の行は今までどおり未設定のままにする。

## Rejected alternatives

- **接頭辞として剥がし続け、パース失敗だけをエラーにする** — `？` を許したままだと、
  予定日時欄に落ちた `？？`（繰り返しブロック）と本物の打ち間違いを区別できない。
  長い方から判定すれば技術的には分けられるが、**判定順序の1行に静かな失敗が全部ぶら下がる**。
  禁止すれば順序に依存しなくなる
- **日時として読めない行を一律に入力エラーにする** — 無音の取りこぼしは全部消えるが、
  空行で項目位置を送る既存の書き方や、板名のような自由入力欄との境界が曖昧な行まで巻き込む。
  実害を確認できているのは `？` だけなので、そこだけ塞ぐ
- **TimeIs（`ーち`）の開始・終了時刻行も同時に禁止する** — こちらも `？` を剥がすが、
  パース失敗は既に `newKFTLInputError` になっていて黙って壊れない（`kftl_timeis.go`）。
  TimeIs には項目位置の先読み機構が無く `？？` は最後の行より後ろにしか書けないので、
  衝突もしない。非対称は承知のうえで据え置く
- **ブロックの先読み（`generateMiBlockNextConstructor`）で `？` を関連時刻行へ回す** —
  板名のような日時でない欄が `？` で始まる文字列を受け取れなくなる。
  `reps.Mi` に RelatedTime 列は無いので、回した先に付け先も無い
- **専用の「変な日時」ラベルを6欄ぶん足す** — ラベルは既存の
  「変な開始 / 変な終了 / 変な期日」で用が足りる。i18n キーは7言語ぶん増えるので足さない

## Consequences

- `？18:00` と書くと、その行がピンクになって保存が止まる（Web クライアント）/
  400 と行番号が返る（`/api/submit_kftl_text`）
- **既存データは無傷。** KFTL はテキストを保存せずパースして書くだけなので、
  影響を受けるのは入力の手癖だけ
- Wear OS / Android にも影響しない。同梱アプリは利用者が打った KFTL を送るだけで、
  `？` 付きの固定テキストを持っていない
- i18n キーが 926 → 927 になった（7言語。Go の embed は `copy_i18n_to_app_embed` が生成する）
- **TimeIs の開始・終了時刻行では `？` が引き続き使える。** 予定日時欄との非対称が残る

## Evidence

実測なし — 構造上の背理。`？` で始まり日時として読めない行は、
Go の `return nil // invalid → skip silently`（Mi 3欄）と `parseMiReKyouTime` の
`return time.Time{}, false`（MiReKyou 3欄）、TS の `if (time !== null)`（6ファイル）を通って
**必ず未設定になる**。この形が6欄すべてで同一であることをコードで確認したうえでの決定。

## Related tests

- `src/server/gkill/api/kftl/kftl_schedule_field_time_test.go`
  - `TestParseScheduleFieldTime`（接頭辞・空行・読めない行・正常な日時）
  - `TestStatement_MiScheduleFieldsRejectRelatedTimePrefix`（Mi 3欄 + ASCII 接頭辞）
  - `TestStatement_MiReKyouScheduleFieldsRejectRelatedTimePrefix`（MiReKyou 3欄）
  - `TestStatement_MiScheduleFieldsAcceptPlainDateTime`（禁止のとばっちりで壊れていないこと）
- `src/server/gkill/api/kftl/kftl_mirekyou_test.go` — `TestApply_MiReKyouTimesRejectPrefix`
- `src/client/__tests__/unit/kftl/kftl-schedule-field-time.test.ts`（Go と対のテーブル）
- `src/client/__tests__/unit/kftl/kftl-submit-emits.test.ts`
  - 「日時に「？」を付けると不正行になり保存されない」
