---
name: gkill-client-tags
description: "Kyou の追加/編集画面のタグ欄（18本共通の edit-kyou-tags-view.vue）の約束。add_tag が完了してから registered_kyou を emit する順序が唯一の防御線で、逆にするとタグ付きで追加した記録がエラーも警告も出ないまま一覧に現れない。tx_id を使わない理由、同名重複のクライアント側排除、タグ→板名の確認順、⊗ の遅延削除も扱う。edit-kyou-tags-view.vue・kyou-tags.ts・use-add-*.ts・use-edit-*.ts・use-confirm-unknown-tag.ts・add-*.vue・edit-*.vue を編集するとき必読。「タグだけ変えたのに更新なしエラー」「同じ名前のタグが増える」の調査でも必読。"
---

# 追加/編集画面のタグ欄の不変条件

対象: `src/client/pages/views/edit-kyou-tags-view.vue` / `src/client/classes/kyou-tags.ts` / `use-add-*.ts` / `use-edit-*.ts` / `use-confirm-unknown-tag.ts` / `pages/dialogs/add-*.vue` / `edit-*.vue`

**このファイルは全文が、実際に起きた事故の再発防止である。該当作業では飛ばさずに読むこと。**
多くは「例外もエラーも出さずに静かに壊れる」種類で、破っても目の前ではエラーにならない。

**Kyou の追加/編集画面のタグ欄**（2026-08-17）。タグが付く追加/編集画面 **18本すべて**（追加7 = kc/lantana/nlog/time-is/ur-log/mi/mi-re-kyou、編集10 = 前記＋kmemo/idf-kyou/re-kyou、ReKyou作成の `confirm-re-kyou-view`）に共通の子ビュー `pages/views/edit-kyou-tags-view.vue` を1行置くだけで、保存の一度の操作で Kyou とタグをまとめて登録できる。子は値を集めるだけで、実際の登録は親の `save()` が `classes/kyou-tags.ts` 経由で行う（`add-notification-for-add-mi-view.vue` と同じ形）。守るべき約束:
- **`add_tag` が完了してから `registered_kyou` を emit する。** 局所挿入（`use-registered-kyou-local-insert.ts`）は渡された Kyou をそのまま使わず `refresh_kyou` で引き直すので、その時点でサーバにタグが入っていれば `attached_tags` 込みで差し込まれる。逆に先に emit すると `kyou-local-insert.ts` の `matches_tags()` が空のタグ列を見て「一致しない」と判定し、**エラーも警告も出ないまま行が現れない**。順序が唯一の防御線
- **編集画面の「更新がなかったらエラー」ガードはタグの変更でも通す**（10本すべて、エラーコードは `*_is_no_update`）。ただし**本体が無変更なら `update_*` を呼ばない** —— 呼ぶと中身の同じ新しい版が1つ増える。判定は各コンポーザブルの `is_body_changed()` に切り出してある
- **タグの変更は `updated_kyou` を出さない。** 反映信号は `requested_reload_kyou` だけなので、タグだけ変えたときも必ず出す
- **`tx_id` は使わない。** TXID指定時のタグ／Kyou は一時リポジトリにしか無いので `add_tag` は `added_tag` を返せず（`handle_add_tag.go`）、`registered_tag` を上げられない。しかも `commit_tx` はDBトランザクションではなく部分確定しうる（`handle_commit_tx.go`）ので束ねても原子性は買えない
- **同じ名前の重複はクライアントで落とす。** サーバの重複チェックはタグIDだけを見る（`usecase/tag.go`）ので、入力欄の中の重複も、削除マークの付いていない既存タグと同名のものも `get_tag_names()` が落とす
- **既存タグは `get_tags_by_target_id` で子ビューが自分で引く。** 編集ビューの `load()` が呼ぶのは `load_typed_datas()` だけで `props.kyou.attached_tags` は空のまま
- **⊗ を押した既存タグは保存を押すまで消さない**（押し間違えを戻せるように）。実削除は `is_deleted=true` の版を足す `update_tag`
- **確認はタグ → 板名の順に1つずつ**（`use-kftl-view.ts` の `do_submit` と同じ）。mi / mi-re-kyou の4画面は `do_save(skip_unknown_tag_check, skip_unknown_mi_board_check)` の再入フラグで表現する。確認ダイアログは非モーダルなので、再入のたびに子ビューから値を取り直すこと
- **タグ欄は既存フィールドより後ろ（アクション行の直前）に置く。** E2E ヘルパ `fillDialogField(dialog, N, ...)` は入力欄の位置インデックスで掴むので、前に挿すと既存 spec が総崩れになる
- 未知タグ確認は共有部品 `pages/dialogs/confirm-unknown-tag-dialog.vue` + `classes/use-confirm-unknown-tag.ts`（板名版と対）。`add-tag-view` / KFTL に手書き複製されていたマークアップと、`add_tag` の手順を12本のコンテキストメニュー・削除確認から寄せた
- **「確認が開いているか」を呼び出し元が持つときは `closed` イベントで倒す。** `unknown_tags` の空判定で代用すると、ブラウザバックで閉じたときに空にならないので開きっぱなし扱いになる（KFTLのタブ操作が永久ロックされる）。ただし **`closed` は `requested_confirm` より先に来る**（ダイアログが `hide()` してから emit するため）ので、確認の続行で読む値（KFTLの `submit_target_tab_id` 等）を `closed` で消してはいけない
- 守るテスト: `kyou-tags.test.ts` / `edit-kyou-tags-view.test.ts` / `add-views.test.ts` の「registered_kyou は add_tag が終わってから emit される」/ `edit-views.test.ts` の「タグ欄」節 / `e2e/add-dialog-crud.spec.ts` の「URLogを本文とタグ入りで一度に追加できる」 順序が唯一の防御線である理由と却下案（tx_id で束ねる等）は [ADR-0032](../../../documents/adr/0032-add-tag-before-registered-kyou.md)。

## 関連スキル

- [gkill-client-foundation](../gkill-client-foundation/SKILL.md) — 必ず併読（中継束・Kyou の再読込）
- [gkill-client-columns](../gkill-client-columns/SKILL.md) — 局所挿入と「その場で作ったタグを列条件へ足す」
- [gkill-client-kftl](../gkill-client-kftl/SKILL.md) — KFTL 側の未知タグ確認（`do_submit` の再入）

## 詳しい設計と却下案（ADR）

- [ADR-0032 add_tag が終わってから registered_kyou](../../../documents/adr/0032-add-tag-before-registered-kyou.md)
