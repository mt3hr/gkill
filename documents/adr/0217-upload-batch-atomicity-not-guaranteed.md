# ADR-0217: アップロードのバッチ原子性は保証しない（ファイル単位の原子性のみ）

| | |
|---|---|
| Status | Accepted |
| Date | 2026-08-30 |
| Sources | `src/server/gkill/api/gkill_server_api/handle_upload_files.go` / `handle_upload_gps_log_files.go` / 2026-08-30 外部監査 F-010 |
| Supersedes | なし |
| Superseded-by | なし |

## Context

`/api/upload_files` は複数ファイルを並列で書き出し、1件でも失敗したらエラーだけを返して
IDFKyou の登録を行わない。**このとき書き出し済みのファイルはディスクに残る**。
GPS 版（`/api/upload_gps_log_files`）も同型で、どちらも doc コメントに明記された既知挙動である。
1ファイル単位では temp+rename の原子性が確保されており（外部監査 M-03 対応）、
`io.Copy` が途中で失敗しても Override 対象の原本は壊れない。

2026-08-30 の外部監査（F-010）が、この挙動を「SQLite から参照されない孤児ファイル・
容量消費・retry 時の衝突・GPS 日次ファイルの部分更新」として P2 指摘し、
staging + 全件成功後の一括 rename + 失敗時 cleanup + 起動時 orphan recovery / manifest の
導入を提案した。対応要否を判断した記録が doc コメントしか無かったため、ここに残す。

## Decision

バッチ原子性は実装しない。**「ファイル単位の原子性 + 部分失敗時は登録スキップ・
書き込み済みファイル残置」を仕様として受容する。**

理由は2つ。第一に、IDF は「rep ディレクトリに置かれたファイルを走査して採番する」rep であり、
部分失敗で残ったファイルは孤児ではなく**次回 IDF 走査の取り込み対象**になる（利用者の意図
＝そのファイルを gkill に入れる、は結果的に満たされる）。第二に、失敗はクライアントへ
エラーとして返るので、利用者の自然な回復手段は再アップロードで、Override 意味論により
同名ファイルは置換されて二重にはならない。

## Rejected alternatives

- **staging ディレクトリ + 全件成功後の一括 rename** — rename が原子的なのはファイル単位だけで、
  「一括 rename」自体がN回の rename であり部分失敗しうる。守れる範囲が1段ずれるだけで、
  実装・テスト規模（L）に見合わない。
- **失敗時 cleanup（書き込めた分を削除する）** — Override アップロードでは rename 完了
  ＝原本置換完了なので、「巻き戻し」は旧内容の復元でありバックアップ機構が別途要る。
  新規分だけ消す分岐は「どれが新規だったか」の判定を持つことになり、判定を誤ると
  **既存ファイルを消す**方向に壊れる。消し過ぎは残し過ぎより重い。
- **起動時 orphan recovery / manifest** — rep ディレクトリは同期ツール・USB・手動コピーで
  持ち回る運用があり、manifest に無いファイル＝孤児とは言えない（外から置かれた正当な
  ファイルを誤判定する）。IDF の走査取り込みという既存の回収経路と競合する。
- **何も記録しない（doc コメントのまま）** — 監査・レビューのたびに「これはバグか仕様か」を
  再検討することになる。今回の監査がまさにそれだった。

## Consequences

- 部分失敗の直後は「ディスクにあるが gkill から見えない」ファイルが残り、容量を消費する。
  次回の UpdateCache（IDF 走査）で採番・取り込みされるので、利用者には
  「失敗したはずのファイルが後から現れた」と見えることがある。
- 対象を失った付随データや取り込み残しの掃除には、既存の dangling cleanup 2バイナリ
  （物理/論理、list→apply の2段階）が使える。
- GPS 日次ファイルの部分更新（複数ファイル中の一部だけ新しい状態）は残る。
  GPS ログは追記マージ主体で、次回アップロードの再送で収束する。

## Evidence

実測なし — 部分失敗の fault injection（disk full・権限失敗・プロセス断）は監査側も安全境界で
未実施であり、本決定でも新たに実施していない。挙動の根拠は
`handle_upload_files.go:196-291`（並列書き出し・エラー集約・登録スキップ）と
`handle_upload_files.go:205-207` の temp+rename コメント。

## Related tests

なし — 受容決定であり新規テストは足していない。挙動の明記は
`src/server/gkill/api/gkill_server_api/handle_upload_files.go` /
`src/server/gkill/api/gkill_server_api/handle_upload_gps_log_files.go` の doc コメント。
