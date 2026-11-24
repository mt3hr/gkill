# ADR-0419: 日付・時刻欄を空に戻す手段は欄の中の × ひとつにし、欄ごとに空にする

| | |
|---|---|
| Status | Accepted |
| Date | 2026-11-24 |
| Sources | [ADR-0414](0414-period-filter-time-of-day-in-sidebar.md)（期間の時刻欄に `clearable` を入れた先例） / `.claude/skills/gkill-client-foundation/SKILL.md`「UI 実装の共通規約」 / `node_modules/vuetify/lib/components/VField/VField.js`・`components/VTextField/VTextField.js`・`composables/proxiedModel.js`（Vuetify 4.3.3 の実装） |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/client/pages/views/add-mi-view.vue` ほか、× を付けた各ビューの最初の入力欄の直前のコメント / `src/client/classes/use-mi-re-kyou-schedule-fields.ts` の `clear_estimate_start_date_time` の直前のコメント |

## Context

日付・時刻の入力欄を空に戻す手段が3通りに割れていた。

- **欄の横の「クリア」ボタン**（Mi / MiReKyou の 開始・終了・期限、TimeIs 編集・終了の終了日時）。
  日付欄と時刻欄をまとめて1つのボタンで空にする。`.gkill-field-side-buttons` の表の中で
  「リセット」「現在日時」と並ぶので、欄1つにつきボタンが2〜3個ぶら下がって縦に嵩む。
- **欄の中の ×**（サイドバー「期間」の開始時刻・終了時刻）。ADR-0414 で入れた Vuetify の `clearable`。
- **手段なし**（サイドバー「時間帯」の開始時刻・終了時刻）。同じ形の時刻ピッカーなのに `clearable` が無く、
  節全体の「クリア」（曜日も落ちる）しか無いので、「開始だけ未指定に戻す」が表せなかった。

利用者が「期間」の × を「コンパクトでわかりやすい」として、これに統一することを選んだ。

## Decision

日付・時刻の入力欄を空に戻す操作は、どの画面でも**欄の中の ×（Vuetify の `clearable`）ひとつ**にする。
欄の横の「クリア」ボタンは撤去し、「リセット」「現在日時」は残す。

- **× は押した欄だけを空にする。** 日付の × は日付だけ、時刻の × は時刻だけ。
- **必須の日時欄には付けない**（関連日時・通知日時・TimeIs の開始日時）。裏の ref が `Ref<Date>` で
  null を取れず、表示 computed に null ガードも無い。
- **`v-model` ではなく `:model-value` + `@click:clear` で結ぶ**（「期間」だけは例外。下記）。
- 「時間帯」の時刻2欄にも × を足す。TimeIs の終了日時は追加・編集・終了の3ダイアログとも付ける。
- **日付欄は × の有無によらず `min-width="160"`、追加/編集ダイアログの時刻欄は `140` に揃える**
  （サイドバーの検索条件の時刻欄は `HH:mm` しか入らないので 120 のまま）。
- `:clearable` には隣の `v-menu` の `:disabled` と同じ式の否定を渡す。送信中に × だけ押せる穴を作らないため。

## Rejected alternatives

- **× も日付と時刻をまとめて空にする（今の「クリア」ボタンと同じ単位にする）** — 1クリックで済むので一見親切。却下理由は、時刻欄の × を押したら日付まで消えるのが読めないこと。日付と時刻は「両方空 or 両方入力」でないと保存時にエラーになるが、その半端な状態は今も（日付だけ選んで時刻を入れないことで）作れて、検証がそれを拾う。× の単位は「欄」のほうが素直。
- **必須の日時欄にも × を付けて揃える** — 見た目は最も揃う。却下理由は、裏の ref が `Ref<Date>`（null 不可）で、表示 computed が `moment(related_date_typed.value).format(...)` と null ガードなしに書かれていること。空にすると保存時に `*_time_is_blank` になるだけで、利用者には「押せるのに保存できない」としか見えない。これらの欄に今も「クリア」が無いのは意図どおり。
- **`v-model` のまま `clearable` を付ける** — 行数は最小。却下理由は2つあり、**どちらも静かに壊れる**。日付欄の `v-model` 先は読み取り専用の computed（`*_date_string`）なので、Vuetify が内部で通す `model.value = null` が「Write operation failed: computed value is readonly」になって値は変わらない。時刻欄の `v-model` 先は `Ref<string>` で、`clearable` が書く `null` が入ると、保存側の `=== ""`（未指定）判定を**全部すり抜ける**。「両方空ならOK、片方だけならエラー」の検証が片方 null のときに誤判定する。
- **時刻 ref を `Ref<string | null>` へ広げて、「期間」と同じく `clearable` に直接 null を書かせる** — パターンが1つに揃うので筋はよい。却下理由は、`=== ""` の判定が `validate()` / `resolve_times()` / `is_body_changed` など20箇所級に散っていて、1つ漏らすと「片方だけ空」の検証が静かに誤ること。`:model-value` + `@click:clear` なら型も判定も1つも変えずに済む。
- **「期間」も `:model-value` + `@click:clear` へ書き換えてパターンを完全に1つにする** — 却下理由は、「期間」の時刻 ref が最初から `Ref<string | null>` で、`parse_time_string` が null を未指定として受けるため、書き換えても挙動が1つも変わらないこと。動いているものを触る理由が無い。走査テストが例外を表で持つ。
- **TimeIs 追加ダイアログの終了日時は「リセット」で空に戻せるので × を足さない** — 初期値が空なので、ここだけは「リセット」が実質クリアになる。却下理由は、兄弟ダイアログ（編集・終了）の同じ欄には × が付くので、同じ操作の見え方がダイアログによって変わること。「リセットは元の値へ戻す」「× は空にする」という2つの意味も混ざる。
- **日付欄の `min-width` は 120 のままにして、× を付けた欄だけ広げる** — 差分は小さい。却下理由は、同じダイアログの中で 開始日付（× なし・120）と 終了日付（× あり・160）が並んで幅が食い違うこと。そもそも 120 では × が無くても `YYYY-MM-DD` の末尾が切れていた。
- **欄ごとのクリア関数をコンポーザブルへ足す** — 命名があるぶん読みやすい。却下理由は、1行の代入に対して関数が20本以上増えること。代わりに「時刻は `''`、日付は `null`」をソース走査テストが表で固定する。

## Consequences

- **`:model-value` を `v-model` に戻すと静かに壊れる。** 日付欄は × が効かず（コンソールに警告は出るが画面は変わらない）、時刻欄は `null` が入って保存側の未指定判定をすり抜ける。`convention-source-scan.test.ts` の「× を付けた日付・時刻欄は :model-value と @click:clear で結んでいる」が見張る。
- **時刻の `@click:clear` に `= null` と書いてはいけない。** 同じテストが `''` を入れていることを検査する。
- **× を付ける欄・付けない欄は表で持つ。** 必須と任意の日時欄が同じダイアログに並ぶ（TimeIs の `timeis_start_*` と `timeis_end_*`）ので、ref 名の形だけでは決まらない。表は `convention-source-scan.test.ts` の `clearable_date_time_fields`。
- Vuetify 4.3.3 の実装に依存している点が3つある。`hasClear` が `!props.disabled` だけを見るので **`readonly` な見せかけ入力でも × は出る**。× の要素は `v-show="dirty"` なので **空の欄には × が出ない**。`onClear` が `e.stopPropagation()` するので **× を押しても activator のピッカーは開かない**。どれかが変わると、見た目か操作が静かに変わる。
- **日付欄は `min-width` を広げないと値が切れる。** × は欄の内側を削るので、`min-width="120"` のままだと
  `YYYY-MM-DD` の10文字が先頭6文字ぶんしか出ない（E2E で実測: 内側 54px に対して必要 95px。時刻の `HH:mm` は 67px で収まる）。
  **× を付けていない日付欄も 120 では末尾が1文字ぶん切れていた**ので、日付欄は全部 `min-width="160"` に揃えた
  （同じダイアログに 開始日付（× なし）と 終了日付（× あり）が並ぶため、揃えないと幅が食い違う）。
  **時刻欄も同じ理由で 140 が要る** —— 「現在日時」が入れるのは `HH:mm:ss` で、ピッカーが入れる `HH:mm` より長い。
  120 のままだと × 付きの欄で内側 56px に対して必要 75px になり、秒まで入れた直後だけ末尾が切れる
  （ピッカーで入れた `HH:mm` では起きないので気づきにくい）。サイドバーの検索条件は `HH:mm` しか入らないので 120 のまま。
  `convention-source-scan.test.ts` の「日付欄は min-width="160"、ダイアログの時刻欄は 140 で揃っている」が見張る。
- **× のアイコンを 24px のままにすると、上の幅でも入り切らない。** `.v-field__clearable .v-icon` を 18px にして
  欄の内側を稼いでいる（戻すと日付・時刻とも末尾が数px切れる）。
- **広げたぶん、横のボタンが1行に収まらなくなる。** ダイアログ本体の幅は `width: min(500px, 85vw)` 固定で
  （`gkill-global.css` の `.gkill-floating-dialog__body`）、行の内幅は468px（E2E で実測）。入力欄280px に対して
  既定の `v-btn`（左右16px）だと「リセット」「現在日時」で193px になり、合わせて473px で**ボタンだけ次の行へ落ちる**。
  `.gkill-field-side-buttons .v-btn` の左右を10pxにして169pxへ詰めてある。
  **ここを戻すとボタン行が折り返す**（縦に嵩んで「コンパクトにする」という狙いが帳消しになる）
- `CLEAR_TITLE` は検索条件の節単位のクリア（「期間」「時間帯」「キーワード」ほか）で引き続き使う。**「期間」の「クリア」は `VDatePicker` の日付レンジを捨てる唯一の手段**なので撤去できない（日付レンジに × は無い）。
- i18n キーは増えない。× のラベルは Vuetify の `$vuetify.input.clear` が持つ。

## Evidence

実測なし — Vuetify 4.3.3 のソース（`VField.js` の `hasClear` と `[[vShow, props.dirty]]`、`VTextField.js` の
`onClear`、`proxiedModel.js` の controlled 判定）を読んだ上での判断。
置換の網羅は、`<v-text-field>` が結んでいる `*_date_string` / `*_time_string` を全件数えて
（コード由来: 68欄のうち × を付けたのは32欄）表と突き合わせるソース走査テストで固定した。

## Related tests

- `src/client/__tests__/unit/classes/convention-source-scan.test.ts` — 「日付・時刻欄の × は表のとおりに付いている」「× を付けた日付・時刻欄は :model-value と @click:clear で結んでいる」「日付・時刻欄の横に「クリア」ボタンを残していない」
- `src/client/__tests__/unit/composables/query-composables.test.ts` — 「時刻の × で未指定（null）へ戻る」
- `src/client/__tests__/e2e/rykv-period-of-time.spec.ts` — 「時刻の × で開始時刻だけが未指定へ戻る」
- `src/client/__tests__/e2e/mi-operations.spec.ts` — 「期限の × は押した欄だけを空にし、横に「クリア」ボタンが無い」
- `src/client/__tests__/e2e/rykv-calendar-period.spec.ts` — 「期間」の × は従来どおり（この変更では触っていない）
