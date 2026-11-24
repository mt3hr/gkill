# ADR-0422: 出力 HTML の中にも閲覧者の手元にも、選んだもの以外を残さない — 地図のキーは保存せず、リポストは列の記録のぶんだけ焼き、警告は焼かない

| | |
|---|---|
| Status | Accepted |
| Date | 2026-11-25 |
| Sources | [ADR-0415](0415-exported-view-is-a-separate-flag.md)（出力用の API と通信の1点止め） / [ADR-0903](0903-export-shell-is-a-second-vite-build.md)（出力のガワ。通信の例外は地図だけ） / [ADR-0417](0417-exported-sidebar-narrows-only.md)（出力のサイドバーは焼いた記録の外を出さない） / `src/client/classes/api/gkill-api.ts` の `GkillAPIForExportedKyou` / `src/server/gkill/api/gkill_server_api/kyou_bundle_collector.go` の `collectTypedDatas` / CodeQL `js/clear-text-storage-of-sensitive-data` |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/client/classes/api/gkill-api.ts` の `google_map_api_key_in_memory` のコメント、`kyou_bundle_collector.go` の `keepByIDs`、`export_plugin_html.go` の `exportWarningMessage` |

## Context

「HTML で出力」したファイルは、出力した本人の手を離れて別の人・別の端末で開かれる。
そのため「選んだものだけが入っている」「開いた人の手元に余計なものを残さない」の2つが、
画面の見た目とは別に守られていなければならない。見直しで、この2つを破っている箇所が3つ見つかった。

1. **閲覧者が入れた Google Maps の API キーを localStorage へ平文で保存していた。**
   出力 HTML は file:// で開かれ、Cookie は書けないので localStorage を選んでいた。
   ところがブラウザによっては file:// の保存領域が、同じブラウザで開いた別のローカル HTML と共有される。
   CodeQL もこの保存を指摘した（`js/clear-text-storage-of-sensitive-data`）。
2. **選んでいないリポストまで、参照先の記録の中身ごと焼き込まれていた。**
   型別データを引く `ReKyouRepositories.FindReKyou` は検索条件を見ずに、利用者のリポストを全件返す。
   収集（`collectTypedDatas`）はそれをそのまま積んでいたので、列に出ていないリポストが焼き込みに入り、
   出力はさらにその参照先（`collectExportReKyouTargets` が辿る）の本文まで入れていた。
   共有ページも同じ収集を通るので、リポストの対応（どの ID がどれを指すか）が載っていた。
3. **出力しきれなかったものの警告文を、焼き込みにも入れていた。** 警告は出力した本人への知らせで、
   渡した相手に見せるものではない（しかも日本語の直書きで、ロケールに関係なく日本語だった）。

## Decision

- **地図のキーは `GkillAPIForExportedKyou` のメモリにだけ持ち、どこにも保存しない。**
  ページを閉じたら消えるので、閲覧者は開くたびに入れ直す。入力欄の案内文もそう書く。
- **型別データのリポスト（ReKyou / MiReKyou）は、列に出た記録の ID だけに絞ってから積む**（`keepByIDs`）。
  rep 側の `FindReKyou` が全件を返す挙動は変えない（MCP の検索は ID の対応表として引くだけで、
  ここを直すと影響が広い）。絞るのは焼き込みの入口で1回。
- **警告は焼き込まない。** 応答の `messages` に warning のメッセージ（`MSG000096`）として、成功の知らせと並べて返す。
  文言は i18n キー（7言語）で、差し込みは vue-i18n と同じ `{count}` の形で書き、サーバで置き換える。
  応答の `warnings` 配列と、それをクライアントが擬似コードのメッセージへ詰め替えていた経路は撤去した。

## Rejected alternatives

- **キーを sessionStorage に置く** — タブを閉じれば消えるが、平文で保存領域に残る点は同じで、file:// の共有の問題も残る。
- **「このブラウザに覚えさせる」を選べるようにする** — 利用者の判断で保存を選べるが、UI とテストが増える。
  Google Maps のブラウザ用キーは入れ直せば済む値で、覚えさせる利点が小さい。
- **今のまま CodeQL を棄却する** — Maps のブラウザ用キーはもともとページに出る前提の値だが、
  file:// の保存領域を共有する別の HTML から読める状態は避けたい。棄却の理由を書くより保存しないほうが安い。
- **`FindReKyou` 自体を検索条件に従わせる** — 本来はそうあるべきだが、rep の実装4本とキャッシュの経路が
  「全件を返す」前提で組まれていて、MCP の検索と Mi の検索の結果まで変わりうる。出力・共有の入口で絞れば、
  漏れを止めるという目的は1か所で果たせる。
- **警告を焼き込んだまま、出力ページでは出さない** — 出さないなら焼く理由が無く、ファイルの中には残る。

## Consequences

- 出力 HTML の地図は、開くたびにキーの入力が要る。入れたキーは画面を閉じるまで使える。
- 出力と共有の焼き込みに入るリポストは、列に出たものだけになる。列に出たリポストの参照先は
  これまでどおり最大3段まで辿って焼く（`exportReKyouTargetMaxRounds`）。
- 応答から `warnings` が消えた。`partial` は残り、何を落としたかは `messages` の warning で分かる。
- `MSG000096` と警告文のキー5本が増えた。差し込みの名前を訳してしまうと、その言語でだけ `{count}` が
  文字のまま見えるので、テストが7言語で置き換え残しが無いことを見る。

## Evidence

実測なし — いずれもコードを読んで経路を確かめ、テストで再現した（コード由来）。リポストは、4段の連鎖を作って
列に最初の1件だけを出したとき、絞る前は連鎖の先の記録まで焼かれ、絞った後は3段で止まることをテストで確かめた。

## Related tests

- `src/client/__tests__/unit/api/exported-kyou-api.test.ts` — 地図のキーを localStorage にも Cookie にも書かない
- `src/server/gkill/api/gkill_server_api/export_rekyou_targets_test.go` — 列に出ていないリポストを型別データに入れず、参照先は上限の段数で止まる
- `src/server/gkill/api/gkill_server_api/export_plugin_html_test.go` — 警告が warning の `MSG000096` で返り、7言語で差し込みが置き換わる
- `src/server/gkill/api/gkill_server_api/export_media_test.go` — メディアの警告が件数を差し込んだ1件で返る
- `src/client/__tests__/unit/composables/export-kyou-html-view.test.ts` — サーバの warning のメッセージを詰め替えずに通知する
