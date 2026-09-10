# ADR-1102: Wear OS の UI 文字列は Android リソースの7言語セット（既定 ja）で持ち、時計へ渡すエラー文言はスマホ側で訳す

| | |
|---|---|
| Status | Accepted |
| Date | 2026-09-10 |
| Sources | `.claude/skills/gkill-mobile/SKILL.md`「Wear OS の UI 文字列は `strings.xml` の7言語セットで持ち、ワイヤコードと KFTL テキストは訳さない」 |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/wear_os/phone_companion/src/main/res/values/strings.xml` / `src/wear_os/watch_app/src/main/res/values/strings.xml` の先頭コメント |

## Context

`src/wear_os/` のウォッチアプリ（`watch_app`）とスマホ側コンパニオン（`phone_companion`）は、
利用者に見える文言がすべて Kotlin に日本語で直書きされていた（watch 約40箇所 / companion 約35箇所）。
`res/values/strings.xml` は `app_name` / `tile_label` の3キーだけで、`stringResource` / `getString` の使用はゼロ、
ロケール別ディレクトリも無かった。gkill 本体（Web / サーバ / マニュアル）は ja, en, zh, ko, es, fr, de の
7言語に対応済みなので、Wear OS の2アプリだけが日本語固定だった。

もう1つ、companion がサーバへ送る `locale_name` は `"ja"` 固定に見えたが、実態はもっと悪かった。
kotlinx.serialization の `Json` は `encodeDefaults=false` なので、既定値 `"ja"` のままの `locale_name` は
**キーごと送られず**、サーバ側 `GetLocalizer` のフォールバック（ja）で日本語になっていた。
さらに `/api/login` の要求クラス `LoginRequest` には `locale_name` の欄自体が無く、
設定画面の「接続失敗: …」に出るログイン失敗の文言（最も利用者の目に触れる経路）は訳す手段が無かった。

文言の発生源と表示先が端末をまたぐのがこのアプリ固有の事情である。
時計に表示されるエラー文言の多くはスマホで生まれる ―― サーバの `error_message`（スマホが送った `locale_name` で訳される）、
`GkillApiClient` の自前メッセージ（`レスポンスが空です` / `empty response` が日英混在で直書き）、
`WearRequestHandler` のコード（`ERROR:login_failed`。時計は `removePrefix("ERROR:")` して**そのまま表示**するので
利用者に `login_failed` が見えていた）。

## Decision

1. **UI 文字列は両モジュールの `res/values/strings.xml`（既定 = 日本語）と `values-{en,zh,ko,es,fr,de}/strings.xml` に置く。**
   Compose は `stringResource`、Activity / Service / Worker は `getString`。既定が日本語なのは Web の `fallbackLocale: 'ja'`・
   サーバの `GetLocalizer` フォールバックと揃えるため。7言語に当たらない端末は gkill 全体で日本語になる。
2. **時計へ渡るエラー文言はスマホ側で訳す。** `WearRequestHandler` と `GkillApiClient` は Android 非依存の純粋クラスのまま、
   失敗を `ERROR:` + ASCII コード（`WIRE_ERR_*`、`WearRequestHandler.kt` 先頭に一元定義）で表す。
   `WearRequestWorker` が時計へ送る直前に `GkillErrorText.localizeWireResponse` で `ERROR:` の後ろだけを訳し、
   `OK` / `DUPLICATE` / JSON はバイト列をそのまま通す。設定画面の表示も同じ `GkillErrorText.localize` を通す。
   時計側は無改修で受信文字列を表示する。
3. **サーバへ送る `locale_name` は UI が解決したロケールをリソースから引く。** `values*/strings.xml` に `server_locale_name`
   （`values/`=`ja`、`values-en/`=`en`、…）を置き、`GkillLocale.serverLocaleName(context)` がそれを返す。
   `GkillApiClient` は `localeName` をコンストラクタで受け取り（既定値は定数 `ja`）、`LoginRequest` を含む全リクエストの
   データクラスから `locale_name` の既定値を外して**常に送信**する。
4. **ビルド設定**: 両モジュールに `androidResources.localeFilters` で7言語を指定し、依存ライブラリの80言語超のリソースを落とす。
   companion は `generateLocaleConfig = true` + `res/resources.properties`（`unqualifiedResLocale=ja`）で
   Android 13+ のアプリ別言語設定に対応する。ウォッチは `LocaleChangedReceiver` が `LOCALE_CHANGED` でタイルの再描画を要求する。

## Rejected alternatives

- **時計側でコードを訳す（ワイヤ上はコードのまま、`watch_app` に照合表を置く）** ―― 照合表がスマホ・時計の2モジュールに
  二重化し（メッセージパスの二重定義と同じ構造）、片方に足し忘れても目の前ではエラーにならない。しかもサーバの
  `error_message` はスマホの `locale_name` で訳されて届くので、時計側で訳しても「自前コードは時計の言語・サーバ文言は
  スマホの言語」の混在になる。Wear OS は通常スマホと同じ言語なので、スマホ側で一括して訳すほうが一貫する。
- **`GkillApiClient` に `Context` を注入して `getString` で日本語を返す** ―― MockWebServer で JVM 単体テストしている
  純粋クラスに Android 依存が入る。テストは `Context` を MockK で用意すれば通るが、`GkillApiClientTest` の28本が
  「文言」ではなく「どのリソースIDに解決されたか」を見る形に書き換わり、HTTP 契約の検査という本来の役目がぼやける。
- **共有 Gradle モジュール（`:shared`）にコードと照合表を置く** ―― 2モジュール10コードのために第3のモジュールと
  その依存配線を増やす。verify_docs の件数集計もモジュール単位で組まれている。過剰。
- **`Locale.getDefault().language` を7言語に写像して `locale_name` を決める** ―― API 24+ の端末は言語の優先リストを持ち、
  リソース解決は「リスト中でアプリが持つ最初の言語」を選ぶ。利用者のリストが `[ar, en]` なら UI は `values-en` になるが
  `Locale.getDefault().language` は `ar` を返す。写像すると `ja`（フォールバック）を送り、UI 英語・サーバ文言日本語に割れる。
  リソースから引けば LocaleList・Android 13 のアプリ別言語設定・スクリプト一致まで全部リソース解決に委ねられ、
  `zh-TW → zh` のような写像コードもテストも要らない。
- **`GkillApiClient` の `localeName` 既定値を `Locale.getDefault()` 由来にする** ―― JVM 単体テストの結果が実行機の
  OS 言語で変わる。既定値は定数 `ja`（= サーバ側フォールバック = 今日までの実挙動）にし、本番の2呼び出し元
  （`MainActivity` / `WearRequestWorker`）が明示的に渡す。
- **`res/xml/locales_config.xml` を手書きして manifest に `android:localeConfig` を書く** ―― AGP 8.1+ の
  `generateLocaleConfig = true` が `values-*` のディレクトリから同じ XML を生成し manifest へ注入する。手書きと併用すると
  ビルドエラー。生成に任せれば `values-xx` を足したときの追随漏れも起きない。
- **既定言語を英語にする（Android の一般的な慣習）** ―― Web の `fallbackLocale: 'ja'`・サーバの `GetLocalizer` の
  フォールバック・マニュアルの原稿言語がすべて日本語で、7言語に当たらない端末の挙動が gkill の中で Wear OS だけ
  違うことになる。既定を ja にすれば `resources.properties` の `unqualifiedResLocale=ja` で機械可読に残る。

## Consequences

- 時計に届くエラー文言はスマホのロケールになる。スマホと時計の言語が違う環境（Wear OS の言語はスマホから同期されるので
  通常は一致する）では、時計の自前文言（`スマホに接続できません` 等）と ERROR: の後ろで言語が割れる。
- `WearRequestHandler` → `WearRequestWorker` の境界はコード固定（`WearRequestHandlerTest` が `ERROR:login_failed` 等を
  完全一致で契約として守る）、スマホ → 時計のメッセージは `ERROR:` の後ろがスマホロケールの人間文。
  コードを足したら `strings.xml` の `error_*` と `GkillErrorText.stringResOf` にも足す（`GkillErrorTextTest` が
  `WIRE_ERROR_CODES` を全件なめて訳漏れを落とす）。
- キーを足すときは7ファイル同時。Android のリソース解決はキーが欠けても例外を出さず既定言語へ静かに落ちるので、
  両モジュールの `StringsParityTest` でしか守れない。aapt2 のエスケープ検査（`\'` / `\"` / `&amp;` / `%%`）は
  `assemble` でしか走らず `test` では黙るので、parity テストが前倒しで検出する。
- `values-zh` は Android 7+ で zh-Hans として扱われ、zh-Hant（台湾・香港）の端末は既定（ja）に落ちる。
  Web も `navigator.language === 'zh'` の完全一致しか受理しないので、gkill の中では一貫した挙動。
- `LoginRequest` に `locale_name` が載るようになり、サーバの `handle_login.go` はそれで `FAILED_LOGIN_MESSAGE` /
  レート制限の文言を訳して返す。既存サーバとの互換性の問題は無い（欄は元から `LoginRequest` 構造体にあった）。
- ウォッチのチップとタイル（`chipWidth = 140dp` 固定）は1行で切り詰められるので、de / fr / es の訳は短く保つ必要がある
  （`⭐️ Ánimo` のように名詞だけにした箇所がある）。

## Evidence

- `locale_name` が送られていなかったこと: `GkillApiClient.kt` の `Json { ignoreUnknownKeys = true }` は encodeDefaults=false、
  データクラスの `locale_name: String = "ja"` は既定値のまま。`GkillApiClientTest` L159/L220 に
  「locale_name may be omitted by kotlinx.serialization when it's the default value」と明記されていた。
  修正後は `localeName_isSentOnEveryRequest` が login / get_application_config / submit_kftl_text / get_kyous /
  get_timeis / update_timeis の6リクエスト本文すべてに `"locale_name":"de"` が載ることを確認する。
- `LoginRequest` に欄が無かったこと: 修正前は `LoginRequest(val user_id: String, val password_sha256: String)`。
  サーバ側 `src/server/gkill/api/req_res/login_request.go` には元から `LocaleName string \`json:"locale_name"\`` がある。
- 時計に生コードが出ていたこと: `WearRequestHandler` が `ERROR:login_failed` を返し、時計 `MainActivity.kt` は
  `result.removePrefix("ERROR:")` を `Screen.Result.error` にそのまま入れて `ResultScreen` が表示していた。
- `generateLocaleConfig = true` の生成物: `_generated_res_locale_config.xml` が `android:defaultLocale="ja"` と
  7言語の `<locale>` を持ち、マージ後の manifest に `android:localeConfig="@xml/_generated_res_locale_config"` が入る
  （`:phone_companion:assembleDebug` で確認）。
- 文言の棚卸し: 修正前の直書きは watch_app の Kotlin に約40箇所（`MainActivity.kt` 23、`screens/*.kt` 14、
  `GkillTileService.kt` 3）、phone_companion に約35箇所（`MainActivity.kt` 16、`GkillApiClient.kt` 15、`WearRequestWorker.kt` 3、
  `GkillServerUrlPolicy.kt` 1）。修正後の Kotlin に残る日本語リテラルは `Log.*` と、到達しない
  `GkillWearClient.fetchTemplates` の例外メッセージだけ。
- テスト件数: Wear OS 合計 200 → 226（companion 120 → 140: `GkillApiClientTest` +1、`GkillErrorTextTest` 12、
  `StringsParityTest` 7。watch 80 → 86: `StringsParityTest` 6）。`StringsParityTest` は `values-de` から
  キーを1つ消すと `values-de: missing=[error_unknown] extra=[]` で落ちることを変異検査で確認した。

## Related tests

- `src/wear_os/phone_companion/src/test/java/com/gkill_android/mobile_app/src/gkill/mt3hr/gkill/wear/companion/StringsParityTest.kt`
- `src/wear_os/watch_app/src/test/java/com/gkill_android/mobile_app/src/gkill/mt3hr/gkill/wear/watch/StringsParityTest.kt`
- `src/wear_os/phone_companion/src/test/java/com/gkill_android/mobile_app/src/gkill/mt3hr/gkill/wear/companion/GkillErrorTextTest.kt`
- `src/wear_os/phone_companion/src/test/java/com/gkill_android/mobile_app/src/gkill/mt3hr/gkill/wear/companion/GkillApiClientTest.kt`
- `src/wear_os/phone_companion/src/test/java/com/gkill_android/mobile_app/src/gkill/mt3hr/gkill/wear/companion/WearRequestHandlerTest.kt`
