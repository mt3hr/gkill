# gkill Architecture Decision Record（ADR）

## この資料集の位置づけ

gkill には「現在どうなっているか（What）」の資料が `documents/reverse/` に24本ある。
ここに置くのは「**なぜそうなっているか（Why）**」——とくに **採らなかった案とその理由** である。

> Reverse docs = What ／ ADR = Why

狙いは1つ。**決定の理由を知らない者が、一見もっともらしい退行を「改善」として提案するのを止めること。**

> 「もっと高速にできそうなので SQL 側に rep 名の条件を入れました！」

このような提案は、実際には十数個のキャッシュrepを数百個の生repに化けさせ、検索の実質CPUの過半を
git に食わせる経路への逆戻りになる。同種の罠が gkill には数十個あり、そのほとんどが
**例外もエラーも出さずに静かに壊れる**。

## 正本の分割規約

同じ不変条件を何箇所にも書くと、実測値とテスト一覧が必ずずれる。層ごとに持ち分を決めてある。

| 層 | 持つもの | 持たないもの |
|---|---|---|
| コードコメント | その場で守る規則と、その場で要る理由 | 事件の経緯、他所の実測値 |
| `AGENTS.md` | 全タスクで要る核（ビルド・命名・横断契約）＋スキルへのルーティング表 | 領域別の禁止文の本文 |
| `.claude/skills/*/SKILL.md` | **領域別の禁止文**＋1行理由＋ADRリンク＋守るテスト名 | 却下案の列挙、実測の内訳、事件の経緯 |
| `CLAUDE.md`・他AI入口ファイル | 導線だけ（Claude Code は `@AGENTS.md` で AGENTS.md を展開） | 規約本文すべて |
| `documents/adr/`（ここ） | **却下案・実測値・事件譚・撤回した決定** | **禁止文の再掲** |
| `ABOUT_TEST.md` | テスト ↔ 規約 ↔ 症状の対応表 | 理由 |

**ADR に禁止文を再掲しない。** これが唯一のドリフト対策である。
「何をしてはいけないか」を知りたいだけの読み手（新しいセッションの AI を含む）は
`AGENTS.md` のルーティング表から該当する `.claude/skills/*/SKILL.md` を読めば足り、
ここへ来るのは「なぜ別の案では駄目なのか」を知りたいときだけでよい。

## 何を ADR にするか

判定基準は1つ。

> **その決定を知らない人が、逆方向の変更を「改善」として提案しうるか。**

しうるなら ADR。しえないならそれは決定ではなく仕様であり、`documents/reverse/` の担当である。

`documents/reverse/design-philosophy.md` との境界も同じ基準で引く。あちらが持つのは
**覆らない大方針**（Append-Only DAO の思想、単一バイナリ配信、KFTL の設計意図など）で、
ここが持つのは**逆方向の変更が提案されうる個別判断**である。

## 書き方

`0000-template.md` をコピーして使う。節構成は固定で、`npm run verify_docs` が機械検査する。

| 節 | 内容 |
|---|---|
| Context | 何が問題だったか。規則そのものは書かない |
| Decision | 何を決めたか。断定形で1〜3文 |
| Rejected alternatives | **この資料の存在理由。** 採らなかった案と理由。空にできない |
| Consequences | 受け入れる制約。守らないと何が起きるか。**静かな壊れ方は必ず書く** |
| Evidence | 実測値。無いなら「実測なし — 脅威モデルからの判断」のように理由を書く |
| Related tests | 守っているテスト。**実在するパスのみ**（verify_docs が検査する） |

### 検査されること

`src/tools/verify_docs.mjs` の `checkADR()` が以下を落とす。

- 6つの必須見出しが揃っているか
- メタ表の `Status`（`Accepted` / `Superseded` / `Deprecated`）・`Date`・`Sources` があるか
- ファイル名の番号と本文 `# ADR-NNNN` が一致し、番号が重複していないか
- `Rejected alternatives` が実質空でないか
- `Related tests` のパスが実在するか（`Status: Superseded` は免除）
- `Superseded-by` の指し先が実在し、相手の `Supersedes` も自分を指しているか
- 下の索引表に全 ADR が1行ずつ載っているか

加えて `docMarkdownFiles()` 経由で、リンク切れ・`src/...` 参照パス・**資料に載っているファイル名の実在**も検査される。

### 削除済みファイルの名前を書くとき

ファイル名の実在検査は ADR にも効くが、ADR は本質的に歴史を語る。
**かつて存在したファイルの名前はコードフェンスで囲むこと。** フェンスの中は実在検査から除外される。

### 図を書かない

`checkMermaid()` は `documents/reverse` をハードコードしており、ADR の Mermaid は検査されない。
壊れた図が検査ゼロで残るので、ADR に Mermaid を書かない。図が要るなら
`documents/reverse/` 側に置いてリンクする。

## 番号の付け方

`documents/reverse` の分類とは別軸で、**「壊れたときに同じ場所を読み直すことになる範囲」＝サブシステム別**に
100番幅で採番する。幅を取ってあるのは、後から同じ帯へ差し込めるようにするため。

| 帯 | サブシステム |
|---|---|
| 0100-0199 | 検索とフィルタの意味論 |
| 0200-0299 | DAO・キャッシュ・並行処理 |
| 0300-0399 | プラグイン |
| 0400-0499 | クライアント（列・入力・状態管理） |
| 0500-0599 | メモ帳（KFTL） |
| 0600-0699 | MCP |
| 0700-0799 | セキュリティ・HTTP契約 |
| 0800-0899 | 開発規約と資料 |
| 0900-0999 | ビルド・テスト・CI |
| 1000-1099 | 運用CLI・配布 |
| 1100-1199 | モバイル（Android / Wear OS） |
| 1200-1299 | ポート（rudbeckia）・画面ホスト |

1200 帯はまだ ADR が無い予約帯で、0件のまま置いてある。新しい領域の決定を書く人が
帯そのものを発明せずに済むようにするため（0900 帯は 2026-09-14 の ADR-0901 で使い始めた）。

**この表に件数を書かない。** ADR が増えるたびに古びる数字を手で持つことになる。
帯ごとの使用数と残り空きは `npm run verify_docs` の `checkADRBands` が実ファイルから数え、
**残り空きが10を切った帯があればそこで落とす。** 満杯になってから別の帯へ逃がすのではなく、
落ちた時点で帯を広げるか新しい帯を切る。逃がすと何が起きるかは
[ADR-0805](0805-adr-numbering-by-subsystem-hundreds.md) にある。

**番号は採番後不変。** 撤回した決定も番号を空けず `Superseded` で残す
（「入れて翌日撤去した」という記録そのものが最良の ADR になる）。
例外は 2026-08-30 の全面再採番の1回だけで、旧番号は下の「旧→新 対応表」から引ける。

**ファイル名の slug は改名しない。** slug は採番時点の決定内容であって現行規則ではない。
改名すると索引・リンク・アンカーコメント・規約スキル（`.claude/skills/`）のリンクを一斉に壊すので、
決定が変わったときは Superseded で新しい番号を採る。
（2026-08-30 の再採番でも slug は1つも変えていない。動いたのは先頭4桁だけ。）

## 決定を覆すとき（Supersede 手順）

決定を覆す実体は「**ガードテストを消すこと**」である。順序を守らないと CI が赤いまま作業することになる。

1. 新しい ADR を新しい番号で書く。`Supersedes` に旧番号を書く
2. 旧 ADR の `Status` を `Superseded` に、`Superseded-by` に新番号を書く
3. 旧 ADR の `Related tests` のうち消したテストを `Removed-tests` の行へ移す
   （`Status: Superseded` は `Related tests` の実在検査を免除されるが、消したことは残す）
4. 下の索引表の Status 列を更新する
5. コードのアンカーコメントを新しい ADR へ向け直す
6. 該当する規約スキル（`.claude/skills/*/SKILL.md`。横断規約なら `AGENTS.md`）の禁止文を書き換える
7. ガードテストを消す／書き換える

## 索引

<!-- ADR を追加したらこの表に1行足すこと。verify_docs が網羅を検査する。 -->

| # | タイトル | Status |
|---|---|---|
| [0101](0101-filter-rep-after-cache.md) | rep名の絞り込みは「検索するrep」ではなく「検索結果」で行う | Accepted |
| [0102](0102-no-rep-name-in-sql.md) | SQL の WHERE へ rep名の条件を押し込まない（暫定的な否決） | Accepted |
| [0103](0103-tag-filter-threshold-32.md) | タグ絞り込みの取得は2経路を持ち、切り替えはタグ名の個数（閾値32）で決める | Accepted |
| [0104](0104-related-tag-ids-only-for-no-tags.md) | 全タグ走査（RelatedTagIDs）は「タグ無し」仮想タグを使う検索のときだけ走らせる | Accepted |
| [0105](0105-chunk-find-query-ids.md) | FindQuery.IDs は分割して渡し、検索が失敗したのに GkillError が空のまま return しない | Accepted |
| [0106](0106-find-query-null-semantics.md) | FindQuery の use_* 有効化フラグを全廃し「値が非nullならフィルタ有効」に一本化する | Accepted |
| [0107](0107-memoize-rekyou-target-resolution.md) | ReKyou / MiReKyou のターゲット解決はリクエスト単位でメモ化する | Accepted |
| [0108](0108-period-of-time-second-of-day.md) | 時間帯フィルタの秒値は「86400未満は秒オブデイ、以上はepoch」の二重解釈にする | Accepted |
| [0109](0109-hide-tags-standalone.md) | hide_tags はタグ絞り込み(tags)の有無と独立に適用する | Accepted |
| [0110](0110-deleted-data-opens-only-with-include-deleted-data.md) | 削除済みの Kyou は include_deleted_data だけが開ける（is_deleted は使わない） | Accepted |
| [0111](0111-drop-never-implemented-query-fields.md) | 実装されなかった検索条件は消す — 受理して無視するより未知キーとして弾く | Accepted |
| [0112](0112-tag-vocabulary-drops-dead-targets.md) | タグ語彙は「生存する対象を持つタグ」だけを返す — カスケード削除はしない | Accepted |
| [0113](0113-word-filter-columns-and-id-prefix.md) | ワード検索は型別の対象列を決め、ID は前方一致だけ、除外語は付随テキストにも効かせ、判定は SQL / Go / プラグイン SDK で1つに揃える | Accepted |
| [0201](0201-append-only-dao.md) | Append-Only DAO — ID 列に主キー制約を置かず、更新も削除も INSERT で表現する | Accepted |
| [0202](0202-rebuild-cache-only-on-db-change.md) | キャッシュのフルリビルドは実DBが変わったときだけ | Accepted |
| [0203](0203-write-through-cache-not-reps-count.md) | 書き込み後のキャッシュ反映は要素数ではなく構築時に控えた CachedReps で判定する | Accepted |
| [0204](0204-keep-journal-mode-delete.md) | 実データDBの journal_mode は DELETE のまま変えない | Accepted |
| [0205](0205-unixepoch-expression-index.md) | 時刻列は unixepoch の式インデックスで引く | Accepted |
| [0206](0206-no-nested-threads-go.md) | threads.Go の入れ子は禁止。集約リポジトリには逐次版を用意する | Accepted |
| [0207](0207-exclude-urlog-thumbnail-from-cache.md) | URLog のサムネイルはインメモリキャッシュに載せない | Accepted |
| [0208](0208-git-repo-detect-by-os-stat.md) | gitリポジトリ判定は PlainOpen のエラー型ではなく os.Stat で行う | Accepted |
| [0209](0209-write-through-normalizes-rep-name.md) | キャッシュへ書き戻す rep 名はクライアントの値を信用せず書き込み側で正規化する | Accepted |
| [0210](0210-latest-data-address-uses-row-rep-name.md) | 最新版アドレス表の rep 名は行ごとの実 rep 名にする（集約名を焼かない） | Accepted |
| [0211](0211-expand-rep-patterns-without-walking.md) | リポジトリ定義のパターン展開は対象ツリーを歩かない | Accepted |
| [0212](0212-derived-cache-scan-lists-directories.md) | 派生キャッシュの一括生成は1件ずつ stat せずディレクトリを列挙する | Accepted |
| [0213](0213-transcode-only-what-the-browser-cannot-play.md) | 互換動画へ変換するのは、原本のまま再生できると言い切れないものだけ | Accepted |
| [0214](0214-thumbnail-decodes-by-content-not-extension.md) | サムネイルは拡張子ではなく中身で作る | Accepted |
| [0215](0215-data-db-synchronous-full.md) | 実データDB・設定DBの synchronous は FULL にする | Accepted |
| [0216](0216-detach-a-broken-rep-but-never-silently.md) | 読めない rep は切り離して動き続ける。ただし黙って落とさない | Accepted |
| [0217](0217-upload-batch-atomicity-not-guaranteed.md) | アップロードのバッチ原子性は保証しない（ファイル単位の原子性のみ） | Accepted |
| [0218](0218-get-kyou-histories-via-cached-reps.md) | `/api/get_kyou` の版履歴はキャッシュ rep を回して集める（`UnWrap()` は rep 名照合のときだけ） | Accepted |
| [0219](0219-commit-tx-is-one-sqlite-transaction.md) | commit_tx は書き込み rep のファイルを ATTACH した1接続の SQLite トランザクションで確定する | Accepted |
| [0220](0220-sqlite-localtime-follows-libc-zone.md) | SQLite の 'localtime' は libc のゾーンで決まるので、Android では libc にも端末のゾーンを教える | Accepted |
| [0221](0221-git-cache-miss-does-not-fall-back-to-raw-walk.md) | Git コミットログのキャッシュ包装は、構築済みなら外れた ID で生リポジトリへ落ちない（生実装は `CommitObject` で存在を引いてから `Log` する） | Accepted |
| [0222](0222-thumbnail-prefers-vips-cli-then-native-then-ffmpeg.md) | 静止画のサムネイルは vips があれば CLI で作り、無ければ Go → ffmpeg で作る（Go で読めて小さい画像は Go から。EXIF 回転は縮小の後ろ、縮小は BiLinear） | Accepted |
| [0301](0301-plugin-cancel-vs-kill.md) | プラグインの打ち切りは「待つのをやめる」と「プロセスを殺す」を分け、期限はスロットを取ってから張る | Accepted |
| [0302](0302-plugin-provides-typed-index.md) | プラグインは provides で型別/付随データを提供でき、アダプタの読み取りは索引から即答する | Accepted |
| [0303](0303-plugin-cache-use-crc32-and-size.md) | Google Takeout は ZIP のまま読み、差分判定は (CRC32, Size)、世代は「フォルダ + 書き出し時刻」 | Accepted |
| [0304](0304-plugin-emits-kyou-false.md) | 記録を返さないプラグインは emits_kyou: false で明示する | Accepted |
| [0305](0305-plugin-background-builder-wal.md) | プラグインの重い構築は常駐ビルダ + WAL + バッチcommit | Accepted |
| [0306](0306-codex-thread-id-from-filename.md) | Codex ロールアウトログのスレッド識別子はファイル名の uuid、会話は event_msg レーンだけ | Accepted |
| [0307](0307-skip-plugin-discovery-when-only-idf-is-needed.md) | IDF のリポジトリしか要らない経路ではプラグインを探索しない | Accepted |
| [0308](0308-plugin-multiple-rep-names.md) | プラグイン1本が複数の rep 名を名乗れる（get_rep_name の rep_names） | Accepted |
| [0309](0309-plugin-provides-git-commit-log.md) | zip の Git リポジトリは provides に git_commit_log を書いて native と同じ経路に載せる | Accepted |
| [0310](0310-chat-export-plugins-read-zip.md) | ChatGPT / Claude.ai の会話履歴はエクスポート ZIP のまま読み、展開済みの JSON は読まない | Accepted |
| [0311](0311-plugin-list-rep-names-are-always-the-query-values.md) | プラグイン一覧の rep_names は常に「query.reps に渡せる値」 | Accepted |
| [0312](0312-fitbit-one-data-source-per-day.md) | fitbit は同じ日に並ぶデータソースを合算せず、時計を優先して1系統だけ採る | Accepted |
| [0313](0313-plugin-logs-through-gkill-log.md) | プラグインのログは gkill_log の別名ファイル（logs/gkill_plugin_<name>*.log）へ出し、stderr には WARN / ERROR の接頭辞行だけを残す | Accepted |
| [0401](0401-do-not-split-search-window-in-client.md) | クライアント側で検索を期間の窓へ刻んで複数回 get_kyous を投げない | Accepted |
| [0402](0402-insert-registered-kyou-locally.md) | 記録の追加は列を再検索せず、その1件をクライアントで判定して差し込む | Accepted |
| [0403](0403-add-tag-before-registered-kyou.md) | タグ欄付きの追加/編集画面は add_tag が完了してから registered_kyou を emit する | Accepted |
| [0404](0404-add-unknown-tag-to-column-filter.md) | 利用者がその場で作ったタグだけを、開いている列の検索条件へ足す | Accepted |
| [0405](0405-column-identity-query-id.md) | 列の同一性は query_id、検索は世代番号で最後の1回だけ書き戻す | Accepted |
| [0406](0406-visualize-before-initial-search.md) | rykv/mi/dashboard は初期取得の完了を待たずに画面を可視化する | Accepted |
| [0407](0407-init-on-application-config-loaded.md) | 列ビューの初期化トリガはサイドバーの @inited ではなく ApplicationConfig.is_loaded | Accepted |
| [0408](0408-props-emit-only-no-pinia.md) | フロントエンドの状態管理は Props/Emit と GkillAPI シングルトンのみ | Accepted |
| [0409](0409-context-menu-position-by-vuetify.md) | コンテキストメニューの位置は手計算せず Vuetify の実測配置に任せる | Accepted |
| [0410](0410-bundle-multi-write-operations-in-tx.md) | 複数書き込みになる画面の操作は tx_id で束ねて commit_tx で確定する | Accepted |
| [0411](0411-error-feed-stays-until-closed.md) | エラー表示は1つのフィードに集約し、閉じるまで残す・コピーできる・握られなかった例外も同じ場所へ出す | Accepted |
| [0501](0501-save-marker-beforeinput-input-pair.md) | KFTL保存マーカーの判定は beforeinput→input の対で行う | Accepted |
| [0502](0502-kftl-errors-are-per-line.md) | メモ帳（KFTL）の失敗は行ごとに返し、入力ミスとサーバ障害を分ける | Accepted |
| [0503](0503-kftl-prefix-misuse-is-an-input-error.md) | 引数の無い／引数を同じ行に書いたメモ帳のプレフィックスは、ゼロ値を書かずに行別エラーにする | Accepted |
| [0504](0504-kftl-missing-configuration-is-an-input-error.md) | KFTL の実行フェーズの失敗も、設定不足なら行別の入力エラーにする | Accepted |
| [0505](0505-schedule-time-field-rejects-related-time-prefix.md) | Mi / MiReKyou の予定日時欄では関連時刻の接頭辞「？」を入力エラーにする | Accepted |
| [0506](0506-kftl-repeat-block-expands-into-records.md) | メモ帳の繰り返し「？？」は実体のレコードへ展開し、複製は送信時にだけ作る | Accepted |
| [0507](0507-kftl-single-implementation-on-server.md) | メモ帳の解釈と書き込みはサーバの1実装に寄せ、Web は行ラベルだけを手元で出す | Accepted |
| [0508](0508-kftl-blank-records-are-input-errors.md) | 保存マーカー行は値の行に数えず、内容の無い記録・付け先の無いメタ情報・読めない予定日時は書く前に行別エラーにする | Accepted |
| [0509](0509-kftl-timeis-end-targets-the-latest-running-record.md) | メモ帳の打刻終了は「開始時刻が最新の実行中1件」を終え、削除済みは候補にしない | Accepted |
| [0510](0510-kftl-idempotency-key-carries-fingerprint-and-result.md) | メモ帳の再送キーは本文の指紋と結果を控え、同じ本文は再生し、別の本文は 409 で断る | Accepted |
| [0601](0601-mcp-request-context-immutable.md) | MCP HTTPモードの1リクエスト文脈は不変オブジェクトを引数で流す | Accepted |
| [0602](0602-mcp-inline-plugin-content.md) | MCP のプラグイン本文は get_kyous へインライン埋め込みし、同一プラグインへ並列に投げない | Accepted |
| [0603](0603-mcp-cursor-pushes-period-end.md) | MCP のページングはカーソルを期間上限へ押し下げ、同一時刻のかたまりを割らない | Superseded |
| [0604](0604-mcp-composite-cursor-strict-limits.md) | MCPのページングは複合カーソル（時刻+ID）にし、Limit/MaxSizeMBを厳密な上限へ戻す | Accepted |
| [0605](0605-mcp-version-history-is-a-dedicated-tool.md) | 1件の版履歴と削除の取り消しは専用ツールで返す（only_latest_data は開けない） | Accepted |
| [0606](0606-idf-file-reaches-ai-through-payload.md) | IDF ファイルの経路は優劣ではなく用途で分かれる — 到達できないツールは消し、画像を見せる base64 は残す | Accepted |
| [0607](0607-attached-data-reps-are-a-separate-list.md) | タグ・テキスト・通知・GPSログの格納先は別枠で返す — Reps へ混ぜない | Accepted |
| [0608](0608-plugin-role-is-emits-kyou-and-provides.md) | プラグインの役割は emits_kyou / provides をそのまま出して表す — capabilities 語彙を新設しない | Accepted |
| [0609](0609-stale-tool-schema-is-warned-only-when-proven.md) | 古いツールスキーマは「証明できるときだけ」警告する | Accepted |
| [0610](0610-mi-projection-depends-on-for-mi.md) | Mi の射影が for_mi に依存することは、潰し込みを外さずに警告と説明で見せる | Accepted |
| [0611](0611-one-table-not-two-forms.md) | 同じ対応表を「表」と「直書き」の2形態で持たない | Accepted |
| [0612](0612-attached-data-follows-kyou-search-rules.md) | 付随データの「その瞬間に走っていたか」は Kyou 検索と同じ規則で判定する | Accepted |
| [0613](0613-add-and-update-share-one-field-table.md) | 追加と更新は1つのフィールド表から作る | Accepted |
| [0614](0614-filter-by-create-app-at-request-level.md) | 「どのアプリが書いたか」の絞り込みは FindQuery ではなく MCP リクエストに置く | Accepted |
| [0615](0615-write-user-comes-from-the-authenticated-session.md) | 書き込みに刻む user は、その要求を認証したセッションから決める | Accepted |
| [0616](0616-update-rejects-an-empty-patch.md) | 更新は「変わる欄が1つも無い」なら書かずに断る | Accepted |
| [0617](0617-oauth-scope-is-one-value-per-server-kind.md) | OAuth の scope はサーバ種別ごとに1値 | Accepted |
| [0618](0618-urlog-outbound-fetch-is-default-on-with-opt-out.md) | urlog の外向き取得は既定で行い、引数で項目別に抑止する | Accepted |
| [0619](0619-mcp-schema-revision-and-stale-tool-list.md) | ツール一覧の世代は gkill_status の schema_revision で見せ、未知の引数名では再接続を案内し、tools/list のバイト量を予算で固定する | Accepted |
| [0620](0620-advertised-schema-omits-deprecated-arguments.md) | 廃止済み引数は公開スキーマに載せず、受理と古さの検出だけ残す | Accepted |
| [0621](0621-cursor-pages-revalidate-mi-against-original-window.md) | カーソル頁では Mi の代表射影を元の期間で再検証する（押し下げで代表が変わり、返却済みの記録が再出現していた） | Accepted |
| [0622](0622-tool-descriptions-are-summaries-details-via-help-tool.md) | ツール説明は要約にとどめ、詳細は gkill_get_mcp_help で取り出す | Accepted |
| [0623](0623-data-types-expands-entity-names-to-projections.md) | data_types はエンティティ名を射影へ展開して受理する | Accepted |
| [0624](0624-mcp-max-size-is-enforced-after-plugin-inline.md) | max_size_mb はプラグイン本文を埋め込んだ後に Node が守り直す | Accepted |
| [0625](0625-mcp-map-filter-requires-all-three-values.md) | 地図条件は MCP 入口で3値を要求し、Go は欠けを警告する | Accepted |
| [0626](0626-mcp-kyou-history-pages-with-offset.md) | gkill_get_kyou_history は offset で続きを読む | Accepted |
| [0627](0627-mcp-for-mi-defaults-include-create-mi.md) | for_mi だけの検索は MCP の入口で include_create_mi を補う | Accepted |
| [0628](0628-mcp-update-rejects-a-no-op-patch-by-value.md) | 更新は現在値と同じ値だけのパッチも書かずに断る | Accepted |
| [0629](0629-mcp-trims-per-entry-overhead.md) | 応答の定常オーバーヘッドを削る（付随 ID はオプトイン・is_deleted は true のときだけ・重複欄を落とす・設定ツリーを圧縮） | Accepted |
| [0630](0630-mcp-file-urls-are-opt-in-with-expiry.md) | 公開ファイルURLは include_file_urls で頼まれたときだけ発行し、期限を添える | Accepted |
| [0631](0631-mcp-lives-in-gkill-server.md) | MCP サーバは gkill_server のサブコマンドとして Go で手書きし、旧 Node 実装とはゴールデンでバイト一致させる | Accepted |
| [0632](0632-config-tree-descriptions-for-mcp.md) | 設定ツリーの全ノードに利用者が書く description を持たせ、MCP は fields:["descriptions"] の平坦な一覧で先に読ませる | Accepted |
| [0633](0633-oauth-redirect-uri-rejects-script-capable-schemes.md) | OAuth の redirect_uri は javascript: などの「移動でスクリプトが走る」scheme を登録でも認可でも拒む | Accepted |
| [0701](0701-argon2id-password-storage.md) | パスワードは Argon2id で保存し、ワイヤ形式（password_sha256）は変えない | Accepted |
| [0702](0702-share-owner-from-session.md) | 共有情報の所有者はリクエスト本文ではなくセッションから決める | Accepted |
| [0703](0703-shared-file-authz-by-query.md) | 共有ページのファイル配信は共有クエリを再評価した許可パス集合にだけ許す | Accepted |
| [0704](0704-safefetch-for-user-urls.md) | 利用者入力URLと og:image の取得は必ず api/safefetch を通す | Accepted |
| [0705](0705-per-user-derived-cache-dir.md) | 派生キャッシュは利用者IDでディレクトリを分ける | Accepted |
| [0706](0706-http-status-from-error-code.md) | HTTP ステータスはエラーコードから一元表で決める | Accepted |
| [0707](0707-redact-environment-specific-strings.md) | 端末固有の文字列は出口で伏せ、プラグインの診断文はAIへ返さない | Accepted |
| [0708](0708-local-only-listen-by-default.md) | 待受の既定はループバック限定 — LAN 公開は設定画面での明示操作にし、既存の設定は移行も拒否もしない | Accepted |
| [0709](0709-api-route-table-single-source.md) | HTTP API のルート表は Go 側の1つの表を正本にし、OpenAPI からの生成は採らない | Accepted |
| [0710](0710-error-kind-and-reason-on-the-wire.md) | `errors` / `messages` は成功時も `[]`、エラーには機械語の `error_kind` と `reason` を載せる | Accepted |
| [0801](0801-perf-judge-by-allocs-not-ns-op.md) | 性能判断は ns/op ではなく allocs/op・B/op・EXPLAIN QUERY PLAN で行う | Accepted |
| [0802](0802-freeze-plaing-spelling.md) | 綴りは「永続に乗るか」で決める — plaing は凍結、agregate は改名して読み込み互換を残す | Superseded |
| [0803](0803-verify-docs-checks-filenames.md) | 資料は件数だけでなく「資料に載っているファイル名の実在」も機械検査する | Accepted |
| [0804](0804-split-claude-md-into-skills.md) | AI向け規約は AGENTS.md（核）と規約スキルへ分割し、入口の肥大化を機械検査で防ぐ | Accepted |
| [0805](0805-adr-numbering-by-subsystem-hundreds.md) | ADR の採番はサブシステム別100番幅にし、帯の空きを機械検査する | Accepted |
| [0806](0806-fix-spellings-instead-of-freezing.md) | 綴りは凍結せず直す — 互換を残さず、旧綴りのデータは一度きりで復旧する | Accepted |

### 0900番台 ビルド・テスト・CI

| 番号 | 決定 | Status |
|---|---|---|
| [0901](0901-release-requires-tested-attestation.md) | リリースはテスト済み attestation と CI / Nightly の緑を機械で要求する（人の記憶に頼らない） | Accepted |

### 1000番台 運用CLI・配布

| 番号 | 決定 | Status |
|---|---|---|
| [1001](1001-log-level-by-severity.md) | ログレベルは事象の重さで決める。既定でエラーだけは必ず残す | Accepted |
| [1002](1002-add-tag-rules-are-find-queries.md) | add_tag のルールは検索条件 JSON そのもの。接頭辞は画面の「記録種別」で表し、曖昧な指定はサーバに触る前に落とす | Accepted |
| [1003](1003-generate-plugin-cache-runs-plugin-standalone.md) | generate_plugin_cache はプラグインを単独モードで直接起動して同期構築する。稼働中サーバにも stdio プロトコルにも足さない | Accepted |
| [1101](1101-wear-mood-goes-through-kftl-text.md) | ウォッチの気分記録は専用メッセージパスを足さず、KFTL テキストを既存の送信経路へ流す | Accepted |
| [1102](1102-wear-ui-strings-in-android-resources-with-ja-default.md) | Wear OS の UI 文字列は Android リソースの7言語セット（既定 ja）で持ち、時計へ渡すエラー文言はスマホ側で訳す | Accepted |
| [1103](1103-wear-app-chips-match-tile.md) | ウォッチアプリのチップはタイルと同じ見た目にし、確認画面は ScalingLazyColumn で収める | Accepted |

## 旧→新 対応表

2026-08-30 の全面再採番（[ADR-0805](0805-adr-numbering-by-subsystem-hundreds.md)）の前後の対応。
過去のコミットメッセージ・レビュー記録・外部のメモが旧番号で書かれているので、ここから引き直す。
slug は変えていないので、新番号から現物へは上の索引で辿れる。

- **0100番台 検索とフィルタの意味論** — 0001→0101, 0002→0102, 0003→0103, 0004→0104, 0005→0105, 0006→0106, 0007→0107, 0009→0108, 0070→0109, 0071→0110, 0072→0111, 0073→0112
- **0200番台 DAO・キャッシュ・並行処理** — 0010→0201, 0011→0202, 0012→0203, 0013→0204, 0014→0205, 0015→0206, 0016→0207, 0017→0208, 0018→0209, 0019→0210, 0100→0211, 0101→0212, 0102→0213, 0104→0214
- **0300番台 プラグイン** — 0020→0301, 0021→0302, 0022→0303, 0023→0304, 0024→0305, 0025→0306, 0103→0307
- **0400番台 クライアント** — 0030→0401, 0031→0402, 0032→0403, 0033→0404, 0034→0405, 0035→0406, 0036→0407, 0038→0408, 0039→0409
- **0500番台 メモ帳（KFTL）** — 0037→0501, 0080→0502, 0081→0503, 0082→0504
- **0600番台 MCP** — 0050→0601, 0051→0602, 0052→0603, 0053→0604, 0054→0605, 0055→0606, 0056→0607, 0057→0608, 0058→0609, 0059→0610, 0063→0611, 0064→0612, 0065→0613, 0074→0614, 0090→0615, 0091→0616
- **0700番台 セキュリティ・HTTP契約** — 0040→0701, 0041→0702, 0042→0703, 0043→0704, 0044→0705, 0045→0706, 0046→0707
- **0800番台 開発規約と資料** — 0008→0801, 0060→0802, 0061→0803, 0062→0804

このうち7本は、番号だけでなく**属する帯そのものが変わっている**。
当時その帯が満杯で、空いている別の帯へ逃がされていたぶんである。

| 旧 | 新 | 旧の帯 | 新の帯 |
|---|---|---|---|
| 0008 | 0801 | 検索とフィルタの意味論 | 開発規約と資料 |
| 0037 | 0501 | クライアント | メモ帳（KFTL） |
| 0063 | 0611 | 開発規約と資料 | MCP |
| 0064 | 0612 | 開発規約と資料 | MCP |
| 0065 | 0613 | 開発規約と資料 | MCP |
| 0074 | 0614 | 検索とフィルタの意味論 | MCP |
| 0103 | 0307 | DAO・キャッシュ | プラグイン |

あわせて帯の名前も2つ広げた。`0206`（threads.Go の入れ子禁止）を抱える帯を
「DAO・キャッシュ・SQLite」から「DAO・キャッシュ・並行処理」へ、`0706`（HTTP ステータスの一元表）を
抱える帯を「セキュリティ」から「セキュリティ・HTTP契約」へ。番号は動かしていない。
