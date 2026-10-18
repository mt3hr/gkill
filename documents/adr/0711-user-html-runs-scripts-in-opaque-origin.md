# ADR-0711: 利用者の HTML ファイルは opaque origin のままスクリプトだけ許す — `allow-same-origin` は付けず、別ポート配信もダウンロード強制も採らない

| | |
|---|---|
| Status | Accepted |
| Date | 2026-10-19 |
| Sources | `.claude/skills/gkill-go-backend/SKILL.md`「利用者ファイル配信の CSP は拡張子で3通り。HTML だけ `allow-scripts` を足し、`allow-same-origin` は付けない」節 |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/api/gkill_server_api/utils.go`（`userContentHTMLSandboxPolicy` / `userContentSecurityPolicy` のコメント） |

## Context

`/files/` と `/zip_cache/` は、取り込んだファイルや ZIP の展開物を拡張子から決めた Content-Type で
gkill と同じオリジンから配信する。セッションクッキー `gkill_session_id` はクライアントの JS が
`document.cookie` で書いている（`gkill-api.ts`）ので HttpOnly を付けられず、同じオリジンでスクリプトが動くと
そのまま読み出せる。これを防ぐため `withUserContentSecurityHeaders`（`utils.go`）は `.pdf` 以外のすべてに
`Content-Security-Policy: sandbox` を付けていた。`allow-scripts` が無いので、中のスクリプトは一切動かない。

その結果、利用者が IDF として取り込んだ**単一ファイルの HTML**（モデルを base64 で内蔵した 3D ビューアのような、
数十 MB 級で外部通信もしないもの）を gkill から開くと、PC のブラウザでも Android アプリでも真っ白になる。
Chrome のコンソールには sandbox がスクリプトを止めた旨（`allow-scripts` が無い）が出る。

自分で取り込んだ HTML を自分の gkill で開けないのは不便だが、「スクリプトを許す」を素直にやると
上のクッキーの問題がそのまま戻る。

## Decision

**`.html` / `.htm` / `.xhtml` にだけ `sandbox allow-scripts allow-downloads allow-modals` を付ける。`allow-same-origin` は付けない。**
それ以外の拡張子（`.svg` を含む）は従来どおり `sandbox`、`.pdf` は従来どおり CSP 無し、`nosniff` は全経路で付ける。
判定は `/files/` と `/zip_cache/` の両方に効く1つの純関数（`userContentSecurityPolicy`）で、`r.URL.Path` を小文字化した拡張子で行う。

**MCP の file-link 配信（`mcp/http_transport.go`）は HTML も `sandbox` のまま。** AI がファイルを取るための経路で、
人が開いてスクリプトを動かす用途が無いので、本体と意図して揃えない。

### なぜ `allow-same-origin` 抜きなら安全と言えるか

- **文書は opaque origin になる。** `document.cookie`・`localStorage`・IndexedDB へのアクセスは例外になり、
  Service Worker の登録もできない。セッションIDを文書から読む手段が無い。
- **`/api` の認証は Cookie ではなく本文の `session_id`**（`auth_middleware.go` の `sessionPeek`）。
  文書はセッションIDを知り得ないので、`/api` へ POST を投げても認証が通らない。
- **Cookie は `SameSite=Lax`**（`gkill-api.ts` が明示している）。opaque origin からのサブリソース要求・fetch・POST は
  クロスサイト扱いで Cookie が付かない。Cookie で認証する `/files/` / `/zip_cache/` も、文書の中からは 403 になる。
- **`/api` は CORS を返さないので応答を読めない。** 例外の2本（ブックマークレット・`get_updated_datas_by_time` の
  `Access-Control-Allow-Origin: *`）は、もともと任意のサイトから届くように作ってある経路で、この文書だけが得る力ではない。
- **フォーム送信・ポップアップ・トップレベルの外枠操作は許さない**（`allow-forms` / `allow-popups` /
  `allow-popups-to-escape-sandbox` / `allow-top-navigation` を付けない）。
- 先例: プラグインの本文を描く `plugin-html-view.vue` の iframe も、同じ理由で `allow-same-origin` を付けずに
  `allow-scripts` だけを許している。

### 残るリスク（受け入れる）

- **外部ホストへの通信は防げない。** opaque origin でも `fetch` や画像読み込みで外へ出られるので、HTML 自身の中身を送り出す・
  開いたことを知らせる・IP を知られる、は起こりうる。取り込んだ HTML を開くことは、その HTML を任意のサイトとして開くのと同じ重さになる。
- **自分自身のトップレベル遷移はできる。** `location` で gkill の URL へ GET 遷移すると Lax の Cookie が付くが、
  これは任意のサイトに貼られたリンクと同じ力で、遷移先は gkill 自身の画面になる。
- ダウンロードとダイアログ（`alert` / `prompt`）の乱発、CPU の消費。
- **同じ rep の別ファイル（外部の `.js`・画像）は読めない。** 文書からの要求に Cookie が付かず 403 になるので、
  動くのは自己完結した単一ファイルの HTML だけ。
- Android の WebView でのダウンロード・`prompt()` が動くかは WebView 側の実装次第（未検証）。

## Rejected alternatives

- **`allow-same-origin` も付ける（普通のページとして動かす）** — スクリプトが gkill のオリジンで動き、
  `document.cookie` からセッションIDを読めて外へ送れる。sandbox を張った理由そのものを捨てることになる。
  ビューアが `localStorage` に設定を残せない、といった不便はこの代償に見合わない。

- **別ポートで配信して別オリジンにする** — Cookie はポートで分かれない（同じホストなら別ポートにも送られ、
  `document.cookie` からも読める）ので、そもそも分離にならない。加えて待受が1本増え、TLS 証明書・
  Android の同梱サーバ・ローカル限定の設定（ADR-0708）をすべて二重に持つことになる。

- **別ホスト名で配信する** — 分離にはなるが、利用者の環境に DNS や hosts の設定と別名の証明書を要求する。
  ループバックや LAN の IP で使う個人のサーバでは用意できない人が多い。

- **HTML は常にダウンロードさせる（`Content-Disposition: attachment`）** — 安全だが見られない。
  Android ではダウンロードした HTML を開くアプリが無く、PC でもローカルファイルとして開くと別の制約（`file://`）を受ける。
  「開くと真っ白」を「開けない」に替えるだけになる。

- **SPA の中に sandbox iframe（`srcdoc` や blob URL）を作って表示する** — 安全性は同じ（`allow-same-origin` 無しの
  sandbox）だが、数十 MB の HTML を JS のメモリに読み込んで iframe へ渡すことになり、新しいタブでの直開きや
  Android からの直開きには効かない。HTTP ヘッダの CSP なら配信経路1箇所で全部の開き方に効く。

- **セッションクッキーを HttpOnly にする** — 根本策ではあるが、今は JS が `document.cookie` から読んで
  `/api` の本文の `session_id` に載せている。ログイン応答で `Set-Cookie` を返し、全 API の認証を Cookie 読みに替える大改修になる。
  この決定はそれを前提にしない。もし HttpOnly へ移ったら、この節の選択肢は見直せる。

## Consequences

**`allow-same-origin` を足す変更は、テストを1つ落とさない限り入らない。** `handle_file_serve_test.go` /
`handle_zip_cache_file_serve_test.go` が HTML の CSP を文字列の完全一致で固定し、さらに `allow-same-origin` を
含まないことを別に検査する。期待値は `utils.go` の定数を参照せず文字列で書いてある（定数を書き換えると期待値も動くのを防ぐ）。
足したときの壊れ方は**静か**で、画面は普段どおり動き、HTML を開いた瞬間にセッションIDが読めるようになるだけである。

**拡張子の一覧を広げるときは、この ADR を読み直すこと。** `.svg` は `<img>` で表示する画像でもあり、
文書として開いたときにスクリプトを動かす理由が無いので入れていない。Content-Type が HTML になる別の拡張子
（環境の MIME 表次第で `.shtml` など）は `sandbox` のままで、スクリプトは動かない側に倒れている。

**MCP の配信経路は本体と値が違う。** `http_transport.go` の `// Mirrors:` コメントに意図して違うことを書いてある。
片方だけ直す変更を「ずれの修正」と誤読しないこと。

## Evidence

実測なし — 脅威モデルからの判断。

実ブラウザでの効き方は E2E（`user-content-sandbox.spec.ts`、Playwright の Chromium）で確かめる。合成の HTML
（インラインの `<script>` が DOM に文字を書き、`document.cookie` と `localStorage` を触るだけのもの）を
`/api/upload_files` で `Files` rep へ上げて `/files/` で開き、スクリプトが動くこと・両方が例外になることを見る。

## Related tests

- `src/server/gkill/api/gkill_server_api/handle_file_serve_test.go`
- `src/server/gkill/api/gkill_server_api/handle_zip_cache_file_serve_test.go`
- `src/client/__tests__/e2e/user-content-sandbox.spec.ts`
