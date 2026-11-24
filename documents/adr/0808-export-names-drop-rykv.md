# ADR-0808: 「HTML で出力」の名前から rykv を外す

| | |
|---|---|
| Status | Accepted |
| Date | 2026-11-24 |
| Sources | [ADR-0806](0806-fix-spellings-instead-of-freezing.md)（綴りは凍結せず直す） / [ADR-0420](0420-split-mi-views-by-mode.md)（mi を顔ごとに分けた決定） / [ADR-0712](0712-export-delivery-is-base64-json.md)（出力の返し方） / `src/client/classes/api/gkill-api.ts` の `GkillAPIForExportedKyou` |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/api/req_res/export_kyou_html_request.go`、`src/client/pages/exported-kyou-page.vue` |

## Context

「HTML で出力」は rykv（ライフログビュー）のためだけに作ったので、名前のほとんどに
`rykv` が入っている —— エンドポイント `/api/export_rykv_html`、`ExportRykvHTMLRequest`、
`HandleExportRykvHTML`、`exported-rykv-page`、イベント `requested_export_rykv_html`、
i18n の `EXPORT_RYKV_HTML_TITLE` ほか3キー、エラーコードの定数名5本。

ところが、この機能は画面に依存しない。焼き込みを読む側の土台は最初から画面を名乗っていない
（`GkillAPIForExportedKyou` / `exported-kyou-payload.ts` / `export_dto.ts` / `export_html_render.go`）。
そこへ mi（タスク画面）からも出力できるようにする作業が始まり、**同じエンドポイントと
同じ焼き込みに mi が乗る**ことになった。名前に `rykv` が残っていると、
読む人は「mi はこれを通らないのでは」と疑うことになる。

gkill には先例がある。かつて綴りの誤りを「永続データと外部契約に乗っているから
直さない」として凍結したが、ADR-0806 でその方針を反転させ、互換を残さず全面改名した。

## Decision

**出力まわりの名前から `rykv` を外し、既にある「kyou」の綴りへ揃える。互換は残さない。**

| 旧 | 新 |
|---|---|
| `/api/export_rykv_html` | `/api/export_kyou_html` |
| `ExportRykvHTMLRequest` / `Response` / `Column` | `ExportKyouHTMLRequest` / `Response` / `Column` |
| `HandleExportRykvHTML`・`handle_export_rykv_html` | `HandleExportKyouHTML`・`handle_export_kyou_html.go` |
| `export-rykv-html-view` / `-dialog` の .vue と props / emits / composable | `export-kyou-html-view` / `-dialog` |
| `exported-rykv-page` / `use-exported-rykv-page` | `exported-kyou-page.vue` / `use-exported-kyou-page.ts` |
| イベント `requested_export_rykv_html` | `requested_export_kyou_html` |
| i18n `EXPORT_RYKV_HTML_TITLE` / `FAILED_EXPORT_RYKV_HTML_MESSAGE` / `SUCCESS_EXPORT_RYKV_HTML_MESSAGE` / `EXPORTED_RYKV_PAGE_NO_DATA_MESSAGE` | 同じ形で `…KYOU…` |
| エラーコードの定数名（`InvalidExportRykvHTML*Error` ほか5本）とメッセージコードの定数名 | 同じ形で `…KyouHTML…` |

**番号は変えない。** `ERR000452`〜`ERR000456` と `MSG000095` は線の上に乗っていて、
ヒント文（`error-hints.ts`）や資料も番号で引いている。変えたのは Go の定数名だけ。

**rykv 専用のものは改名しない。** 出力の rykv 用ラッパ `exported-rykv-view.vue` /
`exported-rykv-view-props.ts` は、隣に `exported-mi-view.vue` が並ぶので名前が正しい。
`GkillAPIForExportedKyou.get_saved_rykv_find_kyou_querys` も mi 版と対になる
rykv 専用メソッドなのでそのまま。ADR の本文に出てくる旧名（ADR-0415 の
`is_exported_rykv_view` など）は、その時点の記録なので書き換えない。

## Rejected alternatives

- **旧名の別名（エイリアス）を残す** — エンドポイントを2本にすると、どちらが正本か読み手に分からなくなる。i18n キーを2本持つと7言語ぶん重複し、片方だけ訳を直す事故が起きる。出力は gkill の中だけで完結しており（外部の利用者が叩く契約ではない）、互換を残して守る相手がいない。ADR-0806 が同じ理由で互換を捨てている。
- **`rykv` のまま、注記で「mi も通る」と書く** — 文書だけが防御線になる。ADR-0802 が綴りの凍結で同じ形を採り、ADR 自身が「一括置換で壊せる状態にある」と書いていた（ADR-0806 で反転）。名前が嘘をついている状態は、読む人が毎回確かめ直すコストを払い続ける。
- **`export_view_html` / `export_screen_html` のような新しい語を作る** — 出力の土台はすでに `GkillAPIForExportedKyou` / `exported-kyou-payload.ts` / `export_dto.ts` と「kyou」で揃っている。別の語を足すと、同じものを指す言葉が2つになる。
- **mi 用に別のエンドポイントを足す（`/api/export_mi_html`）** — 収集も焼き込みもガワも共通で、違うのは「どの画面としてマウントするか」だけ。2本にすると、メディアの予算・同名フォルダ・プラグイン本文の扱いが2箇所になる。
- **mi の対応と同じコミットでまとめて改名する** — 改名は機械的で差分が大きく、mi 対応の中身（焼き込みの印・出力ページの分岐）が読めなくなる。テーマごとに分ける。

## Consequences

- **旧エンドポイントは消えた。** 出力 HTML は焼き込み済みで通信しないので、作ったファイルは影響を受けない。**古いクライアント（Service Worker に残った版）が `/api/export_rykv_html` を叩くと 404 になる**。利用者は画面を再読み込みすれば新しい版になる。
- i18n のキー数は変わらない（4キーを改名しただけ）。`src/server/gkill/api/embed/i18n/locales/` は生成物なので、`npm run copy_i18n_to_app_embed`（`prepare_install` の一部）で追随する。
- `documents/adr/0415` / `0417` / `0418` / `0712` / `0713` の**ファイル名とエンドポイント名の参照行**は新しい名前へ直した（`npm run verify_docs` が参照パスの実在と `Related tests` の実在を検査するため）。**決定の本文は書き換えていない。**
- E2E の `rykv-export-html` の spec は `kyou-export-html.spec.ts` になった。中身が見ているのは「出来上がった HTML が通信しないこと」で、画面には依存していない。
- 出力ダイアログの CSS クラス（`export_rykv_html_button` など、E2E とユニットが掴む目印）も改名した。掴み手は同じコミットの中にある。

## Evidence

実測なし —— 名前が機能の範囲と食い違っている、というソースの読み取りからの判断。
置換の件数（64ファイル・347箇所）はスクリプトが数えたもの（コード由来）。
改名の前後で lint・型検査・ユニットテスト・`test_server`・資料検査・E2E が通ることを確かめた。

## Related tests

- `src/server/gkill/api/gkill_server_api/handle_export_kyou_html_test.go` — 出力エンドポイントの通し（改名後の名前で動くこと）
- `src/server/gkill/api/gkill_server_api/api_routes_test.go` — ルート表と登録の一致（エンドポイント名の正本）
- `src/client/__tests__/unit/api/gkill-api.test.ts` — ルート表から読んだアドレスとクライアントのアドレス定数の一致
- `src/client/__tests__/unit/composables/export-kyou-html-view.test.ts` / `exported-kyou-page.test.ts` — 出力ダイアログと出力ページ
- `src/client/__tests__/e2e/kyou-export-html.spec.ts` — 出力した HTML を `file://` で開く通し
- `src/locales/ABOUT_TEST.md` の i18n キー一致（7言語で同じキーを持つこと）
