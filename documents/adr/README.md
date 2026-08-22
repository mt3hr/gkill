# gkill Architecture Decision Record（ADR）

## この資料集の位置づけ

gkill には「現在どうなっているか（What）」の資料が `documents/reverse/` に24本ある。
ここに置くのは「**なぜそうなっているか（Why）**」——とくに **採らなかった案とその理由** である。

> Reverse docs = What ／ ADR = Why

狙いは1つ。**決定の理由を知らない者が、一見もっともらしい退行を「改善」として提案するのを止めること。**

> 「もっと高速にできそうなので SQL 側に rep 名の条件を入れました！」

このような提案は、実際には11個のキャッシュrepを約940個の生repに化けさせ、検索の実質CPUの6割を
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
10番刻みで採番する。刻んであるのは、後から同じ帯へ差し込めるようにするため。

| 帯 | 範囲 |
|---|---|
| 0001-0009 | 検索とフィルタの意味論 |
| 0010-0019 | DAO・キャッシュ・SQLite |
| 0020-0029 | プラグイン |
| 0030-0039 | クライアント（列・入力・状態管理） |
| 0040-0049 | セキュリティ |
| 0050-0059 | MCP |
| 0060-0069 | 開発規約と資料 |

**番号は採番後不変。** 撤回した決定も番号を空けず `Superseded` で残す
（「入れて翌日撤去した」という記録そのものが最良の ADR になる）。

**ファイル名の slug は改名しない。** slug は採番時点の決定内容であって現行規則ではない。
改名すると索引・リンク・アンカーコメント・規約スキル（`.claude/skills/`）のリンクを一斉に壊すので、
決定が変わったときは Superseded で新しい番号を採る。

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
| [0001](0001-filter-rep-after-cache.md) | rep名の絞り込みは「検索するrep」ではなく「検索結果」で行う | Accepted |
| [0002](0002-no-rep-name-in-sql.md) | SQL の WHERE へ rep名の条件を押し込まない（暫定的な否決） | Accepted |
| [0003](0003-tag-filter-threshold-32.md) | タグ絞り込みの取得は2経路を持ち、切り替えはタグ名の個数（閾値32）で決める | Accepted |
| [0004](0004-related-tag-ids-only-for-no-tags.md) | 全タグ走査（RelatedTagIDs）は「タグ無し」仮想タグを使う検索のときだけ走らせる | Accepted |
| [0005](0005-chunk-find-query-ids.md) | FindQuery.IDs は分割して渡し、検索が失敗したのに GkillError が空のまま return しない | Accepted |
| [0006](0006-find-query-null-semantics.md) | FindQuery の use_* 有効化フラグを全廃し「値が非nullならフィルタ有効」に一本化する | Accepted |
| [0007](0007-memoize-rekyou-target-resolution.md) | ReKyou / MiReKyou のターゲット解決はリクエスト単位でメモ化する | Accepted |
| [0008](0008-perf-judge-by-allocs-not-ns-op.md) | 性能判断は ns/op ではなく allocs/op・B/op・EXPLAIN QUERY PLAN で行う | Accepted |
| [0010](0010-append-only-dao.md) | Append-Only DAO — ID 列に主キー制約を置かず、更新も削除も INSERT で表現する | Accepted |
| [0011](0011-rebuild-cache-only-on-db-change.md) | キャッシュのフルリビルドは実DBが変わったときだけ | Accepted |
| [0012](0012-write-through-cache-not-reps-count.md) | 書き込み後のキャッシュ反映は要素数ではなく構築時に控えた CachedReps で判定する | Accepted |
| [0013](0013-keep-journal-mode-delete.md) | 実データDBの journal_mode は DELETE のまま変えない | Accepted |
| [0014](0014-unixepoch-expression-index.md) | 時刻列は unixepoch の式インデックスで引く | Accepted |
| [0015](0015-no-nested-threads-go.md) | threads.Go の入れ子は禁止。集約リポジトリには逐次版を用意する | Accepted |
| [0016](0016-exclude-urlog-thumbnail-from-cache.md) | URLog のサムネイルはインメモリキャッシュに載せない | Accepted |
| [0017](0017-git-repo-detect-by-os-stat.md) | gitリポジトリ判定は PlainOpen のエラー型ではなく os.Stat で行う | Accepted |
| [0020](0020-plugin-cancel-vs-kill.md) | プラグインの打ち切りは「待つのをやめる」と「プロセスを殺す」を分け、期限はスロットを取ってから張る | Accepted |
| [0021](0021-plugin-provides-typed-index.md) | プラグインは provides で型別/付随データを提供でき、アダプタの読み取りは索引から即答する | Accepted |
| [0022](0022-plugin-cache-use-crc32-and-size.md) | Google Takeout は ZIP のまま読み、差分判定は (CRC32, Size)、世代は「フォルダ + 書き出し時刻」 | Accepted |
| [0023](0023-plugin-emits-kyou-false.md) | 記録を返さないプラグインは emits_kyou: false で明示する | Accepted |
| [0024](0024-plugin-background-builder-wal.md) | プラグインの重い構築は常駐ビルダ + WAL + バッチcommit | Accepted |
| [0025](0025-codex-thread-id-from-filename.md) | Codex ロールアウトログのスレッド識別子はファイル名の uuid、会話は event_msg レーンだけ | Accepted |
| [0030](0030-do-not-split-search-window-in-client.md) | クライアント側で検索を期間の窓へ刻んで複数回 get_kyous を投げない | Accepted |
| [0031](0031-insert-registered-kyou-locally.md) | 記録の追加は列を再検索せず、その1件をクライアントで判定して差し込む | Accepted |
| [0032](0032-add-tag-before-registered-kyou.md) | タグ欄付きの追加/編集画面は add_tag が完了してから registered_kyou を emit する | Accepted |
| [0033](0033-add-unknown-tag-to-column-filter.md) | 利用者がその場で作ったタグだけを、開いている列の検索条件へ足す | Accepted |
| [0034](0034-column-identity-query-id.md) | 列の同一性は query_id、検索は世代番号で最後の1回だけ書き戻す | Accepted |
| [0035](0035-visualize-before-initial-search.md) | rykv/mi/dashboard は初期取得の完了を待たずに画面を可視化する | Accepted |
| [0036](0036-init-on-application-config-loaded.md) | 列ビューの初期化トリガはサイドバーの @inited ではなく ApplicationConfig.is_loaded | Accepted |
| [0037](0037-save-marker-beforeinput-input-pair.md) | KFTL保存マーカーの判定は beforeinput→input の対で行う | Accepted |
| [0038](0038-props-emit-only-no-pinia.md) | フロントエンドの状態管理は Props/Emit と GkillAPI シングルトンのみ | Accepted |
| [0039](0039-context-menu-position-by-vuetify.md) | コンテキストメニューの位置は手計算せず Vuetify の実測配置に任せる | Accepted |
| [0040](0040-argon2id-password-storage.md) | パスワードは Argon2id で保存し、ワイヤ形式（password_sha256）は変えない | Accepted |
| [0041](0041-share-owner-from-session.md) | 共有情報の所有者はリクエスト本文ではなくセッションから決める | Accepted |
| [0042](0042-shared-file-authz-by-query.md) | 共有ページのファイル配信は共有クエリを再評価した許可パス集合にだけ許す | Accepted |
| [0043](0043-safefetch-for-user-urls.md) | 利用者入力URLと og:image の取得は必ず api/safefetch を通す | Accepted |
| [0044](0044-per-user-derived-cache-dir.md) | 派生キャッシュは利用者IDでディレクトリを分ける | Accepted |
| [0050](0050-mcp-request-context-immutable.md) | MCP HTTPモードの1リクエスト文脈は不変オブジェクトを引数で流す | Accepted |
| [0051](0051-mcp-inline-plugin-content.md) | MCP のプラグイン本文は get_kyous へインライン埋め込みし、同一プラグインへ並列に投げない | Accepted |
| [0052](0052-mcp-cursor-pushes-period-end.md) | MCP のページングはカーソルを期間上限へ押し下げ、同一時刻のかたまりを割らない | Accepted |
| [0060](0060-freeze-plaing-spelling.md) | 綴りは「永続に乗るか」で決める — plaing は凍結、agregate は改名して読み込み互換を残す | Accepted |
| [0061](0061-verify-docs-checks-filenames.md) | 資料は件数だけでなく「資料に載っているファイル名の実在」も機械検査する | Accepted |
| [0062](0062-split-claude-md-into-skills.md) | AI向け規約は AGENTS.md（核）と規約スキルへ分割し、入口の肥大化を機械検査で防ぐ | Accepted |
