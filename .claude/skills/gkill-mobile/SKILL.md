---
name: gkill-mobile
description: "Android APK ラッパ（src/android/）と Wear OS（src/wear_os/）の約束。同梱 gkill_server は 127.0.0.1 限定で起動（無指定だと LAN の第三者が無認証で全記録を読める）、jniLibs からの実行と useLegacyPackaging、configChanges、Wear companion の TLS TOFU/ピン留め、KFTL 送信の冪等キー（handle_submit_kftl_text.go と対。内容ハッシュにしない・markDone 配線を落とさない）、ウォッチの気分記録が送る KFTL テキストの3行（ASCII接頭辞・関連時刻行）、Wear OS の UI 文字列の7言語 strings.xml（既定 ja・locale_name はリソース由来・ワイヤコードは訳さない）を扱う。src/android/・src/wear_os/・handle_submit_kftl_text.go を編集するとき、SDK やビルド設定を上げるとき、文言を足す・直すとき必読。「打刻が二重登録される」「時計にエラーコードがそのまま出る」の調査でも必読。"
---

# Android / Wear OS の不変条件

対象: `src/android/**` / `src/wear_os/**` / `src/server/gkill/api/gkill_server_api/handle_submit_kftl_text.go`

**このファイルは全文が、実際に起きた事故の再発防止である。該当作業では飛ばさずに読むこと。**
多くは「例外もエラーも出さずに静かに壊れる」種類で、破っても目の前ではエラーにならない。

**Android同梱サーバはループバック限定**（2026-08-21、監査 S3-android-main）。`MainActivity` の ProcessBuilder は `--address 127.0.0.1:9999` を渡す（無指定だと全インターフェース待受＝LANの第三者が無認証で全記録を読み書きできる）。activity に `configChanges`（回転で Activity を再生成させず、SQLite書き込み中の `kill -9` を防ぐ。起動はポート先行プローブで既存サーバを再利用）と `onBackPressedDispatcher`（WebView goBack）。cleartext は `network_security_config.xml` で localhost 限定、外部ストレージ権限は `maxSdkVersion` 付き・起動ゲートは非ブロッキング（M-15）。**Wear companion の TLS は TOFU/ピン留め**（H-05）: `GkillServerTrust.kt` がプラットフォーム既定で検証し、失敗時のみ保存済み SHA-256 フィンガープリント一致で許可（trust-all は全廃）。ピン学習は companion の「保存&接続テスト」で利用者承認時のみ。ウォッチ→電話の送信は `WearRequestWorker`（WorkManager）で Service 破棄を跨ぎ、`WearSubmitLedger` で重複再配送を確認へ回す（S3-wear）。KFTL送信にはサーバ側冪等キーも付く: `GkillWearableListenerService` がメッセージ1件ごとに UUID を採番して WorkRequest の不変入力に載せるので、同じ要求のワーカー再送では同じキーになり `handle_submit_kftl_text.go` の `kftlIdempotencyStore`（TTL 10分・成功時のみ記録）が二重登録を畳む。意図的な再送は別メッセージ＝別キーなので畳まれない。**冪等キーを内容ハッシュにしないこと**（意図的な同一内容の再送が畳まれて記録できなくなる）。**ハンドラでの `markDone` 配線を落とさないこと** ―― ストア単体テストは `markDone` を直接呼ぶので配線漏れを見逃す（ビルドも vet も素通しする＝`idempotencyKey` は `alreadyDone` 分岐で使われ未使用にならない）。守るテストは `kftl_idempotency_test.go`（ストア単体）/ `handle_submit_kftl_text_test.go`（2回叩いて2回目が畳まれる end-to-end。同一キー=1件・別キー=2件・キー無し=2件）/ `WearRequestHandlerTest.kt`。

**Android**: APK wrapper (WebView) bundling the gkill_server binary as `jniLibs/arm64-v8a/libgkill_server.so` and exec'ing it from `nativeLibraryDir` — required because targetSdk 29+ forbids executing files under the app's data dir (W^X). Needs `packaging { jniLibs { useLegacyPackaging = true } }` so the `.so` is extracted as a real file. compileSdk 37 (androidx 1.19.x requires it), targetSdk 36, minSdk 26. **Wear OS**: Gradle multi-module project (phone_companion + watch_app), communicates via Wearable Data Layer. The Gradle wrapper is committed under `src/wear_os/`, so no copying is needed; `npm run setup_wear_os_gradle` re-syncs it from `src/android/` if it ever breaks.

**ウォッチの気分記録は KFTL テキストで送り、ASCII接頭辞と関連時刻行を省かない**（2026-09-10）。
星5個の UI が作る文字列は `?yyyy-MM-dd HH:mm:ss` / `/mood` / 値 の3行で、組み立ては
`LantanaKftl.kt` の `buildLantanaKftlText` に集約する（送信そのものは `/gkill/submit` の使い回しで、
companion もサーバも無改修）。**全角の `ーら` / `？` を使わないこと** ―― 接頭辞の判定は完全一致なので、
長音符(U+30FC)を漢数字の一や全角ハイフンと取り違えると行が本文へ落ち、エラーも警告も出ないまま
Kmemo が2件書かれる。**関連時刻の行を落とさないこと** ―― `WearSubmitLedger` はテキスト完全一致・TTL24時間で
重複を判定するので、時刻が無いと「同じ日の2回目の同じ気分値」が毎回 `DUPLICATE` になる。
送信が WorkManager で後回しになったときにタップ時刻ではなくサーバ受信時刻が残る問題も同時に防いでいる。
**日時にオフセット付き ISO8601 を使わないこと** ―― `kftl_related_time_statement_line.go` の `dateFormats` に
その形は無く、パースに失敗して送信全体が行エラーになる。**気分値 0 を送らないこと**（0 は「未入力」の意味で、
KFTL パーサは 0 を受理してしまうため最低値の記録が黙って1件書かれる）。星と値の対応（左半分=2N-1 / 右半分=2N）は
Web 版 `use-lantana-flowers-view.ts` と同じ式にする。画面状態を足したら `MainActivityTest.kt` の状態数も直す
（`Screen` が file-private なので件数の突き合わせでしか守れない）。守るテストは `LantanaKftlTest.kt`（星と値の対応・
文字列の完全一致・範囲外の拒否）/ `kftl_statement_test.go` の `TestStatement_LantanaFromWearOS`（サーバ側で
同じ文字列から Lantana 1件・mood・related_time が出ること）。却下案は
[ADR-1101](../../../documents/adr/1101-wear-mood-goes-through-kftl-text.md)。

**ウォッチから書いた記録は `create_app="gkill_wear"` で、Kotlin の `create_app` に既定値を付けない**（2026-09-11）。
`/api/submit_kftl_text` はサーバが `gkill_kftl` を固定で書いていたが、`SubmitKFTLTextRequest` の任意項目 `create_app`
で申告できるようにし、companion は `GkillApiClient.APP_NAME`（`gkill_wear`。打刻終了の `update_app` と同じ定数）を送る。
**データクラス側の `create_app` に既定値を書かないこと** ―― kotlinx.serialization は encodeDefaults=false なので、
既定値のままだとキーごと落ち、サーバはエラーも警告も出さずに `gkill_kftl`（メモ帳と同じ値）へ戻す
（`locale_name` で 2026-09-10 まで実際に起きていた壊れ方と同型。既定値を付けると
`GkillApiClientTest.submitKFTLText_success_returnsNull` が落ちることを実測済み）。**サーバ側の既定値 `gkill_kftl` を変えないこと**
―― MCP と旧 companion は `create_app` を送らないので、既定値が動くと既存の記録と値が割れる。
2026-09-11 より前にウォッチから書いた記録は `create_device` もサーバ名で Web と同じなので識別できず、遡って直せない。
守るテストは `handle_submit_kftl_text_test.go` の `TestHandleSubmitKFTLText_CreateApp`（指定値が書かれる・無指定と空白は
`gkill_kftl`）/ `GkillApiClientTest.submitKFTLText_success_returnsNull`（本文に `create_app` が載る）。

**Wear OS の UI 文字列は `strings.xml` の7言語セットで持ち、ワイヤコードと KFTL テキストは訳さない**（2026-09-10）。
両モジュールの `res/values/strings.xml`（既定 = 日本語。Web の `fallbackLocale: 'ja'` とサーバの `GetLocalizer` の
フォールバックと同じ）と `values-{en,zh,ko,es,fr,de}/strings.xml` が正本で、Kotlin に文言を直書きしない
（Compose は `stringResource`、Activity / Service / Worker は `getString`）。**キーを足したら7ファイル同時に足すこと** ――
Android のリソース解決はキーが欠けても例外を出さず既定言語へ静かに落ちるので、混在した画面が出るまで気づけない。
守るのは両モジュールの `StringsParityTest.kt`（キー集合・`%1$s` の個数・`\'` `\"` `&amp;` `%%` のエスケープ。
aapt2 のエスケープ検査は `assemble` でしか走らず `test` では黙っている）。**サーバへ送る `locale_name` を
`Locale.getDefault()` から自前で判定しないこと** ―― API 24+ の言語優先リスト（例 `[ar, en]`）では UI は `values-en` に
解決されるのに `getDefault()` は `ar` を返し、UI 英語・サーバ文言日本語に割れる。`GkillLocale.serverLocaleName(context)` =
`R.string.server_locale_name`（各 `values-xx` に `xx`）から引く。`GkillApiClient` の既定値は定数 `ja`（環境依存にすると
JVM テストが実行機の OS 言語で変わる）で、`LoginRequest` を含む全リクエストが `locale_name` を明示して送る
（kotlinx.serialization は encodeDefaults=false なので既定値のままだとキーごと落ちてサーバ側フォールバックになる。
2026-09-10 まで実際にそうなっていた）。**`WearRequestHandler` と `GkillApiClient` に日本語を戻さないこと** ――
時計へ返す失敗は `ERROR:` + `WIRE_ERR_*` の ASCII コード（`WearRequestHandler.kt` 先頭に一元定義）で持ち、
時計に見せる文言へは `WearRequestWorker` が送信直前に `GkillErrorText.localizeWireResponse` で訳す（時計側では訳さない。
サーバの `error_message` もスマホの `locale_name` で訳されて届くので、時計に出る文言はすべてスマホのロケールで一貫する）。
コードを足したら `strings.xml` の `error_*` と `GkillErrorText.stringResOf` にも足す（`GkillErrorTextTest` が
`WIRE_ERROR_CODES` を全件なめて訳漏れを落とす。忘れると時計に `login_failed` のような生コードが出る）。
`OK` / `DUPLICATE` / JSON 本文と KFTL テキストは訳さない。依存ライブラリの80言語超のリソースは
`androidResources.localeFilters` で7言語に絞る（`"ja"` を含めるのは依存側の `values-ja` を残すため）。companion は
`generateLocaleConfig = true` + `res/resources.properties`（`unqualifiedResLocale=ja`）で Android 13+ のアプリ別言語設定に出る
―― **手書きの `locales_config.xml` と併用しないこと**（ビルドエラー）。タイルは `onTileRequest` 時の文言で固定されるので
`LocaleChangedReceiver` が `LOCALE_CHANGED` で再描画を要求する。却下案は
[ADR-1102](../../../documents/adr/1102-wear-ui-strings-in-android-resources-with-ja-default.md)。

## 関連スキル

- [gkill-go-backend](../gkill-go-backend/SKILL.md) — サーバ側の冪等ストアと HTTP セキュリティ
- [gkill-build-test](../gkill-build-test/SKILL.md) — ビルドパイプライン（embed 3コピー）
- [gkill-client-kftl](../gkill-client-kftl/SKILL.md) — KFTL の接頭辞と行の解釈（ウォッチが組み立てるテキストの受け手）
