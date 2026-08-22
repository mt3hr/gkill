---
name: gkill-client-foundation
description: "gkill フロントエンド（src/client/）全域の約束。他の gkill-client-* スキルに該当するかに関わらず src/client を触るなら読む。コンポーザブル規約と CRUD 中継束（kyou-view-relay.ts）、Kyou の再読込手順（kyou-reload.ts）、タグ変更は updated_kyou を出さない、コンテキストメニュー位置・チェックツリー単一パス・端末種別・自動フォーカス・行高さ・HTML リテラル禁止・設定の適用、Web Share Target の二重保存対策、check_auth のログイン画面ガードを扱う。src/client/ の .vue / classes/*.ts を編集・新規作成するとき必読。「タグを足しても表示が変わらない」「Android 共有が2件保存される」の調査でも必読。"
---

# クライアント全域の約束（src/client/）

対象: `src/client/**` すべて。**他の gkill-client-* スキルに該当する場合も、このファイルの約束はすべて効く。**

**このファイルは全文が、実際に起きた事故の再発防止である。該当作業では飛ばさずに読むこと。**
多くは「例外もエラーも出さずに静かに壊れる」種類で、破っても目の前ではエラーにならない。

### Frontend (Vue 3 + TypeScript) — `src/client/`

Stack: Vue 3 + Vuetify 4 + Vue Router 5 + vue-i18n 11 + Vite 8 + TypeScript 6 + PWA (vite-plugin-pwa + Workbox)

- `router/index.ts` — 13 page routes (login, kftl, mi, rykv, kyou, mkfl, plaing, saihate, dashboard, rudbeckia, set_new_password, register_first_account, shared_page) + 2 redirect-only routes（`/regist_first_account` → `/register_first_account`、`/shared_mi` → `/shared_page`。どちらも query を引き継ぐ）。**旧パスの吸収は redirect でやること** ―― コンポーネントの setup から `router.replace` すると、`<script setup>` に top-level await があるページでは初回ナビゲーションが完了しなくなる（`/shared_mi` が実際にそうなっていた。share_id 無しで throw して setup ごと落ちていたため、redirect が一度も走らず露見していなかった）
- `pages/views/` — 202 view components, `pages/dialogs/` — 116 dialog components (Escape key closes via `useFloatingDialog`), including ZIP contents browser, plugin HTML views (`plugin-html-view.vue`, `plugin-html-context-menu.vue`, `plugin-config-dialog.vue`), and Dnote trend/correlation graph components (client-side aggregation, no server API)
- `classes/api/gkill-api.ts` — Singleton `GkillAPI` class (~3,400 lines), client-side API wrapper
- `classes/cascade-delete-kyou.ts` — cascade delete for Kyou. The attached Tag / Text / Notification and the ReKyou / MiReKyou that reference the Kyou are looked up in reverse via `GetReKyousByTargetID` / `GetMiReKyousByTargetID` and logically deleted together with it. Depth cap 32 (`max_cascade_depth`), 16 lookups in flight per level (`request_chunk_size`). **The Kyou itself is deleted last** (deleting it first makes the server's `FindKyous` drop the referencing records from its results, so the reverse lookup can no longer find them). No TXID / `commit_tx` is used, so a partial commit is possible. On failure: ERR900093 `cascade_delete_depth_exceeded` / ERR900094 `cascade_delete_failed`, i18n key `FAILED_CASCADE_DELETE_KYOU_MESSAGE`
- `serviceWorker.ts` — PWA service worker (Workbox precaching, POST caching, push notifications, Web Share Target; `/zip_cache/.*` on NavigationRoute denylist)

**State management:** Props/Emit only. No Pinia/Vuex. `GkillAPI` singleton for backend communication. Pinia/Vuex を入れない理由と却下案は [ADR-0038](../../../documents/adr/0038-props-emit-only-no-pinia.md)。

**Android共有（Web Share Target）の二重保存対策**（2026-08-16）。`POST /share-target` は `serviceWorker.ts` が
`respondWith` で丸ごと処理し、その場で `add_urlog` / `add_kmemo` を叩いてから `/saihate` へ 303 で送る。
**Android はタスク（アプリ履歴）から復帰すると同じ共有インテントを再配送する**ので、初回とビット単位で同じ
multipart POST がもう一度届き、素直に保存すると2件目ができる。台帳は `classes/share-target-dedup.ts`。守るべき約束:
- **再配送と意図的な再共有は内容から区別できない。** 見分ける手は「保存済みの内容を覚えておく」以外に無いので、
  台帳（`gkill-share-dedup-cache` の `/__gkill_share_dedup/ledger`、直近100件・24時間）と内容の完全一致で判定する
- **台帳へ載せるのは保存が成功したときだけ。** 応答を見ずに載せると、保存できていないのに次の共有が24時間弾かれる
  （`is_successful_gkill_response` で HTTP ok と `errors` 空を確認する）
- **重複は黙って捨てず最果てで確認を出す**（`confirm-save-duplicated-shared-data-dialog.vue`）。
  黙って捨てると意図的な再共有が理由なく消える
- **「それでも保存する」も `/share-target` を通す**（フォームに `gkill_force` を立て、応答は 303 ではなく JSON）。
  ページ側で `add_urlog` / `add_kmemo` を組み立て直すと保存が2実装に割れる
- **共有由来のクエリ（`is_saved` / `share_result`）は最果てに入った瞬間に落とす。** 残したままアプリ履歴から
  開き直されると「保存しました」がもう一度出て、保存していないのに二重保存に見える
- 台帳は Kyou キャッシュとは**別のキャッシュ**に置く（`activate` が `KYOU_CACHE_NAME` を丸ごと消すので、
  同居させると版が上がるたびに台帳が飛ぶ）
- 守るテスト: `share-target-dedup.test.ts`

**Composable pattern:** ロジックを持つ view/dialog は必ず対応する `classes/use-*.ts` にコンポーネントのロジックを置き、`.vue` の `<script setup>` は「import・`defineProps`・`defineEmits`・コンポーザブル呼び出しの分割代入・`defineExpose`」だけにする（dialogs は **116/116** がこの形）。例外は**ロジックが1行も無いコンポーネント** ―― props と emits を宣言してテンプレートから直接 emit するだけの`sidebar-header.vue` / `manage-share-task-list-view.vue` / `confirm-delete-share-task-list-view.vue` は、空の `return {}` を返すだけのコンポーザブルを置かない（実際に3本そういう死んだファイルが残っていた）。 Props and emits are defined in `-props.ts` / `-emits.ts` files alongside the `.vue` file (many components share a common props/emits type instead of having their own pair). Template refs to child components use the `ComponentRef` type (`classes/component-ref.ts`). Never hand-write the CRUD relay handler bundle that only forwards child events to the parent: call `build_kyou_view_relay(emits, overrides?)` (view layer, 18 events) or `build_kyou_dialog_relay(emits, overrides?)` (dialog layer, those 18 plus `focused_kyou` / `clicked_kyou` = 20 events) from `classes/kyou-view-relay.ts`, assign it as `const crudRelayHandlers = build_kyou_view_relay(emits)`, and pass it to the template as `v-on="crudRelayHandlers"` (one line replaces the whole `@…` list). Swap out only the events whose behavior differs via `overrides`. Of the 21 events in `KyouViewEmits`, `requested_close_dialog` is not relayed (the dialog wires it to itself with `@requested_close_dialog="hide()"`), and `focused_kyou` / `clicked_kyou` are not relayed by the view layer (a nested view would fire them twice — only the dialog layer relays them). When adding an event, add it to **both** `KyouViewRelayArgs` and `kyou_view_relay_event_names`; adding it to only one breaks the build through the `Exclude` exhaustiveness check. どちらの束を使うかの基準は「ダイアログかどうか」ではなく **「自分がフォーカスの発火源かどうか」**。子が上げてきた `focused_kyou` / `clicked_kyou` を素通しするだけの中間層（`dnote-item-list-view` 等）は自分では発火しないので `build_kyou_dialog_relay` が正しい（名前に `dialog` と付いているせいで誤読されやすい）。ページ最上位の `RykvDialogHost` には `build_kyou_dialog_host_handlers(required, overrides?)` を使う ―― ページには emit 先の親がいないので未指定イベントは no-op で埋まるが、`updated_kyou` / `deleted_kyou` / `requested_reload_kyou` / `requested_open_rykv_dialog` / `closed` の5件は型で必須にしてあり、書き忘れるとコンパイルエラーになる。`__tests__/unit/classes/relay-bundle-source-scan.test.ts` が「`v-on` で渡した束と `@中継イベント` を同じ要素に併記していないか（両方登録されて二重に発火する）」をソース走査で検査する。

**Kyou の再読込:** `classes/kyou-reload.ts` の `refresh_kyou(kyou, query?)` / `refresh_kyou_in_list(list, kyou, options?)` を使い、**手順を手書きしない**。正しい手順は「SWキャッシュ削除 → `reload(true, query)` → `is_typed_data_loaded = false` → `load_all(query, true)`」の4つで、1つでも欠けると引き直しに失敗する。とくに `load_all` の第2引数(`force_attached`)を落とすと、`Kyou.clone()` が `is_attached_tags_loaded` を引き継ぐせいで `InfoBase.load_attached_tags(false)` が早期returnし、**添付タグを一度も引き直さない**（「タグを足しても表示が変わらない」の正体）。書き戻し方は呼び出し元が選ぶ。**既定の in-place splice を使うこと** ―― `model_value` が親の配列そのものになるダイアログで必須なだけでなく、rykv/mi の列でも必須。列の配列を作り直すと `focused_kyous_list`（= `match_kyous_list[focused_column_index]` へのエイリアス）が黙って切れ、件数カレンダーと Dnote がフォーカス列に追随しなくなる（2026-08-18 に rykv/mi を `replace` から in-place へ戻した。守るテストは `rykv-view-search-routing.test.ts` の「reload_kyouは列の配列を作り直さない」）。`replace` による copy-on-write が正しいのは、配列そのものを `Ref` に持ち他からエイリアスされない dashboard / plaing / shared-mi だけ。書き戻す位置は `refresh_kyou_in_list` が **await のあとに id で取り直す**（待っている間に局所挿入や削除でリストが動くため、待つ前のインデックスで splice すると別の行を潰す）。

引き直しの**合流には `new_reload_batch()` の値を渡す**。1回の更新でリスト・focused・開いているダイアログが独立に引き直すので、`reload_kyou` の先頭で1回だけ取って全部に同じ値を渡すと往復が1回で済む（`refresh_kyou` の第3引数 / `refresh_kyou_in_list` の `options.requested_at`）。渡さないと呼び出し時刻が使われ、飛行中の引き直しには相乗りしない。**無条件に合流させてはいけない**: `open_rykv_dialog` はダイアログを開くたびに引き直しを投げるので、それがまだ飛行中のうちに保存すると、更新前の Kyou を掴んで列・focused・ダイアログを一斉に古い内容へ戻していた（「ダイアログを閉じたら一覧が更新されない」の正体）。

引き直しは**呼び出し元のダイアログや行より長生きさせる**。`fetch_refreshed_kyou` は作業用クローンに専用の `AbortController` を入れ直しており、`KyouView` の `onUnmounted` abort で保存直後の引き直しが道連れになることはない。引き直し中は `is_kyou_reloading(id)` が真になり、`KyouView` が**中身を残したまま**右上にスピナーを重ねる（`.kyou_reloading`。中身を差し替える `show_loading_indicator` とは別物で、消すと行がちらつき詳細ビューでは高さが跳ねる）。状態は id キーのモジュールレベルに置く ―― `KyouView` は id が同じなら再マウントされず props が差し替わるだけなので、コンポーネントローカルに持つと倒せない。

**タグ/テキスト/通知の変更は `updated_kyou` を出さない。** 唯一の信号が `requested_reload_kyou` なので、Kyou を抱えて表示するコンポーネントは必ずこれを処理すること。

**ログイン画面ではセッション無効の飛ばしを止める**（`is_on_login_page`、`gkill-api.ts` の `check_auth`）。
`check_auth` はセッション無効系のエラーコード（`ERR000013` / `ERR000002` / `ERR000238` / `ERR000373`）を見つけると
`clear_browser_datas()` してから `location.replace("/")` する。ところが**ログイン失敗も同じコード帯を通る**
（存在しないユーザIDは `ERR000002` `AccountNotFoundError`、無効化されたアカウントは `ERR000238`）。
素直に飛ばすとページごと作り直され、`login-page.vue` がいま出したばかりのエラー表示が消える。
利用者からは「画面が一瞬光って、理由も出ないまま元のまま」に見える。
- **行き先は同じ `/` なので、飛ばさないことで失うものは無い。** セッションIDのクリアだけは行う
- 判定は `pathname === '/'`（と空文字）だけ。**共有ページを足してはいけない** ―― 共有ページは
  セッションではなく `share_id` で認証するので、そもそもこのコード帯を通らない
- ガードは `location.replace` の**手前**に置く。順序は `check-auth-login-page.test.ts` がソース走査で固定する
  （`check_auth` の本体を切り出し、ガードの位置が replace より前であることを添字で比べる）
- 守るテスト: `check-auth-login-page.test.ts` / `e2e/login.spec.ts` の
  「login with invalid credentials shows error」

### UI 実装の共通規約

**Context menus:** never compute the popup position by hand. Call `useContextMenuPosition()` (`classes/use-context-menu-position.ts`) for `is_show` / `menu_target` / `open_at(e)`, and bind the template as `<v-menu v-model="is_show" :target="menu_target" location="bottom start">`. Vuetify's connected location strategy measures the rendered menu and flips/shifts it to stay inside the viewport. The previous approach copy-pasted `left: min(innerWidth - 130, x); top: min(max(50, innerHeight - (8 + 48 * N)), y)` into all 25 context-menu composables — the 130px width was unrelated to the real list width, and `N` had to be hand-synced with the template's item count (the struct-family menus sat at `N=2` while actually having 5 items). `.gkill_context_menu_list { max-height: 70vh; overflow-y: scroll }` in `App.vue` still caps very long menus. 手計算をやめた経緯（実測しない限り正しい定数は書けない）は [ADR-0039](../../../documents/adr/0039-context-menu-position-by-vuetify.md)。

**チェックツリーへの適用は単一パスで:** サイドバーのチェック状態をツリーへ書き戻すときは `classes/foldable-struct-check.ts` の `apply_check_state_to_struct(root, items, state, pre_uncheck_all)` を使い、**「項目1つごとにツリー全体を再帰走査」を書かない**。旧実装は O(項目数×ノード数) で、全ノード訪問が Vue の deep reactive proxy 越しになるため、rep 376個の実環境では列フォーカス切替の click が同期6.5秒（トレース実測）に達していた（2026-08-10 修正。rep/tag/timeis の3コンポーザブルと `apply_rep_summary_to_detaul` が対象。旧実装との等価性は `foldable-struct-check.test.ts` が担保）。なお `InfoBase` 系データクラスに TS `private` フィールドを足してはいけない —— `ref()` の UnwrapRef が private を落とし `Ref<Array<Kyou>>` への代入が全所で型エラーになる（ES `#` も reactive Proxy 越しの `this` で壊れる）。内部フィールドは underscore 公開 + getter/setter にする。

**端末種別の判定:** `'ontouchstart' in window || navigator.maxTouchPoints > 0` を手書きしない。`useDeviceKind()`（`classes/use-device-kind.ts`）の `device_kind` / `is_pc` / `is_tablet` / `is_smart_phone` / `has_touch` を使う。**ドラッグ&ドロップの可否は `is_pc`、タッチ端末向けの代替導線（ロングプレスでの contextmenu 補完など）は `has_touch` と、見る値が違う**。以前は「タッチできるか」の2値を両用途に兼用していたため、タッチパネル搭載 Windows ノートで D&D が無効になっていた。判定は「UAでスマホ確定 → `any-pointer: fine` かつ `any-hover: hover` なら PC → UAでタブレット確定 → 画面短辺600px」の順で、この順序自体が仕様（iPad + トラックパッドは PC 扱い、スタイラス対応スマホはスマホ扱い）。状態はモジュールレベルのシングルトンで、`foldable-struct.vue` のようにノード数ぶん再帰インスタンス化されるコンポーネントから呼んでも購読は2本のまま。純関数 `classify_device_kind(env)` が判定の実体で、テストはここを突く。D&D を無効化してもコンテキストメニューの「上へ / 下へ / フォルダへ移動」（`classes/foldable-struct-move.ts`）で操作できる。

**ダイアログの自動フォーカス:** ダイアログを開いたら本文の最初のテキスト入力欄にカーソルが載る。実装は `useFloatingDialog` の `autofocus` オプション（既定 true）1箇所で、候補選びは `classes/dialog-autofocus.ts` の純関数 `find_autofocus_target()`。**view 側に `autofocus` を撒いてはいけない** ―― 入力欄を持つ view はサイドバーやページ直下でも使われるので、ページ読込時にフォーカスを奪う。探索は `.gkill-floating-dialog__body` の中だけ（ヘッダには透過トグルの `v-checkbox` と×が必ず先頭にあり、ルートから探すと必ずそれを掴む）。既に `autofocus` を書いた要素があるダイアログでは何もせず Vuetify に任せる。`readonly`（日付ピッカーの見せかけ入力）・`disabled`・非表示・`.v-selection-control` 配下・`inputmode="none"` の `v-select` は候補から外す（`v-autocomplete` / `v-combobox` は打ち込めるので対象）。入力欄が内側の `v-if` で遅れて生えるダイアログのために `MutationObserver` で2秒だけ見張り、一度当てたら切る。自前でフォーカス先を決めているダイアログ（`save-clipboard-to-file-dialog` は保存ボタン）は `autofocus: false` で切る。

**行判定の高さにパーセントを渡さない:** `classes/kyou-row-height.ts` の `is_row_height()` は高さを `Number.parseFloat` して 120 未満なら一覧の行とみなす。`parseFloat('80%')` は **80** なので、詳細ペインやダイアログでパーセントを渡すと行扱いになり、`mi-re-kyou-view.vue` の参照先ブロック（`v-if="!is_compact"`）が丸ごと消える。行ではない場所は `'unset'` か `'auto'` を渡すこと。例外は画像一覧（`kyou-list-view.vue` の `is_image_only`）だけで、200pxのセルに詰めるため意図的に `'100%'` を渡している。`__tests__/unit/classes/kyou-view-height-source-scan.test.ts` がソース走査で守る。

**表示用の文字列に HTML タグのリテラルを埋めない**（2026-08-21）。表示側は `{{ }}` 補間で Vue がエスケープするので、剥がし忘れた画面では**タグが文字としてそのまま見える**。実例: `classes/format-date-time.ts` の `format_duration()` が `"23時間 6分 <br>（23.1時間）"` を返し、3つの描画点が銘々 `.replace("<br>", …)` で剥がす一方、**Dnote の集計リスト（`aggregated-list-item.vue`）と相関グラフ（`dnote-correlation-graph-view.vue` の `point_description`）は剥がしておらず `<br>` が見えていた**。
- 区切りが要るなら**本物の改行**（`format-date-time.ts` の `DURATION_LINE_SEPARATOR`）を使う。`white-space` が既定のままの場所では空白1個へ畳まれるので、**剥がし忘れの壊れ方が「今までどおり1行で出る」に変わる**のが要点
- 改行として見せたい場所**だけ**が `white-space: pre-line` で opt-in する（今は `aggregated-list-item.vue` の値 span1つだけ）。値が2行になると `<td>` 既定の `vertical-align: middle` で前後の prefix/suffix が中間へ浮くので、同じ scoped style で `top` に揃えてある
- 1行で見せる場所は `to_single_line()` に通す。**`.replace(文字列, …)` を手書きしない**（最初の1個しか置換しないので区切りが増えると黙って壊れる）。相関グラフの SVG `<title>` は**ネイティブのツールチップとして改行を実際に描画する**ので、ここだけは CSS 任せにできず畳む処理が必須
- 守るテスト: `format-date-time.test.ts` / `aggregate-targets.test.ts`（TimeIs の合計・平均）/ `convention-source-scan.test.ts` の「表示文字列に HTML タグのリテラルを埋めていない」「集計リストの値が pre-line で改行を見せている」。走査の対象は `classes/**/*.ts` だけ（`.vue` のテンプレートには `time-is-view.vue` の `<br />` のような正当な用例がある）

**設定は「適用」を押すまでサーバへ送らない:** ApplicationConfig の子ダイアログ（tag/rep/rep_type/device/kftl_template/mi_board の struct 系6つ + Dnote / Ryuu / Dashboard / PlaingTimeIs）の「適用」は**組み立てだけ**で、API を1つも呼ばない。送信は `use-application-config-view.ts` の `update_application_config()` の1箇所に閉じている。子の適用先は `cloned_application_config` **だけ**で、`props.application_config` を直接書いてはいけない（書くと設定画面のキャンセルが効かなくなる）。props の差し替え（板ツリー/タグツリーの追随）で未適用の編集が消えないようにするのは `has_pending_child_edits` の役目。ロケールとダークテーマは選ばせるために即時プレビューするので、`cancel_pending_changes()` で開いた時点の値へ戻す（×・Escape・キャンセルのどれでも通るよう `application-config-dialog.vue` の `hide()` から呼ぶ）。**例外は ServerConfigDialog** ―― 別エンティティ（`update_server_config`）なので自分の「適用」で送る。

## 関連スキル

- [gkill-client-columns](../gkill-client-columns/SKILL.md) — rykv/mi/dashboard の列と検索
- [gkill-client-kftl](../gkill-client-kftl/SKILL.md) — KFTL（メモ帳）
- [gkill-client-tags](../gkill-client-tags/SKILL.md) — 追加/編集画面のタグ欄
- [gkill-client-rudbeckia](../gkill-client-rudbeckia/SKILL.md) — ポートとフローティングウィンドウ
- [gkill-plugin](../gkill-plugin/SKILL.md) — プラグイン本文（`plugin-html-view.vue`）の iframe 注入の正本
- [gkill-find-query](../gkill-find-query/SKILL.md) — `FindQuery` の null 意味論（TS 側の `undefined` 禁止を含む）

## 詳しい設計と却下案（ADR）

- [ADR-0038 Props/Emit のみ（Pinia を入れない）](../../../documents/adr/0038-props-emit-only-no-pinia.md)
- [ADR-0039 コンテキストメニュー位置は Vuetify に任せる](../../../documents/adr/0039-context-menu-position-by-vuetify.md)
