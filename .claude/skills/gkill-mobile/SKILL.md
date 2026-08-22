---
name: gkill-mobile
description: "Android APK ラッパ（src/android/）と Wear OS（src/wear_os/）の約束。同梱 gkill_server は 127.0.0.1 限定で起動（無指定だと LAN の第三者が無認証で全記録を読める）、jniLibs からの実行と useLegacyPackaging、configChanges、Wear companion の TLS TOFU/ピン留め、KFTL 送信の冪等キー（handle_submit_kftl_text.go と対。内容ハッシュにしない・markDone 配線を落とさない）を扱う。src/android/・src/wear_os/・handle_submit_kftl_text.go を編集するとき、SDK やビルド設定を上げるとき必読。「打刻が二重登録される」の調査でも必読。"
---

# Android / Wear OS の不変条件

対象: `src/android/**` / `src/wear_os/**` / `src/server/gkill/api/gkill_server_api/handle_submit_kftl_text.go`

**このファイルは全文が、実際に起きた事故の再発防止である。該当作業では飛ばさずに読むこと。**
多くは「例外もエラーも出さずに静かに壊れる」種類で、破っても目の前ではエラーにならない。

**Android同梱サーバはループバック限定**（2026-08-21、監査 S3-android-main）。`MainActivity` の ProcessBuilder は `--address 127.0.0.1:9999` を渡す（無指定だと全インターフェース待受＝LANの第三者が無認証で全記録を読み書きできる）。activity に `configChanges`（回転で Activity を再生成させず、SQLite書き込み中の `kill -9` を防ぐ。起動はポート先行プローブで既存サーバを再利用）と `onBackPressedDispatcher`（WebView goBack）。cleartext は `network_security_config.xml` で localhost 限定、外部ストレージ権限は `maxSdkVersion` 付き・起動ゲートは非ブロッキング（M-15）。**Wear companion の TLS は TOFU/ピン留め**（H-05）: `GkillServerTrust.kt` がプラットフォーム既定で検証し、失敗時のみ保存済み SHA-256 フィンガープリント一致で許可（trust-all は全廃）。ピン学習は companion の「保存&接続テスト」で利用者承認時のみ。ウォッチ→電話の送信は `WearRequestWorker`（WorkManager）で Service 破棄を跨ぎ、`WearSubmitLedger` で重複再配送を確認へ回す（S3-wear）。KFTL送信にはサーバ側冪等キーも付く: `GkillWearableListenerService` がメッセージ1件ごとに UUID を採番して WorkRequest の不変入力に載せるので、同じ要求のワーカー再送では同じキーになり `handle_submit_kftl_text.go` の `kftlIdempotencyStore`（TTL 10分・成功時のみ記録）が二重登録を畳む。意図的な再送は別メッセージ＝別キーなので畳まれない。**冪等キーを内容ハッシュにしないこと**（意図的な同一内容の再送が畳まれて記録できなくなる）。**ハンドラでの `markDone` 配線を落とさないこと** ―― ストア単体テストは `markDone` を直接呼ぶので配線漏れを見逃す（ビルドも vet も素通しする＝`idempotencyKey` は `alreadyDone` 分岐で使われ未使用にならない）。守るテストは `kftl_idempotency_test.go`（ストア単体）/ `handle_submit_kftl_text_test.go`（2回叩いて2回目が畳まれる end-to-end。同一キー=1件・別キー=2件・キー無し=2件）/ `WearRequestHandlerTest.kt`。

**Android**: APK wrapper (WebView) bundling the gkill_server binary as `jniLibs/arm64-v8a/libgkill_server.so` and exec'ing it from `nativeLibraryDir` — required because targetSdk 29+ forbids executing files under the app's data dir (W^X). Needs `packaging { jniLibs { useLegacyPackaging = true } }` so the `.so` is extracted as a real file. compileSdk 37 (androidx 1.19.x requires it), targetSdk 36, minSdk 26. **Wear OS**: Gradle multi-module project (phone_companion + watch_app), communicates via Wearable Data Layer. The Gradle wrapper is committed under `src/wear_os/`, so no copying is needed; `npm run setup_wear_os_gradle` re-syncs it from `src/android/` if it ever breaks.

## 関連スキル

- [gkill-go-backend](../gkill-go-backend/SKILL.md) — サーバ側の冪等ストアと HTTP セキュリティ
- [gkill-build-test](../gkill-build-test/SKILL.md) — ビルドパイプライン（embed 3コピー）
