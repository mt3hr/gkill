# ADR-0416: 出力 HTML の IDF 本文はバイト列のまま焼き込み、文字コードの判定はクライアントの1実装に残す

| | |
|---|---|
| Status | Accepted |
| Date | 2026-11-22 |
| Sources | `src/client/classes/use-idf-kyou-view.ts` の `load_text_content` / `src/client/classes/decode-text.ts` / [ADR-0415](0415-exported-view-is-a-separate-flag.md)（出力モードと通信の1点止め） / [ADR-0113](0113-word-filter-columns-and-id-prefix.md)（同じ規則を複数実装に写さない） |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/client/classes/api/gkill-api.ts` の `resolve_media_url` / `get_exported_idf_text_bytes` の doc コメント、`src/client/classes/markdown-to-html.ts` の `resolve_asset` 引数 |

## Context

「HTML で出力」したファイルは通信をしない（ADR-0415）。ところが IDF のテキスト / Markdown は、
画面が `fetch(file_url)` で本文を取りに行く作りになっている（`use-idf-kyou-view.ts` の
`load_text_content`）。

この経路は**静かに壊れる**。`watch(..., { immediate: true })` で .txt / .md が一覧に出た瞬間に
自動で走り、`res.ok` を見ているだけなので、失敗しても例外も警告も出ない。
出力 HTML では「画像は出るのにテキストの中身だけ空」という形になる。

本文の中の相対参照も同じ問題を持つ。`markdown_to_safe_html` が `base_url` で解決した URL を
`img[src]` と `a[href]` に書き戻すが、出力 HTML ではその先に gkill が居ない。

## Decision

**本文はバイト列のまま焼き込む。** サーバはデコードせず base64 で渡し、文字コードの判定は
従来どおりクライアントの `detect_and_decode_text`（`decode-text.ts`）が行う。

受け口は `GkillAPI` の2つのフックにする。どちらも**基底は何もしない**ので、本体の挙動は変わらない。

| フック | 基底 | 出力 HTML |
|---|---|---|
| `resolve_media_url(url)` | 恒等関数 | 焼き込んだ `data:` URI / サイドカーの相対パスへ差し替える |
| `get_exported_idf_text_bytes(id)` | `null` | 焼き込んだバイト列を返す（`fetch` を通らない） |

`markdown_to_safe_html` には省略可能な第3引数 `resolve_asset` を足し、`base_url` で解決した
あとの URL をもう一度通す。`build_media_url` の戻り値も `resolve_media_url` を通す。

**`resolve_media_url` は URL を正規化してから引く。** `markdown-to-html.ts` が渡してくるのは
`base_url` で解決した**オリジン付きの絶対 URL**（`http://host/files/...`）で、
`build_media_url` が渡すのは相対パス（`/files/...`）なので、形が2つある。
前方一致だけで判定すると、markdown 経由の参照が素通りして**通信してしまう**。

焼き込みに無かったときの扱いは次のとおり。

- 同じオリジン（= gkill が配っていた経路）か `file:` … **`about:blank`**。
  そのまま残すと `file://` で無関係なローカルパスを読み、`/files/` 経由では Cookie が付かず 403 になる
- 外部サイトの `http(s)` … **そのまま残す**。潰すと記録の中身を失う。
  読み込みが起きるのは閲覧者がその記録を開いたときだけ
- `data:` / `mailto:` など … そのまま

## Rejected alternatives

- **サーバでデコードして文字列を焼き込む** — Go 側に文字コード判定の2実装目ができる。`decode-text.ts` は Shift_JIS 等の判定を持っており、同じ規則を写すと必ずずれる（ワード照合で同じ事故を踏んでいる。ADR-0113）。ずれたときは「この記録だけ文字化けする」形で出て、どちらが正しいのか分からなくなる。
- **`fetch(file_url)` のまま残し、`file_url` を `data:` URI に差し替える** — `fetch('data:...')` は仕様上 GET で許され主要ブラウザで動くが、`/files/` 経由で開いたときは opaque origin の CSP 次第になる。通信をしないという約束を、ブラウザの実装と CSP に預けることになる。
- **サーバで Markdown を HTML へ変換して焼き込む** — DOMPurify を通せない。サニタイズの実装もクライアントとサーバで2つになる。
- **`resolve_media_url` を前方一致（`url.startsWith('/files/')`）だけで判定する** — 実装してテストを書いた時点で落ちた。markdown から渡るのは絶対 URL なので前方一致が外れ、焼き込みを引かずに元の URL を返す＝出力 HTML が通信する。
- **焼き込みに無い URL をすべて `about:blank` にする** — 記録の本文に書かれた外部サイトへのリンクまで潰れる。出力 HTML が「読み込み時に通信しない」ことと、「閲覧者がクリックした先へ行けない」ことは別の話。
- **`use-idf-kyou-view.ts` の `fetch` を出力モードのフラグで分岐する** — `props.gkill_api` に口を1つ足すほうが、分岐が1箇所で済み、テストからも差し替えやすい。画面側が出力モードを知る必要も無い。

## Consequences

- **`props.gkill_api` が必ず要る。** `use-idf-kyou-view.ts` が2つのフックを呼ぶようになったので、props を手で組むテストは `gkill_api` を入れること（入れ忘れると `load_text_content` の中の TypeError が `catch {}` に握られ、本文が空のまま通る）。
- 本文の焼き込みは**メディアの埋め込みモードとは独立**。テキストは軽く、これが無いと本文が空になるので、`埋め込まない` を選んでも焼き込む。
- サーバ側は「どの IDF がテキストか」をクライアントの `TEXT_EXTENSIONS` / `is_markdown_file_name` と同じ判定にする必要がある。ずれると「画面には出るのに本文だけ空」になる。
- `base64_to_bytes` を `file-base64.ts` へ足し、`base64_to_blob` はそれを使う形にした（同じ復号を2回書かない）。戻り値は `Uint8Array<ArrayBuffer>` と書くこと —— `Uint8Array` のままだと `ArrayBufferLike` へ広がり `Blob` へ渡せない。

## Evidence

実測なし — `load_text_content` が `watch(..., { immediate: true })` で自動的に走り `res.ok` しか見ていないこと、
`decode-text.ts` が唯一の文字コード判定であることからの判断。
`resolve_media_url` の前方一致では markdown 経由が素通りすることは、テストを書いた時点で落ちて分かった。

## Related tests

- `src/client/__tests__/unit/composables/idf-kyou-view.test.ts` — 「本文の焼き込み」（焼き込みがあれば `fetch` を呼ばない・無ければ従来どおり `fetch` する・メディアの URL が `resolve_media_url` を通る）
- `src/client/__tests__/unit/classes/markdown-to-html.test.ts` — 「`resolve_asset`」（省略時は従来どおり・`img` と `a` の両方に効く・渡るのは解決後の絶対 URL）
- `src/client/__tests__/unit/api/exported-kyou-api.test.ts` — 「メディアの URL は焼き込みへ差し替える」（相対形と絶対形の両方で引ける・未焼き込みの gkill 経路は `about:blank`・外部サイトは残す）
- `src/client/__tests__/unit/classes/exported-kyou-payload.test.ts` — メディアの対応表と、本文をバイト列のまま持つこと
