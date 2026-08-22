---
name: gkill-client-columns
description: "rykv / mi / dashboard の列と検索の不変条件。列の同一性 query_id、初期化順序、検索を期間の窓で刻まない、記録追加の局所挿入（kyou-local-insert.ts）、その場で作ったタグを列条件へ足す、画面間の変更伝播（kyou-change-bus.ts）、Mi の板名の並び順、ツリーの入れ物除外を扱う。rykv と mi はコピー由来の対称実装なので修正は必ず両方へ。use-rykv-view.ts・use-mi-view.ts・use-dashboard-page.ts・use-kyou-list-view.ts・kyou-local-insert.ts・use-registered-kyou-local-insert.ts・use-registered-tag-column-filter.ts・kyou-change-bus.ts・mi-board-names.ts・use-foldable-struct.ts を編集するとき必読。「検索結果が別の列に出る」「追加した記録が一覧に出ない・直後に消える」「画面が読み込み中のまま固まる」の調査でも必読。"
---

# rykv / mi / dashboard の列×検索の不変条件

対象: `src/client/classes/use-rykv-view.ts` / `use-mi-view.ts`（**対称実装。修正は必ず両方へ**）/ `use-dashboard-page.ts` / `use-kyou-list-view.ts` / `kyou-local-insert.ts` / `use-registered-kyou-local-insert.ts` / `use-registered-tag-column-filter.ts` / `kyou-change-bus.ts` / `mi-board-names.ts` / `use-foldable-struct.ts` / サイドバーの各クエリエディタ

**このファイルは全文が、実際に起きた事故の再発防止である。該当作業では飛ばさずに読むこと。**
多くは「例外もエラーも出さずに静かに壊れる」種類で、破っても目の前ではエラーにならない。

**rykv/mi の「列×検索」不変条件**（2026-08-09 の誤配送根絶で確立。崩すと「検索結果が別の列に出る」が再発する）: 列の同一性は `query_id`（**列の誕生時のみ採番、以後不変**）。`:key`・DOM id・テンプレート ref 逆引き（`get_kyou_list_view`）・`abort_controllers` / `search_seqs`（Map）のキーは全部これ。列リロード・画像トグル・サイドバー clear 系で**再採番してはいけない**（列が remount され、検索結果の帰属も切れる）。検索は列ごとの世代番号 `search_seqs` で「最後の検索だけが書き戻せる」。`focused_query` を更新してよいのはフォーカス列の検索だけ（無条件更新するとサイドバーが別列条件に乗っ取られ query_id 重複→誤配送になる）。サイドバー編集の宛先は `focused_column_index` ではなく `querys.findIndex(query_id)` で解決する。検索ボタンはサイドバーの `generate_query(列のquery_id)` で「今見えている条件」から検索する（`rykv_hot_reload` OFF 時は編集が列に保存されないため。なお既定は **ON**、サーバ既定値 true）。サイドバーの `emits_current_query` は**再生成結果が同期済みクエリと同値なら emit しない**（フォーカス切替で子ビューのprops同期の残響が機械的に届くのを検索にしないための値比較ガード。これが破れると「検索中の列をクリック→飛行中の検索がabortされ最初からやり直し」が再発する。`generate_query` は同期済みクエリに対して恒等であること —— とくに `include_*_mi` をtrue固定でドリフトさせない。`rykv-sidebar-mechanical-emission.test.ts` が守る）。`use-rykv-view.ts` と `use-mi-view.ts` はコピー由来の対称実装なので**修正は必ず両方へ**（`rykv-view-search-routing.test.ts` / `mi-view-search-routing.test.ts` が対で守る）。フォーカス切替時の検索抑止は**コールバック式 `run_with_sidebar_search_suppressed(fn)` だけ**を使う（`skip=true → fn()でリアクティブ書き込み → nextTick(解除)` の順が本質。書き込みより先に nextTick を登録すると、Vue の nextTick が resolvedPromise へ直結して解除がウォッチャ flush より先に走り、抑止が一度も効かない — 2026-08-10 のタブフリーズ回帰の正体）。サイドバーの子クエリビュー（Rep/Tag/TimeIs/Map/Calendar）は **props 同期では emit しない**：TimeIs は同期経路に `disable_emits=true` + `pre_uncheck_all=true`（累積させない）、Map は同期時 emit なし + radius ウォッチャの値ガード、Calendar は `clicked_date` の同値エコーガード（`sidebar-child-query-sync-emission.test.ts` が守る）。`use-kyou-list-view.ts` の `scroll_to` は世代カウンタ + 上限（2秒）つきで、打ち切り時は scrollTop を直接代入する（無限リトライに戻すと 0 件列への 50ms 周期の強制レイアウトが増殖する）。 誤配送が起きた経緯と却下案は [ADR-0034](../../../documents/adr/0034-column-identity-query-id.md)。

**rykv/mi/dashboard の初期化順序**（2026-08-17 に「初期取得の完了まで全画面を隠す」段階を外して確立）。順序は **列の骨組みを確定 → 可視化 → 検索**で、`inited` / `is_loading` は初期検索の完了に依存しない。守るべき約束:
- **`init()` の起動条件は `props.application_config.is_loaded` の watch**。サイドバーの `@inited` へ戻してはいけない ―― あれは「その節が描けた」の集約でしかなく、節を1つ画面から外すだけで画面ごとスピナーで固まる（経緯は ADR-0036）
- **サイドバーの `inited` 集約は無い**が、`inited_*_for_query_sidebar` の各フラグは**残す**。子へ `:inited` prop として降り、子が「初回同期か再同期か」を判定している（消すと props 同期のたびにチェックが列をまたいで累積する）
- **`init()` は hot reload の ON/OFF で分岐せず、列の骨組み（`querys` / `querys_backup` / `match_kyous_list`）を検索より前に確定させる**。1本ずつ足すと「列が確定した瞬間」が定義できず、復元中にユーザが列を足したとき `search(i, ...)` の固定 index と衝突する。`querys_backup` を先に埋めるのは、機械的な残響が `search()` の `deep_equals` 早期returnで確実に落ちるようにするため
- **`init()` で `skip_search_this_tick` を立てっぱなしにしない。** あれは「1tick分の残響を捨てる」短命フラグで、初期化全体の門番に流用すると機械的な emit が1つ届いただけで `onSidebarUpdatedQuery` が消費し、複数列のとき1列目の完了で抑止が途中で解ける。抑止は `run_with_sidebar_search_suppressed` だけを使う
- **復元の検索は `preserve_scroll=true`** で呼ぶ。`inited` が早期に立つので、落とすと `search()` が `scroll_to(0)` を撃って保存済みの復元先を潰す
- **`onSidebarUpdatedQuery` に「初期化が終わるまで捨てる」早期returnを置かない。** 初期検索の飛行中でもユーザの編集は通す。同じ `query_id` を共有するので `abort_controllers` が復元を中断し、`search_seqs` の世代照合が遅れて届いた復元結果を捨てる（＝ユーザが勝つ）
- **`onColumnScrollList` は検索中の列の通知を捨てる。** リストを空にした副作用のスクロール通知を取り込むと `preserve_scroll` の復元先が0で潰れ、保存位置にも焼き付く
- **E2E の準備完了信号はルート要素の `data-gkill-view-ready`**（`is_view_ready` = `inited && !is_restoring_columns && running_search_count === 0`）。全画面オーバーレイが初期検索を待たなくなったので、`.v-overlay .v-progress-circular` の `.first()` は列スピナーを掴み「出る前に `toBeHidden` が通る」窓ができる。`crud-helpers.ts` の `waitForColumnViewReady` がこの属性を待つ。**真偽値をそのまま bind してはいけない**（Vue は false のとき属性ごと消すので「属性の有無」で判定するセレクタが壊れる）
- **①（ApplicationConfig 待ち）は残す。** 未ロードで既定クエリを作ると既定期間と強制非表示タグが黙って落ちる（`generate_default_query_for_rykv` が設定を読むため。初回起動のユーザだけが踏む）
- **設定取得の失敗は永久スピナーにしない。** `application_config_load_failed` を立ててオーバーレイの中身をエラー＋再試行ボタンへ差し替える（文言は既存の `FAILED_GET_APPLICATION_CONFIG_MESSAGE` / `RELOAD_TITLE`）
- dashboard は列を持たないが同じ方針。初回ロードは日付変更時と同じ `fetch_for_date()` に寄せ、パネル単位のローディングで進行を見せる
- 守るテスト: `rykv-view-initial-load.test.ts` / `mi-view-initial-load.test.ts`（対）/ `column-view-init-source-scan.test.ts`（ソース走査）/ `dashboard-page-reload.test.ts` の「ApplicationConfig 取得の失敗」節 / `e2e/column-view-initial-load.spec.ts` 却下案と、@inited が偶然機能していた理由（→ ADR-0036）は [ADR-0035](../../../documents/adr/0035-visualize-before-initial-search.md)。

**検索を期間の窓へ刻んで複数回 `get_kyous` を投げてはいけない**（2026-08-18 に入れて 08-19 に撤去）。
総時間が伸び、スピナーが回り続けているように見え、**境界のレコードがどちらの窓にも入らず静かに落ちる**
（SQL は `.Unix()` の秒切り捨て、`passesPeriodFilter` はナノ秒で、精度が違う）。
狙いをどう外したかの内訳は [ADR-0030](../../../documents/adr/0030-do-not-split-search-window-in-client.md)。

ピークメモリのために分割するなら、固定費を1回で済ませられる**サーバの中**でやること。
**列に部分的な結果を出さないこと** ―― 件数カレンダー・Dnote・Ryuu・フッタの件数はどれも
「列が全件を持っている」前提で書かれていて、部分状態ではエラーも出さずに間違える
（Ryuu は列の配列をサーバ検索の代わりに使うので、間違った「近くの記録」を返す）。
フッタの件数は `has_loaded` が立つまで出さない。守るテストは
`rykv-view-search-routing.test.ts` の「検索は期間が広くても1回で引く」「検索中の列に部分的な結果を出さない」。

**記録の追加は再検索せず、その1件を列へ差し込む**（2026-08-15）。追加系ビューは `registered_kyou` だけを出し、`requested_reload_list` は「サーバが Kyou を返さなかった」ときのフォールバックに退いた。受け口は `use-registered-kyou-local-insert.ts` 1つで、rykv / mi / dashboard がこれを使う。判定と整列は純関数 `classes/kyou-local-insert.ts` にあり、意味論は `server/gkill/api/find_filter.go` の写し。守るべき約束:
- **`/api/get_kyou` は FindQuery を受けない**ので、「その列の条件に一致するか」はクライアントで判定する。判定できるのは rep / タグ（完全一致・大小無視、`"no tags"` 番兵つき）/ カレンダー（両端含む）/ 時間帯 / mi の板名（大小**区別**）・完了状態・`include_*_mi`。判定できないのは本文検索・TimeIs・地図・plaing・画像のみ・`rep_types`（rep_name→rep_type の写像がクライアントに無い）で、**これらを使う列だけ従来どおり再検索する**。判定を足すときは「判定できないものを判定できると誤って宣言しない」ことがすべて —— 誤ると例外もエラーも出ずに黙って一致しない行が出る
- 並び順は非mi=`RelatedTime` 降順（**`.Unix()` 相当に秒へ切り捨ててから**比較。ミリ秒のままだと同一秒の隣接行で位置がずれる）、mi=ソート基準の時刻の昇順で**未設定は末尾**。mi の「未設定」は `typed_mi` ではなく `data_type` の接尾辞で判定する（一覧の既存行は `typed_mi` が未ロードなので、`typed_mi` を要求する比較子は既存行に対して動かない）
- 差し込みは **in-place `splice`**。`focused_kyous_list` は `match_kyous_list[focused_column_index]` へのエイリアスなので、参照ごと差し替えると件数カレンダーや Dnote と縁が切れる（30万件のコピーも避けられる）。ただし Dnote は命令的 reload なので配列を触るだけでは追随せず、明示的に呼び直すこと
- `add_*` の応答は **hydrate を通っていない生 JSON**（`related_time` が文字列、`clone()` も無い）。受け口で必ず実体化する
- KFTL は送信全体を tx で包むが、**tx 中の `add_*` は `added_kyou` を返せない**。リクエストクラスが id だけ積み（`get_result_kyou_ids()`）、`commit_tx` の**あと**に `get_kyou` で引いてから emit する。「終了」系は既存 TimeIs の更新なので `updated_kyou`
- 守るテスト: `kyou-local-insert.test.ts` / `kyou-local-insert-mi-parity.test.ts`（Go の `find_filter_mi_test.go` と対）/ `registered-kyou-local-insert.test.ts`（rykv・mi 両方でパラメタライズ）/ `kftl-submit-emits.test.ts` 判定できない条件の一覧と却下案は [ADR-0031](../../../documents/adr/0031-insert-registered-kyou-locally.md)。

**利用者がその場で作ったタグは、開いている列の検索条件へ足す**（2026-08-19）。
直している不具合は **「タグを付けて追加した記録が、追加した直後に一覧から消える」**。
根本原因は既定クエリがタグ宇宙を列挙で物質化して凍ること（詳細は ADR-0033）。タグが1つも無い時期に作られた列は
`tags = ["no tags"]` だけになり、**タグの付いた記録が1件も通らない**（サーバ検索と局所挿入の両方で落ち、
**エラーも警告も出ない**）。実装は `classes/use-registered-tag-column-filter.ts` の1つで、
rykv / mi が同じものを使う。守るべき約束:
- **使ってよいのは「そのタグがタグツリーに無かった」という決定可能な事実だけ。** 未知だった＝利用者がついさっき
  作った＝「意図的にチェックを外した」ことは原理的にありえない。逆に**既知のタグは触らない**（「保存後に増えたタグ」と
  「利用者が外したタグ」は現状の保存データでは区別できない）。推測しないので新しい永続フィールドは要らない
- **判定は `emits('registered_tag', …)` より前に、同期で行う。** emit 先（`use-rykv-page.ts` / `use-mi-page.ts`）の
  `check_tag_update` がタグツリーへ足したあとでは、「ついさっき作った」ことを二度と知れない。
  順序は `column-view-init-source-scan.test.ts` がソース走査で固定する
- **`tags_and === true` の列には足してはいけない。** AND は `query.tags.every(...)` なので `["no tags", "新タグ"]` の積は
  必ず空になる（新タグ付きの記録は `has_no_tags === false` で落ち、タグ無しの記録は `has_tag_name(新タグ)` で落ちる）。
  足しても目当ての記録は救えず、ほかの記録を巻き込んで**列を丸ごと消す**。`tags === null` の列も触らない（元から通る）
- **`querys` と `querys_backup` を同じ tick で揃え、`run_with_sidebar_search_suppressed` で包む。** 控えがずれると
  サイドバーの機械的な残響が `search()` の `deep_equals` 早期returnで落ちなくなる
- **localStorage へは自分で書かない。** `search()` が必ず `set_saved_*` を通るので、引き直しを通せば揃う。
  自前で書くと「条件だけ変わって引き直さない」経路が生まれ、**次回起動時だけ列が変わる**という最悪の非対称になる
- **1tick ぶん溜めてから1回だけ流す。** `add_tags_to_target` はタグを1件ずつ登録して1件ずつ emit するので
  （`kyou-tags.ts`）、まとめないと新タグ3つで列あたり3本 `search()` が走り2本が abort されるだけになる
- **ポート（rudbeckia）へも配る。** タグが検索条件に効くようになったので `kyou-change-bus.ts` の
  「タグは配る必要が無い」という前提が崩れた。publish するのは**未知と判定した発生元だけ**で、
  受け手は既知判定を**やり直さない**（届く頃にはツリーに載っているので必ず取りこぼす）。`apply_registered_tag` は
  optional（dashboard / plaing は列のタグ絞り込みを持たない）。受け手では `reload_list` の畳み込みより**先**に適用する
  （逆だと旧条件のまま全件取り直す）
- **今回の修正では直らないもの**（区別できる情報が保存データに無い）: 他端末で作られたタグ、過去に作ったタグ、
  プラグインKyouのタグ。本命の対処（既定クエリの物質化をやめる）が別件である理由は ADR-0033
- 守るテスト: `registered-tag-column-filter.test.ts` / `new-tag-column-search.test.ts`（rykv・mi 両方）/
  `column-view-init-source-scan.test.ts` / `kyou-change-bus.test.ts` /
  `e2e/regression-fixes.spec.ts` の「新規タグを付けて追加した記録が、画面遷移せずに一覧へ残る」
  （**画面遷移しないことが本質** ―― 遷移すると既定クエリを作り直すのでこの不具合をすり抜ける） 既定クエリの物質化という根本原因と、この修正で直らない範囲は [ADR-0033](../../../documents/adr/0033-add-unknown-tag-to-column-filter.md)。

**画面間の変更伝播（`classes/kyou-change-bus.ts`）**
- **購読側へ渡してよいのは emit を含まない適用関数だけ。** 中継束（`crudRelayHandlers`）を渡すと適用のたびに `emits(...)` が走ってホストが再 publish し、通知が無限に往復する。そのために `onDeletedKyou` は `apply_deleted_kyou`（適用のみ）と `onDeletedKyou`（適用＋emit）に割ってある
- **通知は seq 付きの追記ログ。** スカラー（最新の1件）だと同じ tick に複数件起きたとき最後の1件しか見えず、残りが黙って落ちる（KFTLの複数行保存が典型）
- **`KyouChangeBus.last_seq` は Ref ではなくメソッド。** Ref をそのまま公開すると、チャネルのオブジェクトが `reactive()` に包まれたとき Vue が自動アンラップして `.value` が `undefined` になり、**伝播が黙って効かなくなる**（テストのハーネスが実際に踏んだ）
- **バスは props で配る。`provide`/`inject` にしない。** 既存のテストは `useRykvView({props, emits})` をコンポーネントインスタンスの外から素で呼ぶので、`inject()` は警告を出して既定値へ落ちる ＝ テストでは伝播が効かないのに緑になる
- **`new_reload_batch()` は発生元のビューが採番**し、自分のローカル適用と通知の両方に同じ値を使う。ホスト側で採番すると `kyou-reload.ts:226` の合流条件（`started_at >= requested_at`）に間に合わず、同じ Kyou を画面の枚数ぶん取りに行く
- **自分が出した通知は受けない**（`origin_id`）。受けると発生元が二重適用する。追加は `insert_kyou_sorted` の id 重複判定で救われるが、削除と引き直しは救われない
- **`reload_list` は1ドレイン1回に畳む。** 畳まないと1回の KFTL 保存で開いている画面ぶんの全件検索が走る
- **後から開いたウィンドウは購読開始時点の最大 seq から始める。** 0 から始めると過去の変更を全部再生する
- **`requested_reload_kyou` を握り潰さない。** タグ・テキスト・通知の変更はこれしか出さないので、配らないと付随データの変更が他の画面に一切届かない
- **`requested_update_check_kyous` は配らない。** 列ごとの選択状態であり、rykv/mi では未実装（throw する）
- ポートのFABから追加した記録も配る（発生元 `rudbeckia-page`）。配らないと「＋から足したのに並べている一覧に出ない」になる
- 守るテスト: `e2e/rudbeckia.spec.ts` / `kyou-change-bus.test.ts` / `kyou-change-propagation.test.ts` / `dashboard-view-reload.test.ts` / `dashboard-page-reload.test.ts` / `column-view-init-source-scan.test.ts` / `kftl-dialog-host.test.ts` / `rykv・mi-view-initial-load.test.ts`（対）

**Mi の板名の並び順と板ツリーのクリック**（2026-08-16）。純関数は `classes/mi-board-names.ts`、守るテストは `mi-board-names.test.ts` / `mi-board-query.test.ts`。
- **板名プルダウンの並び順は ApplicationConfig の板ツリーが正。** `get_mi_board_list` は Go の map を回して集めているので**順序を保証しない**（`dao/reps/mi_repositories.go` / `mi_re_kyou_repositories.go`。interface の doc コメントにも明記）。素で `:items` に渡すと読み込むたびに並びが入れ替わるので、5箇所の `v-select`（add/edit Mi・add/edit MiReKyou・設定の既定の板）はどれも `sort_mi_board_names_by_config_order()` を通す
- 並べ替えは **`computed`** で表現する。`use-edit-mi-view.ts` / `use-mi-re-kyou-schedule-fields.ts` には `application_config` の watch が無く、ある側の watch も**参照同一性**しか見ないので（設定ダイアログでの並べ替えのような deep な変更では発火しない）、代入時に1回ソートすると板を並べ替えても追随しない。API の生の一覧は `mi_board_names_source` に持ち、`update_board_name()` の push 先もそちら
- 設定画面だけは並べ替えの基準が **`cloned_application_config`**。子の板構成ダイアログの「適用」は clone にだけ書くので、props を見ると並べ替え直後に既定板のプルダウンだけ古い順で取り残される
- **設定にしか無い名前を候補へ足さない。** とくに「すべて」は `append_all_mi_board()` が入れる仮想ノードで実在の板ではないので、Mi/MiReKyou の板名候補に混ぜてはいけない
- **サイドバーの板ツリーのルート行では何も開かない**（`resolve_clicked_mi_board_names()`）。ルートは `folder_name=''` で描いていて見た目は空白だが `.tree_item { min-width: 200px }` のクリック領域が残っており、踏むと `click_group_by_user()` が**自分自身の key（`__root__`）を含めて**サブツリー全部の key を上げてくる。素通しすると `__root__` という名前の列 + 板の数だけの列が一度に開く。判定は「フォルダ扱いのノード（`is_dir` か `board_name` が空）の key が混ざっていたらグループ行のクリック → 何も開かない」。**ツリーに無い key は開く** ―― 作った直後で `append_not_found_mi_boards()` がまだ拾えていないだけかもしれず、落とすと「板をクリックしても何も起きない」（エラーも出ない）になる
- 共有の `use-foldable-struct.ts` の `click_group_by_user` は**触らない**。自分の key を含める挙動は tag/rep/timeis のチェックボックス経路も使っており、そちらは「フォルダ行クリックで配下を一括チェック」が意図された機能（`foldable-struct-check.test.ts` が等価性を固定している）。直すのは読み取り側（下記）

**ツリーの「入れ物」を検索条件へ混ぜない**（2026-08-16）。判定は `pages/views/foldable-struct-model.ts` の **`is_struct_container_node(struct)`**（`is_dir` またはキーが `__root__`）の1つだけ。守るテストは `foldable-struct-selected-items.test.ts` / `collect-inited-tag-names.test.ts`。
- ルートとフォルダは並べ替えのための器でしかないのに、`key` にはフォルダ名が（ルートは `__root__` が）そのまま入る。チェックの入ったノードの key は**そのまま検索条件（`tags` / `reps` / `devices_in_sidebar` / `rep_types_in_sidebar` / `timeis_tags`）として流れる**ので、入れ物を混ぜると実在しない名前の条件が紛れ込む。**OR検索では無害だが AND検索（`tags_and` 等）では必ず0件になる**ため、症状が出るときだけ出る
- ルート行は `folder_name=''` の空白帯（`.tree_item { min-width: 200px }`）としてクリックでき、踏むと `click_group_by_user` が `__root__` を載せてくる。**誤クリックだけで条件に入る**
- 除外する場所は4つ: `use-foldable-struct.ts` の `get_selected_items()`（対話経路。tag/rep/timeis の3コンポーザブルが**唯一この関数から**条件を読む）、`collect-inited-tag-names.ts`、`find-kyou-query.ts` の `device_name_walk` / `rep_type_name_walk`（既定クエリ経路）、同 `apply_rep_summary_to_detaul` の `collect_checked_keys`。**最後のものは `indeterminate=false` のクリアが入れ物にも要るので、walk は打ち切らず集合へ入れないだけにする**
- **フォルダ名と同名のタグが実在しても条件は落ちない。** `apply_check_state_to_struct` が key 一致でツリー全体を走査して葉のほうにもチェックを入れるため。実運用の `TAG_STRUCT` ではフォルダの大半が同名タグを持たない純粋な入れ物で、しかもその一部は `check_when_inited=true` で保存されている ―― つまり誤クリックしなくても既定の検索条件に幽霊タグが入りうる
- 入れ物かどうかを `is_dir` だけで見ないこと。保存済みJSONのルートに `is_dir` が無い実例があり（`gkill-api.ts` は `children` が falsy のときしか立てない）、そのときルートは**葉として描かれて `__root__` がそのまま条件に入る**

## 関連スキル

- [gkill-client-foundation](../gkill-client-foundation/SKILL.md) — 必ず併読（中継束・Kyou の再読込・updated_kyou）
- [gkill-client-tags](../gkill-client-tags/SKILL.md) — `add_tag` → `registered_kyou` の順序（局所挿入の前提）
- [gkill-client-rudbeckia](../gkill-client-rudbeckia/SKILL.md) — 複数枚の保存キー分離・ウィンドウホスト
- [gkill-find-query](../gkill-find-query/SKILL.md) — 局所挿入の判定は `find_filter.go` の意味論の写し

## 詳しい設計と却下案（ADR）

- [ADR-0030 検索を期間の窓へ刻まない](../../../documents/adr/0030-do-not-split-search-window-in-client.md)
- [ADR-0031 追加は局所挿入](../../../documents/adr/0031-insert-registered-kyou-locally.md)
- [ADR-0033 未知タグを列条件へ足す](../../../documents/adr/0033-add-unknown-tag-to-column-filter.md)
- [ADR-0034 列の同一性は query_id](../../../documents/adr/0034-column-identity-query-id.md)
- [ADR-0035 可視化を初期検索より先に](../../../documents/adr/0035-visualize-before-initial-search.md)
- [ADR-0036 init は ApplicationConfig ロード後](../../../documents/adr/0036-init-on-application-config-loaded.md)
