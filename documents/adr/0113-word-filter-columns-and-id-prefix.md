# ADR-0113: ワード検索は型別の対象列を決め、ID は前方一致だけ、除外語は付随テキストにも効かせ、判定は SQL / Go / プラグイン SDK で1つに揃える

| | |
|---|---|
| Status | Accepted |
| Date | 2026-09-12 |
| Sources | `.claude/skills/gkill-find-query/SKILL.md`「ワード検索の照合規則は SQL（`GenerateFindSQLCommon`）と Go（`find_word.MatchLoweredWords`。プラグイン SDK も同じ関数）で揃える」節 / `.claude/skills/gkill-plugin/SKILL.md`「ワード検索の判定はプラグインが唯一の判定者」節 |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/api/find_word/match_words.go`（パッケージコメント）/ `src/server/gkill/dao/sqlite3impl/sqlite3impl_util.go`（ワード検索の SQL 追記の直前）/ `src/server/gkill/api/find_filter.go`（`findKyous` の除外語の2経路）/ `src/server/gkill/plugin/sdk/match_words.go` |

## Context

ワード検索（`FindQuery.words / not_words / words_and`）は型ごとに見る列がばらばらで、テキスト列を持たない
Lantana は「ID 列だけを部分一致」という実装になっていた。その結果、次の歪みがあり、いずれもエラーにならず静かに起きていた。

- **ID の部分一致が短い英数字語でノイズを生む。** `1` / `a` / `cafe` のような `[0-9a-f]` だけの語は UUID に含まれうるので、
  Lantana が「ランダムに」出る。除外側も同じで `-1` は UUID に 1 を含む記録を本文と無関係に消していた。
- 付随テキスト経由で合流した記録には除外語が効かず、付随テキストに除外語があっても本体は残っていた。
- 空文字の語 `[""]` をサーバで弾いておらず、`words:[""]` は全件、`not_words:[""]` は0件になっていた（画面は空語を作らないが MCP / API 直叩きで届く）。
- プラグイン: SDK に共通判定が無く、4プラグインが同じループをコピペし、fitbit は独自実装（空文字の語で結果が変わる）、
  雛形の `gkill_example` はワードを完全に無視していた。gkill 本体はプラグインが返した Kyou のワードを再判定しない
  （本文を持たない）ので、プラグインが唯一の判定者だが、その契約はどこにも書かれていなかった。
  型別アダプタ（`plugin_typed_adapters.go`）の `findKyous` もワードを見ておらず、`rep_types` 指定・ForMi・ReKyou の委譲では
  語に関係なくプラグインの記録が全件当たっていた。
- `lantana_repository.go` の doc「ワード条件が有効なら常に0件」、MCP スキーマの「`[]` matches nothing」がどちらも実装と食い違っていた。

## Decision

型別の対象列を決める（Lantana は MOOD、Nlog は AMOUNT、KC は NUM_VALUE、Mi は BOARD_NAME を足す。数値列は SQLite の暗黙変換で
文字列として LIKE する）。肯定語は「対象列に含む OR ID が語で**始まる**」、除外語は「対象列に含まない」で **ID は見ない**。
除外語は付随テキスト経由にも効かせ、語なし・除外語だけのときは「素通しした全体から除外語のヒットを引く」。
空文字・空白だけの語は Kyou 検索の入口で捨てる。Go 側の判定は依存ゼロの `api/find_word` に1つだけ置き、本体（IDF / git /
型別アダプタ）とプラグイン SDK（`Query.MatchText`）の両方がそれを使う。

## Rejected alternatives

- **Lantana はワード条件の対象外（常に素通し）にする** — 「テキストを持たない型はワードで絞れない」という筋は通るが、
  キーワード列に気分記録が全部混ざり、キーワード欄へ ID を貼る検索でも Lantana が全件出る。利用者が却下。
- **Lantana は語ありなら常に0件（現状維持）** — 除外語だけなら全件残るのに肯定語なら0件という非対称が残り、
  doc とも食い違ったまま。気分値を文字列として照合すれば `7` で気分7が引け、他の数値列と同じ規則で説明できる。
- **タグ名にもヒットさせる** — タグは別次元のフィルタ（`tags`）で既に絞れる。合流経路をもう1本足す実装の大きさに対して
  利用者が見送り。将来の検討。
- **数値列は完全一致にする** — REAL 由来の `1500.0` と `1500` が外れ、`65.4` に `65` が当たらない。部分一致なら
  保存型（TEXT / INTEGER / REAL）によらず当たる（`TestGenerateFindSQLCommon_NumericColumnMatchesAsText`）。
  `1` が 1 と 10 に当たる程度の緩さは受け入れる。
- **ID は部分一致のまま** — UUID は hex なので、`[0-9a-f]` だけの語は偶然含まれる。1文字なら約 86% の UUID が当たり
  （30桁の hex に特定の1文字が含まれない確率 (15/16)^30 ≈ 14%）、3文字でも約 0.7%。本文ヒットに隠れて気付かないが、
  本文列の無い Lantana ではそれが唯一の照合だったので露骨に出ていた。前方一致なら UUID 丸ごとの貼り付けと
  git の短縮ハッシュ（前方一致）はそのまま引ける。
- **除外語も ID を見る** — 「`-1` で UUID に 1 を含む記録が消える」の再発。ID を除外したい用途は無い。
- **除外語の再検査は ID 再検索に `NotWords` を直接足す** — ReKyou の委譲（`isWordFilterEnabled`）が `IDs` 付きで走り、
  参照先を `IDs` で引けずに付随テキストが当たった ReKyou が全部落ちる。除外語を肯定語として検索して引くなら
  ReKyou は保守的に残る。
- **語ありのときも「全体から引く」に統一する** — `foo -の` のような頻出除外語で除外集合が本検索と同規模になり、
  走査と実体化が2回になる。SQL の `NOT LIKE` は同じ走査の中で済むので、語ありは従来どおり SQL で除外し、
  付随テキストで合流した記録だけ再検査する。
- **除外語の再検査で ID 照合を切らない** — 除外語を肯定語として検索するので、旗（`WordsSkipIDMatch`）が無いと
  「ID が除外語で始まる記録」まで除外され、`-a` で 1/16 の記録が消える。
- **Go 側の判定を `api/find` に置く** — `find` は `gkill_log` → `gkill_options` を引く。プラグイン SDK は gkill のパッケージを
  1つも import しない方針（`cache_path.go` の「SDK と本体の依存を混ぜない」）なので、標準ライブラリだけの葉 `api/find_word` に分けた。
  （追記 2026-09-20: 同じ「標準ライブラリだけの葉」として `gkill_log` / `gkill_options` も SDK から import するようになった。[ADR-0313](0313-plugin-logs-through-gkill-log.md)）
- **SDK 内に判定を複製して「直すときは両方」コメントにする（`cache_path.go` 方式）** — 判定規則は今回まさに変わった。
  複製は必ずずれ、rep 種別によって結果が食い違う静かな壊れ方になる。
- **型別索引にプラグインの照合テキストを持たせるプロトコル拡張** — 索引が本文サイズぶん膨らむ。型別アダプタは native と
  同じ列で判定するほうが説明できる（fitbit の "Fitbit" という語が `rep_types` 指定では当たらない差は資料に明記する）。

## Consequences

- 判定規則を変えるときは SQL（`GenerateFindSQLCommon`）と Go（`find_word.MatchLoweredWords`）の両方と、
  実 SQLite に投げる `sqlite3impl_util_test.go` を同時に直す。片方だけ変えても**エラーは出ず、rep 種別によって結果が食い違う**。
- プラグインが SDK の `Query.MatchText` を使わずに自前判定を書いても gkill 側では検出できない。雛形 `gkill_example` が
  SDK 経由になったので、写した第三者プラグインも同じ規則になる。
- 型別アダプタ経由（`rep_types` 指定・ForMi）とプラグイン本体経由で照合対象が違いうる（後者はプラグインが決める）。
- プラグイン本体 rep には `WordsSkipIDMatch` が届かない（`PluginQuery` に無い）ので、除外語の再検査で
  「ID が除外語で始まるプラグイン記録」だけは消えうる。実害が出たら `PluginQuery` に旗を足す。
- `timeis_words` / `timeis_not_words`（TimeIs 側のワード）は今回対象外で、従来どおり SQL の `NOT LIKE` だけ。
- chatgpt / claudeai は会話タイトルも照合対象になった。既存キャッシュの作り直しは不要（照合時に JOIN で載せる）。
  プラグインの配布は本タスクの外で、バイナリを更新するまでは旧規則で動く。

## Evidence

- ID の部分一致が偶然当たる確率（UUID の hex 30桁、語が hex だけの場合）: 1文字 ≈ 86%、2文字 ≈ 11%、3文字 ≈ 0.7%、4文字 ≈ 0.04%。
  非 hex 文字（g〜z、日本語）を含む語は当たらない。
- 性能の実測なし — 語なし・除外語だけの検索は「素通し（全件）＋除外語の肯定検索」の2走査で、除外語の検索は通常の
  ワード検索1回ぶん。語ありの検索は従来どおり1走査＋付随テキスト分の ID 検索。

## Related tests

- `src/server/gkill/api/find_word/match_words_test.go`
- `src/server/gkill/api/find/find_query_test.go`
- `src/server/gkill/dao/sqlite3impl/sqlite3impl_util_test.go`
- `src/server/gkill/dao/reps/find_word_match_test.go`
- `src/server/gkill/dao/reps/plugin_typed_adapters_test.go`
- `src/server/gkill/api/gkill_server_api/get_kyous_word_filter_test.go`
- `src/server/gkill/plugin/sdk/match_words_test.go`
- `src/plugins/gkill_plugin_chatgpt/find_kyous_test.go`
- `src/plugins/gkill_plugin_claudeai/find_kyous_test.go`
- `src/plugins/gkill_plugin_claudecode/find_kyous_test.go`
- `src/plugins/gkill_plugin_codex/find_kyous_test.go`
- `src/plugins/gkill_plugin_fitbit/find_kyous_test.go`
