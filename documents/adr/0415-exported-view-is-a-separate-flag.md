# ADR-0415: 出力モードは共有モードと別のフラグにし、通信は `gkill_fetch` の1点で止める

| | |
|---|---|
| Status | Superseded |
| Date | 2026-11-22 |
| Sources | `src/client/classes/api/gkill-api.ts` の `GkillAPIForExportedKyou` の doc コメント / [ADR-0903](0903-export-shell-is-a-second-vite-build.md)（出力のガワ） / [ADR-0407](0407-init-on-application-config-loaded.md)（列の初期化トリガ） / [ADR-0418](0418-split-rykv-views-by-mode.md) / [ADR-0420](0420-split-mi-views-by-mode.md) |
| Supersedes | なし |
| Superseded-by | ビュー分岐部分のみ [ADR-0418](0418-split-rykv-views-by-mode.md) / [ADR-0420](0420-split-mi-views-by-mode.md) |
| Anchors | `src/client/classes/exported-kyou-payload.ts` の `build_exported_api`、`src/client/classes/api/gkill-api.ts` の `GkillAPIForExportedKyou` |

## Context

ビュー側のフラグと computed による分岐は ADR-0418 / ADR-0420 のモード別ビュー分割で置き換え済み。`GkillAPIForExportedKyou` と `gkill_fetch` の1点遮断・`ApplicationConfig.for_export_kyou` は現行も有効。本書の後続の記述は決定当時の設計を示す。

「HTML で出力」した画面は共有ページ（`/shared_page`）とよく似ている。どちらも読み取り専用で、
データをサーバの検索ではなく手元の配列から返す。実際、共有ページには
`RykvViewProps.is_shared_rykv_view` と `GkillAPIForSharedKyou`（通信をしない `GkillAPI`）という
土台が既にある。

しかし2つ違う。

- **共有ページは集計（Dnote）を落としている**（`rykv-view.vue` のトグルと本体の両方）。
  出力は集計を残す。同じフラグで両方は表せない。
- **出力は区画ごとに選べる**（集計・マップ・検索条件サイドバー）。共有には無い軸。

`GkillAPIForSharedKyou` もそのままは使えない。クエリを見ずに全件返す `get_kyous`、空を返す
通知のスタブ、`""` 固定の API キー、そして **`get_plugin_content_html` は上書きされていないので
本体の `fetch` に落ちる**。

そして、上書きされていないメソッドは共有ページでも本体の `fetch` に落ちる。数えると
**本体で通信する93本のうち、共有版が上書きしているのは71本**で、残りは素通しだった。
共有ページはサーバが生きている前提なので実害が出にくいが、出力 HTML では
「開いた本人には何も起きず、オフラインの相手の手元でだけ壊れる」形になる。

## Decision

**出力モードは `is_exported_rykv_view` という別の props にする。** そのうえで、
template が見るのは次の4つの computed だけにし、組み合わせの判断を1箇所へ閉じる。

| computed | 意味 |
|---|---|
| `is_readonly` | 共有 **または** 出力。編集系の導線・設定・ヘルプを落とす |
| `is_dnote_available` | 共有では出さない。出力は選ばれたときだけ |
| `is_map_available` | 共有では出す。出力は選ばれたときだけ |
| `is_sidebar_available` | 共有では出さない。出力は選ばれたときだけ |

深い子（`gps-log-map.vue` など）には props が届かないので、`ApplicationConfig.for_export_kyou` を
足して載せる（既存の `for_share_kyou` と同じ流儀）。

**`GkillAPIForExportedKyou` は `GkillAPIForSharedKyou` を継承し、違うところだけを上書きする。**
そのうえで、**通信は `gkill_fetch` の1点で止める**。`GkillAPI` の `gkill_fetch` を `private` から
`protected` へ変え、出力用の派生がそこを上書きして例外にする。

## Rejected alternatives

- **`is_shared_rykv_view` を流用する** — 共有は集計を落としている。`rykv-view.vue` から集計の分岐を外すと、今度は共有ページに集計が出る。どちらかが壊れる。
- **`is_shared_rykv_view` に「集計を出すか」のオプションを足す** — フラグの意味が「共有である」から「共有かつ集計なし」へ滑る。出力の区画が3つあるので、同じことをあと2回やることになる。
- **template で `is_shared_rykv_view || is_exported_rykv_view` を直に書く** — 27箇所に散っていた分岐が、2つの条件の組み合わせで更に散る。どの区画がどちらのモードで出るのかが読めなくなる。
- **`GkillAPIForExportedKyou` を `GkillAPI` から書き直す** — 上書き漏れの心配は無くなるが、80本以上のメソッドを写すことになる。本体に API が増えたとき、写しの側は**古いまま通る**（`extends GkillAPI` なので新しい API は本体の fetch に落ちる）。重複は減らず、ドリフトの源が1つ増える。上書き漏れはテストで1点ずつ固定するほうが安い。
- **上書きされていない22本を1本ずつ塞ぐ** — 塞いだ時点では正しいが、本体に API が増えるたびに黙って漏れる。漏れたことは「オフラインの相手の手元でだけ壊れる」ので誰も気づかない。`gkill_fetch` を1点で止めれば、将来の API も自動的に止まる。
- **`gkill_fetch` を上書きせず、通信したら握って空を返す** — 止まったことが誰にも分からない。呼んだ経路の名前が出ないので、上書きすべきメソッドも特定できない。
- **出力でもプラグインの rep 名を本物の一覧から引く** — `/api/get_plugin_list` を取りに行く。出力 HTML は全部が読み取り専用で、プラグイン由来かどうかで出し分けるものが無い（編集メニューがそもそも出ない）。空で返す。

## Consequences

- **`init()` はサイドバーの template ref が埋まっていないと列を作らずに戻る**（ADR-0407 の経路）。サイドバーを含めない出力でも `v-navigation-drawer` は mount したままにし、**開く導線（アプリバーのアイコン）だけを落とす**。`v-if` で消すと**読み込み中オーバーレイが消えない**（列が1本も作られない）。
- **焼き込んだ `ApplicationConfig` は `is_loaded` と `rykv_hot_reload` を強制で true にする。** 前者が false だと `init()` の watch が発火せず永久スピナー、後者が false だと**列の枠だけ出て中身が空**になる（利用者がその設定を切っていると、出力した本人の手元では再現しない）。どちらも焼き込み側の値には頼らず `build_exported_api` で立てる。
- **`v-app` で包むこと。** `v-app-bar` / `v-navigation-drawer` はレイアウトの中でしか描けず、包まないと「[Vuetify] Could not find injected layout」で rykv ごと落ちる。骨格（`#control-height` と `VLocaleProvider`）は `App.vue` と同じにする。
- `is_readonly` にしたことで、列のフッタ（件数・リロード・画像切替・✕）も出力では出なくなる。共有ページと同じ見え方になる。
- `gkill_fetch` が `protected` になった。派生から呼べるが、本体の挙動は変わっていない。
- 出力用 API はシングルトンなので、テストは状態をテスト間で持ち越す。`columns` などを都度リセットすること。

## Evidence

実測なし — 本体で通信する93本のうち共有版が上書きしているのが71本、という数はソースを機械的に数えたもの
（コード由来）。出力 HTML をブラウザで開いて、列ごとに違う記録が出ること・gkill の API 呼び出しが
1本も出ないこと（Google Maps を除く）を確かめた。`gkill_fetch` を塞ぐ前は
`/api/get_plugin_list` が1本出ていた。

## Related tests

- `src/client/__tests__/unit/api/exported-kyou-api.test.ts` — 上書き1点ずつ（query_id で列を引く・焼き込んだ設定を返す・通知を焼き込みから引く・プラグイン本文が fetch へ落ちない・API キーが localStorage と退避先）
- `src/client/__tests__/unit/classes/exported-kyou-payload.test.ts` — 焼き込みの実体化と、列の `kyou_ids` からの解決
- `src/client/__tests__/unit/composables/exported-kyou-page.test.ts` — 焼き込みが読めないときに「記録が含まれていません」側へ落ちること
