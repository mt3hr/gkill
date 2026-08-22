# ADR-0010: Append-Only DAO — ID 列に主キー制約を置かず、更新も削除も INSERT で表現する

| | |
|---|---|
| Status | Accepted |
| Date | 2026-08-21 |
| Sources | `documents/reverse/design-philosophy.md`「3. Append-Only DAO」 / `bb364253`（監査 H-07） / `.claude/skills/gkill-go-backend/SKILL.md`「型別 GetXxx(id, nil) は最新版を返す」節 |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/dao/reps/repository.go` |

## Context

gkill は**複数端末から同一のDBファイルへ書く**（端末ごとの `.db` を持ち回り、同期ツールでまとめる）。この形で排他制御を前提にした `UPDATE` / `DELETE` を使うと、同期の衝突が「どちらかの編集が消える」形で現れる。

旧システム群は CRUD の U/D が未実装で、そもそも更新も削除もできなかった。gkill はこれを実現するのが目的の1つだった。

## Decision

データの追加・更新・削除を**すべて `INSERT` で表現する**。ID 列に主キー制約を置かず、同一 ID のレコードが複数行共存する。

- 更新 = 同一 ID で新しい `UPDATE_TIME` の行を足す
- 削除 = `IS_DELETED = TRUE` の行を足す（論理削除）
- 有効データ = 同一 ID の中で `UPDATE_TIME` が最新の版

## Rejected alternatives

- **素直に `UPDATE` / `DELETE` を使う** — 複数端末が同一DBを触るので排他制御が要る。持ち回りの同期では排他が張れず、衝突が「片方の編集が黙って消える」形になる。追記だけなら、同期がファイルをマージし損ねても**最悪でも古い版が残るだけ**で、書いたものは消えない。

- **端末ごとにDBを分け、更新は「所有端末だけができる」ことにする** — 記録は端末をまたいで編集したい（スマホで撮った写真にPCでタグを付ける、が日常操作）。所有権を持ち込むと、この操作が「できない」になる。

- **論理削除ではなく物理削除する** — 削除も同期の衝突対象になる。削除した端末とまだ削除していない端末が同期すると、行が復活する（＝削除が効かない）。`IS_DELETED` の版を足す形なら、削除も「新しい版」として通常の最新版判定に乗る。

## Consequences

**「最新版を取る」がすべての読み取りの前提になる。** これを外すと、古い版が現在版として見える／削除した記録が復活する、という壊れ方をする。しかも例外は出ない。

- 型別の単体取得 `GetXxx(id, nil)` は `onlyLatestData := query.OnlyLatestData`（`false` 固定にしない）と `slices.MaxFunc(UpdateTime)`（`&xxx[0]` を返さない）で最新版を選ぶ。格納順の先頭は多くの場合いちばん古い版なので、`kyous[0]` を返すと静かに間違える
- ディスク容量は版の数だけ増える。`optimize` サブコマンドがこれを畳む
- 「対象が実在しない付随データ」（削除された Kyou に付いたタグなど）が残りうる。物理/論理で別々の掃除ツールを用意してある

削除しても行は残るので、**IDを再利用してはいけない**。タグIDが（対象ID, タグ名）の UUIDv5 になっているのはそのためで、この性質のおかげで `auto_tag` の再実行がタグを重複させない代わりに、**手動で削除したタグは二度と復活しない**（削除された行が同じIDを持つため）。

## Evidence

実測なし — 構造からの判断（複数端末が持ち回る同一DBに対して排他制御が張れない）。

## Related tests

- `src/server/gkill/dao/reps/get_kyou_latest_version_test.go`
- `src/server/gkill/dao/reps/get_typed_latest_version_test.go`
- `src/server/gkill/dao/reps/gkill_repositories_get_kyou_test.go`
- `src/server/gkill/dao/reps/cached_find_only_latest_test.go`
- `src/server/gkill/dao/reps/rows_err_check_test.go`
