# ADR-0038: フロントエンドの状態管理は Props/Emit と GkillAPI シングルトンのみ

| | |
|---|---|
| Status | Accepted |
| Date | 2026-08-21 |
| Sources | `documents/reverse/frontend-architecture.md`「5. 状態管理」 / `CLAUDE.md`「State management」「Composable pattern」節 |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/client/classes/kyou-view-relay.ts` |

## Context

Vue 3 のアプリで状態管理ライブラリ（Pinia / Vuex）を入れないのは、現代的な構成としては少数派である。**「なぜ入れないのか」を知らない人には、入れるほうが改善に見える。**

gkill は Props/Emit と `GkillAPI` シングルトンだけで組まれており、その代わりに**中継の網羅性を型で保証する**構造になっている。

## Decision

状態管理は Props/Emit と `GkillAPI` シングルトンのみ。**Pinia / Vuex を導入しない。**

画面をまたぐ変更の伝播が要る場所（ポート＝複数ウィンドウ）は、ストアではなく**変更バス**（`kyou-change-bus.ts`。seq 付きの追記ログ）で解く。

## Rejected alternatives

- **Pinia を導入して Kyou の状態を集約する** — 既存の**約63箇所の中継**（`build_kyou_view_relay` / `build_kyou_dialog_relay` で束ねた18〜20イベント）が総崩れになる。中継束は `Exclude` による網羅性チェックで「イベントを足したら両方に足さないとビルドが通らない」ようになっており、この保証はストアに移すと失われる。

- **Pinia を部分的に入れる（新しい画面だけ）** — 状態の所在が2系統になる。「この値はどっちで持っているのか」を毎回調べることになり、Props/Emit 側の網羅性チェックも効かなくなる。

- **`provide` / `inject` で配る** — 既存のテストは `useRykvView({props, emits})` を**コンポーネントインスタンスの外から素で呼ぶ**ので、`inject()` は警告を出して既定値へ落ちる ＝ **テストでは伝播が効かないのに緑になる**。変更バスは props で配る。

- **変更バスをスカラー（最新の1件）にする** — 同じ tick に複数件起きたとき最後の1件しか見えず、残りが黙って落ちる（KFTL の複数行保存が典型）。seq 付きの追記ログにする。

- **`KyouChangeBus.last_seq` を Ref で公開する** — チャネルのオブジェクトが `reactive()` に包まれたとき Vue が自動アンラップして `.value` が `undefined` になり、**伝播が黙って効かなくなる**（テストのハーネスが実際に踏んだ）。メソッドにする。

## Consequences

ロジックは必ず `classes/use-*.ts` に置き、`.vue` の `<script setup>` は「import・`defineProps`・`defineEmits`・コンポーザブル呼び出しの分割代入・`defineExpose`」だけにする（dialogs は 116/116 がこの形）。これが「ストアが無くてもテストできる」ことの根拠になっている。

中継束を手書きしない。`build_kyou_view_relay(emits)` / `build_kyou_dialog_relay(emits)` を使い、`v-on="crudRelayHandlers"` で渡す。イベントを足すときは `KyouViewRelayArgs` と `kyou_view_relay_event_names` の**両方**へ足す（片方だけだと `Exclude` の網羅性チェックでビルドが落ちる ＝ 気付ける）。

購読側へ渡してよいのは **emit を含まない適用関数だけ**。中継束を渡すと適用のたびに `emits(...)` が走ってホストが再 publish し、通知が無限に往復する。

## Evidence

実測なし — 既存構造からの判断（中継約63箇所と、setup 外からコンポーザブルを呼ぶテスト群が前提になっている）。

## Related tests

- `src/client/__tests__/unit/classes/kyou-view-relay.test.ts`
- `src/client/__tests__/unit/classes/relay-bundle-source-scan.test.ts`（`v-on` の束と `@中継イベント` の併記を検出）
- `src/client/__tests__/unit/classes/kyou-change-bus.test.ts`
