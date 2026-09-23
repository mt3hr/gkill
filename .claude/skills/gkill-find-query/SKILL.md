---
name: gkill-find-query
description: "検索条件 FindQuery の null 判定セマンティクスとワード検索の照合規則。null=フィルタ未使用・非nullの空配列=0件指定、use_* フラグ全廃、ゲートヘルパ経由の判定、TypeScript 側の undefined 禁止、旧形式JSONの移行3実装（Go の find_query_legacy_json.go / client の normalize-legacy-find-kyou-query-json.ts / MCP の constants.go）、ワード検索の型別の対象列・ID は前方一致・除外語は ID を見ない・SQL と Go（find_word）とプラグイン SDK の3実装を揃えることを扱う。src/server/gkill/api/find/・find_word/・find_filter.go・src/client/classes/api/find_query/・src/server/gkill/mcp/constants.go を触るとき、「条件を足したら0件になった」「フィルタが効かない」「無関係な記録がランダムに出る」を調べるとき必読。"
---

# FindQuery の null 判定セマンティクス（Go / TypeScript / MCP 共通）

対象: `src/server/gkill/api/find/**` / `src/server/gkill/api/find_word/**` / `src/server/gkill/api/find_filter.go` / `src/client/classes/api/find_query/**` / `src/server/gkill/mcp/constants.go`

**このファイルは全文が、実際に起きた事故の再発防止である。該当作業では飛ばさずに読むこと。**
多くは「例外もエラーも出さずに静かに壊れる」種類で、破っても目の前ではエラーにならない。

**検索条件（FindQuery）の null 判定セマンティクス:** かつて存在した `use_*` 有効化フラグ（14個）は**全廃**され、いまは **値フィールドが非null（Go では非nil）ならそのフィルタが有効**。`FindQuery` は 55→39 フィールド。間違えると例外もエラーも出ずに静かに 0 件になるので、以下は規約として守ること。
- `null` / `nil` = フィルタ未使用、**非nullの空配列 `[]` = フィルタ有効かつ0件指定**。唯一の例外は `timeis_words: []` で「任意の TimeIs に覆われた Kyou」を意味する
- 3値そろって初めて有効になるグループがある（地図の `map_latitude` / `map_longitude` / `map_radius`）。Go 側は `HasWordFilter()` / `HasTimeIsFilter()` / `HasCalendarFilter()` / `HasMapFilter()` / `HasPeriodOfTimeFilter()` の**ゲートヘルパ経由で判定する**（生の nil 比較を書き散らさない）
- `PeriodOfTimeWeekOfDays` は **nil を先行ガードで弾く**こと。`len==0` / `len!=7` の分岐へ落とすと全件が消える（`find_filter.go` の `sortAndTrimKyousMap` と `sqlite3impl_util.go` の両方に同じ罠がある）
- **Web のサイドバーは曜日を `[]` では送らない**（2026-09-16）。`use-period-of-time-query.ts` の `get_period_of_time_week_of_days()` は画面で1つも押していなければ全7曜日（=曜日制限なし）を返し、全7曜日が props で戻ってきてもローカルの未選択は保つ（全点灯へ書き換えると、次に1つ押した瞬間に「その曜日だけ」ではなく「その曜日を外した6つ」になる）。時間帯にチェックを入れた直後に `[]`（0件指定）を送っていたころは、曜日を押すまで黙って対象なしだった。`[]`＝0件の API 意味論はそのまま（MCP / `add_tag` は変えない）。守るテストは `__tests__/unit/composables/query-composables.test.ts` と `__tests__/e2e/rykv-period-of-time.spec.ts`
- TypeScript 側で `undefined` は禁止。`JSON.stringify` でキーが落ち、localStorage 往復でコンストラクタ既定値が復活し、`deep_equals` のキー数比較が壊れてサイドバーの機械的 re-emit ガードが死ぬ。未使用は必ず `null` で表現する。**禁止の対象は「永続化・比較される値」**（`FindQuery` のフィールド、localStorage へ入るオブジェクト、`deep_equals` に掛ける値）。関数の省略可能引数（`show(query?: FindKyouQuery)`）や Vuetify の `:color="… ? 'error' : undefined"`（prop 既定値を効かせる用法）は対象外
- ただし `FindKyouQuery` のコンストラクタ既定は `tags` / `reps` だけ **`null` ではなく `[]`**（旧 `use_tags=true` + 空配列と厳密等価にするため）
- Mi の板名は `mi_board_name: null` が「すべて」。番兵は `classes/mi-board-names.ts` の **`MI_ALL_BOARD_KEY`（= ハードコードの `"すべて"`。ロケール非依存）** でサイドバー専用、null への変換は `use-mi-query-editor-sidebar.ts` の1点に集約されている。**i18n の訳語（`MI_ALL_BOARD_NAME_TITLE`）と比較してはいけない** ―― ツリーが emit するのはノードの `key` で、それは `append_all_mi_board()` が入れた `"すべて"` 固定なので、訳語と比べると日本語以外のロケールで「すべて」が全件に戻らず 0 件になる（表示名だけが `ALL_MI_BOARD_NAME` / `MI_ALL_TITLE`）
- **削除済みを含めたいときは `IncludeDeletedData`（JSON `include_deleted_data`）を使う。** Kyou 検索の削除除外は `find_filter.go` の1箇所で、既定（false）は従来どおり最新版が削除済みのIDを丸ごと落とす。かつて紛らわしい `IsDeleted` と `HideTimeIsTags` が定義だけ存在し（前者は git の実装が「削除済みのみ」という逆の意味で読んでいた）、どちらも Kyou 検索では一度も参照されなかったが、送っているクライアントが実在しなかったので 2026-08-24 に削除した。MCP の語彙からも外してあるので、送ると未知キーとしてエラーになる。なお rekyou / mirekyou は rep の内部で削除済みを弾いており、この旗の対象外。**プラグインプロトコルの `sdk.Query.IsDeleted` だけは公開 API として残っており、gkill 本体からは常に false が渡る**
- 旧形式JSONの移行は3実装が**同じ16キー**を扱う: Go `api/find/find_query_legacy_json.go`、client `classes/api/find_query/normalize-legacy-find-kyou-query-json.ts`、MCP `mcp/constants.go` の `LEGACY_USE_FLAG_KEYS`。どれかが欠けると、そのフラグを送る古いクライアントの保存クエリが移行されない（MCP では未知キー扱いで throw する）。共有URL用の `share_kyou_info.db` は起動時にスキーマ 1.0.0→1.1.0 で**保存済みJSONそのものを書き換える**（共有URLは配布済みで再発行できないため） 却下案（フラグを残す／値が空ならフラグを無視する）は [ADR-0106](../../../documents/adr/0106-find-query-null-semantics.md)。

**ワード検索の照合規則は SQL（`GenerateFindSQLCommon`）と Go（`find_word.MatchLoweredWords`。プラグイン SDK も同じ関数）で揃える:** 片方だけを変えるとリポジトリ種別によって検索結果が食い違い、エラーにならない。規則は次のとおりで、変えるときは `dao/sqlite3impl/sqlite3impl_util_test.go` の実 SQLite 検査と `api/find_word/match_words_test.go` を同時に直すこと。
- 型別の対象列: kmemo=CONTENT / urlog=URL,TITLE,DESCRIPTION / nlog=TITLE,SHOP,**AMOUNT** / timeis=TITLE / kc=TITLE,**NUM_VALUE** / mi=TITLE,**BOARD_NAME** / lantana=**MOOD**（気分値を文字列として。テキスト列が無いからといって0件や素通しにしない）/ idf=ファイル名＋rep内相対パス＋`.md/.txt` 本文 / git=コミットメッセージ / rekyou・mirekyou=参照先へ委譲。数値列は SQLite の暗黙変換で文字列として LIKE する（CAST 不要）
- **肯定語は「対象列に含む OR ID が語で始まる」（ID は前方一致だけ、しかも語が7文字以上のときだけ）。除外語は「対象列に含まない」で ID は見ない。** 語長のしきい値は `find_word.MinIDPrefixMatchLength` の1か所で、SQL も `find_word.IsIDPrefixMatchWord` で判定する（別々に書くと rep 種別で結果が割れる）。前方一致でも短い語は先頭に当たり、気分の `8` で git のコミットが出ていた（[ADR-0114](../../../documents/adr/0114-word-filter-id-prefix-needs-seven-chars.md)）。完全一致（`ID = ?`）は長さを問わない。 部分一致だったころは `1` / `a` / `cafe` のような hex だけの短い語が UUID に偶然含まれ、無関係な記録が「ランダムに」出て、`-1` が無関係な記録を消していた（1文字なら ~86% の UUID が当たる）。除外語を肯定語として再検索する内部クエリは `FindQuery.WordsSkipIDMatch`（JSON に出ない）を立て、Go 側は id に空文字を渡す
- **除外語は付随テキスト経由にも効く**（`find_filter.go` の `findKyous`）。語なし・除外語だけなら「素通しした全体から除外語のヒット（本体・付随テキスト・ReKyou の参照先）を引く」、語ありなら SQL の NOT LIKE に加えて付随テキストで合流した記録の本体側を再検査し、付随テキストに除外語がある記録も落とす。語ありまで引き算に統一しないのは `foo -の` のような頻出除外語で除外集合が巨大になるため
- 空文字・空白だけの語は Kyou 検索の入口 `FindFilter.FindKyous` が `WithNormalizedWords` で捨てる（`[""]` は LIKE '%%' で全件、除外語なら0件になる）。rep の直叩きやプラグイン SDK は入口を通らないので、SDK の `Query.MatchText` は自前でも捨てる
- プラグイン本体 rep はプラグインが唯一の判定者（gkill は本文を持たず再判定しない。[gkill-plugin](../gkill-plugin/SKILL.md)）。型別アダプタ経由（`rep_types` 指定・ForMi）は上の型別の列で判定するので、プラグインが独自に足した照合対象（fitbit の "Fitbit" 等）はそこでは当たらない

却下案（Lantana 素通し／常に0件、タグ名へのヒット、数値列の完全一致、ID 部分一致の維持、語ありも引き算に統一、SDK への複製、索引へ照合テキストを持たせる）は [ADR-0113](../../../documents/adr/0113-word-filter-columns-and-id-prefix.md)。

## 関連スキル

- [gkill-go-backend](../gkill-go-backend/SKILL.md) — 検索フィルタの実装側（rep名・タグ・IDsチャンク）
- [gkill-plugin](../gkill-plugin/SKILL.md) — プラグイン側のワード判定（SDK の `Query.MatchText`）
- [gkill-client-columns](../gkill-client-columns/SKILL.md) — 局所挿入の判定は `find_filter.go` の意味論の写し
- [gkill-mcp](../gkill-mcp/SKILL.md) — MCP の旧形式キー移行（`LEGACY_USE_FLAG_KEYS`）

## 詳しい設計と却下案（ADR）

- [ADR-0106 FindQuery の null 意味論](../../../documents/adr/0106-find-query-null-semantics.md)
- [ADR-0113 ワード検索の型別の対象列と ID の前方一致](../../../documents/adr/0113-word-filter-columns-and-id-prefix.md)
- [ADR-0114 ID の前方一致は7文字以上の語だけ](../../../documents/adr/0114-word-filter-id-prefix-needs-seven-chars.md)
