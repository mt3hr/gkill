# ADR-0903: 「HTML で出力」のガワは第2の Vite ビルドにし、自己完結を機械検査する

| | |
|---|---|
| Status | Accepted |
| Date | 2026-11-22 |
| Sources | `vite.export.config.ts` / `src/tools/build_export_template.mjs` / [ADR-0711](0711-user-html-runs-scripts-in-opaque-origin.md)（利用者由来 HTML の sandbox） / [ADR-0902](0902-npm-audit-allowlist-for-unfixed-dev-dependencies.md)（開発用依存を増やす重み） |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `vite.export.config.ts` の冒頭コメント、`src/tools/build_export_template.mjs` の冒頭コメントと自己完結検査、`src/client/export-main.ts` の「本体の main.ts から写すときに持ち込んではいけないもの」 |

## Context

rykv の画面を「ファイル1枚を渡せば相手の手元で完結して見られる」形で出したい。出力した HTML は
**通信をしない**（例外は、閲覧者が自分の API キーを入れたときの Google Maps だけ。そのキーは保存しない。ADR-0422）。

本体のビルド（`vite.config.ts`）はこの用途にそのままは使えない。

- `VitePWA` が `injectRegister: 'auto'` で**全エントリの HTML に Service Worker 登録を注ぐ**。
  このプラグインは `command` で分岐していないので、プラグイン配列を共有している限り必ず入る。
  出力 HTML に SW 登録が入れば、開いた瞬間に `serviceWorker.js` を取りに行く。
- コード分割が効いているので、動的 import（mermaid / marked / dompurify / ja 以外のロケール）は
  別チャンクになる。単一ファイルにすると、その別チャンクを fetch しようとして落ちる。
- `@mdi/font` の woff2 は `assets/*.woff2` として別ファイルに出る。

`vite-plugin-singlefile` のような既成プラグインを入れる案もあるが、
**VitePWA を切るためにどのみち config を分ける必要がある**。config が分かれるなら、
プラグインを1つ増やして得るものは JS と CSS のインライン化だけで、
CSS の中の `url(/assets/*.woff2)` は残る（フォント用の自前処理がどうせ要る）。
`npm audit` の例外リスト（ADR-0902）と `verify_release_gate` の対象を増やす重みもある。

そして何より、「出力 HTML が通信してしまう」は**開いた本人には何も起きない**。
オフラインの相手の手元でだけ画面が壊れる。文書で守るのではなく、ビルドで落とす必要がある。

## Decision

出力のガワを**第2の Vite ビルド**（`vite.export.config.ts` → `dist_export/`）にし、
畳むのは自前のスクリプト（`src/tools/build_export_template.mjs` → `resources/export/template.html`）が行う。

- `vite.export.config.ts` は `VitePWA` と `vueDevTools` を入れない。
  `mdiWoff2Only()` は `vite.config.ts` から **export して共有する**（写すと
  `@mdi/font` の CSS の形が変わったときに片方だけ直してドリフトする）。
- `build.rollupOptions.output.codeSplitting: false` で動的 import も1チャンクへ畳む
  （エントリが1本なので成立する。本体ビルドでは成立しない）。
- `cssCodeSplit: false` で CSS を1枚に寄せる。畳む側がチャンクグラフを読まなくて済む。
- i18n は `@/i18n` → `src/client/export-i18n.ts` の alias で差し替え、**ja だけを内蔵する**。
  他言語はサーバが `<script type="application/json" id="gkill-export-i18n">` へ差し込む。
  差し替えないと、7言語ぶんのメッセージが記録0件のガワにも常駐する。
- 畳むスクリプトは `<script src>` / `<link rel=stylesheet>` をインライン化し、CSS 内の
  `url(/assets/*)` を data: URI にし、残った `<link>`（favicon など）を捨てる。
  最後に**外部参照が1つでも残っていればビルドを落とす**。
- 検査は**HTML の骨格と、畳んだ JS / CSS の中身で規則を分ける**。畳んだ JS には
  ``href="`+e+`"`` のような文字列テンプレートが普通に入っているので、HTML 用の正規表現を
  そのまま当てると必ず誤検知する。JS 側で見るのは「別チャンクを読みに行く形」
  （`/assets/` への import、絶対パスの動的 import、`__vitePreload`）と
  **Service Worker の登録**だけ。`navigator.serviceWorker.ready` は Web Push の購読コードが
  持っており、出力 HTML では呼ばれないので登録だけを見る。

## Rejected alternatives

- **本体ビルドの2エントリにする（`rollupOptions.input` に `export.html` を足す）** — `VitePWA` が出力側の HTML にも SW 登録を注ぐ。プラグイン配列を共有している限り避けられない（`command` で分岐していない）。`codeSplitting: false` も本体エントリに効いてしまう。
- **`vite-plugin-singlefile` を入れる** — VitePWA を切るために config はどのみち分かれる。畳むのは JS と CSS だけで、CSS 内の woff2 の `url()` は残るので自前処理が要る。依存が1つ増えると `npm audit` の例外リスト（ADR-0902）と `verify_release_gate` の対象も増える。自前なら「外部参照が残っていたら落とす」検査を同じスクリプトに同居させられる。
- **サーバ側（Go）で実行時に `embed/html` の成果物を結合する** — 何を結合すればよいかはチャンクグラフ次第で、Go 側がそれを知ることになる。ビルドの知識がビルドの外へ漏れる。外部参照が残ったことにも気づけない。
- **自己完結を文書だけで守る** — ADR-0802 が綴りの凍結で同じ形を採り、ADR 自身が「文書だけが防御線で、一括置換で壊せる状態にある」と書いていた（ADR-0806 でその方針は反転した）。出力 HTML の通信は開いた本人には見えないので、文書は防御線にならない。
- **Mermaid をスタブへ差し替えてガワを軽くする** — 図を含む記録を開いたときだけ使う 3MB 級の依存で、落とせばガワは3分の1近くになる。却下したのは利用者の判断（図が描けないほうが困る）。コードブロックのまま見せる案は実装して測ったうえで捨てた。

## Consequences

- **ガワは記録0件でも常に乗る下駄になる。** 内訳は JS が大半で、Vuetify の CSS と
  `@mdi/font` の woff2（base64 で 4/3 に膨らむ）が続く。Mermaid を含めた判断により
  JS は3倍近くになっている。実際の出力では記録とメディアが支配的になるので、
  この下駄を削るより先にメディアの埋め込み方（サムネのみ／原寸／埋め込まない）で効かせる。
- **`export-main.ts` は `gkill-api` を画面より先に import しなければならない。**
  データクラス（`InfoBase` 系）と `req_res` は `gkill-api` と循環 import になっており、
  単一チャンクではエントリの import 順がそのまま評価順になるので、逆だと
  `class extends value undefined` で mount の前に落ちる。本体ビルドでは
  コード分割の順序で偶然成立していた（`mermaid-render.test.ts` も同じ理由で先に import している）。
- **`export-main.ts` に本体の `main.ts` から写してはいけないものがある**：`router`（ルートが無く、
  `@vue/devtools-api` が付いてくる）、`registerSW` / `virtual:pwa-register`（通信する）、
  `GkillAPI.apply_locale()`（Cookie を読んでロケールを動的 import する）。
- `dist_export/` と `resources/export/` は生成物なので `.gitignore` に入れ、**ESLint の
  無視リストにも入れる**（minify 済みのバンドルを対象にすると数千件の指摘が出る）。
- **`npm run prepare_install` を通さない裸の `go build` では `embed/export/` が空になる。**
  `embed/html` と同じ性質で、出力 API だけが落ちる形にすること（起動時に panic させると、
  出力を使わない人のサーバまで立たなくなる）。
- ガワを確かめるには **HTTP で配る必要がある**（`file://` は Chrome 拡張も内蔵ブラウザも開けない）。
  `resources/export` を静的に配って `template.html` を開く（`.claude/launch.json` は追跡されないので、
  各自の環境で `export-template-view` のような設定を置くか、その場で静的サーバを立てる）。

## Evidence

ガワ単体をブラウザで開いて測った（ビルド出力由来）: 単一 HTML 1枚で Vue と Vuetify が mount され、
文書自身の取得以外の**リソース要求は0件**、`@mdi/font` は data: URI から読まれて `document.fonts.size` が 1。
データの差し込みが無い状態では「記録が含まれていません」を出して終わる。
自己完結の検査は、分ける前は畳んだ JS 内の `href="…"` と `navigator.serviceWorker.ready` を
誤検知して落ちていた（規則を骨格と中身で分けた理由）。

## Related tests

自動テストは無い。守っているのは `src/tools/build_export_template.mjs` の自己完結検査で、
`npm run build_export_shell` → `npm run build_export_template`（`prepare_install` の一部）が
外部参照を見つけた時点でビルドを落とす。テストではなくビルドに置いたのは、
出力 HTML が通信するかどうかは成果物の性質であって、実行して確かめるものではないため。
