# ADR-0037: KFTL保存マーカーの判定は beforeinput→input の対で行う

| | |
|---|---|
| Status | Accepted |
| Date | 2026-08-19 |
| Sources | `1acc9481`（1回目の修正） / `a3f349ca`（本修正） / `4cceccd0`（回帰テスト） / `CLAUDE.md`「保存マーカーの判定は beforeinput で控えた本文と input 時点の本文を比べ」節 |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/client/classes/use-kftl-view.ts` |

## Context

KFTL は本文の末尾に保存マーカー行（`！` など）を打つと自動で送信する。これが**IMEで入力したときだけ効かなかった**。「バックスペースを押すと効くのに、順当に打つと効かない」という再現条件で、しかも修正を1回外している。

真因は DOM の仕様にあった。**同じ `input` イベントのリスナーとリスナーの間ではマイクロタスクが走る**（各リスナーの呼び出しごとに microtask checkpoint が入る）。そのため Vue の post flush ＝ 本文の `watch` が、`@input` ハンドラより**先**に新しい本文を観測することがある。

IMEで「変換の確定」と「改行」を続けて打つとこの順序になり、`watch` 側の「利用者が打った印」を中間の値を見た回が食べてしまって、確定した値を見る回には印が残らず**判定そのものが走らなかった**。

```
input  てすと\n！      ← compositionend から Vue が合成した input
watch  てすと\n！      ← ここで印を消費（増分0なので発火しない）
watch  てすと\n！\n    ← 印が無いので判定に入らない
input  てすと\n！\n    ← @input はこの後に走る
```

## Decision

判定を **`beforeinput` → `input` の対**へ移す。`beforeinput` で「変わる直前の本文」を控え、`input` で**確定したマーカー行の増分**を見る（`count_save_marker_lines`）。

この2つのイベントの間には何も割り込めないので、フラッシュ窓の張り方にもリスナーの実行順にも依存しない。

## Rejected alternatives

- **`watch`（本文の変化）に判定を置く（改修前）** — 2つの理由で駄目。
  1. `watch` は `flush: 'post'` なので、**1回のフラッシュ窓の中で本文が2回変わると1回しか呼ばれず**、中間の値（マーカーで終わっている本文）は一度も観測されない
  2. 上記のマイクロタスク順序により、`watch` が `@input` より先に新しい値を見る。IMEでは必ず起きる

- **「末尾がマーカーか」で判定する** — 上記(1)で黙って落ちるうえ、**1行目のマーカーを拾えず**、マーカーの後ろに空行が1本あるだけで効かない。

- **`watch` の内容変化そのものを入口にする** — `text_area_content` はアクティブタブへの computed なので、タブ切替・localStorage からの復元でも発火し、**末尾にマーカーが残ったタブをクリックしただけで保存が走る**。入口は「利用者が選んだ操作」＝ textarea の `@input` とテンプレート貼り付けの2つだけにする。

- **テンプレート貼り付けも `@input` の印に相乗りさせる** — `watch` は `new_value === old_value` で早期 return するので、貼る前のタブの本文がテンプレートと同一文字列だと**黙って発火しない**。テンプレートは `paste_template()` から直接判定を呼ぶ。

- **Playwright の `pressSequentially` で回帰テストを書く** — **常に緑になる。** 打鍵ごとにイベントループが回るので中間の本文を必ず観測してしまい、問題の順序を再現できない。IMEは CDP の `Input.imeSetComposition` でしか再現しない。

## Consequences

増分で見ることにより、副次的に3つが同時に成り立つ。1行目のマーカーも拾える／既にマーカーが残っている本文を編集しただけでは再送信しない／マーカーの後ろに空行があっても効く。

「確定した」＝ その行の後ろに改行がある、なので `！` を打った時点では走らない。

**この不具合は「再現方法そのもの」が難所だった。** `pressSequentially` で書いた回帰テストは修正前のコードでも緑になるので、テストがあること自体が守りにならない。回帰テストが CDP を使っている理由をここに残す。

## Evidence

- CDP で実際に IME 合成を起こして再現させ、実装へ一時ログを入れてイベント順序を確定させた（上記のログ）
- 1回目の修正（末尾一致 → 行数の増分、`1acc9481`）では直らなかった。判定を `watch` に置いたままだったため

## Related tests

- `src/client/__tests__/unit/kftl/kftl-submit-emits.test.ts`（「KFTLの保存マーカー」節。実際のDOMの順序に合わせて `beforeinput` → 代入 → `input` の形で書いてある）
- `src/client/__tests__/e2e/kftl-tabs.spec.ts`（「IMEで確定してから改行しても自動で保存される」。CDP の `Input.imeSetComposition` を使う）
