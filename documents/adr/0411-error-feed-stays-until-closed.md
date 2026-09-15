# ADR-0411: エラー表示は1つのフィードに集約し、閉じるまで残す・コピーできる・握られなかった例外も同じ場所へ出す

| | |
|---|---|
| Status | Accepted |
| Date | 2026-09-15 |
| Sources | `src/client/classes/use-gkill-message-feed.ts` / `src/client/pages/views/gkill-message-feed-view.vue` / `.claude/skills/gkill-client-foundation/SKILL.md`「エラー / メッセージの表示」節 |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/client/classes/use-gkill-message-feed.ts` |

## Context

画面右上のエラー / メッセージ表示は、`use-*-page.ts` 16 本と `*-page.vue` 15 本に同じ実装がコピーされていた。
その実装は `show_keep` で「閉じられるか」と「自動で消えるか」を決めていたが、**サーバの JSON には `show_keep` が
無い**ので `undefined` → 閉じられない・2.5 秒で消える。クライアントの入力検証（`new GkillError()` は
`show_keep=true`）は常駐するので、**本物の障害ほど早く消える**逆転が起きていた。エラーコードはホバーの
ツールチップだけで、スマホでは見えなかった。

さらに、JSON でない応答（プロキシの HTML エラーページ）や TypeError は `main.ts` の `unhandledrejection`
リスナが abort 以外を素通しし、コンソールにしか出なかった。利用者には「押しても何も起きない」にしか見えない。

## Decision

- モジュール単位のシングルトン `use-gkill-message-feed.ts` に一覧を1つだけ置き、16 本の `write_errors` /
  `write_messages` はそこへ委譲する。表示は `gkill-message-feed-view.vue` の1コンポーネントで、
  15 ページはそれを置くだけ。
- **エラーと warning は閉じるまで残す。自動で消えるのは info（成功・完了の知らせ、2.5 秒）だけ。**
- 同じ code + 本文の連続は1枚にまとめて `×N` を出す（オフライン中の連打で画面が埋まらないように）。
- 1枚に本文・ヒント（`error_kind` / `reason` から i18n。ADR-0710）・`コード · reason` のフッター・
  「詳細をコピー」ボタン（コード・reason・本文・ヒント・時刻・パス）を出す。
- `main.ts` の `unhandledrejection`（abort 以外）・`window.onerror`・`app.config.errorHandler` から
  `push_client_exception` で同じフィードへ出す（ERR900101）。`gkill_fetch` は Content-Type が JSON でない
  応答を `bad_response`（ERR900100）の合成応答にする。
- 中断（`reason: canceled`）は出さない。

## Rejected alternatives

- **ページごとのコピーを直す**（16+15 箇所へ同じ修正）— 次に直すときも 31 箇所。しかもコンポーネントの外
  （`main.ts`）から同じ表示へ流せない。
- **Pinia を入れて store にする** — ADR-0408 のとおり入れない。必要なのは「1つの配列と数個の関数」で、
  `GkillAPI` シングルトンと同じ流儀のモジュールで足りる。
- **エラーも N 秒で自動クローズする** — 読む前に消える問題がそのまま。エラーは利用者が読んで対処するもので、
  閉じるのも利用者。積み上がりは `×N` のまとめで抑える。
- **`show_keep` をサーバの JSON に足す** — 「閉じられるか」はクライアントの表示方針で、サーバが決めることではない。
  レベル（`info` / `warning`）だけをサーバが返す。
- **コードをホバーのツールチップに残す** — スマホにホバーは無い。フッターに常時出す。

## Consequences

- **ページや hosted view に `write_errors` の実装を新しく書かない。** `useGkillMessageFeed()` の
  `push_errors` / `push_messages` を呼ぶ（既存の `received_errors` → ページの emit 連鎖はそのまま）。
- E2E は `.v-alert[role="alert"]` でエラーを掴んでいる（6 spec）。エラーだけ `role="alert"` の形を変えないこと。
- `app.config.errorHandler` を置くと Vue 自身のコンソール出力が止まるので、ハンドラ内で `console.error` を出し直す。
- フィードはモジュール単位の状態なので、単体テストは `reset_feed_items()` で始める。

## Evidence

実測（2026-09-15）:

- `write_errors` の同型実装 16 本（`grep -rn "function write_errors" src/client/classes`）、
  `alert_container` のテンプレート 15 本
- サーバ由来エラーの表示時間 2,500 ms、閉じるボタン無し（`closable: errors_[i].show_keep` → `undefined`）
- `unhandledrejection` で握っていたのは abort 系だけ（`main.ts`）。それ以外は `console` 止まり

## Related tests

- `src/client/__tests__/unit/classes/use-gkill-message-feed.test.ts` — 閉じるまで残る / info は自動で消える / warning は残る / ×N / ヒント / null・中断は出さない / 例外 / コピー文
- `src/client/__tests__/unit/classes/error-hints.test.ts` — kind / reason → ヒントの i18n キー
- `src/client/__tests__/unit/api/gkill-api-bad-response.test.ts` — JSON でない応答
- `src/client/__tests__/unit/composables/dashboard-page-reload.test.ts` — ページの `received_errors` がフィードへ積まれる
- `src/client/__tests__/e2e/login.spec.ts` — `.v-alert[role="alert"]` でログイン失敗のエラーが見える
