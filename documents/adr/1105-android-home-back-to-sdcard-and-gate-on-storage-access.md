# ADR-1105: Android の $GKILL_HOME を /sdcard/gkill に戻し、共有ストレージの権限が許可されるまでサーバを起動しない

| | |
|---|---|
| Status | Accepted |
| Date | 2026-09-24 |
| Sources | `c1200a3b` / `b971b0a3`（2026-08-03 にアプリ専用領域へ移した側） / `.claude/skills/gkill-mobile/SKILL.md`「Android同梱サーバの待受・TLS・画面のアドレスは ServerConfig に従う」 |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/android/app/src/main/java/com/gkill_android/mobile_app/src/gkill/mt3hr/gkill/MainActivity.kt` の `GKILL_HOME` / `decideStorageGate` / `copyAppPrivateHomeIfNeeded` の KDoc |

## Context

APK の `MainActivity` は同梱の gkill_server に `--gkill_home_dir` で `/sdcard/gkill` を渡していた。
`common.go` のディレクトリ構成により、この配下には全 Kyou のデータベースに加えて、パスワードハッシュと
リセットトークンを持つアカウント DB、ログ、TLS の秘密鍵がすべて置かれる。共有ストレージなので、
全ファイルアクセス権を持つ他アプリ・USB/MTP 接続・ファイラーアプリのどれからも中身が読め、
マニフェストの `allowBackup=false` はデータがアプリ専用領域にあって初めて意味を持つので効いていなかった。

2026-08-03（v1.1.7）にこれを理由に置き場をアプリ専用領域（`filesDir` 配下の `gkill`）へ移し、
初回起動時に `/sdcard/gkill` を専用領域へ複製する移行を入れた（複製元は残した。この移行の決定には ADR が無く、
経緯は `b971b0a3` の本文だけにある）。さらに 2026-08-21 の
指摘対応（M-15）で、データが専用領域にあることを前提に起動ゲートを非ブロッキングにした
（共有ストレージの権限は、写真などを指すファイルリポジトリのためにだけ要るものになっていた）。

ところがアプリ専用領域は利用者からも見えない。ファイラーや USB 接続から辿れないので、データベースの
退避・PC との複製・別端末からの持ち込み・共有ストレージ上のフォルダをファイルリポジトリとして直接扱うといった、
利用者がファイルを自分で扱う運用がすべてできなくなった。gkill のデータは利用者自身のライフログで、
利用者が持ち出せることが前提にある。利用者は共有ストレージの読める状態を承知のうえで、置き場を戻すことを選んだ。

一方で、共有ストレージへ書くなら権限が要る。権限なしで起動すると gkill_server はデータ置き場を作れずに落ちる。
専用領域にデータを置いていた版から更新する端末では、そのデータを `/sdcard/gkill` へ運ぶ必要もある。

## Decision

`$GKILL_HOME` を `/sdcard/gkill` に戻す。共有ストレージへの権限（Android 11 以上は全ファイルアクセス、
10 以下は `WRITE_EXTERNAL_STORAGE`）が許可されるまでサーバを起動しない。権限の要求画面を自動で出すのは
Activity ごとに1回だけで、以後は起動画面の説明文と「許可する」ボタンから出す。許可後の再判定は `startServerWhenStorageAccessible` の
1か所に集める（Android 11 以上は設定画面から戻ったときの `onResume`、10 以下は `onRequestPermissionsResult` から呼ぶ）。
専用領域にデータを置いていた版からの更新では、`/sdcard/gkill` が無いか空のときだけ専用領域の中身を
一時ディレクトリへ複製してから改名し、複製元は消さない。複製に失敗したらサーバを起動しない。

## Rejected alternatives

- **アプリ専用領域のまま置く（2026-08-03 の置き場を維持する）** — 他アプリ・USB から読めない守りは確かに強い。しかし利用者がファイルを直接扱えない。ライフログを利用者が退避・複製・持ち出しできないことのほうを、利用者は問題とした。守りの差は Consequences に書いたとおり承知のうえで受け入れている。
- **非ブロッキングの起動ゲートのまま、権限が無くてもサーバを起動する** — M-15 の非ブロッキング化はデータが専用領域にあることが前提だった。共有ストレージでは、権限なしで起動した gkill_server が置き場を作れずに落ち、起動のたびに「異常終了」のトースト（画面下に数秒だけ出る表示）だけが出る。利用者が自力で端末の設定を開いて許可するまで一度も使えないうえ、許可されても起動し直さなければ画面は読み込み中のまま。権限を先に取り、取れたときにだけ起動するほうが単純で、失敗の見え方も1つになる。
- **SAF（Storage Access Framework）で利用者にフォルダを選ばせ、`content://` で扱う** — Go サーバは `content://` を扱えず、SQLite も実パスでしか開けない。SAF の許可は Android の ContentResolver を通してだけ効くもので、別プロセスの gkill_server には渡らない。同じ理由で、メディア系の細かい権限でも足りない（`MainActivity` の `hasSharedStorageAccess` のコメント）。
- **権限が許可されるまで `onResume` のたびに要求画面を出す** — Android 11 以上の全ファイルアクセスは設定アプリの画面で、許可せずに戻ると `onResume` が走ってまた設定画面へ送り返される。利用者がアプリから抜けられない。自動で出すのは1回にし、以後は画面のボタンから出す。
- **専用領域から `/sdcard/gkill` へ直接複製する（一時ディレクトリを介さない）** — 途中で止まると、中途半端な `/sdcard/gkill` が次回の起動で「中身あり」と見なされ、以後二度と複製されずに正として使われる。一時ディレクトリへ複製してから改名すれば、失敗は「`/sdcard/gkill` が無いまま」で終わり、次回は続きから埋められる。
- **複製に失敗してもサーバを起動する（次回に再試行する）** — 起動した gkill_server が空の `/sdcard/gkill` を作り、次回から「中身あり」になって専用領域のデータへ二度と戻れない。エラーは出ない。これが起動ゲートに複製の成否を噛ませている理由で、失敗時はトーストを出して止まる（Android の通知領域には残らない）。
- **複製が済んだら専用領域を消す** — 複製が壊れていたときに戻れない。2026-08-03 の移行も複製元を残しており、同じ方針にした。
- **`/sdcard/gkill` に中身があっても専用領域の新しいデータで上書きする、または2つを併合する** — どちらが新しいかを機械で決められない。2026-08-03 の移行で `/sdcard/gkill` に残った古い複製と、その後に専用領域で書いた記録が両方あるとき、DB の併合は Append-Only の履歴を壊す。`/sdcard/gkill` を正として何もせず、専用領域を残すことで手で戻せるようにした。

## Consequences

- 共有ストレージなので、全ファイルアクセス権を持つ他アプリ・USB/MTP 接続・ファイラーから、全 Kyou のデータベース・アカウント DB・ログ・TLS の秘密鍵が読める。`allowBackup=false` はこの置き場には効かない。これは承知のうえで受け入れている（`GKILL_HOME` の KDoc と `user-guide.md` の Android 節に書いてある）。
- 権限が許可されるまで gkill は起動しない。起動ゲートを非ブロッキングに戻すと、gkill_server が置き場を作れずに落ちる。
- 複製に失敗したのにサーバを起動すると、空の置き場が作られて専用領域のデータへ戻れなくなる。**エラーも警告も出ない。** `MainActivityUnitTest.kt` の `migration_*` が固定するのは、複製の判定と失敗時に例外を投げるところまで（無い・空なら複製して複製元を残す、中身があれば何もしない、複製元が無ければ何もしない、中断後の一時ディレクトリから完了する、同名ファイルは消さずに例外）。
  呼び出し側の `startGkillServer` が複製の失敗（`migrateAppPrivateHome` が false）を受けて `return@Thread` で起動を止める配線には、テストが無い（実機でも未確認）。この分岐を消しても、ビルドもユニットテストも通る。
- 2026-08-03 の移行で `/sdcard/gkill` に古い複製が残っている端末では、更新後にその古いデータで起動し、専用領域にしか無い記録が見えなくなる。エラーは出ない。専用領域は消さないので、`/sdcard/gkill` を退避してから起動し直せば複製される（`user-guide.md` の Android 節が更新前の退避を案内している）。
- Android 10 の端末で `/sdcard` へ実パスで書けるよう `AndroidManifest.xml` に `requestLegacyExternalStorage` が要る（11 以上では無視される）。外すと 10 でだけ書けない。
- 権限の要求画面は Activity ごとに1回しか自動で出ない。要求画面へ導く説明文とボタンを起動画面から外すと、一度許可せずに戻った利用者は端末の設定から自分で辿るしかなくなる。
- コミット時点で、権限画面の行き来と `/sdcard` 上での複製・改名は実機で未確認。

## Evidence

実測なし — 利用者の要望と、Android の権限モデル（SAF の許可は別プロセスへ渡らない・全ファイルアクセスは設定アプリでしか許可できない）、
gkill_server が実パスでしか読み書きできないことからの判断。複製の判定は `MainActivity` の companion object に置いた純関数 `copyAppPrivateHomeIfNeeded` に
切り出してユニットテストで固定した。

## Related tests

- `src/android/app/src/test/java/com/gkill_android/mobile_app/src/gkill/mt3hr/gkill/MainActivityUnitTest.kt` — `gkillHome_isSdcardGkill` / `migration_*` / `storageGate_startsOnlyWithAccessAndRequestsOnce` / `storageGate_neverStartsTwice` / `manifest_declaresSharedStorageAccessForAllSupportedVersions`
