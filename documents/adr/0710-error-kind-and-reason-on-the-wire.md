# ADR-0710: `errors` / `messages` は成功時も `[]`、エラーには機械語の `error_kind` と `reason` を載せる

| | |
|---|---|
| Status | Accepted |
| Date | 2026-09-15 |
| Sources | `src/server/gkill/api/message/gkill_error.go` / `error_kind.go` / `error_reason.go` / `.claude/skills/gkill-go-backend/SKILL.md`「エラーの種類と理由（2026-09 導入）」節 |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/api/message/gkill_error.go` / `src/client/classes/api/message/error-hints.ts` |

## Context

利用者の指摘は3つだった。「粒度が荒くて何をすれば直るのか分からない」「利用者側で試せることがあるのに分からない」
「エラーが起きたことを知らせるだけ」。外部レビュー（2026-09-13、#8）は「成功時の `errors: null` は API 設計ミス。
クライアント180箇所で守るより境界で `[]` にすればいい」と指摘した。

実測（HEAD 12f086f2）:

- `GkillError` を組み立てる箇所は **633**（`api/gkill_server_api` 94ファイル・`usecase` 16ファイル）。
  利用者向け文言の i18n ID は 130 種で、`INTERNAL_SERVER_ERROR_MESSAGE`「内部エラー」が 26 回、
  `FAILED_GET_KYOU_MESSAGE` が 17 回。同じ「メモ追加に失敗しました」が、リクエスト JSON 不正（400）・
  同じ ID が既にある（409）・DB 書き込み失敗（500）の3つで使われていた。
- 原因の `err` は 633 箇所のうち **521** で手元にあった（`if err != nil` の直下）が、`slog` の Debug に
  出すだけで応答には載らず、既定のログレベル（error）では 1 行も残らなかった。
- 「書き込み先 rep が未設定」「USB ディスクが外れた」「DB がロック中」「検索が時間切れ」は打つ手が違うのに、
  利用者には全部「〜に失敗しました」に見えていた。dao 層の型付きエラーは `ErrPluginBusy` /
  `ErrPluginReturnedErrors` / `ErrNotGitRepository` / `CommitTxWriteRepMissingError` の4つだけで、
  SQLite の結果コード（BUSY / CANTOPEN / CORRUPT / READONLY / FULL）はどこでも見ていなかった。
- tx を使わない `add_*` で `WriteXxxRep` が nil だと nil ポインタ参照で panic し、
  「内部エラーが発生しました」だけが出ていた。
- 成功時の `errors` / `messages` は `null`。Web（約180箇所の `?? []` / `&& length` ガード）・MCP・CLI・
  Wear がそれぞれ null を吸収しており、単体テストのモックも `errors: null` で書かれていた。

## Decision

**文面（633箇所の `ErrorMessage`）は書き換えない。** 足りない「種類」「理由」「次の一手」は別のフィールドで
機械的に付ける。

- `GkillError` に `Cause error`（`json:"-"`）を持たせ、`if err != nil` の中で組み立てる箇所には
  `Cause: err` を付ける（521 箇所は codemod で一括、以後は `gkill_error_cause_scan_test.go` が強制）。
- `MarshalJSON` が `error_kind`（誰の問題か。9 種）と `reason`（何が起きたか。14 種）を付ける。
  kind はエラーコードから表で決め（`KindOf`。既定は HTTP ステータス、500 のうち設定不備だけ `config`）、
  reason は `Cause` を `errors.Is` / `errors.As` で分類する（`ReasonOf`。文字列照合はしない）。
  ワイヤは機械語のトークンだけで、ヒント文は消費者側（Web は `error-hints.ts` + i18n）が引く。
- `errors` / `messages` は名前付きスライス型 `GkillErrors` / `GkillMessages` にし、nil を `[]` で出す。
  **Web 側のガードは残す**（古い PWA が新旧どちらのサーバと話しても壊れないため）。
- `GkillMessage` に `level`（`info` / `warning`）を足し、rep 読み込み失敗とプラグイン警告は `warning`。
- `writeErrorStatus` が出す `request failed` の1行に `reasons` と `causes` を載せる。
  中断（`canceled`）だけの 500 は Debug に落とす。
- `WriteRepMissingError`（ERR000422）を新設し、usecase の `Add*` / `Update*` が nil の書き込み rep を
  panic の前に弾く。

## Rejected alternatives

- **クライアント 180 箇所を `?? []` へ書き換える** — レビューの指摘どおり、サーバの偶然の実装詳細を
  全消費者が防御する形が残る。境界で `[]` に揃えれば新しい消費者は何も知らなくてよい。
  ガード自体を消すことも採らない（SW キャッシュに残った古い応答・古いサーバとの組み合わせがある）。
- **サーバ側でヒント文を訳して返す** — `message` パッケージは `api`（localizer）を import できず、
  marshal 時にロケールが無い。521 箇所へロケールも配る codemod か、ctx にロケールを積むミドルウェアが
  要る。ヒントを要るのは Web だけ（MCP はトークンのほうが都合がよく、CLI は運用者向け、Wear は本文のみ）
  なので、消費者側で引く。
- **エラーコードの名前から kind を推論する** — ADR-0706 と同じ理由で導けない（`Invalid*` が 400 と 500 に
  跨る）。`NotFoundTLSCertFileError` は名前は not_found だが設定不備（`config`）。
- **`error_message` 本文をコード→文言の表から生成し直す** — 633 箇所の既存文言が死に、
  「メモ追加に失敗しました」のような操作の文脈が失われる。文面は残し、kind / reason を足すほうが安い。
- **reason を文字列照合で分類する**（`strings.Contains(err.Error(), "database is locked")`）—
  ドライバや OS の文言が変わった回にだけ静かに分類が外れる（ADR-0208 と同じ罠）。`*sqlite.Error` の
  `Code()`、`fs.ErrNotExist`、`context.DeadlineExceeded` のように値・型で判定する。
- **手元の `err` を 521 箇所へ配らず、usecase 16 ファイル＋主要ハンドラ約 30 箇所だけ手で付ける** —
  書き込み・検索・アップロードは拾えるが、それ以外は kind だけになる。codemod は AST で位置を決めて
  テキストを挿す形なので安全で、以後はソース走査テストが守る。
- **通信例外（JSON でない応答）を各呼び出し元で catch する** — `res.json()` は 86 箇所。
  `gkill_fetch` の 1 箇所で Content-Type を見て `bad_response` の合成応答にする。

## Consequences

- **新しいハンドラ・ユースケースは `if err != nil` の中の `GkillError` に `Cause: err` を付ける。**
  付け忘れは `gkill_error_cause_scan_test.go` が落とす。付いていないと、そのエラーだけ reason が出ず、
  ログの1行にも cause が載らない（応答は従来どおり返るので目の前では気付かない）。
- **エラーコードを足したら `http_status.go` の表へ1行**（従来どおり）。kind はそこから導かれる。
  設定画面で直せる不備なら `errorCodeKindOverride` へ、Cause 無しで理由が決まるなら `errorCodeReason` へ足す。
- **reason / kind の語彙を増やしたら Go（`reasonTokens` / `errorKinds`）と Web（`error-hints.ts`）と
  i18n（7言語 `ERROR_HINT_*`）を揃える。** `error-hints.test.ts` が Go のソースと突き合わせて落とす。
- dao の番兵エラーは `message.Reasoner`（`ErrorReason() string`）を実装する。`errors.Is` の同一性は保つ。
- `errors: null` を前提にしたテストのモックはそのまま動く（Web のガードが残るため）。
  ただし `TestResponseStatus_SuccessIsStill200` は `[]` を固定する。
- `usecase/kyou.go` の警告は `level: warning`。Web は閉じるまで残す（ADR-0411）。

## Evidence

実測（2026-09-15、このリポジトリのソース走査）:

- `grep -rn "ErrorMessage:" --include=*.go src/server/gkill | grep -v _test | wc -l` → 635（うち `GkillError` リテラル 633）
- `if err != nil` 直下のリテラル → 521（codemod の適用件数。scan test の検出数と一致）
- 利用者向け文言の i18n ID → 130 種（`INTERNAL_SERVER_ERROR_MESSAGE` 26 回が最多）
- 成功時 `errors:null` を吸収するクライアントのガード → 約 180 箇所（AGENTS.md の記述）
- 分類のテストは modernc.org/sqlite が実際に返すエラーで CANTOPEN(14) / NOTADB(26) / READONLY(8) / BUSY(5) を
  起こして確認した（`error_reason_test.go`）

## Related tests

- `src/server/gkill/api/message/gkill_error_test.go` — `[]` / `error_kind` / `reason` の形、`Cause` が漏れないこと
- `src/server/gkill/api/message/error_kind_test.go` — 全コードに kind、上書き表の実在、件数
- `src/server/gkill/api/message/error_reason_test.go` — 分類の表、SQLite 結果コード、コードからの既定、語彙
- `src/server/gkill/api/gkill_server_api/gkill_error_cause_scan_test.go` — `if err != nil` 内の `Cause` 付け忘れ
- `src/server/gkill/api/gkill_server_api/response_status_log_test.go` — ログの1行に reasons / causes、canceled は Debug
- `src/server/gkill/api/gkill_server_api/response_status_test.go` — 成功時の `errors` が `[]`
- `src/client/__tests__/unit/classes/error-hints.test.ts` — Go と Web の語彙の一致、ヒントの i18n キーの実在
- `src/client/__tests__/unit/api/gkill-api-bad-response.test.ts` — JSON でない応答が `bad_response` になる
