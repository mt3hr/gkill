# ADR-0039: コンテキストメニューの位置は手計算せず Vuetify の実測配置に任せる

| | |
|---|---|
| Status | Accepted |
| Date | 2026-08-08 |
| Sources | `9beff2ff` / `.claude/skills/gkill-client-foundation/SKILL.md`「Context menus」節 |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/client/classes/use-context-menu-position.ts` |

## Context

コンテキストメニューの表示位置を、25箇所のコンポーザブルが**それぞれ手計算していた**。中身はコピペで、

```
left: min(innerWidth - 130, x)
top:  min(max(50, innerHeight - (8 + 48 * N)), y)
```

`130` はメニューの実際の幅と無関係な定数で、`N` は**テンプレートの項目数と手で同期する不文律**だった。構成ツリー系のメニューは `N=2` のまま実際には5項目あり、**画面下端からはみ出していた**。

## Decision

位置を手計算しない。`useContextMenuPosition()` が返す `is_show` / `menu_target` / `open_at(e)` を使い、テンプレートは

```
<v-menu v-model="is_show" :target="menu_target" location="bottom start">
```

とだけ書く。Vuetify の connected location strategy が**レンダリング後のメニューを実測して**、はみ出さないよう反転・シフトする。

## Rejected alternatives

- **手計算を残して定数を正しくする** — 幅は中身（項目のラベル）で決まり、i18n で言語ごとに変わる。項目数も条件付き表示で動く。**実測しない限り正しい値は書けない。**

- **項目数をテンプレートから自動で数える** — 数えられたとしても、高さは項目数だけでは決まらない（区切り線・サブヘッダ・折り返し）。

- **メニューの最大高さだけ CSS で抑える** — 位置の問題は解けない。ただし**併用はする**: `.gkill_context_menu_list { max-height: 70vh; overflow-y: scroll }` を `App.vue` に置いて、非常に長いメニューを抑える。

## Consequences

25箇所のコンポーザブルからコピペが消え、位置計算のバグが構造的に起きなくなった。

**新しいコンテキストメニューで位置を手書きしないこと。** 手書きは動いているように見えて、項目が増えた日にはみ出す。

## Evidence

実測なし — 構造からの判断（実測しない限り正しい定数は書けない）。

症状は実機で観測（構成ツリー系メニューが `N=2` のまま5項目あり、画面下端からはみ出していた）。

## Related tests

- `src/client/__tests__/unit/classes/use-context-menu-position.test.ts`
- `src/client/__tests__/unit/composables/context-menus.test.ts`
- `src/client/__tests__/e2e/context-menu-viewport.spec.ts`
