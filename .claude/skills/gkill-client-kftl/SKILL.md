---
name: gkill-client-kftl
description: "KFTL（メモ帳）の約束。タブ（kftl-tabs.ts / use-kftl-tabs.ts）、保存マーカーの beforeinput/input 対、複数ウィンドウの二重送信防止と送信タブの排他、メモ帳ダイアログの複数枚化（KFTLDialogHost・slot 採番）、KFTL の解釈と書き込みがサーバの1実装（Go）だけであること（TS は行ラベルの分類器、ピンクと確認は /api/parse_kftl_text の応答）を扱う。src/client/classes/kftl/・kftl-tabs.ts・use-kftl-tabs.ts・use-kftl-view.ts・kftl-view.vue・mkfl-view.vue・kftl-dialog.vue・use-kftl-dialog-host.ts・src/server/gkill/api/kftl/ を編集するとき必読。「メモ帳が二重登録される」「別のタブへ保存された」「行ラベルが消える」「タブをクリックしただけで保存が走る」「Web だけ書き間違いが通る」の調査でも必読。"
---

# KFTL（メモ帳）の不変条件

対象: `src/client/classes/kftl/**` / `kftl-tabs.ts` / `use-kftl-tabs.ts` / `use-kftl-view.ts` / `kftl-view.vue` / `mkfl-view.vue` / `kftl-dialog.vue` / `use-kftl-dialog-host.ts` / `src/server/gkill/api/kftl/**`

**このファイルは全文が、実際に起きた事故の再発防止である。該当作業では飛ばさずに読むこと。**
多くは「例外もエラーも出さずに静かに壊れる」種類で、破っても目の前ではエラーにならない。

- `gkill/api/kftl/` — KFTL custom text format parser (single package, no sub-packages). Supports both Japanese (。！？、ーー etc.) and ASCII (#!?,-- ~~ /mi /mood /expense /num /url /start /end /timeis /end? /endt /endt?) prefixes. **解釈（何が正しい入力か）と書き込みはここだけが持つ**（ADR-0507）
- `classes/kftl/` — 行ラベルのための行分類器 (53 statement types; the Go side has 50). Accepts the same Japanese/ASCII prefixes as the Go parser; ASCII constants and match/strip helpers centralized in `kftl-prefixes.ts`. 検証・リクエスト組み立て・API 呼び出しは**持たない**

## 解釈と書き込みはサーバの1実装（2026-09-15、[ADR-0507](../../../documents/adr/0507-kftl-single-implementation-on-server.md)）

2026-09-15 まで KFTL は TS（Web: 解析して `add_*` を tx で fan-out）と Go（Wear / MCP: `/api/submit_kftl_text`）の2実装だった。
ADR-0503 の「`/mood` 単独で気分0を書かない」は Go だけに入り、**Web は3週間、気分0を書き続けた**（2026-09-14 に実測）。
2083 年の打刻・支出の関連時刻がタグに乗らない事故も「片方だけ直した／片方だけ壊れた」型。守ること:

- **TS `classes/kftl/` は行ラベルの分類器だけ。** 検証ルール・リクエスト組み立て・`add_*` 呼び出し・繰り返しの展開を**戻さない**。
  ラベルは打鍵と同時に出す（利用者の要件）が、「おかしな行」のピンクは `/api/parse_kftl_text` の `invalid_lines`
  （打鍵が止まって `KFTL_INVALID_LINE_DEBOUNCE_MS`=300ms 後に1回、世代トークン + AbortController、通信失敗は前回値を保持）
- **`line_label_styles` の watch は `line_label_datas` と `invalid_line_numbers` の両方を見る。** ピンクはサーバから
  打鍵の後に着地するので、ラベルだけを見ていると次にラベルが変わるまでピンクが出ない
- **送信は `do_submit` で「送信対象タブの本文を `parse_kftl_text` → `invalid_lines` があれば止める → `tags` / `mi_board_names` で
  未知タグ・未知板名の確認 → `submit_kftl_text`（`idempotency_key` は送信ごとに新しい UUID）→ `created[]` を
  `fetch_committed_kyou` で引き直して `registered_kyou` / `updated_kyou`」。** 表示用の `invalid_line_numbers` で送信の可否を決めない
  （アクティブタブのもので、しかも遅れて着地する）。**`saved_kyou_by_kftl` は `created[].related_time` の最大値で、引き直し（`get_kyou`）の前に出す**
  —— 板・タグツリーの取り直しがこの合図で走るので、引き直しの後ろへ回すと保存直後に一覧へ移ったとき新しいタグが
  ツリーに無いまま絞られ、記録が見えない（`kftl-submit-emits.test.ts`「引き直しより前に」が固定する）
- **Go の `Analyze`（parse）と `GenerateAndExecuteRequests`（submit）は同じ `prepareRequests` を通す。** 片方の入口にだけ検査を足さない
  （`TestAnalyze_ReportsTheSameInvalidLinesAsExecute` が固定する）。`Analyze` は repos=nil で `expandRepeats` まで回す
  （既存判定は `repositoriesOf` が nil を「既存なし」と扱う）ので、件数の上限も解析で分かる
- **板名は `miBoardNameProvider`（`MiBoardName()`）で Mi / MiReKyou だけが実装する。** `KFTLRequest` インタフェースに足して
  空実装を撒くと、型を足したときに「返し忘れ」がコンパイルエラーにならず確認が黙って抜ける。返すのは**利用者が書いたとおりの板名**
  （既定板へ解決しない。確認ダイアログは書いた名前で聞く）
- **`/end` 系の対象検索は設定の playing 検索条件を通す**（`playingTimeIsQueryFromConfig` + `findPlayingTimeIsEntries`）。
  語の条件は rep の SQL が見るが、**タグ・非表示タグは Kyou 検索の層（`api.FindFilter`）でしか効かない**ので、ハンドラが
  `KFTLStatement.FindKyous` に閉包で渡す（kftl → api の import は作らない）。閉包が無い（テスト・直叩き）ときは語だけで絞る。
  2026-09-15 まで Web だけが条件を適用し、Wear / MCP の `/end` は条件外の打刻も終わらせていた
- **接頭辞を足す・変えるときは Go `kftl_factory.go` と TS `kftl-prefixes.ts` の両方を同じコミットで直す。**
  TS 側がずれても保存は壊れないが、ラベルが嘘になる（`kftl-type-detection.test.ts` が守る）
- **契約は足すだけ。** `SubmitKFTLTextRequest` / `Response` の既存フィールドを変えると Wear（`GkillApiClient.kt`）と MCP（`write-handlers.mjs`）が壊れる。
  `parse_kftl_text` は wrapAuth（repositories 不要。DB を読まないので打鍵のたびに呼ばれても軽い）
- 守るテスト: `kftl-submit-emits.test.ts`「サーバが不正行を返したら送信せず…」「おかしな行の表示」/
  Go `kftl_analyze_test.go` / `handle_parse_kftl_text_test.go`

**Go 側のエラーは行ごとに返し、入力ミスとサーバ障害を分ける**（2026-08-24、[ADR-0502](../../../documents/adr/0502-kftl-errors-are-per-line.md)）。`kftl_statement.go` は3フェーズ直列で、**行をリクエストへ適用するフェーズまでは1バイトも書かない**。だからそこは最初の1件で止めず**全行を評価して `errors.Join` で束ねる**（利用者が1往復で全部直せる。TS 側は元からそうなっていて Go だけが遅れていた）。実行フェーズは書き込みが起きるので最初の失敗で止める。振り分けは `KFTLInputError` と `errors.As` で、入力ミスは `ERR000416`(**400**)・サーバ障害は `ERR000351`(500)。**打ち間違いを 500 で返さないこと** —— ステータスを見る層からサーバ障害と区別できなくなる。応答の `created[]` は**書き込みが成功した直後にリクエスト側が控えた**もので、`requestMap` を事前に列挙して作ってはいけない（支払いの後ろの空行が作る空の Nlog は何も書かずに成功し、打刻の終了は既存レコードの更新なので嘘になる）。失敗したら何も残らない（`CommitTx` の1トランザクション。ADR-0219）。多言語キーは各失敗に1対1で7言語ぶん既にあるので**新設しない**（例外は ADR-0508 の「付け先の無いメタ情報」1件）。`api/kftl/` に .go を足したら `api/README.md` の件数を同じコミットで直す（`verify_docs` がファイル数を数えている）。

**KFTL（メモ帳）のタブ**（2026-08-16）。`kftl-view.vue` がタブのホストで、`/kftl` ページ・各画面のメモ帳ダイアログ（`kftl-dialog.vue`）・打刻メモ帳（`mkfl-view.vue`）の**3系統すべて**に効く。純関数は `classes/kftl-tabs.ts`、状態は `classes/use-kftl-tabs.ts`。守るべき約束:
- **`v-window` を使わず、アクティブなタブ1枚だけを描画する。** 非表示の textarea は `clientWidth` が0になり、`kftl-statement-line.ts` の `1 + parseInt(text_width / 0)` が **`NaN`**（`Infinity` ではない）を返して行ラベルが丸ごと消える
- **タブの一覧と中身はモジュールシングルトン、いま映しているタブはビューごと。** メモ帳ダイアログは複数枚開けるので、インスタンスごとに配列を持つと片方の古い配列で localStorage を丸ごと上書きしてもう片方のタブが消える（単一文字列だった頃は「上書き」で済んでいた）。逆に `active_tab_id` までストアに置くと、全ウィンドウが同じタブを映して並べて見られない。ストアが持つのは `last_active_tab_id`（次に開くウィンドウの初期表示）だけ。単体テストは `reset_kftl_tabs_for_test()` + `localStorage.clear()` を `beforeEach` で呼ぶこと
- **タブが消えたらビューは追随する。** 別のウィンドウがタブを閉じたり保存で閉じたりすると、そのタブを映していたビューの `active_tab_id` が宙に浮く。`watch(() => tabs.map(t => t.id))` で新旧を比べ、旧インデックスへクランプして隣のタブへ移す（放置すると本文が空文字を返し続ける）
- **送信対象タブは `do_submit(target_tab_id, ...)` の引数で渡す。** `do_submit()` は未知タグ確認・未知板名確認でいったん抜けて応答を待つので、1回の保存で2〜3回呼ばれる。gkill のフローティングダイアログは非モーダル（`App.vue` の `.gkill-float-scrim` が `pointer-events: none`）なので、確認中でも背後のタブバーは押せる。持ち越し用の `submit_target_tab_id` を**引数ではなく直接読ませてはいけない** ―― 確認ダイアログを Escape / ブラウザバックで閉じられると古い値が残り、次の保存が別のタブへ誤配送される。新しい送信（`submit()`）は必ずアクティブなタブを渡す
- **タブ操作のロックは `is_submitting || show_confirm_unknown_tag_dialog`。** `is_requested_submit` は設定の読み込みが終わるまで `true` なので鍵に使えない（起動直後にタブを追加できなくなる）。板名確認をロック条件に入れてもいけない ―― `unknown_mi_boards` はブラウザバックで閉じても空にならないので**永久ロック**になる（タグ確認は `useDialogHistoryStack` がどの閉じ方でも false にするので安全）
- **不正行の判定は送信対象タブから引き直す。** 表示用の `invalid_line_numbers` はアクティブタブのもので、しかも await をまたいで遅れて着地する。これを送信の可否に使うと、タブを切り替えた直後の保存が「おかしな行があります」で止まる
- **保存マーカーの判定は `beforeinput` で控えた本文と `input` 時点の本文を比べ、「確定したマーカー行が増えたか」で行う（`count_save_marker_lines`）。watch（本文の変化）に置いてはいけない。** watch は `flush: 'post'` で中間の値を観測しないうえ、**同じ `input` イベントのリスナー間でマイクロタスクが走る**ので `@input` より先に新しい本文を見てしまう（IMEでは必ず起きる）。「末尾がマーカーか」で見るのも不可。理由の詳細と却下案は [ADR-0501](../../../documents/adr/0501-save-marker-beforeinput-input-pair.md)。「確定した」= その行の後ろに改行がある、なので `！` を打った時点では走らない。守るテストは `kftl-submit-emits.test.ts` の「KFTLの保存マーカー」節と `e2e/kftl-tabs.spec.ts` の「IMEで確定してから改行しても自動で保存される」（**IMEはCDPの `Input.imeSetComposition` でしか再現しない**。`pressSequentially` は打鍵ごとにイベントループが回るので中間の本文を必ず観測してしまい、常に緑になる）
- **保存マーカーによる自動送信の入口は「利用者が選んだ操作」の2つだけ ―― textarea の `@input` と、テンプレート貼り付け。** 判定関数（`maybe_submit_by_save_marker`）は1つのまま、入口だけ2つにすること。`text_area_content` はアクティブタブへの computed なので、判定を watch の内容変化そのものに戻すとタブ切替・localStorage からの復元でも発火し、末尾にマーカーが残ったタブを**クリックしただけで保存が走る**。watch は `@input` が立てた印（`user_input_tab_id`。立てるのは `onTextAreaInput()` **だけ**）が付いているときしか判定しない。**テンプレートはこの印に相乗りさせず `paste_template()` から直接呼ぶ** ―― watch は `new_value === old_value` で早期returnするので、貼る前のタブの本文がテンプレートと同一文字列だと黙って発火しない（タブ化する前も同じ理由で取りこぼしていた）。さらに watch は `flush: 'post'` かつ await を挟むので、判定までにタブを切り替えられると「印のタブ == アクティブタブ」が偽になってこれも黙って落ちる
- **同じタブを2枚のウィンドウが同時に保存しても、登録は1回。** `is_submitting` / `is_requested_submit` は**ビューごと**なので、同じタブを映した別ウィンドウの保存ボタンは止められない（KFTLはtxで束ねて送るので二重送信するとKyouが丸ごと重複する）。送信中のタブidは共有ストアが持ち、`do_submit` の冒頭で `tabs_store.try_begin_submit(target_tab_id)`、`finally` で `end_submit(target_tab_id)`。**claim は `is_requested_submit` ガードの「後」かつ `try` の「外」で取る** ―― 前に置くとガードの return が `finally` を通らず永久ロック、`try` の中で取ると掴めなかった側の `finally` が**勝ったウィンドウの分を解放**する。確認ダイアログで抜けるときは手放し、`confirm_submit` / `confirm_mi_board_submit` からの再入で取り直す（**持ち越すと自己デッドロックする**）。この印はリアクティブにせず localStorage にも出さない（`is_tab_locked` に混ぜると「送信中のタブから切り替えられない」、永続化するとリロードで掴んだままのタブが二度と保存できなくなる）。テンプレート経路は毎回一意な新しいタブを作り、それをアクティブにするのは貼ったウィンドウだけなので、この排他とは無関係に重複しない
- `update_line_labels()` / `refresh_invalid_lines()` は **await をまたぐので世代トークンで最後の1回だけ書き戻す**（前のタブぶんの結果が後から着地して行ラベルが化けるのを防ぐ）
- タブのストアは**独立した `effectScope(true)` の中で作る**。setup の中で素に `watch` を張ると最初に呼んだコンポーネントのスコープに属し、そのコンポーネントが unmount された時点で**永続化ごと止まる**（メモ帳ダイアログを閉じる／画面を移るで再現する）
- DOM は id 引きではなく**テンプレート ref** で掴む（`TextAreaInfo.text_area_element`）。textarea の `id` は複数枚で重複しないよう `useId()` で採番するので、**E2E はクラス `.kftl_text_area` で掴む**
- タブ列の `×` は `v-btn` ではなく `v-icon` にする（`v-tab` は `<button>` を描画するので `<button>` が入れ子になる）。`@click.stop` が無いと閉じると同時にそのタブがアクティブ化される
- **タブ列はタイトル行に同居させる**（別の行にするとテキストエリアの縦が40px削られる）。`resize()` が引くのは `title_height` だけで、`tab_bar_height` は `v-tabs` の `:height` に渡すためだけの定数。`title_height` の実寸は `.kftl_title` の CSS で固定する（`v-card-title` に `height` prop は無いので、渡すだけでは効かない）。測った値をフィードバックすると `kftl-dialog.vue` の ResizeObserver が縮小ループに入る
- 守るテスト: `kftl-tabs.test.ts` / `kftl-tab-store.test.ts` / `kftl-submit-emits.test.ts` の「KFTLのタブ」「KFTLを複数のウィンドウで開く」/ `e2e/kftl-tabs.spec.ts`

**メモ帳ダイアログの複数枚化**（2026-08-16）。5画面（rykv / mi / dashboard / saihate / playing）は `<KFTLDialogHost>` を1個置くだけで、＋メニューを選ぶたびにウィンドウが増える。呼び出し側は従来どおり `kftl_dialog.value?.show()` のまま（`show()` の意味が「開く or 再フォーカス」から「1枚増やす」に変わった）。配列の持ち方は `rykv-dialog-host` と同じ。守るべき約束:
- **`useFloatingDialog` のキーはウィンドウごとに分ける。** キーは `${key}:pos` / `:size` / `:transparent` の保存先そのものなので、同じキーで複数枚出すと位置とサイズを奪い合う。スロット番号（空いている最小の番号）で `kftl-dialog` / `kftl-dialog-2` … と分け、`centerOffset` に `slot_index * 28` px を渡してずらす（全ダイアログが `centerMode: "always"` なので、ずらさないとピクセル単位で完全に重なる）
- **z-index は「開いているダイアログの並び順」から出す。単調増加のカウンタにしてはいけない** ―― Vuetify の overlay（メニュー / ツールチップ）が 2400 なので、上へ伸ばし続けるとダイアログの中のメニューが下へ潜る。伸びるのは同時に開いている枚数ぶんだけ
- **前面化は自分と子孫をまとめて上げる。** 確認ダイアログは `Teleport to="body"` で親の**兄弟**になるので、素朴に前面化すると親をクリックしただけで確認が後ろへ隠れる。親子は `provide`/`inject` で持つ（コンポーネント木は Teleport をまたいでも保たれる）
- **バックと Escape が閉じるのは「見た目の最前面」。** `use-dialog-history-stack.ts` の `stack` は積んだ順なので、前面化で見た目とずれる。ずれたままだと奥のダイアログが閉じる。前面化のたびに `raise_dialog_history_entries()` で履歴エントリも同じ順へ並べ替える。2つのコンポーザブルは**同じコンポーネントの setup で呼ばれる**ので、`getCurrentInstance()` を鍵にして結んでいる（`stack` の長さは変わらないので履歴の深さ計算は無傷）
- `closed` は `useDialogHistoryStack(is_show_dialog, { onClosed })` から出す。×・Escape・ブラウザバックのどれでも1回だけ上がるので、ホストはこれ1本で一覧から外せる
- ホストは `v-for` でルートが複数になる。呼び出し側が渡してくる Kyou 系の中継束のうちメモ帳が出さないイベントは行き場が無いので、`defineOptions({ inheritAttrs: false })` で黙って捨てる（従来と同じ挙動）
- E2E で `clickFabButton()` を使ってはいけない ―― 先に `dismissFloatingDialogs()` を呼ぶので、開いているメモ帳ウィンドウを閉じてしまい枚数が増えない
- 守るテスト: `floating-dialog-z-order.test.ts` / `kftl-dialog-host.test.ts` / `e2e/kftl-multi-dialog.spec.ts`

## 書式ミスは黙って通さない

プレフィックスの判定は**完全一致**（`kftl_factory.go` の `generateDefaultConstructor`）で、値は次の行に書く。外したときは**書き込みの前に**行別エラーへ倒すこと（[ADR-0503](../../../documents/adr/0503-kftl-prefix-misuse-is-an-input-error.md)）。

- **単独プレフィックス**（次に値の行が無い）は `requireNextLineText` で弾く。放置すると `/mood` 単独が**気分値 0（最低）の記録を黙って1件書き**、`/num` 単独が空の数値記録を書く。他は無言で0件になる
- **保存マーカー「！」の行は「次の行」に数えない。** `generateKFTLLines` は `strings.Split` の直後に最初のマーカー行（1行目は除く）で本文を切り詰めてから `NextStatementLineText` を組み立てる。ループ末尾の break に戻すと、Web の「！」で保存する経路だけ `ーち`+「！」が `requireNextLineText` を素通りしてタイトル空のまま `DoRequest` に届く（2026-09-15 の利用者報告。`ーら`+「！」は気分値0を書き、`ーか`+「！」は 500 だった。[ADR-0508](../../../documents/adr/0508-kftl-blank-records-are-input-errors.md)）
- **内容の無い記録と付け先の無いメタ情報は、書く前に `validateRequestContents` が行別エラーにする。** `KFTLRequest.ValidateContent()` は基底に既定実装を置かない（型を足したら「何を空とみなすか」を書かないとコンパイルが通らない）。`prepareRequests` が全行を適用した後・`expandRepeats` の前に呼ぶので Analyze（ピンク）と送信で同じ結果になる。`DoRequest` の `title == "" → return nil` に戻さない —— 空のメモ・打刻・タスク・ブックマークは旧 Web の `ERR9000xx` と同じ文言で**送信全体が止まる**（`メモ`,`、`,空行 は先頭のメモも書かれない）。kmemo の空は「全行が空白」（`joinLines` は `["",""]` を改行1文字にする）。残ったプロトタイプ（`。タグ` だけ・`？時刻` だけ・`ーー` だけ）は `KFTL_META_INFO_NO_TARGET_MESSAGE_TITLE`。例外は `？時刻` の直後の `ーん` で、関連時刻を取り込んだあと `KFTLRequestMap.Delete` で外す（残すと誤爆する）
- **予定日時欄（Mi / MiReKyou の見積開始・見積終了・期限）の、空でないのに読めない行は入力エラー**（`parseScheduleFieldTime`。ADR-0505 が据え置いた部分を ADR-0508 で改めた）。空行だけが「未設定」。`、` で6行を埋めずに次の記録へ移ると、以前は `、` が見積開始に食われ後ろの本文も残りの欄に黙って消えていた
- Web は本文が空白だけなら `parse_kftl_text` を投げずピンクを消す（`fetch_invalid_lines`）。投げると**新しい空のタブの1行目が常にピンク**になる。送信はサーバに聞くので空のタブで保存を押せばエラーが出る
- **プレフィックス＋同じ行の引数**（`/mood 8`）は `prefixWrittenWithArgument` で弾く。完全一致判定なので本文へ落ち、気分記録のつもりが本文「/mood 8」のメモ1件になっていた
- 判定は**長いプレフィックスから**見る（短い側からだと `/end?` が「`/end` に引数 `?`」に化ける）。タグ（`。`/`#`）と関連時刻（`？`/`?`）は**前方一致で受理する設計なので対象に入れない** —— 入れると `# 見出し` や `?` 始まりの英文が壊れる
- 検査は `ApplyThisLineToRequestMap` のフェーズで行う。**まだ1バイトも書いていない**ので全行を評価して束ねられる。`DoRequest` まで持ち越すと前の行は既に書かれている
- 打ち間違いは `newKFTLInputError` を使う。`fmt.Errorf` のままだと ERR000351（HTTP 500）＋英語の生文言になる。**メッセージIDは既存の i18n キーを探してから足すこと**（キー追加は7言語 + Go の embed コピー + 件数を書いた資料4箇所に波及する）
- **Mi / MiReKyou の予定日時欄（見積開始・見積終了・期限）で行頭の `？`/`?` を剥がさない。入力エラーにする。** 剥がすと、残りがパースに失敗しても未設定として握り潰されるので、`？18:00` の打ち間違いも `？？`（繰り返しブロック）の書き損じも**エラーも警告も出ないまま日付だけが入らない**。Mi 本体は作られるので保存が成功したように見える。判定は Go `parseScheduleFieldTime` / TS `parse_schedule_field_time` の**1箇所ずつに集約**する（6欄が同じ形で壊れていた）。TimeIs（`ーち`）の時刻行は `？` を剥がすままだが、そちらはパース失敗が既に行エラーなので黙って壊れない。却下案は [ADR-0505](../../../documents/adr/0505-schedule-time-field-rejects-related-time-prefix.md)
- **`api/kftl/` に .go を足したら、`api/README.md` の件数と `kftl/README.md` の表を同じコミットで直す。** `verify_docs.mjs` がファイル数を数えて突き合わせる（**テストファイルも数に入る**）

**KFTL の実行フェーズの失敗でも、原因が「利用者が直せる状態」なら `newKFTLInputError` に載せる。** `fmt.Errorf` のままだと `kftl_statement.go` の `errors.As` に引っかからず、`ERR000351`（HTTP 500）の「メモ帳のテキストの記録に失敗しました」だけが返って**行番号も理由も出ない**。2026-08-25 の実利用レビューは `~~`（リポストタスク）がこれで3回とも同じ文言で落ち、原因を MCP 経路の不具合と誤診した（実際は繋いだアカウントに `mirekyou` 型の rep が1件も無く、Web UI からでも同じく失敗する状態だった）。**`MessageID` を空にしないこと** —— 空だと `formatKFTLInputErrorMessage` が `Cause` の英文をそのまま応答へ載せ、利用者IDと端末名が漏れる（[ADR-0707](../../../documents/adr/0707-redact-environment-specific-strings.md)）。境界と却下案は [ADR-0504](../../../documents/adr/0504-kftl-missing-configuration-is-an-input-error.md)。

**`~~` は既存レコードをタスク化できない。** 対象IDは `ctx.ThisStatementLineTargetID` ＝**同じ送信テキストの直前の行が採番したUUID**で、gkill に既にある記録を指す構文は無い。ツール説明にそう書き戻さないこと。

**KFTL 経由で書いた記録は `create_app="gkill_kftl"` / `create_device=<サーバのdevice>`。** MCP から書いても同じで、手打ちのメモ帳と**区別する欄が無い**（`gkill_add_*` は `gkill_mcp_readwrite` / `mcp`）。`create_apps:["gkill_mcp_readwrite"]` を「MCP で作った記録の探し方」と案内しないこと。例外は Wear OS で、companion が `SubmitKFTLTextRequest` の任意項目 `create_app` に `gkill_wear` を載せるので `create_app="gkill_wear"` になる（2026-09-11〜。サーバは空なら `gkill_kftl` に落とす。詳細は [gkill-mobile](../gkill-mobile/SKILL.md)）。失敗した送信は何も残らない（確定は `CommitTx` の1つのトランザクション、ADR-0219）のでそのまま送り直せばよいが、**成功した送信の再送**（応答だけ落ちた等）を畳むために `idempotency_key` を同じ値で渡す（受け口は `SubmitKFTLTextRequest`。渡さないと丸ごと二重登録になる）。

## 繰り返し「？？」

`？？` の単独行で開いて同じ記号で閉じる4行ブロック（条件・回数/終了日・既存時・起点）。
**直前に書いた記録を日付を変えて何度も作る。** 設計と却下案は [ADR-0506](../../../documents/adr/0506-kftl-repeat-block-expands-into-records.md)。

- **展開（複製の生成）を行の解釈でやらない。`prepareRequests` の最後（実行ループの直前）だけ。** 行の解釈に置くと
  「？？」をブロックのどこに書くかで結果が変わる。展開は Go だけが持つ（TS には無い。戻さない）
- **複製ではレコードIDと「テキストのID」を採り直す。** テキストIDを使い回すと同じIDのテキストを回数ぶん書くことになり、
  append-only なので**最後の1件以外が黙って消える**
- **`？？` の判定は `？` の前方一致より前に置く。** 後ろだと `？？` が関連時刻行に食われ、
  「？」を1つ剥がした残りのパース失敗になる。完全一致だけをブロックの開始にし、
  `？？ 金 3` のように引数を同じ行に書いたものは記号の書き方のエラーへ倒す
- **4行を書き終えたあとの位置は受け皿。行ラベルは「`**********`」を並べ、空行は見逃す。**
  行ラベルの先読み（`generate_line_label_data`）は次の行のコンストラクタがある限り空行を
  上限50行ぶん組み立てるので、ここを「繰り返し↓」にすると**書いてもいない行のラベルで列が埋まる**
  （`～～` で同じ事故が起きて `KFTLMiReKyouNoneStatementLine` を足したのと同じ）。
  ラベルが「何も無い行」と言う以上、空行でエラーにもしない。テキストの行は今までどおり
  `KFTL_REPEAT_NOT_CLOSED_MESSAGE_TITLE` で弾く（飲み込むと閉じ忘れたときに本文が繰り返し指定に化ける）。
  エラーは Go、ラベルは TS —— **ラベルの受け皿（TS）とエラーの受け皿（Go）を同じ位置に置くこと**。
  ずれると「ピンクなのにラベルは何も無い行」になる。受け皿はブロックの中に留まるので、空行のあとの `？？` は今までどおり閉じる
- **リポストタスク（`～～`）の中では付け先を明示する。** ブロックの中の `target_id` は
  「タスク化される元の記録」を指していて、リポストタスク自身は別のIDで登録されている。
  引かせると**元の記録のほうが繰り返される**（タグ行が `req` を持ち回っているのと同じ理由）
- **支出（`ーん`）はブロック全体で1グループ。** 店名・関連時刻と同じ扱いで、
  `kftlNlogRequest.SetRepeatSpec` がブロックへ流す。複製では**ブロックも複製する** ――
  共有したままだと回ごとにずらした関連時刻を最後の1回が上書きして全レコードが同じ日時になる
- **繰り返せない型を開けない。** `ーた`（打刻開始のみ）は end_time の無い打刻をN個作り、
  終わりの無いスタンプが以降の全記録を覆う。終了4種（`ーえ` `ーいえ` `ーたえ` `ーいたえ`）は
  「いま走っている1件」しか見ないので繰り返す意味がない。どちらも `SetRepeatSpec` で断る
- **`CloneForRepeat` の既定実装を基底に置かない。** 置くと新しい型で
  実装を忘れても通り、**日時のずれない複製が黙って書かれる**。コンパイルエラーで気づく形を保つ
- **既存判定（3行目の既定 `no`）の照合キーは書き込みと同じ値を使う。** 板名は既定板への解決を
  同じ関数（`resolvedBoardName`）に通すこと。ずれると毎回重複する
- **存在しない日は飛ばす。丸めない。** `毎月31` の2月、`第5金` の無い月。最も近い日へ丸めると
  `毎月30` や `第4金` と重複して二重に作られる
- **打刻（`ーち`）のアンカーは related_time（基底の `AnchorTimeForRepeat`）。** 2026-09-10 まで TS 側は
  `start_time`（`do_request` まで空）を基準にしていて、1970 からの日数ぶん（約 20,700 日）ずらされ、
  **Web から送った打刻 5 件が 2083-05-17〜 で登録された**（エラーも警告も出ない。Go は起きなかった）。
  TS の展開は消えたが、**アンカーの欄・ずらす欄・DoRequest が書く欄は同じ欄を指す**という約束は Go でも同じ。
  8 型を監査した結果（2026-09-10）: kmemo / lantana / kc / urlog は related_time、支出はブロックの related_time、
  タスク / リポストタスクは予定3欄、打刻は related_time（→ start_time）+ end_time で一致。`ーた` と終了4種は繰り返し自体を断るので起きない。
  新しい型を足すときは **DoRequest が実際に書く値**を `handle_submit_kftl_text_test.go`
  `TestHandleSubmitKFTLText_RepeatWritesShiftedTimes` の表へ足すこと（request オブジェクトの欄だけ見る表では捕まらない）。
  守るテスト: Go `TestExpand_TimeIsRepeatShiftsStartAndEndTogether`（年を明示的に見る）
- **Go の `doBaseRequest` へ関連時刻は引数で渡す（`r.GetRelatedTime()`）。基底の中で `b.GetRelatedTime()` を引かない。**
  埋め込み基底のメソッドは外側の override を見ない（Go は仮想ディスパッチしない）ので、支出（`ーん`）がブロック共有の時刻を
  override で返していても基底で引くと `？`行の時刻がタグ・テキストに乗らず「今」で書かれる（2026-09-10 まで TS 側は届いていたので
  Web と Wear / MCP で結果が違っていた —— 2実装だった頃の典型）。
  守るテスト: Go `TestHandleSubmitKFTLText_RepeatWritesShiftedTimes`「支出の関連時刻がタグにも乗る」

## 関連スキル

- [gkill-client-foundation](../gkill-client-foundation/SKILL.md) — 必ず併読（中継束・ダイアログ規約）
- [gkill-client-rudbeckia](../gkill-client-rudbeckia/SKILL.md) — メモ帳ウィンドウの slot 採番はアプリ全体で1つ
- [gkill-client-tags](../gkill-client-tags/SKILL.md) — 未知タグ確認の共有部品と確認の順序

## 詳しい設計と却下案（ADR）

- [ADR-0501 保存マーカーは beforeinput/input の対で](../../../documents/adr/0501-save-marker-beforeinput-input-pair.md)
- [ADR-0502 メモ帳の失敗は行ごとに返す](../../../documents/adr/0502-kftl-errors-are-per-line.md)
- [ADR-0503 引数の書き方を誤ったプレフィックスは行別エラー](../../../documents/adr/0503-kftl-prefix-misuse-is-an-input-error.md)
- [ADR-0504 実行フェーズでも設定不足は行別の入力エラー](../../../documents/adr/0504-kftl-missing-configuration-is-an-input-error.md)
- [ADR-0505 予定日時欄では関連時刻の接頭辞「？」を入力エラーにする](../../../documents/adr/0505-schedule-time-field-rejects-related-time-prefix.md)
- [ADR-0506 繰り返し「？？」は実体のレコードへ展開し、複製は送信時にだけ作る](../../../documents/adr/0506-kftl-repeat-block-expands-into-records.md)
- [ADR-0507 メモ帳の解釈と書き込みはサーバの1実装に寄せる](../../../documents/adr/0507-kftl-single-implementation-on-server.md)
- [ADR-0508 保存マーカー行は値の行に数えず、内容の無い記録・付け先の無いメタ情報・読めない予定日時は書く前に行別エラーにする](../../../documents/adr/0508-kftl-blank-records-are-input-errors.md)
