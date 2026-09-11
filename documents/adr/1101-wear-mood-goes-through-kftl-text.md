# ADR-1101: ウォッチの気分記録は専用メッセージパスを足さず、KFTL テキストを既存の送信経路へ流す

| | |
|---|---|
| Status | Accepted |
| Date | 2026-09-10 |
| Sources | `.claude/skills/gkill-mobile/SKILL.md`「ウォッチの気分記録は KFTL テキストで送り、ASCII接頭辞と関連時刻行を省かない」 |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | なし — 組み立てを `LantanaKftl.kt` の1関数へ閉じ込め、そこの KDoc が理由を持っている |

## Context

ウォッチアプリ（`src/wear_os/watch_app/`）に気分（Lantana）の記録を足すことになった。
既存のウォッチ→スマホ→サーバの経路は、KFTL テンプレートの送信（`/gkill/submit`）と
実行中 TimeIs の終了（`/gkill/end_timeis`）の2系統で、どちらも
`WearRequestHandler.handle()` の `when` の枝と `KNOWN_REQUEST_PATHS` の登録が対になっている。

この経路には、目に見えないところで効いている仕掛けが2つ乗っている。

1. **サーバ側の冪等キー** — `GkillWearableListenerService` がメッセージ1件ごとに UUID を採番し、
   WorkRequest の不変入力に載せる。ワーカーの再送では同じキーになり、
   `handle_submit_kftl_text.go` の `kftlIdempotencyStore` が二重登録を畳む。
2. **`WearSubmitLedger`** — 直近成功した KFTL テキストを覚えておき、完全一致・TTL24時間で
   重複を検出して「それでも送信しますか」の確認へ回す。結果だけ届かなかった場合の
   二重登録を、利用者に見える形で止める最後の砦。

新しい記録種別を足すたびにメッセージパスを増やすと、この2つを毎回配線し直すことになる。
配線を1本落としても**ビルドは通り、vet も素通りし、目の前では正常に見える**。
壊れるのは「圏外から復帰したときだけ二重に記録される」という再現しにくい場面である。

もう1つの制約は KFTL パーサ側にある。接頭辞の判定は**完全一致**で、
どの接頭辞にも一致しない行は既定でテキストメモ（Kmemo）になる。
つまり接頭辞を1文字間違えた機械生成テキストは、エラーにならずに別種の記録を作る。

## Decision

ウォッチの気分記録は専用のメッセージパスを足さず、`?<日時>` / `/mood` / `<値>` の3行の
KFTL テキストを組み立てて既存の `/gkill/submit` へ流す。
接頭辞は ASCII（`?` と `/mood`）を使い、関連時刻の行は必ず付ける。
送信中・重複確認の画面状態は「どの画面から来たか」ではなく KFTL テキストだけを持つ形に一般化し、
テンプレート記録と気分記録で送信経路を1本に保つ。

## Rejected alternatives

- **`/gkill/submit_mood` のような専用パスを足す** — ウォッチ側の定数、`WearRequestHandler` の
  `when` の枝、`KNOWN_REQUEST_PATHS`、companion 側の応答パスの4箇所が対になっており、
  1つ落とすと無言で疎通しない（メッセージパスは `GkillWearClient.kt` と `WearRequestHandler.kt` に
  重複定義されている）。加えて冪等キーと `WearSubmitLedger` を新経路へ配線し直す必要があり、
  落としても目の前ではエラーにならない。得られるものは「気分だけ別扱い」でしかない。
- **companion に `/api/add_lantana` を呼ぶメソッドを生やす** — サーバ側の書き込み経路が
  KFTL 一括パスとは別系統になる。`GkillApiClient` は現在ログイン・設定取得・KFTL送信・
  TimeIs の4用途しか持たず、記録種別ごとに増やし始めると Web クライアントの fan-out を
  スマホ側に作り直すことになる。ウォッチが増やしたい記録種別のぶんだけ companion の
  APK を配り直す必要も生む。
- **全角の接頭辞 `ーら` / `？` を使う（コードベースは日本語なので自然に見える）** —
  `ーら` の先頭は長音符 U+30FC で、漢数字の一（U+4E00）や全角ハイフンマイナス（U+FF0D）と
  見分けがつかない。取り違えると完全一致の判定が外れ、`ーら` と値の2行がそのまま本文へ落ちて
  **Kmemo が2件書かれる**。エラーも警告も出ない。人が手で書く画面ではこの危険は
  IME が吸収するが、機械生成テキストで曖昧さを持ち込む理由がない。
- **関連時刻の行を省き、サーバの受信時刻に任せる** — `WearSubmitLedger` はテキスト完全一致で
  重複を見るので、`/mood` と値だけだと「同じ日の2回目の同じ気分値」が毎回 `DUPLICATE` になり、
  確認画面を挟まないと記録できない。気分値は語彙が10通りしかなく、テンプレート名と違って
  衝突が例外ではなく常態である。さらに送信は WorkManager 経由なので、圏外なら復帰後に届く。
  受信時刻に任せると「朝つけた気分が夕方の記録になる」。
  秒まで入れておくと、意図的な2回は別テキスト・同一メッセージの再配送は同一テキストのままなので、
  台帳はむしろ本来の精度で働く。
- **`？2025-06-01T10:00:00+09:00` のようにオフセット付き ISO8601 で書く（既存テストがその形）** —
  `kftl_related_time_statement_line.go` の `dateFormats` にオフセット付きの書式は無い。
  `TestStatement_LantanaWithRelatedTime` はラベル名しか検査していないためこの形で通っているが、
  実際に値を取ろうとすると `KFTL_INVALID_PARSE_RELATED_TIME_ERROR_MESSAGE` で行エラーになり、
  送信全体が落ちる。`yyyy-MM-dd HH:mm:ss` を使う。
- **末尾に保存文字 `！` を付ける（ウォッチのテンプレート絞り込みが `endsWith("！")` なので必要に見える）** —
  `！` は「以降の行を打ち切る」マーカーであって保存トリガではない（`kftl_statement.go` の
  save character の分岐）。テンプレート側の `endsWith` は「そのまま送れるテンプレートだけ一覧に出す」
  ためのウォッチ側の都合で、サーバの要件ではない。付けても害はないが、
  台帳が見るテキストに意味の無い1行を混ぜることになる。
- **星を5段階（値 2/4/6/8/10）にしてタップ目標を倍にする** — 丸い画面で半分の当たり判定は
  約16dp しかなく、押しやすさの理由はある。ただし Web 版は左半分/右半分で 1〜10 を全部選べるので、
  ウォッチからつけた記録だけ奇数値を持てなくなる。同じ人が同じ尺度で続ける記録なので、
  入力チャネルによって値域が変わるほうが害が大きい（縦方向の当たり判定を広げて補う）。

## Consequences

- 気分記録は `create_app="gkill_kftl"` になり、メモ帳から手で書いた記録と**区別する欄が無い**。
  「ウォッチからつけた気分」を後から検索で絞ることはできない。
  → **2026-09-11 に解消。** `SubmitKFTLTextRequest` に任意の `create_app` を足し、companion が
  `gkill_wear`（`GkillApiClient.APP_NAME`。打刻終了の `update_app` と同じ値）を送るようにした。
  無指定は従来どおり `gkill_kftl`（MCP と旧 companion）。それ以前にウォッチから書いた記録は
  `create_device` もサーバ名で Web と同じなので識別できず、遡って直す手段は無い。
- 送信画面の状態が KFTL テキストを持つようになったので、テンプレート記録側も
  `TemplateNode` ではなくテキストを持つ。テンプレートの表示名を送信中・重複確認の画面に
  出したくなったら、状態にラベルを足し直す必要がある。
- 接頭辞と日時書式は Kotlin 側の文字列とサーバ側のパーサに二重に存在する。
  どちらかだけを変えると**サーバは 200 を返し、記録だけが黙って別物になる**。
  これは `LantanaKftlTest.kt`（文字列の完全一致）と
  `kftl_statement_test.go` の `TestStatement_LantanaFromWearOS`（同じ文字列からの解釈結果）を
  対で置くことでしか守れない。片方だけ残すと守れていないので、消すときは両方消すこと。
- ウォッチの時刻とサーバのローカルタイムゾーンが違う環境では、related_time がずれる
  （`parseDateTime` はサーバの `time.Local` で解釈する）。Web のメモ帳も同じ性質なので
  新しい制約ではないが、ウォッチを旅行先の時刻に合わせると差が出る。

## Evidence

- `WearSubmitLedger` の重複判定は「テキスト完全一致・上限100件・TTL 24時間」
  （`WearSubmitLedger.kt`。`isDuplicate` は `it.at >= threshold && it.text == kftlText`）。
  気分値の語彙は10通りなので、時刻を含めない場合の衝突は日常的に起きる。
- `dateFormats` は12個の書式を順に試すが、いずれもタイムゾーンオフセットを持たない
  （`kftl_related_time_statement_line.go`）。
- 接頭辞判定が完全一致であることと、外れた行が Kmemo になることは
  `kftl_factory.go` の `generateDefaultConstructor` と `kftl_none_statement_line.go`。
  同種の事故（`/mood 8` と1行に書いて本文になる）は `prefixWrittenWithArgument` の
  コメントに 2026-08-24 の実利用報告として残っている。
- 経路の再利用によって、この機能で追加・変更したファイルは watch_app の5ファイルのみ。
  companion（10ファイル）とサーバの本番コードは1行も変えていない。

## Related tests

- `src/wear_os/watch_app/src/test/java/com/gkill_android/mobile_app/src/gkill/mt3hr/gkill/wear/watch/data/LantanaKftlTest.kt`
- `src/server/gkill/api/kftl/kftl_statement_test.go`
- `src/wear_os/watch_app/src/test/java/com/gkill_android/mobile_app/src/gkill/mt3hr/gkill/wear/watch/MainActivityTest.kt`
