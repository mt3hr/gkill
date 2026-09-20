# ADR-0632: 設定ツリーの全ノードに利用者が書く description を持たせ、MCP は fields:["descriptions"] の平坦な一覧で先に読ませる

| | |
|---|---|
| Status | Accepted |
| Date | 2026-09-20 |
| Sources | 利用者の要求「タグや Rep、RepStruct、プロファイル、板構造や KFTL テンプレートに Description を追加し、MCP から取得できるように。Description は ApplicationConfig の各構造から編集できるように。MCP に運用方法を伝え、効率よく必要なデータを引き出せるようにすることが目的」。途中の判断: 対象は 6 ツリーの全ノード（フォルダ・葉）、MCP は「ツリー + 平坦な一覧射影」、Web は編集ダイアログのみ（サイドバーのツールチップは不要） |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/client/classes/datas/config/tag-struct-element-data.ts` ほか 6 クラス（`description`）/ `src/client/classes/use-edit-tag-struct-element-view.ts` ほか 6 本の `apply()` / `src/client/classes/use-edit-mi-board-struct-element-view.ts`（板の編集ダイアログ、新設）/ `src/server/gkill/mcp/read_handlers.go`（`structIdentityKeys` / `FilterAppConfigStructs` / `BuildAppConfigDescriptions` / `compactStructNode`）/ `src/server/gkill/mcp/help_topics.go`（topic `config`）/ `src/server/gkill/mcp/internal/fakegkill/fixtures.go` |

## Context

MCP 経由の AI は `gkill_get_application_config` で設定ツリーを読めるが、そこにあるのは名前と表示フラグだけで、
「このタグは何のためのものか」「この板には何を入れているか」「このテンプレートはいつ使うか」という利用者の運用は
名前から推測するしかなかった。推測が外れると検索条件が外れ、AI は大きなツリー（実測で `rep_struct` だけで
25,000 トークン超。ADR-0629）を何度も読み直す。

設定ツリーの実体は `APPLICATION_CONFIG` テーブルの KEY/VALUE に JSON 文字列として 1 行ずつ入り、Go 側は
`*json.RawMessage` で素通しする（`dao/user_config/application_config.go`）。ノードの形の正本は Web の TS クラスで、
Go 本体で型付きに読むのは KFTL の `forceHideTagNames`（読み取り専用）だけ。つまり **欄を足すのに DB もサーバも
触る必要が無い** —— 読み込み・`clone()`・D&D・保存はすべて JSON を丸ごと往復させるので、未知の欄は自然に生き残る。

落ちる場所は 1 つだけあった。各ツリーの要素編集ダイアログの `apply()` は既存ノードを in-place で直さず、
`new XxxStructElementData()` に既知の欄だけを詰め直して id 一致で splice 差し替えする。ここに新しい欄を写さないと、
「適用」のたびにその欄がエラーも警告も出ずに消える。

作業中に MCP 側の既存の穴も見つかった。偽 gkill（`internal/fakegkill/fixtures.go`）が設定ツリーを **配列ルート + 別語彙
（`tag` / `device` / `rep_type`）** で書いていたため、実データ（ルート 1 オブジェクト `{name:"__root__", children:[…]}` +
`tag_name` / `device_name` / `rep_type_name`）では、`contains` が 1 件も刈らず（本セッションの実測: タグの葉 347 件が
そのまま返る）、compact の `name` 落としも tag / device / rep_type の 3 ツリーで効いていなかった。テストは全部通っていた。

## Decision

- 6 ツリー（TagStruct / RepStruct / RepTypeStruct / DeviceStruct / MiBoardStruct / KFTLTemplate）の **全ノード（フォルダ・葉）**
  に `description: string`（既定 `""`）を持たせる。正本は TS クラスで、Go 本体・DB スキーマは無変更。
- Web の各ツリーの要素追加 / 編集ダイアログに「説明」欄を足し、`apply()` の詰め直しに `description` を**必ず**写す。
  板（MiBoard）は要素編集ダイアログが無かったので新設し（板名は実データ由来なので読み取り表示、編集できるのは説明だけ）、
  コンテキストメニューに「編集」を足す。ルート行は開かない（`update_*_struct` の walk は子しか差し替えないので、開けても適用が消える）。
  フォルダの説明は作った直後にコンテキストメニュー「編集」（葉・フォルダ共用の編集ダイアログ）で書く。
- MCP `gkill_get_application_config`:
  - 各ノードの `description` はそのまま通し、`compact:true` は**空文字だけ**落とす（「無い欄 = 書いていない」）。
  - `fields:["descriptions"]` を仮想欄として足す。6 ツリーを深さ優先で歩き、`description` が非空のノードだけを
    `{struct, name, path, is_dir(true のときだけ), description}` の平坦な配列で返す（`name` は葉なら識別欄・フォルダなら表示名、
    `path` はルートを除く祖先の表示名を `/` で連結）。**fields で明示したときだけ**載せ、既定の全量には含めない
    （ツリー側に同じ文が載るため）。`contains` は一覧にも掛かる（name / path / description）。
  - `structIdentityKeys` を実データの語彙（`tag_name` / `rep_name` / `rep_type_name` / `device_name` / `board_name` / `title`）に直し、
    `FilterAppConfigStructs` はルート 1 オブジェクトの `children` を刈って同じルートへ戻す（残らなければ `children:[]`）。
    偽 gkill のフィクスチャも実データと同じ形に書き直す。
  - 本文は `gkill_get_mcp_help topic:config`（6 ツリーの形と識別欄、フラグの意味、description は運用メモで検索前に読むこと、
    ツリー→検索条件の対応）に置き、ツール説明文には在処案内だけ足す（ADR-0622）。

## Rejected alternatives

- **サーバ側の Repository 定義（`REPOSITORY` テーブル）に列を足す** — スキーマ移行が要り、編集の口が設定画面の構造ダイアログの外
  （管理者の rep 管理）へ出る。利用者は「ApplicationConfig の各構造から編集」を求めており、rep の運用メモは RepStruct の葉に置けば足りる。
- **説明専用の MCP ツールを足す** — ツール数が 3 サーバで動き、資料の表・予算・ゴールデンが増える割に、`gkill_get_application_config`
  の `fields` 射影で同じことができる。
- **ツリーの全量読みで足りるとする（一覧射影を作らない）** — 実環境の `rep_struct` だけで 25,000 トークン超。運用メモを読むためだけに
  毎回それを読ませるのは、この機能の目的（効率よく引き出す）と逆になる。
- **`descriptions` を既定の全量応答にも載せる** — ツリー側にも同じ文が載るので二重になる。`fields` を省いた読み取りは「全部」の意味で
  使われており、そこに仮想欄を増やすとコンテキストを常時食う。
- **サイドバーのツリーに説明のツールチップを出す** — 利用者が今回不要と判断した。要素データのフィールドは描画に使っていないので、
  後から `foldable-struct.vue` に `:title` を足すだけで済む。
- **フォルダ追加ダイアログにも説明欄を足す** — 5 ツリー共用の `add-new-folder-view.vue` と `FolderStructElementData` と 5 本の
  `add_folder_struct_element()` を触ることになる。フォルダの説明は編集ダイアログで書けるので、触るファイルを減らす側を採った。

## Consequences

- 設定画面で書いた説明は、`update_application_config` の JSON に載って保存され、再読込・clone・並べ替え・保存で消えない。
  **新しい欄をノードに足すときは 6 本の `use-edit-*-struct-element-view.ts` の `apply()` 全部へ写すこと**
  （`struct-element-description.test.ts` が表駆動で固定する）。
- MCP の応答形は変わらない（`description` が非空のノードに現れるだけ）。`fields` の enum に `descriptions` が増え、
  `gkill_get_mcp_help` の enum に `config` が増えるので tools/list の予算は +822 バイト（40,033 / 59,278 / 83,760 → 40,855 / 60,100 / 84,582）、
  `schema_revision` と 12 本のゴールデンが動いた。
- 実環境で `contains` が初めて刈れるようになる。`contains` の照合は名前（識別欄と表示名）であって説明ではない。説明で探すときは
  `fields:["descriptions"]` と `contains` を組み合わせる。
- 説明は MCP から書けない（書き込みツールは無い）。運用メモは利用者が設定画面で書く。

## Evidence

- 本セッションの実測（2026-09-20、実環境の read サーバ）: `fields:["tag_struct"]` + `contains` で葉 347 件が 1 件も減らず、
  `compact:false` の出力で識別欄が `tag_name` / `device_name` / `rep_type_name` であることを確認。
- 偽 gkill を実データの形に直した後の `contains` のゴールデン: `app config: contains` が「生活/日記」の 1 枝だけを返し、
  `app config: contains no match` は各ツリーのルートだけ（`children` 無し）になった。
- tools/list の予算: read 40,033 → 40,855、write 59,278 → 60,100、readwrite 83,760 → 84,582（各 +822）。

## Related tests

- `src/server/gkill/mcp/read_handlers_test.go`（`TestHandleReadToolCallGetApplicationConfigCompactContainsMaxSize`: ルート 1 オブジェクトの
  contains / 空の description だけ落とす / `fields:["descriptions"]` / 一覧への contains / 既定の全量には載らない / ツリー未設定で `[]`）
- `src/server/gkill/mcp/help_topics_test.go`（topic `config` の語句の実在）
- `src/server/gkill/mcp/tool_handlers_test.go`（`gkill_get_application_config` の説明文が `gkill_get_mcp_help` を案内する）
- `src/server/gkill/mcp/golden_test.go`（`app config: fields descriptions` / `app config: descriptions with contains` / `help: config`）
- `src/client/__tests__/unit/composables/struct-element-description.test.ts`（6 本の `apply()` と 5 本の追加ダイアログが `description` を写す、
  板の `update_mi_board_struct` と「ルートは開かない」）
