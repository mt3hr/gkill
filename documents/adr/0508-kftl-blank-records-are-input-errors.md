# ADR-0508: 保存マーカー行は値の行に数えず、内容の無い記録・付け先の無いメタ情報・読めない予定日時は書く前に行別エラーにする

| | |
|---|---|
| Status | Accepted |
| Date | 2026-09-15 |
| Sources | 2026-09-15 の利用者報告（「`ーち` で内容を入力しなくてもエラーにならない」）と、旧 TS（`99998673^`）と新 Go に同じ 104 入力を流した実測。[ADR-0503](0503-kftl-prefix-misuse-is-an-input-error.md) / [ADR-0505](0505-schedule-time-field-rejects-related-time-prefix.md) / [ADR-0507](0507-kftl-single-implementation-on-server.md) |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/api/kftl/kftl_statement.go`（`generateKFTLLines` のマーカー切り詰め / `validateRequestContents`）/ `src/server/gkill/api/kftl/kftl_request.go`（`ValidateContent`）/ `src/server/gkill/api/kftl/kftl_prototype_request.go` / `src/server/gkill/api/kftl/kftl_related_time_statement_line.go`（`parseScheduleFieldTime`）/ `src/client/classes/use-kftl-view.ts`（空白だけの本文は解析に投げない） |

## Context

ADR-0507 で Web のメモ帳も Go の `/api/submit_kftl_text` へ送るようにした翌日、利用者から
「`ーち` で内容を入力しなくてもエラーにならない。画面（旧 TS）の頃はエラーだった」と報告があった。

旧 TS と新 Go に同じ 104 入力を流して突き合わせると、甘くなった行は次の2群だった。

**1. 保存マーカー「！」の穴。** Web はマーカー行を含めたまま送る。Go の `generateKFTLLines` は
`NextStatementLineText` に生の次行を入れてからマーカー行で break していたので、`ーち`+「！」は
「次の行に値がある」と判定され（ADR-0503 の `requireNextLineText` を素通り）、タイトル空のまま
`DoRequest` に届いて `title == "" → return nil`。結果は 200「保存しました」でタブが閉じ、何も書かれない。

| 入力（Web の「！」保存） | 旧 TS | Go（2026-09-15 まで） |
|---|---|---|
| `ーち`+「！」／`ーた`／`ーみ`／`ーう` | ERR9000xx「内容がない○○の保存がスキップされました」で送信中止 | 200・0件・タブが閉じる |
| `ーら`+「！」 | 気分値0を書く（旧も同じ） | **気分値0を書く**（ADR-0503 が塞いだ事故の復活） |
| `ーか`+「！」／`ーか`,`タイトル`（値の行無し） | 空の数値記録を書く | rep の `kc num_value must not be empty` が生 error のまま **500 ERR000351**（行番号も理由も無い） |

マーカー無し（保存ボタン）なら `requireNextLineText` が止めるので、**「！」で保存する人だけが踏む**穴だった。

**2. `DoRequest` の「空なら黙って nil」が旧 TS の送信時エラーを置き換えていなかった。** 旧 TS は
kmemo / 打刻 / タスク / ブックマークの空を `ERR900012` `ERR900016` `ERR900013` `ERR900020` で止め、
tx を捨てていた（`メモ`,`、`,空行 は先頭のメモも書かれない）。Go は空メモだけ黙って飛ばしていた。

あわせて、旧 TS も新 Go も黙っていた穴が3つ見えた。`。タグ` だけ・`？時刻` だけ（付け先の記録が無い
プロトタイプ）は0件成功、`ーん`,`店` は支払いが1件も無いのに0件成功、`ーみ`,`タイトル`,`板`,`abc` は
予定日時が読めないまま「未設定」で保存（ADR-0505 が据え置いた部分。`、` で6行を埋めずに次の記録へ
移ろうとすると `、` が見積開始に食われ、後ろの本文も残りの欄に消える）。

利用者の指示は「甘くなった行は旧の挙動へ、黙っていた行も改善」。

## Decision

- **保存マーカー行は値の行に数えない。** `generateKFTLLines` は `strings.Split` の直後に最初のマーカー行
  （1行目は除く）で `lineTexts` を切り詰めてから `NextStatementLineText` を組み立てる。
  `ーち`+「！」は保存ボタンと同じ「次の行に値を書いてください」になる
- **内容の無い記録と付け先の無いメタ情報は、書く前に行別エラーにする。** `KFTLRequest` に
  `ValidateContent() error` を足し（基底に既定実装を置かない）、`prepareRequests` が全行を適用した後・
  `expandRepeats` の前に `validateRequestContents` で全リクエストを回す。行エラーが1件でもあれば呼ばない
  （欠けた map で偽のエラーを重ねない）。`Analyze`（ピンク）と送信が同じ関数を通る。
  メッセージIDは旧 TS が使っていた既存キー（`KFTL_KMEMO_BLANK_SKIP_SAVE_MESSAGE_TITLE` 等）。
  残ったプロトタイプだけ新設の `KFTL_META_INFO_NO_TARGET_MESSAGE_TITLE`（i18n 984 → 985）
- kmemo の空は「全行が空白」（`strings.TrimSpace(joinLines(...)) == ""`）。`joinLines` は `["",""]` を
  `"\n"` にするので `== ""` では改行だけのメモが本文 `"\n"` として書かれる
- `DoRequest` は `ValidateContent` をもう一度呼ぶ（書き込みフェーズまで来た経路の二重の防御線）。
  `ーえ` / `ーたえ` の題名・タグ無し、`～～` の対象無し（requestMap だけで判定できる3検査）も
  `DoRequest` から `ValidateContent` へ移し、Analyze でピンクになるようにした
- `？時刻` の直後の `ーん` は関連時刻をブロックへ取り込んだあと `KFTLRequestMap.Delete` でプロトタイプを外す
  （残すと「付け先の無い関連時刻」に誤爆する）
- KC のタイトル行と支出の店名行にも `requireNextLineText` を置く（値の行無し・店名だけを Apply フェーズで止める）
- `parseScheduleFieldTime` は空でないのに読めない行を `KFTL_TIMEIS_INVALID_PARSE_TIME_ERROR_MESSAGE_TITLE`
  の入力エラーにする。ADR-0505 の却下案「日時として読めない行を一律に入力エラーにする」の採用。
  0505 の決定（`？` の禁止）はそのまま。空行は今までどおり未設定
- Web は本文が空白だけなら `parse_kftl_text` を投げずピンクを消す（投げると新しい空のタブの1行目が常にピンクになる）。
  送信はサーバに聞くので、空のタブで保存を押せば「内容がないメモの保存がスキップされました」が出る（旧 Web と同じ）

## Rejected alternatives

- **`DoRequest` の中だけで入力エラーにする** — 送信は 400 になるが `Analyze` は `DoRequest` を呼ばないので
  ピンクにならず、ADR-0507 の「parse で通った入力は submit で弾かれない」（`TestAnalyze_ReportsTheSameInvalidLinesAsExecute`）
  が崩れる。書く前の1箇所（`prepareRequests`）に置く
- **各 start 行の `requireNextLineText` を外して `ValidateContent` に一本化する** — `ーら`+「！」の気分値0は
  値の行を通らずに `mood` がゼロ値のまま届く型で、`ValidateContent` に `hasMood` を持たせれば止まるが、
  Apply フェーズの検査（ADR-0503）を外す理由が無い。二重にして両方から守る
- **マーカー行を `NextStatementLineText` に残し、各 start 行で「次の行がマーカーなら空扱い」にする** —
  同じ if が7箇所に散り、型を足すたびに書き忘れる。切り詰めは1箇所
- **空メモを黙ってスキップのまま** — `メモ`,`、`,空行 で先頭のメモが保存される今の挙動のほうが親切とも言えたが、
  旧 Web は送信全体を止めていたし、本文無しで「！」を打っただけで「保存しました」と出てタブが閉じるのは
  「何が起きたか分からない」の典型。利用者が旧の挙動を指定した
- **プロトタイプの残存を許す（`。タグ` だけは0件成功のまま）** — 旧 Web も黙っていたので回帰ではないが、
  `、` の後ろにタグ行だけを書いた送信が黙って0件になる。利用者が改善を指定した
- **`ーー` だけ（テキスト0行のプロトタイプ）は許す** — 「付け先が無い」に変わりはなく、中身の有無で分岐を増やす
  価値が無い
- **予定日時欄の読めない行は据え置く（ADR-0505 のまま）** — 0505 の却下理由「空行で項目位置を送る書き方や
  板名との境界が曖昧な行まで巻き込む」は、空行を未設定に残し、タグ・テキスト・`？？` を先読みが拾う今の
  形なら当たらない。実害は `、` で早く閉じた本文が黙って消える方が大きい
- **`KFTLRequestMap.Delete` の代わりにプロトタイプへ「取り込み済み」フラグ** — `Analyze.RecordCount` が
  書かないプロトタイプを数え続ける。消すほうが件数も正しくなる

## Consequences

- `ーち`+「！」は行1「次の行に値を書いてください」で止まり、タブは閉じない（マーカーだけ取り除かれる）
- 空のテキスト・空行だけ・`。タグ`+空行・`メモ`,`、`,空行 は「内容がないメモの保存がスキップされました」で
  **送信全体が止まる**（旧 Web と同じ。先頭のメモも書かれない）。`/api/submit_kftl_text` に空文字を送ると 400
- `。タグ` だけ・`？時刻` だけ・`ーー` だけ・`、` の後ろにタグ行だけ は「タグ・テキスト・関連時刻を付ける記録がありません」
- `ーん`,`店` は行2「次の行に値を書いてください」。支払いの後ろの空行は今までどおり許す
- `ーみ` / `～～` の予定日時欄に空でない読めない行を書くとエラー。**6行を埋めずに `、` で次の記録へ移る書き方は
  今までも黙って壊れていたが、これからはエラーになる**（空行で欄を飛ばす書き方はそのまま）
- `ーえ` / `ーたえ` / `～～` の欠けは打鍵中にピンクになる（今までは送信で初めて 400）
- Wear OS の気分（`?time\n/mood\n5`）・MCP（trim 済みの非空文字列）・E2E の `submitKftlText`（値のある本文）は
  どれも該当せず、契約（`SubmitKFTLTextRequest/Response`, `parse_kftl_text`）は変えていない。
  Wear の利用者テンプレートが `接頭辞\n！` の形なら新たにエラーになる（意図どおり）
- `KFTLRequest` に型を足すときは `ValidateContent` を書かないとコンパイルが通らない。「何を空とみなすか」を
  必ず決めさせるための形で、nil を返す型（`ーん` の支払い・`ーら`）にもその理由を書く

## Evidence

- 2026-09-15 の実測（旧 TS は worktree で vitest、新 Go はテストハーネスの HTTP）。修正前:
  `ーち\n！\n` → parse `invalid_lines=[]`・submit 200・`created=[]`／`ーら\n！\n` → 200・`created=[lantana]`／
  `ーか\n！\n` → 500 ERR000351／`""` → 200・0件（旧 TS はどれも ERR9000xx）。修正後: すべて 400 で `created=[]`、
  `ーら\n！\n` で lantana の件数が増えない（`TestHandleSubmitKFTLText_SaveMarkerAfterBarePrefixIsInputError`）
- 同じ 104 入力の「判定が一致した」群（日時の解釈失敗・気分範囲外・金額・繰り返し11種・none 行 等）は
  修正前後で変わらない
- `？2026-09-15 10:00\nーん\n店\n品\n100\nお茶\n120` は `RecordCount=2`（支払い2件・プロトタイプは数えない）

## Related tests

- `src/server/gkill/api/kftl/kftl_analyze_test.go`
  - `TestAnalyze_ReportsTheSameInvalidLinesAsExecute`（マーカー・空メモ・付け先無し・読めない予定欄・終了の欠けを parse と submit で同じ集合に）
  - `TestAnalyze_BlankRecordsAreInputErrors`（型ごとの文言と行番号）
  - `TestAnalyze_SaveMarkerLineIsNotAValueLine`（14 接頭辞 × 2 マーカー。値があれば通る）
  - `TestAnalyze_RelatedTimeBeforeExpenseBlockIsNotAnOrphan`（`Delete` の誤爆防止）
  - `TestAnalyze_ExpenseShopNameNeedsAnItemLine` / `TestAnalyze_UnparsableScheduleFieldIsInputError`
- `src/server/gkill/api/kftl/kftl_statement_test.go` `TestStatement_BlankRecordAbortsWholeSubmissionBeforeWriting`
- `src/server/gkill/api/kftl/kftl_schedule_field_time_test.go` `TestParseScheduleFieldTime`「読めない行は入力エラーにする」
- `src/server/gkill/api/kftl/kftl_nlog_test.go` `TestNlogBlock_TrailingBlankLineIsIgnored`（支払いの後ろの空行は据え置き）
- `src/server/gkill/api/gkill_server_api/handle_submit_kftl_text_test.go` `TestHandleSubmitKFTLText_SaveMarkerAfterBarePrefixIsInputError`
- `src/server/gkill/api/gkill_server_api/gkill_server_api_test.go` `TestHandleSubmitKFTLText_EmptyText`（空文字は 400）
- `src/client/__tests__/unit/kftl/kftl-submit-emits.test.ts`「内容が空のメモ（ADR-0508）」節
- `src/client/__tests__/e2e/kftl-tabs.spec.ts`「接頭辞だけの行を保存マーカーで保存するとエラーになり、タブは閉じない」
