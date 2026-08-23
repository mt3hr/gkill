# ADR-0009: 時間帯フィルタの秒値は「86400未満は秒オブデイ、以上はepoch」の二重解釈にする

| | |
|---|---|
| Status | Accepted |
| Date | 2026-08-23 |
| Sources | `5c976332` |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/api/find/period_of_time.go` |

## Context

`period_of_time_start_time_second` / `period_of_time_end_time_second` には歴史的に2つの表現が届いていた。

- **Webクライアント**は「今日のローカル h:m:s」の絶対epoch秒を送る
  （`use-period-of-time-query.ts` の `moment().startOf("day").hour(h)...unix()`）。
  サーバは `time.Unix(v, 0).In(time.Local)` の時分秒を抽出するので、これは正しく往復する。
- **MCPスキーマ**は最初から「0-86399 の秒オブデイ」を宣言し、値も 0-86399 で検証していた。
  サーバはこれもepochとして解釈するため、`32400`（09:00のつもり）は
  1970-01-01T09:00:00Z → JST 18:00 として評価され、**MCP経由の時間帯検索だけが+9時間ずれた窓で
  1年間動いていた**（1年分124,056件の外部監査で発覚。窓がずれているだけでフィルタ自体は
  正しく効いていたため、「返却行にフィルタが効かない」ように見えて誤診を誘った）。

さらにこの秒解釈は find_filter.go / sqlite3impl_util.go / git_commit_log_repository_local_dir_impl.go の
3箇所に写経されており、契約の食い違いにどこからも気付けなかった。

## Decision

- `find.NormalizeSecondOfDay` を唯一の正本とし、**0..86399 は秒オブデイそのまま、
  それ以外は従来どおり「絶対epoch秒 → ローカル時刻の時分秒」**で解釈する
- 根拠: 0..86399 の絶対epoch秒は 1970-01-01/02 UTC にしか存在せず、
  時間帯フィルタとしてその時刻を指定する実クライアントは存在しない。
  よって二重解釈は既存のWeb・保存済み検索条件を**一切壊さずに** MCP契約を成立させる
- 3箇所の写経はアクセサ（`PeriodStartSecondOfDay` / `PeriodEndSecondOfDay`）へ収束させる。
  SQL経路のバインド値は正規化済み秒を `SecondOfDayToHHMMSS` で "HH:MM:SS" 文字列にして
  列側の `strftime('%H:%M:%S', ...)` と文字列比較する
- 既存の時間帯テストは「1日全体を覆う窓」しか無く秒解釈が変わっても赤くならないため、
  **狭窓（09:00-10:00）と夜跨ぎ（23:00-01:00）の回帰テストを両表現×両経路×キャッシュON/OFFで**新設する

## Rejected alternatives

- **MCP(Node)側で秒オブデイ→epochへ変換する** — サーバの「epochの時分秒を取る」という
  奇妙な契約が残り、次の非Webクライアントが同じ罠を踏む。変換はNodeプロセスのTZに依存し、
  サーバと別TZで動かすと再びずれる。
- **サーバを秒オブデイ解釈へ全面変更する** — Webは epoch を送るので、`epoch % 86400` 相当の
  解釈になり**Webの時間帯検索がJSTで9時間ずれる**。既存テストは1日全体窓のため赤くならず、
  静かに壊れる最悪の形。
- **MCPスキーマの範囲検証(0-86399)を撤去してepochを要求する** — AIクライアントにとって
  「時間帯」を絶対epochで表現させるのは不自然で、エラーの発見も遅れる。宣言済みの契約を
  サーバ側が受け入れるほうが正しい。

## Consequences

MCPの時間帯検索が宣言どおりに動く。Webの挙動は不変。
仮に epoch 0..86399（1970-01-01/02 UTC の実時刻）を指定したいクライアントが現れた場合は
秒オブデイとして解釈されるが、その用途は時間帯フィルタの目的上存在しない。

## Evidence

- 1年分の監査実測: `period_of_time_start_time_second: 0, end: 21599`（00:00-05:59のつもり）が
  JST 09-14時台の109件と完全一致し、`32400-53999` が JST 18-23時台の実測131件と完全一致した
- Webの符号化: `src/client/classes/use-period-of-time-query.ts` は常に当日epoch（≥1.7e9）を送る

## Related tests

- `src/server/gkill/api/find/period_of_time_test.go`
- `src/server/gkill/api/find_filter_test.go`
- `src/server/gkill/dao/sqlite3impl/sqlite3impl_util_test.go`
- `src/server/gkill/dao/reps/git_commit_log_repository_local_dir_impl_test.go`
- `src/server/gkill/api/gkill_server_api/get_kyous_period_of_time_test.go`
