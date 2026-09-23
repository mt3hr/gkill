# ADR-1104: Android 同梱サーバの待受・TLS・画面のアドレスは上書きせず ServerConfig に従わせる

| | |
|---|---|
| Status | Accepted |
| Date | 2026-09-24 |
| Sources | `.claude/skills/gkill-mobile/SKILL.md`「Android同梱サーバの待受・TLS・画面のアドレスは ServerConfig に従う」 |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/android/app/src/main/java/com/gkill_android/mobile_app/src/gkill/mt3hr/gkill/MainActivity.kt` の `buildGkillServerArgs` と `SERVER_URL_LINE_PREFIX` の KDoc / `src/server/gkill/api/gkill_server_api/close.go` の `PrintStartedMessage` のコメント |

## Context

APK の `MainActivity` は同梱の gkill_server を `--address 127.0.0.1:9999 --disable_tls` 付きで起動していた
（指摘 S3-android-main への対応。当時は設定 DB の既定の待受が全インターフェースだったので、無指定だと LAN に開いた）。
どちらも設定 DB を書き換えない実行時上書きなので、サーバは `ENABLE_THIS_DEVICE` の行を読みながら、
待受アドレスと TLS だけはその行を無視していた。設定画面で待受アドレスや TLS を変えて保存しても、
Android では何も変わらず、エラーも出ない。画面（WebView）のアドレスも、サーバが標準出力に出す
`Access your record space at : ` 行から拾ってはいたが、上書きのせいで常に `http://localhost:9999` になり、
さらに Kotlin 側にもポート 9999 が直書きされていた（既存サーバの先行プローブと、行が来ないときの代わり）。

利用者から「画面のアドレスは ServerConfig を見て決めてほしい」と要望され、待受も含めてすべて
ServerConfig に従わせることを選んだ。一方、2026-09-14 の [ADR-0708](0708-local-only-listen-by-default.md) で
新しい設定 DB の既定は `127.0.0.1:9999` ＋「ローカルアクセスのみ許可」になっており、上書きが無くても
既定では LAN に開かない。

## Decision

`MainActivity` は `--address` と `--disable_tls` を渡さず、Kotlin にポートもスキームも持たない。
WebView のアドレスは、サーバが ServerConfig から組み立てて標準出力に出す起動行の URL だけから取り、
行が出直したとき（サーバ設定の保存でサーバが作り直されたとき）はオリジンが変わった場合だけ開き直す。
TLS 有効時の自己署名証明書は、ループバックのホストに限って受け入れる。

## Rejected alternatives

- **`--address 127.0.0.1:9999` を残し、セキュリティのためにループバックへ固定し続ける** — 設定画面の待受アドレスと TLS が Android でだけ黙って効かない状態が続く。LAN に開かない守りは、新しい設定 DB の既定（ADR-0708）と、上書きできない「ローカルアクセスのみ許可」が引き継いでいる。
- **ポートと TLS だけ ServerConfig に従わせ、待受のホストはループバックに固定する（Go に専用フラグを足す）** — 利用者が選ばなかった。設定画面の待受アドレスのホスト部が Android でだけ効かないという、今回直したのと同じ種類の食い違いが残る。
- **Kotlin で `server_config.db` を読んでポートと TLS を決める** — `ENABLE_THIS_DEVICE` の行の選び方、キーが欠けたときの既定値、ホストを `localhost` に置き換える組み立て方を Kotlin にもう1つ作ることになる。デスクトップ版・CLI・起動行の3か所が同じ組み立て方をしている（ADR-0708 の Evidence）ところへ、ずれうる4つ目を足すだけになる。
- **URL を出力するだけのサブコマンドを足し、起動前に呼ぶ** — 新しい置き場ではサーバが最初に起動するまで設定の行が無く、失敗する。起動のたびにバイナリを2回動かすことにもなる。起動行はサーバ設定の保存のたびに出直すので、起動後の変化にも追従できる。
- **自己署名証明書を、`TLS_CERT_FILE` の証明書と一致したときだけ受け入れる（ピン留め）** — 証明書のパスを知るには Kotlin で設定 DB を読む必要があり、上の二重実装に戻る。ループバックへの http はサーバを認証せずに受け入れているので、ループバックに限って証明書を検証しなくても守りは弱くならない。

## Consequences

- 起動行の文言（`close.go` の `PrintStartedMessage`）が Android との契約になる。変えると WebView が開かないまま読み込み中で止まる（60 秒たつと「起動を確認できません」の通知が出る）。
- 古い設定 DB や PC から写した設定 DB で ADDRESS が `:9999` や `0.0.0.0:9999` なら、Android の同梱サーバも LAN に開く。アプリ層の「ローカルアクセスのみ許可」は別に効く。
- ADDRESS に特定の LAN の IP アドレスだけを書くと、ループバックで待ち受けないので WebView（`localhost`）から届かない。起動行はホストを常に `localhost` にするという既存の仕様による。
- 設定画面で待受のポートや TLS を変えるとオリジンが変わり、WebView の保存領域が分かれるのでログインし直しになる。
- TLS を有効にした設定で証明書ファイルが無いと、サーバは起動に失敗して終了する（以前は `--disable_tls` がこれを隠していた）。
- ADR-0708 の Context と Evidence にある「Android 同梱サーバは `--address 127.0.0.1:9999`」の記述は、この ADR で覆る。
- 上書きのフラグや 9999 を Kotlin に書き戻すと、設定画面の値が Android でだけ効かなくなる。エラーは出ない。`MainActivityUnitTest.kt` が起動引数にフラグが無いことを確かめている。

## Evidence

実測なし — 仕様の突き合わせからの判断。起動行の組み立ては `close.go` の `PrintStartedMessage`（`ServerAddressPortSuffix` と `EnableTLS && !DisableTLSForce`）、
上書きの効き方は `gkill_options` の `ResolveServerAddress` を読んで確かめた。実機での確認は、この ADR を書いた時点ではしていない。

## Related tests

- `src/android/app/src/test/java/com/gkill_android/mobile_app/src/gkill/mt3hr/gkill/MainActivityUnitTest.kt`
- `src/server/gkill/main/common/gkill_options/option_test.go`
- `src/server/gkill/api/gkill_server_api/close_bind_address_test.go`
