# ADR-0507: メモ帳（KFTL）の解釈と書き込みはサーバの1実装に寄せ、Web は行ラベルだけを手元で出す

| | |
|---|---|
| Status | Accepted |
| Date | 2026-09-15 |
| Sources | 2026-09-13 の外部レビュー（「KFTL の二重実装はコンパイラを2個保守しているのと同じ」）と 2026-09-14 の実測。[ADR-0503](0503-kftl-prefix-misuse-is-an-input-error.md) / [ADR-0506](0506-kftl-repeat-block-expands-into-records.md) / [ADR-0219](0219-commit-tx-is-one-sqlite-transaction.md) |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/api/kftl/kftl_statement.go`（`prepareRequests` / `Analyze`）/ `src/server/gkill/api/gkill_server_api/handle_parse_kftl_text.go` / `src/server/gkill/api/kftl/kftl_timeis.go`（`playingTimeIsQueryFromConfig` / `findPlayingTimeIsEntries`）/ `src/client/classes/use-kftl-view.ts` / `src/client/classes/kftl/`（行ラベルの分類器のみ） |

## Context

メモ帳（KFTL）は1つの言語仕様に対して実装が2つあった。

| 経路 | 解釈 | 書き込み |
|---|---|---|
| Web | TypeScript `src/client/classes/kftl/`（5,830行・53行種別） | ブラウザから `add_*` を tx で束ねて fan-out |
| Wear OS / MCP | Go `src/server/gkill/api/kftl/`（5,099行・50行種別） | `/api/submit_kftl_text` |

共有のゴールデンコーパスは無く、テストも別々（Go 3,780行 / TS 4,600行）だった。
結果として「片方だけ直した／片方だけ壊れた」が繰り返し起きていた。

- [ADR-0503](0503-kftl-prefix-misuse-is-an-input-error.md)（2026-08-24）は「`/mood` 単独で気分0を書かない」を
  **Go だけ**に入れた。2026-09-14 に TS 側を探針テストで流すと `invalid=[]`・`mood: 0` ―― 一番使う経路の
  Web は3週間、気分0を書き続けていた（`/mood 8` もただのメモになっていた）
- 2026-09-10 の「打刻 5 件が 2083 年で登録された」は TS 側だけの事故（アンカーの欄が Go と違った）
- 「支出の `？`行の時刻がタグに乗らない」は Go 側だけの事故（埋め込み基底の仮想ディスパッチ）
- ADR-0502 / 0504 はどちらも「TS 側は元からそうなっていて Go だけが遅れていた」と書いている

一方、TS 側のパーサは送信時のコンパイラであるだけでなく、**打鍵のたびに走るエディタの前段**でもあった。
行ラベル（「メモ」「タスク」「板名」… と、まだ書いていない行の先読み50行）、「おかしな行」のピンク、
送信前の未知タグ・未知板名の確認は、すべて TS のパース結果から出ていた。
利用者の要件として、行ラベルは従来どおり打鍵と同時に表示されなければならない。

## Decision

- **「何が正しい入力か」「何を書くか」は Go の1実装だけが持つ。** Web の送信は `/api/submit_kftl_text`
  1往復に変え、TS 側の解釈・リクエスト組み立て・`add_*` fan-out・tx・繰り返しの展開・既存判定を消す
  （13 のリクエストクラス・全行クラスの `apply_this_line_to_request_map` と、消したファイル:

  ```
  kftl-request.ts / kftl-request-base.ts / kftl-request-map.ts
  kftl_repeat/kftl-repeat-expand.ts / kftl_repeat/kftl-repeat-duplicate.ts / kftl_prototype/kftl-prototype-request.ts
  ```
  ）
- **TS `src/client/classes/kftl/` は行ラベルのための分類器としてだけ残す**（行の種別と次の行の決め方、
  ラベル文字列の決定に要る読み方だけ）。ラベルは打鍵と同時に出る
- **「おかしな行」のピンクと未知タグ・未知板名の確認は、新設の `POST /api/parse_kftl_text`（解析のみ、書かない、wrapAuth）
  の応答で行う。** 打鍵が止まって 300ms 後に1回投げ（世代トークン + AbortController）、通信に失敗したら前回の表示を残す。
  保存の直前には送信対象タブの本文で改めて投げ、`invalid_lines` があれば送らない
- **parse と submit は Go の同じ `prepareRequests`（行の解釈 → 全行の適用 → 繰り返しの展開）を通す。**
  解析で通った入力が送信で弾かれることも、その逆も起きない
- `/end` 系の対象検索は設定の playing 検索条件（`playing_timeis_json_data`）を Go でも適用する。
  タグ・非表示タグは Kyou 検索の層でしか効かないので、ハンドラが `api.FindFilter` を閉包で kftl に渡す
- Wear OS / Android / MCP の契約は変えない（`SubmitKFTLTextRequest` / `Response` は既存のまま。`created[]` に
  `related_time` を**足す**だけ —— Web が「、、」でずらした分を実行中画面へ渡す `saved_kyou_by_kftl` を、引き直しを
  待たずに従来どおり保存直後に出すため）

## Rejected alternatives

- **Go のパーサを WebAssembly にしてブラウザでも同じコードを動かす** — 唯一「1実装のまま、ラベルも判定も手元で即時」を
  満たす案。だが `api/kftl` は `dao/reps`・`dao/sqlite3impl`・`dao/reps/cache` を import しており（`go list -deps` で
  424 パッケージ、sqlite 系 17 件）、解析と実行を別パッケージに割る大工事なしには `js/wasm` でビルドできない。
  将来の候補として残す
- **行ラベルもサーバの dry-run で出す** — TS のパーサを丸ごと消せる。だが利用者が「行ラベルは従来どおりリアルタイム」を
  要件として明示した。ネットワーク往復ぶん遅れ、オフラインでは出ない
- **2実装のまま共有ゴールデンコーパス（同じ KFTL テキスト → 期待レコード列）で同期する** — 乖離を赤にできるが、
  「片方だけ直す」構造そのものは残る。移行後は判定が1つなので、コーパスは要らない
- **TS 側に検証（apply）を残して、ピンクを手元で即時に出す** — ピンクは即時になるが、検証ルールが2箇所に残り続ける。
  ユーザー判断でサーバ判定を採った（ピンクだけが 300ms + RTT 遅れる。ラベルは即時のまま）
- **Web だけ「空のタイトル/URL/本文は送信エラー」を TS で持ち続ける** — Go 側の `requireNextLineText`（ADR-0503）が
  「次の行が空白だけ」も弾くので、Web を Go に寄せても黙って0件にはならない。TS 独自の検査は不要
- **サーバの KFTL に tx を足して部分保存を防ぐ** — 計画時点では Go 側が実 rep へ直書きで部分保存が残る設計だったが、
  同日の [ADR-0219](0219-commit-tx-is-one-sqlite-transaction.md) で Go 側 KFTL も temp rep + `CommitTx`（1つの SQLite
  トランザクション）になり、失敗したら何も残らない。Web が tx を手放しても原子性は落ちない
- **`context.WithoutCancel` で書き始めたら止めない** — Web が1000件を1往復で送るようになるので途中の切断を心配したが、
  ROLLBACK で何も残らない今は、切断で止まるほうが安全（続けると利用者は失敗と思って再送し、二重登録になる）

## Consequences

- Web からの書き込みは Wear OS / MCP と同じ Go の経路になり、KFTL の判定ルールを足す場所は Go だけになった
- TS `classes/kftl/` は 83 ファイル 5,830 行から 66 ファイルへ縮み、API を1つも呼ばない。
  行の種別（接頭辞と次の行の決め方）だけは Go と揃える必要が残る（ずれてもラベルが嘘になるだけで、書き込みは壊れない）
- ピンク表示は打鍵後 300ms + RTT だけ遅れる。オフラインでは前回の表示のまま
- 1000件の繰り返しは「ブラウザから直列1000往復」から「1往復」になった
- 日時の解釈はブラウザのタイムゾーンではなくサーバのそれになる（Wear / MCP と同じ）
- Web の `/end` で設定の playing 条件が効くのは従来どおり。Wear / MCP でも同じ条件で対象を探すようになった
- 新しいエンドポイント `parse_kftl_text`（ERR000420 / ERR000421）が1本増えた。Web が叩くエンドポイントに
  `submit_kftl_text` と `parse_kftl_text` が加わり、`gkill-api.test.ts` の「Web 以外専用」許容リストから `submit_kftl_text` が外れた
- `saved_kyou_by_kftl` は `created[].related_time` から計算し、引き直し（`get_kyou`）の前に出す。実装の途中で
  引き直しの後ろへ動かしたところ、保存直後に一覧へ移る E2E（`edit-dialog-crud`）が「新しいタグがツリーに無いまま
  絞られて記録が見えない」で不安定になった —— 板・タグツリーの取り直しがこの合図で走るので、順序は約束の一部

## Evidence

- 2026-09-14 の探針（vitest で TS パーサに `/mood` と `/mood 8` を流した）:
  `{"invalid": [], "kinds": ["KFTLLantanaRequest"], "mood": 0}` / `{"invalid": [], "isKmemo": true}`。
  ADR-0503 の Anchors は全部 Go 側で、TS には対応する検査が無かった
- `go list -deps ./gkill/api/kftl | wc -l` = 424、`grep -c -i "sqlite\|modernc"` = 17（WASM 化の却下根拠）
- Go 側の Apply フェーズは `Repositories` を一切参照しない（参照は `DoRequest` / `FindExistingForRepeat` /
  打刻終了の `FindTimeIs` の中だけ）ので、解析専用の入口は DB 無しで安全に組める

## Related tests

- `src/server/gkill/api/kftl/kftl_analyze_test.go`
  - `TestAnalyze_ReportsTheSameInvalidLinesAsExecute`（parse と submit で同じ行エラー集合）
  - `TestAnalyze_ListsTagsBoardsAndCount` / `TestAnalyze_ListsTagsAndBoardsInsideBlocks`（確認に使うタグ・板名）
  - `TestAnalyze_ExpandsRepeatsWithoutRepositories` / `TestAnalyze_BlankKmemoIsNotAnError`
  - `TestStatement_PrefixFollowedByBlankValueLineIsInputError`（空白だけの値の行）
  - `TestPlayingTimeIsQueryFromConfig` / `TestFindPlayingTimeIsEntries_TagFilterGoesThroughFindKyous`（`/end` の設定条件）
- `src/server/gkill/api/gkill_server_api/handle_parse_kftl_text_test.go`（HTTP 200 のまま `invalid_lines`、何も書かれない、空は `[]`）
- `src/client/__tests__/unit/kftl/kftl-submit-emits.test.ts`（解析→送信の順、不正行で送らない、`tags` / `mi_board_names` からの確認、デバウンス、オフライン時の前回値）
- `src/client/__tests__/unit/api/gkill-api.test.ts`（ルート表との突き合わせ。`submit_kftl_text` / `parse_kftl_text` の対）
