# ADR-0032: タグ欄付きの追加/編集画面は add_tag が完了してから registered_kyou を emit する

| | |
|---|---|
| Status | Accepted |
| Date | 2026-08-17 |
| Sources | `.claude/skills/gkill-client-tags/SKILL.md`「Kyou の追加/編集画面のタグ欄」節 / `src/client/pages/views/edit-kyou-tags-view.vue` |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/client/classes/kyou-tags.ts` |

## Context

タグが付く追加/編集画面18本すべてに共通の子ビュー `edit-kyou-tags-view.vue` を置き、保存の一度の操作で Kyou とタグをまとめて登録できるようにした。

このとき **emit の順序が唯一の防御線**になる。局所挿入（→ ADR-0031）は渡された Kyou をそのまま使わず `refresh_kyou` で引き直すので、その時点でサーバにタグが入っていれば `attached_tags` 込みで差し込まれる。

## Decision

**`add_tag` が完了してから `registered_kyou` を emit する。**

## Rejected alternatives

- **先に `registered_kyou` を emit して、タグは後から付ける** — `kyou-local-insert.ts` の `matches_tags()` が空のタグ列を見て「一致しない」と判定し、**エラーも警告も出ないまま行が現れない**。タグで絞り込んでいる列で必ず起きる。

- **`tx_id` で Kyou とタグを束ねる** — **使えない。** TXID指定時のタグ／Kyou は一時リポジトリにしか無いので `add_tag` は `added_tag` を返せず（`handle_add_tag.go`）、`registered_tag` を上げられない。しかも `commit_tx` はDBトランザクションではなく**部分確定しうる**ので、束ねても原子性は買えない。

- **重複するタグ名をサーバに任せる** — サーバの重複チェックは**タグIDだけ**を見る（`usecase/tag.go`）。入力欄の中の重複も、削除マークの付いていない既存タグと同名のものも通ってしまう。クライアントの `get_tag_names()` が落とす。

- **既存タグを `props.kyou.attached_tags` から読む** — 編集ビューの `load()` が呼ぶのは `load_typed_datas()` だけで、`attached_tags` は空のまま。子ビューが `get_tags_by_target_id` で自分で引く。

- **⊗ を押した既存タグを即座に消す** — 押し間違いを戻せない。保存を押すまで消さず、実削除は `is_deleted=true` の版を足す `update_tag`（→ ADR-0010）。

## Consequences

編集画面の「更新がなかったらエラー」ガードは**タグの変更でも通す**（10本すべて、エラーコードは `*_is_no_update`）。ただし**本体が無変更なら `update_*` を呼ばない** —— 呼ぶと中身の同じ新しい版が1つ増える。判定は各コンポーザブルの `is_body_changed()` に切り出してある。

**タグの変更は `updated_kyou` を出さない。** 反映信号は `requested_reload_kyou` だけなので、タグだけ変えたときも必ず出す。

確認はタグ → 板名の順に1つずつ。確認ダイアログは非モーダルなので、再入のたびに子ビューから値を取り直すこと。

タグ欄は**既存フィールドより後ろ**（アクション行の直前）に置く。E2E ヘルパ `fillDialogField(dialog, N, ...)` は入力欄の位置インデックスで掴むので、前に挿すと既存 spec が総崩れになる。

## Evidence

実測なし — 構造からの判断（`matches_tags()` が引き直しの結果を見るので、順序が結果を決める）。

## Related tests

- `src/client/__tests__/unit/classes/kyou-tags.test.ts`
- `src/client/__tests__/unit/composables/edit-kyou-tags-view.test.ts`
- `src/client/__tests__/unit/composables/add-views.test.ts`（「registered_kyou は add_tag が終わってから emit される」）
- `src/client/__tests__/unit/composables/edit-views.test.ts`（「タグ欄」節）
- `src/client/__tests__/e2e/add-dialog-crud.spec.ts`
